package service

import (
	"context"
	"fmt"
	"time"

	"github.com/brady1408/atlinks/internal/audit"
	"github.com/brady1408/atlinks/internal/models"
	"github.com/brady1408/atlinks/internal/store"
	"github.com/jackc/pgx/v5/pgxpool"
)

type OrderService struct {
	pool         *pgxpool.Pool
	orderStore   *store.OrderStore
	vehicleStore *store.VehicleStore
	audit        *audit.Service
}

func NewOrderService(pool *pgxpool.Pool, orderStore *store.OrderStore, vehicleStore *store.VehicleStore, audit *audit.Service) *OrderService {
	return &OrderService{
		pool:         pool,
		orderStore:   orderStore,
		vehicleStore: vehicleStore,
		audit:        audit,
	}
}

// Valid status transitions: forward and revert
var validTransitions = map[string][]string{
	"Waiting":   {"Scheduled"},
	"Scheduled": {"Loaded", "Waiting"},
	"Loaded":    {"Delivered", "Scheduled"},
	"Delivered":  {"Confirmed", "Loaded"},
	"Confirmed": {"Delivered"},
}

// statusDateColumn maps status to the date column to set
var statusDateColumn = map[string]string{
	"Scheduled": "scheduled_date",
	"Loaded":    "loaded_date",
	"Delivered": "delivered_date",
	"Confirmed": "confirmed_date",
}

// UpdateVehicleStatus transitions a vehicle's status and syncs order counts atomically.
func (s *OrderService) UpdateVehicleStatus(ctx context.Context, vehicleID int, newStatus string, confirmedBy *string) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	// 1. Get current vehicle
	v, err := s.vehicleStore.GetByIDTx(ctx, tx, vehicleID)
	if err != nil {
		return fmt.Errorf("get vehicle: %w", err)
	}

	// 2. Validate transition
	allowed := validTransitions[v.Status]
	valid := false
	for _, s := range allowed {
		if s == newStatus {
			valid = true
			break
		}
	}
	if !valid {
		return fmt.Errorf("invalid status transition: %s -> %s", v.Status, newStatus)
	}

	// 3. Determine date column and value
	dateCol, ok := statusDateColumn[newStatus]
	if !ok {
		// Reverting to a previous status — clear the date of the status we're leaving
		dateCol = statusDateColumn[v.Status]
	}

	var dateVal any
	if isForwardTransition(v.Status, newStatus) {
		now := time.Now()
		dateVal = &now
	} else {
		// Revert: clear the date
		dateVal = nil
	}

	// 4. Update vehicle status + date
	if err := s.vehicleStore.UpdateStatusTx(ctx, tx, vehicleID, newStatus, dateCol, dateVal); err != nil {
		return err
	}

	// For Confirmed status, also set confirmed_by
	if newStatus == "Confirmed" && confirmedBy != nil {
		_, err := tx.Exec(ctx, "UPDATE order_vehicles SET confirmed_by=$1 WHERE id=$2", confirmedBy, vehicleID)
		if err != nil {
			return fmt.Errorf("set confirmed_by: %w", err)
		}
	}

	// When reverting FROM Confirmed, clear confirmed_by
	if v.Status == "Confirmed" && newStatus != "Confirmed" {
		_, err := tx.Exec(ctx, "UPDATE order_vehicles SET confirmed_by=NULL WHERE id=$1", vehicleID)
		if err != nil {
			return fmt.Errorf("clear confirmed_by: %w", err)
		}
	}

	// 5. Recalculate order counts
	counts, err := s.vehicleStore.CountByOrderTx(ctx, tx, v.OrderID)
	if err != nil {
		return err
	}

	// 6. Update order counts
	if err := s.orderStore.UpdateCounts(ctx, tx, v.OrderID, counts); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}

	return nil
}

// SyncOrderCounts recalculates and updates the count fields on an order.
func (s *OrderService) SyncOrderCounts(ctx context.Context, orderID int) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	counts, err := s.vehicleStore.CountByOrderTx(ctx, tx, orderID)
	if err != nil {
		return err
	}

	if err := s.orderStore.UpdateCounts(ctx, tx, orderID, counts); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

// isForwardTransition returns true if this is a forward status progression.
func isForwardTransition(from, to string) bool {
	order := map[string]int{
		"Waiting":   0,
		"Scheduled": 1,
		"Loaded":    2,
		"Delivered": 3,
		"Confirmed": 4,
	}
	return order[to] > order[from]
}

// RevertVehicleStatus reverts a vehicle to its previous status.
func (s *OrderService) RevertVehicleStatus(ctx context.Context, vehicleID int) error {
	v, err := s.vehicleStore.GetByID(ctx, vehicleID)
	if err != nil {
		return err
	}

	var prevStatus string
	switch v.Status {
	case "Scheduled":
		prevStatus = "Waiting"
	case "Loaded":
		prevStatus = "Scheduled"
	case "Delivered":
		prevStatus = "Loaded"
	case "Confirmed":
		prevStatus = "Delivered"
	default:
		return fmt.Errorf("cannot revert from status: %s", v.Status)
	}

	return s.UpdateVehicleStatus(ctx, vehicleID, prevStatus, nil)
}

// CreateVehicleAndSync creates a vehicle and syncs order counts.
func (s *OrderService) CreateVehicleAndSync(ctx context.Context, v *models.OrderVehicle) error {
	if err := s.vehicleStore.Create(ctx, v); err != nil {
		return err
	}
	return s.SyncOrderCounts(ctx, v.OrderID)
}

// DeleteVehicleAndSync deletes a vehicle and syncs order counts atomically.
func (s *OrderService) DeleteVehicleAndSync(ctx context.Context, vehicleID int) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	v, err := s.vehicleStore.GetByIDTx(ctx, tx, vehicleID)
	if err != nil {
		return fmt.Errorf("get vehicle: %w", err)
	}
	if err := s.vehicleStore.DeleteTx(ctx, tx, vehicleID); err != nil {
		return fmt.Errorf("delete vehicle: %w", err)
	}

	counts, err := s.vehicleStore.CountByOrderTx(ctx, tx, v.OrderID)
	if err != nil {
		return fmt.Errorf("count vehicles: %w", err)
	}
	if err := s.orderStore.UpdateCounts(ctx, tx, v.OrderID, counts); err != nil {
		return fmt.Errorf("update order counts: %w", err)
	}

	return tx.Commit(ctx)
}
