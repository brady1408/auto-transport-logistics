package connectrpc

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	"github.com/brady1408/auto-transport-logistics/internal/audit"
	pb "github.com/brady1408/auto-transport-logistics/internal/gen/atlinks/v1"
	"github.com/brady1408/auto-transport-logistics/internal/gen/atlinks/v1/atlinkspbconnect"
	"github.com/brady1408/auto-transport-logistics/internal/models"
	"github.com/brady1408/auto-transport-logistics/internal/service"
)

type vehicleStore interface {
	ListByOrder(ctx context.Context, orderID int) ([]models.OrderVehicle, error)
	GetByID(ctx context.Context, id int) (*models.OrderVehicle, error)
	Update(ctx context.Context, v *models.OrderVehicle) error
}

type VehicleServer struct {
	atlinkspbconnect.UnimplementedVehicleServiceHandler
	store    vehicleStore
	orderSvc *service.OrderService
	audit    *audit.Service
}

func NewVehicleServer(store vehicleStore, orderSvc *service.OrderService, audit *audit.Service) *VehicleServer {
	return &VehicleServer{store: store, orderSvc: orderSvc, audit: audit}
}

func (s *VehicleServer) ListVehicles(ctx context.Context, req *connect.Request[pb.ListVehiclesRequest]) (*connect.Response[pb.ListVehiclesResponse], error) {
	items, err := s.store.ListByOrder(ctx, int(req.Msg.OrderId))
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("list vehicles: %w", err))
	}

	vehicles := make([]*pb.Vehicle, len(items))
	for i := range items {
		vehicles[i] = vehicleToProto(&items[i])
	}

	return connect.NewResponse(&pb.ListVehiclesResponse{Vehicles: vehicles}), nil
}

func (s *VehicleServer) GetVehicle(ctx context.Context, req *connect.Request[pb.GetVehicleRequest]) (*connect.Response[pb.GetVehicleResponse], error) {
	v, err := s.store.GetByID(ctx, int(req.Msg.Id))
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("vehicle %d not found", req.Msg.Id))
	}
	return connect.NewResponse(&pb.GetVehicleResponse{Vehicle: vehicleToProto(v)}), nil
}

func (s *VehicleServer) CreateVehicle(ctx context.Context, req *connect.Request[pb.CreateVehicleRequest]) (*connect.Response[pb.CreateVehicleResponse], error) {
	if req.Msg.OrderId == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("order_id is required"))
	}

	v := createVehicleReqToModel(req.Msg)
	// Use OrderService to create + sync counts atomically
	if err := s.orderSvc.CreateVehicleAndSync(ctx, v); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("create vehicle: %w", err))
	}

	s.audit.Log(ctx, "order_vehicles", v.ID, "INSERT", nil, v)

	created, err := s.store.GetByID(ctx, v.ID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("fetch created vehicle: %w", err))
	}
	return connect.NewResponse(&pb.CreateVehicleResponse{Vehicle: vehicleToProto(created)}), nil
}

func (s *VehicleServer) UpdateVehicle(ctx context.Context, req *connect.Request[pb.UpdateVehicleRequest]) (*connect.Response[pb.UpdateVehicleResponse], error) {
	old, err := s.store.GetByID(ctx, int(req.Msg.Id))
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("vehicle %d not found", req.Msg.Id))
	}

	v := updateVehicleReqToModel(req.Msg)
	if err := s.store.Update(ctx, v); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("update vehicle: %w", err))
	}

	s.audit.Log(ctx, "order_vehicles", v.ID, "UPDATE", old, v)

	updated, err := s.store.GetByID(ctx, v.ID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("fetch updated vehicle: %w", err))
	}
	return connect.NewResponse(&pb.UpdateVehicleResponse{Vehicle: vehicleToProto(updated)}), nil
}

func (s *VehicleServer) DeleteVehicle(ctx context.Context, req *connect.Request[pb.DeleteVehicleRequest]) (*connect.Response[pb.DeleteVehicleResponse], error) {
	// Use OrderService to delete + sync counts atomically
	if err := s.orderSvc.DeleteVehicleAndSync(ctx, int(req.Msg.Id)); err != nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("vehicle %d not found", req.Msg.Id))
	}
	s.audit.Log(ctx, "order_vehicles", int(req.Msg.Id), "DELETE", nil, nil)
	return connect.NewResponse(&pb.DeleteVehicleResponse{Success: true}), nil
}

func (s *VehicleServer) UpdateVehicleStatus(ctx context.Context, req *connect.Request[pb.UpdateVehicleStatusRequest]) (*connect.Response[pb.UpdateVehicleStatusResponse], error) {
	if req.Msg.Status == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("status is required"))
	}

	if err := s.orderSvc.UpdateVehicleStatus(ctx, int(req.Msg.Id), req.Msg.Status, nil); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	v, err := s.store.GetByID(ctx, int(req.Msg.Id))
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("fetch updated vehicle: %w", err))
	}
	return connect.NewResponse(&pb.UpdateVehicleStatusResponse{Vehicle: vehicleToProto(v)}), nil
}
