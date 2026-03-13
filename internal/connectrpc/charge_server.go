package connectrpc

import (
	"context"
	"fmt"
	"time"

	"connectrpc.com/connect"
	"github.com/brady1408/auto-transport-logistics/internal/audit"
	pb "github.com/brady1408/auto-transport-logistics/internal/gen/atlinks/v1"
	"github.com/brady1408/auto-transport-logistics/internal/gen/atlinks/v1/atlinkspbconnect"
	"github.com/brady1408/auto-transport-logistics/internal/models"
)

type chargeStore interface {
	ListByOrder(ctx context.Context, orderID int) ([]models.OrderCharge, error)
	GetByID(ctx context.Context, id int) (*models.OrderCharge, error)
	Create(ctx context.Context, c *models.OrderCharge) error
	Update(ctx context.Context, c *models.OrderCharge) error
	Delete(ctx context.Context, id int) error
}

type ChargeServer struct {
	atlinkspbconnect.UnimplementedChargeServiceHandler
	store chargeStore
	audit *audit.Service
}

func NewChargeServer(store chargeStore, audit *audit.Service) *ChargeServer {
	return &ChargeServer{store: store, audit: audit}
}

func (s *ChargeServer) ListCharges(ctx context.Context, req *connect.Request[pb.ListChargesRequest]) (*connect.Response[pb.ListChargesResponse], error) {
	items, err := s.store.ListByOrder(ctx, int(req.Msg.OrderId))
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("list charges: %w", err))
	}

	charges := make([]*pb.OrderCharge, len(items))
	for i := range items {
		charges[i] = chargeToProto(&items[i])
	}

	return connect.NewResponse(&pb.ListChargesResponse{Charges: charges}), nil
}

func (s *ChargeServer) GetCharge(ctx context.Context, req *connect.Request[pb.GetChargeRequest]) (*connect.Response[pb.GetChargeResponse], error) {
	c, err := s.store.GetByID(ctx, int(req.Msg.Id))
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("charge %d not found", req.Msg.Id))
	}
	return connect.NewResponse(&pb.GetChargeResponse{Charge: chargeToProto(c)}), nil
}

func (s *ChargeServer) CreateCharge(ctx context.Context, req *connect.Request[pb.CreateChargeRequest]) (*connect.Response[pb.CreateChargeResponse], error) {
	if req.Msg.OrderId == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("order_id is required"))
	}

	c := createChargeReqToModel(req.Msg)
	if err := s.store.Create(ctx, c); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("create charge: %w", err))
	}

	s.audit.Log(ctx, "order_charges", c.ID, "INSERT", nil, c)

	return connect.NewResponse(&pb.CreateChargeResponse{Charge: chargeToProto(c)}), nil
}

func (s *ChargeServer) UpdateCharge(ctx context.Context, req *connect.Request[pb.UpdateChargeRequest]) (*connect.Response[pb.UpdateChargeResponse], error) {
	old, err := s.store.GetByID(ctx, int(req.Msg.Id))
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("charge %d not found", req.Msg.Id))
	}

	c := updateChargeReqToModel(req.Msg)
	if err := s.store.Update(ctx, c); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("update charge: %w", err))
	}

	s.audit.Log(ctx, "order_charges", c.ID, "UPDATE", old, c)

	updated, err := s.store.GetByID(ctx, c.ID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("fetch updated charge: %w", err))
	}
	return connect.NewResponse(&pb.UpdateChargeResponse{Charge: chargeToProto(updated)}), nil
}

func (s *ChargeServer) DeleteCharge(ctx context.Context, req *connect.Request[pb.DeleteChargeRequest]) (*connect.Response[pb.DeleteChargeResponse], error) {
	if err := s.store.Delete(ctx, int(req.Msg.Id)); err != nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("charge %d not found", req.Msg.Id))
	}
	s.audit.Log(ctx, "order_charges", int(req.Msg.Id), "DELETE", nil, nil)
	return connect.NewResponse(&pb.DeleteChargeResponse{Success: true}), nil
}

func chargeToProto(c *models.OrderCharge) *pb.OrderCharge {
	var orderID int32
	if c.OrderID != nil {
		orderID = int32(*c.OrderID)
	}
	return &pb.OrderCharge{
		Id:          int32(c.ID),
		OrderId:     orderID,
		VehicleId:   ip(c.VehicleID),
		TripId:      ip(c.TripID),
		Description: sp(c.Description),
		Amount:      sp(c.Amount),
		ItemCode:    sp(c.ItemCode),
		Qty:         ip(c.Qty),
		Rate:        sp(c.Rate),
		CalcType:    sp(c.CalcType),
		Taxable:     c.Taxable,
		Billable:    c.Billable,
		ApPayable:   c.APPayable,
		CreatedAt:   c.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   c.UpdatedAt.Format(time.RFC3339),
	}
}

func createChargeReqToModel(msg *pb.CreateChargeRequest) *models.OrderCharge {
	orderID := int(msg.OrderId)
	return &models.OrderCharge{
		OrderID:     &orderID,
		VehicleID:   i32p(msg.VehicleId),
		TripID:      i32p(msg.TripId),
		Description: sp(msg.Description),
		Amount:      sp(msg.Amount),
		ItemCode:    sp(msg.ItemCode),
		Qty:         i32p(msg.Qty),
		Rate:        sp(msg.Rate),
		CalcType:    sp(msg.CalcType),
		Taxable:     msg.Taxable,
		Billable:    msg.Billable,
		APPayable:   msg.ApPayable,
	}
}

func updateChargeReqToModel(msg *pb.UpdateChargeRequest) *models.OrderCharge {
	orderID := int(msg.OrderId)
	return &models.OrderCharge{
		ID:          int(msg.Id),
		OrderID:     &orderID,
		VehicleID:   i32p(msg.VehicleId),
		TripID:      i32p(msg.TripId),
		Description: sp(msg.Description),
		Amount:      sp(msg.Amount),
		ItemCode:    sp(msg.ItemCode),
		Qty:         i32p(msg.Qty),
		Rate:        sp(msg.Rate),
		CalcType:    sp(msg.CalcType),
		Taxable:     msg.Taxable,
		Billable:    msg.Billable,
		APPayable:   msg.ApPayable,
	}
}
