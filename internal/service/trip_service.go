package service

import (
	"context"
	"fmt"
	"time"

	"github.com/brady1408/auto-transport-logistics/internal/audit"
	"github.com/brady1408/auto-transport-logistics/internal/models"
	"github.com/brady1408/auto-transport-logistics/internal/store"
	"github.com/jackc/pgx/v5/pgxpool"
)

type TripService struct {
	pool         *pgxpool.Pool
	tripStore    *store.TripStore
	loadStore    *store.LoadDetailStore
	vehicleStore *store.VehicleStore
	orderStore   *store.OrderStore
	audit        *audit.Service
}

func NewTripService(
	pool *pgxpool.Pool,
	tripStore *store.TripStore,
	loadStore *store.LoadDetailStore,
	vehicleStore *store.VehicleStore,
	orderStore *store.OrderStore,
	audit *audit.Service,
) *TripService {
	return &TripService{
		pool:         pool,
		tripStore:    tripStore,
		loadStore:    loadStore,
		vehicleStore: vehicleStore,
		orderStore:   orderStore,
		audit:        audit,
	}
}

// AssignVehicleToTrip creates a load_detail row and updates the vehicle's trip reference.
// If bayNumber is empty, auto-computes the next bay number.
func (s *TripService) AssignVehicleToTrip(ctx context.Context, tripID, vehicleID int, bayNumber string) error {
	// Auto-compute bay number if not provided
	if bayNumber == "" {
		next, err := s.loadStore.NextBayNumber(ctx, tripID)
		if err == nil {
			bayNumber = next
		}
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	// 1. Get vehicle (for denormalized fields)
	v, err := s.vehicleStore.GetByIDTx(ctx, tx, vehicleID)
	if err != nil {
		return fmt.Errorf("get vehicle: %w", err)
	}

	// 2. Get trip (for load_number) — inside tx for consistency
	trip, err := s.tripStore.GetByIDTx(ctx, tx, tripID)
	if err != nil {
		return fmt.Errorf("get trip: %w", err)
	}

	// 3. Create load detail with denormalized vehicle info
	bayNum := &bayNumber
	if bayNumber == "" {
		bayNum = nil
	}
	status := "Scheduled"
	ld := &models.LoadDetail{
		TripID:    tripID,
		OrderID:   &v.OrderID,
		VehicleID: &v.ID,
		VIN:       v.VIN,
		Year:      v.Year,
		Make:      v.Make,
		Model:     v.Model,
		Color:     v.Color,
		Weight:    v.Weight,
		Category:  v.Category,
		BayNumber: bayNum,
		Status:    &status,
	}
	if err := s.loadStore.CreateTx(ctx, tx, ld); err != nil {
		return err
	}

	// 4. Update vehicle: set trip_id, load_number, bay_number, status → Scheduled
	loadNum := &trip.LoadNumber
	if err := s.vehicleStore.UpdateTripAssignmentTx(ctx, tx, vehicleID, &tripID, loadNum, bayNum, "Scheduled"); err != nil {
		return err
	}

	// 5. Set scheduled_date on vehicle
	now := time.Now()
	if err := s.vehicleStore.UpdateStatusTx(ctx, tx, vehicleID, "Scheduled", "scheduled_date", &now); err != nil {
		return fmt.Errorf("set scheduled date: %w", err)
	}

	// 6. Sync order counts
	counts, err := s.vehicleStore.CountByOrderTx(ctx, tx, v.OrderID)
	if err != nil {
		return err
	}
	if err := s.orderStore.UpdateCounts(ctx, tx, v.OrderID, counts); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

// UnassignVehicle removes a vehicle from a trip and reverts its status.
func (s *TripService) UnassignVehicle(ctx context.Context, loadDetailID int) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	// 1. Get load detail
	ld, err := s.loadStore.GetByID(ctx, loadDetailID)
	if err != nil {
		return fmt.Errorf("get load detail: %w", err)
	}

	// 2. Delete load detail
	if err := s.loadStore.DeleteTx(ctx, tx, loadDetailID); err != nil {
		return err
	}

	// 3. Revert vehicle: clear trip_id, load_number, bay_number, status → Waiting
	if ld.VehicleID != nil {
		if err := s.vehicleStore.UpdateTripAssignmentTx(ctx, tx, *ld.VehicleID, nil, nil, nil, "Waiting"); err != nil {
			return err
		}

		// 4. Sync order counts
		if ld.OrderID != nil {
			counts, err := s.vehicleStore.CountByOrderTx(ctx, tx, *ld.OrderID)
			if err != nil {
				return err
			}
			if err := s.orderStore.UpdateCounts(ctx, tx, *ld.OrderID, counts); err != nil {
				return err
			}
		}
	}

	return tx.Commit(ctx)
}

// AssignAllFromOrder assigns all waiting, unassigned vehicles from an order to a trip.
// Returns the count of vehicles assigned.
func (s *TripService) AssignAllFromOrder(ctx context.Context, tripID, orderID int) (int, error) {
	// Get all waiting unassigned vehicles for this order
	vehicles, err := s.vehicleStore.ListUnassignedByOrder(ctx, orderID)
	if err != nil {
		return 0, fmt.Errorf("list unassigned vehicles: %w", err)
	}
	if len(vehicles) == 0 {
		return 0, nil
	}

	// Get next bay number start
	bayStr, _ := s.loadStore.NextBayNumber(ctx, tripID)
	bayStart := 1
	fmt.Sscanf(bayStr, "%d", &bayStart)

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	trip, err := s.tripStore.GetByIDTx(ctx, tx, tripID)
	if err != nil {
		return 0, fmt.Errorf("get trip: %w", err)
	}

	for i, v := range vehicles {
		bay := fmt.Sprintf("%d", bayStart+i)
		status := "Scheduled"
		ld := &models.LoadDetail{
			TripID:    tripID,
			OrderID:   &v.OrderID,
			VehicleID: &v.ID,
			VIN:       v.VIN,
			Year:      v.Year,
			Make:      v.Make,
			Model:     v.Model,
			Color:     v.Color,
			Weight:    v.Weight,
			Category:  v.Category,
			BayNumber: &bay,
			Status:    &status,
		}
		if err := s.loadStore.CreateTx(ctx, tx, ld); err != nil {
			return 0, err
		}

		loadNum := &trip.LoadNumber
		if err := s.vehicleStore.UpdateTripAssignmentTx(ctx, tx, v.ID, &tripID, loadNum, &bay, "Scheduled"); err != nil {
			return 0, err
		}
	}

	// Sync order counts once
	counts, err := s.vehicleStore.CountByOrderTx(ctx, tx, orderID)
	if err != nil {
		return 0, err
	}
	if err := s.orderStore.UpdateCounts(ctx, tx, orderID, counts); err != nil {
		return 0, err
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("commit: %w", err)
	}
	return len(vehicles), nil
}
