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

type truckStore interface {
	List(ctx context.Context, f models.TruckFilter) (*models.TruckListResult, error)
	GetByID(ctx context.Context, id int) (*models.Truck, error)
	Create(ctx context.Context, t *models.Truck) error
	Update(ctx context.Context, t *models.Truck) error
	Delete(ctx context.Context, id int) error
}

type TruckServer struct {
	atlinkspbconnect.UnimplementedTruckServiceHandler
	store truckStore
	audit *audit.Service
}

func NewTruckServer(store truckStore, audit *audit.Service) *TruckServer {
	return &TruckServer{store: store, audit: audit}
}

func (s *TruckServer) ListTrucks(ctx context.Context, req *connect.Request[pb.ListTrucksRequest]) (*connect.Response[pb.ListTrucksResponse], error) {
	filter := protoToTruckFilter(req.Msg)
	result, err := s.store.List(ctx, filter)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("list trucks: %w", err))
	}

	trucks := make([]*pb.Truck, len(result.Items))
	for i := range result.Items {
		trucks[i] = truckToProto(&result.Items[i])
	}

	return connect.NewResponse(&pb.ListTrucksResponse{
		Trucks: trucks,
		Pagination: &pb.PaginationResponse{
			TotalCount: int32(result.TotalCount),
			Page:       int32(result.Page),
			PageSize:   int32(result.PageSize),
		},
	}), nil
}

func (s *TruckServer) GetTruck(ctx context.Context, req *connect.Request[pb.GetTruckRequest]) (*connect.Response[pb.GetTruckResponse], error) {
	t, err := s.store.GetByID(ctx, int(req.Msg.Id))
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("truck %d not found", req.Msg.Id))
	}
	return connect.NewResponse(&pb.GetTruckResponse{Truck: truckToProto(t)}), nil
}

func (s *TruckServer) CreateTruck(ctx context.Context, req *connect.Request[pb.CreateTruckRequest]) (*connect.Response[pb.CreateTruckResponse], error) {
	if req.Msg.TruckNumber == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("truck_number is required"))
	}

	t := createTruckReqToModel(req.Msg)
	if err := s.store.Create(ctx, t); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("create truck: %w", err))
	}

	s.audit.Log(ctx, "trucks", t.ID, "INSERT", nil, t)

	return connect.NewResponse(&pb.CreateTruckResponse{Truck: truckToProto(t)}), nil
}

func (s *TruckServer) UpdateTruck(ctx context.Context, req *connect.Request[pb.UpdateTruckRequest]) (*connect.Response[pb.UpdateTruckResponse], error) {
	if req.Msg.TruckNumber == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("truck_number is required"))
	}

	old, err := s.store.GetByID(ctx, int(req.Msg.Id))
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("truck %d not found", req.Msg.Id))
	}

	t := updateTruckReqToModel(req.Msg)
	if err := s.store.Update(ctx, t); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("update truck: %w", err))
	}

	s.audit.Log(ctx, "trucks", t.ID, "UPDATE", old, t)

	updated, err := s.store.GetByID(ctx, t.ID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("fetch updated truck: %w", err))
	}
	return connect.NewResponse(&pb.UpdateTruckResponse{Truck: truckToProto(updated)}), nil
}

func (s *TruckServer) DeleteTruck(ctx context.Context, req *connect.Request[pb.DeleteTruckRequest]) (*connect.Response[pb.DeleteTruckResponse], error) {
	if err := s.store.Delete(ctx, int(req.Msg.Id)); err != nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("truck %d not found", req.Msg.Id))
	}
	s.audit.Log(ctx, "trucks", int(req.Msg.Id), "DELETE", nil, nil)
	return connect.NewResponse(&pb.DeleteTruckResponse{Success: true}), nil
}

// --- Truck converters ---

func truckToProto(t *models.Truck) *pb.Truck {
	return &pb.Truck{
		Id:                   int32(t.ID),
		TruckNumber:          t.TruckNumber,
		TruckMake:            sp(t.TruckMake),
		TruckModel:           sp(t.TruckModel),
		TruckYear:            sp(t.TruckYear),
		TrailerNumber:        sp(t.TrailerNumber),
		TrailerMake:          sp(t.TrailerMake),
		TrailerModel:         sp(t.TrailerModel),
		TrailerYear:          sp(t.TrailerYear),
		Driver1:              sp(t.Driver1),
		Driver2:              sp(t.Driver2),
		TruckRate:            sp(t.TruckRate),
		TruckCalcType:        sp(t.TruckCalcType),
		Active:               t.Active,
		LeasedTruck:          t.LeasedTruck,
		Class:                sp(t.Class),
		FleetNumber:          sp(t.FleetNumber),
		TruckLicense:         sp(t.TruckLicense),
		TruckLicenseExp:      timeStr(t.TruckLicenseExp),
		InsuranceExpDate:     timeStr(t.InsuranceExpDate),
		InsuranceCoverageAmt: sp(t.InsuranceCoverageAmt),
		CargoCoverageAmt:     sp(t.CargoCoverageAmt),
		TareWeight:           ip(t.TareWeight),
		WePayDriver:          t.WePayDriver,
		Straps:               t.Straps,
		ExcludeFuel:          t.ExcludeFuel,
		CreatedAt:            t.CreatedAt.Format(time.RFC3339),
		UpdatedAt:            t.UpdatedAt.Format(time.RFC3339),
	}
}

func protoToTruckFilter(msg *pb.ListTrucksRequest) models.TruckFilter {
	f := models.TruckFilter{}
	if msg.Pagination != nil {
		f.Page = int(msg.Pagination.Page)
		f.PageSize = int(msg.Pagination.PageSize)
	}
	if msg.Search != nil {
		f.Search = *msg.Search
	}
	if msg.Active != nil {
		f.Active = *msg.Active
	}
	if msg.LeasedTruck != nil {
		f.LeasedTruck = *msg.LeasedTruck
	}
	if msg.Class != nil {
		f.Class = *msg.Class
	}
	return f
}

func createTruckReqToModel(msg *pb.CreateTruckRequest) *models.Truck {
	return &models.Truck{
		TruckNumber:   msg.TruckNumber,
		TruckMake:     sp(msg.TruckMake),
		TruckModel:    sp(msg.TruckModel),
		TruckYear:     sp(msg.TruckYear),
		TrailerNumber: sp(msg.TrailerNumber),
		Driver1:       sp(msg.Driver1),
		Driver2:       sp(msg.Driver2),
		TruckRate:     sp(msg.TruckRate),
		TruckCalcType: sp(msg.TruckCalcType),
		Active:        msg.Active,
		LeasedTruck:   msg.LeasedTruck,
		Class:         sp(msg.Class),
		FleetNumber:   sp(msg.FleetNumber),
		TruckLicense:  sp(msg.TruckLicense),
		WePayDriver:   msg.WePayDriver,
		Straps:        msg.Straps,
		ExcludeFuel:   msg.ExcludeFuel,
	}
}

func updateTruckReqToModel(msg *pb.UpdateTruckRequest) *models.Truck {
	return &models.Truck{
		ID:            int(msg.Id),
		TruckNumber:   msg.TruckNumber,
		TruckMake:     sp(msg.TruckMake),
		TruckModel:    sp(msg.TruckModel),
		TruckYear:     sp(msg.TruckYear),
		TrailerNumber: sp(msg.TrailerNumber),
		Driver1:       sp(msg.Driver1),
		Driver2:       sp(msg.Driver2),
		TruckRate:     sp(msg.TruckRate),
		TruckCalcType: sp(msg.TruckCalcType),
		Active:        msg.Active,
		LeasedTruck:   msg.LeasedTruck,
		Class:         sp(msg.Class),
		FleetNumber:   sp(msg.FleetNumber),
		TruckLicense:  sp(msg.TruckLicense),
		WePayDriver:   msg.WePayDriver,
		Straps:        msg.Straps,
		ExcludeFuel:   msg.ExcludeFuel,
	}
}
