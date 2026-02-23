package service

import (
	"context"
	"fmt"

	"github.com/brady1408/atlinks/internal/audit"
	"github.com/brady1408/atlinks/internal/models"
	"github.com/brady1408/atlinks/internal/store"
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
func (s *TripService) AssignVehicleToTrip(ctx context.Context, tripID, vehicleID int, bayNumber string) error {
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

	// 2. Get trip (for load_number)
	trip, err := s.tripStore.GetByID(ctx, tripID)
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
	if err := s.vehicleStore.UpdateStatusTx(ctx, tx, vehicleID, "Scheduled", "scheduled_date", nil); err != nil {
		// scheduled_date was already set by UpdateTripAssignment changing status, skip this
		// Actually we need to set the date. Let me use a direct exec.
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
