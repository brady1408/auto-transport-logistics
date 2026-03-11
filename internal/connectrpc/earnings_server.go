package connectrpc

import (
	"context"
	"fmt"
	"time"

	"connectrpc.com/connect"
	"github.com/brady1408/atlinks/internal/audit"
	pb "github.com/brady1408/atlinks/internal/gen/atlinks/v1"
	"github.com/brady1408/atlinks/internal/gen/atlinks/v1/atlinkspbconnect"
	"github.com/brady1408/atlinks/internal/models"
)

type earningsStore interface {
	ListDriver(ctx context.Context, f models.EarningsAdjFilter) (*models.DriverEarningsAdjResult, error)
	GetDriverByID(ctx context.Context, id int) (*models.DriverEarningsAdj, error)
	CreateDriver(ctx context.Context, a *models.DriverEarningsAdj) error
	UpdateDriver(ctx context.Context, a *models.DriverEarningsAdj) error
	DeleteDriver(ctx context.Context, id int) error
	ListTruck(ctx context.Context, f models.EarningsAdjFilter) (*models.TruckEarningsAdjResult, error)
	GetTruckByID(ctx context.Context, id int) (*models.TruckEarningsAdj, error)
	CreateTruck(ctx context.Context, a *models.TruckEarningsAdj) error
	UpdateTruck(ctx context.Context, a *models.TruckEarningsAdj) error
	DeleteTruck(ctx context.Context, id int) error
}

type EarningsServer struct {
	atlinkspbconnect.UnimplementedEarningsServiceHandler
	store earningsStore
	audit *audit.Service
}

func NewEarningsServer(store earningsStore, audit *audit.Service) *EarningsServer {
	return &EarningsServer{store: store, audit: audit}
}

// --- Driver Earnings ---

func (s *EarningsServer) ListDriverEarnings(ctx context.Context, req *connect.Request[pb.ListDriverEarningsRequest]) (*connect.Response[pb.ListDriverEarningsResponse], error) {
	filter := protoToDriverEarningsFilter(req.Msg)
	result, err := s.store.ListDriver(ctx, filter)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("list driver earnings: %w", err))
	}

	items := make([]*pb.DriverEarningsAdj, len(result.Items))
	for i := range result.Items {
		items[i] = driverEarningsToProto(&result.Items[i])
	}

	return connect.NewResponse(&pb.ListDriverEarningsResponse{
		Items: items,
		Pagination: &pb.PaginationResponse{
			TotalCount: int32(result.TotalCount),
			Page:       int32(result.Page),
			PageSize:   int32(result.PageSize),
		},
	}), nil
}

func (s *EarningsServer) GetDriverEarnings(ctx context.Context, req *connect.Request[pb.GetDriverEarningsRequest]) (*connect.Response[pb.GetDriverEarningsResponse], error) {
	a, err := s.store.GetDriverByID(ctx, int(req.Msg.Id))
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("driver earnings %d not found", req.Msg.Id))
	}
	return connect.NewResponse(&pb.GetDriverEarningsResponse{Item: driverEarningsToProto(a)}), nil
}

func (s *EarningsServer) CreateDriverEarnings(ctx context.Context, req *connect.Request[pb.CreateDriverEarningsRequest]) (*connect.Response[pb.CreateDriverEarningsResponse], error) {
	if req.Msg.EmployeeId == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("employee_id is required"))
	}

	a := createDriverEarningsReqToModel(req.Msg)
	if err := s.store.CreateDriver(ctx, a); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("create driver earnings: %w", err))
	}

	s.audit.Log(ctx, "driver_earnings_adj", a.ID, "INSERT", nil, a)

	return connect.NewResponse(&pb.CreateDriverEarningsResponse{Item: driverEarningsToProto(a)}), nil
}

func (s *EarningsServer) UpdateDriverEarnings(ctx context.Context, req *connect.Request[pb.UpdateDriverEarningsRequest]) (*connect.Response[pb.UpdateDriverEarningsResponse], error) {
	old, err := s.store.GetDriverByID(ctx, int(req.Msg.Id))
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("driver earnings %d not found", req.Msg.Id))
	}

	a := updateDriverEarningsReqToModel(req.Msg)
	if err := s.store.UpdateDriver(ctx, a); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("update driver earnings: %w", err))
	}

	s.audit.Log(ctx, "driver_earnings_adj", a.ID, "UPDATE", old, a)

	updated, err := s.store.GetDriverByID(ctx, a.ID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("fetch updated driver earnings: %w", err))
	}
	return connect.NewResponse(&pb.UpdateDriverEarningsResponse{Item: driverEarningsToProto(updated)}), nil
}

func (s *EarningsServer) DeleteDriverEarnings(ctx context.Context, req *connect.Request[pb.DeleteDriverEarningsRequest]) (*connect.Response[pb.DeleteDriverEarningsResponse], error) {
	if err := s.store.DeleteDriver(ctx, int(req.Msg.Id)); err != nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("driver earnings %d not found", req.Msg.Id))
	}
	s.audit.Log(ctx, "driver_earnings_adj", int(req.Msg.Id), "DELETE", nil, nil)
	return connect.NewResponse(&pb.DeleteDriverEarningsResponse{Success: true}), nil
}

// --- Truck Earnings ---

func (s *EarningsServer) ListTruckEarnings(ctx context.Context, req *connect.Request[pb.ListTruckEarningsRequest]) (*connect.Response[pb.ListTruckEarningsResponse], error) {
	filter := protoToTruckEarningsFilter(req.Msg)
	result, err := s.store.ListTruck(ctx, filter)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("list truck earnings: %w", err))
	}

	items := make([]*pb.TruckEarningsAdj, len(result.Items))
	for i := range result.Items {
		items[i] = truckEarningsToProto(&result.Items[i])
	}

	return connect.NewResponse(&pb.ListTruckEarningsResponse{
		Items: items,
		Pagination: &pb.PaginationResponse{
			TotalCount: int32(result.TotalCount),
			Page:       int32(result.Page),
			PageSize:   int32(result.PageSize),
		},
	}), nil
}

func (s *EarningsServer) GetTruckEarnings(ctx context.Context, req *connect.Request[pb.GetTruckEarningsRequest]) (*connect.Response[pb.GetTruckEarningsResponse], error) {
	a, err := s.store.GetTruckByID(ctx, int(req.Msg.Id))
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("truck earnings %d not found", req.Msg.Id))
	}
	return connect.NewResponse(&pb.GetTruckEarningsResponse{Item: truckEarningsToProto(a)}), nil
}

func (s *EarningsServer) CreateTruckEarnings(ctx context.Context, req *connect.Request[pb.CreateTruckEarningsRequest]) (*connect.Response[pb.CreateTruckEarningsResponse], error) {
	if req.Msg.TruckId == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("truck_id is required"))
	}

	a := createTruckEarningsReqToModel(req.Msg)
	if err := s.store.CreateTruck(ctx, a); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("create truck earnings: %w", err))
	}

	s.audit.Log(ctx, "truck_earnings_adj", a.ID, "INSERT", nil, a)

	return connect.NewResponse(&pb.CreateTruckEarningsResponse{Item: truckEarningsToProto(a)}), nil
}

func (s *EarningsServer) UpdateTruckEarnings(ctx context.Context, req *connect.Request[pb.UpdateTruckEarningsRequest]) (*connect.Response[pb.UpdateTruckEarningsResponse], error) {
	old, err := s.store.GetTruckByID(ctx, int(req.Msg.Id))
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("truck earnings %d not found", req.Msg.Id))
	}

	a := updateTruckEarningsReqToModel(req.Msg)
	if err := s.store.UpdateTruck(ctx, a); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("update truck earnings: %w", err))
	}

	s.audit.Log(ctx, "truck_earnings_adj", a.ID, "UPDATE", old, a)

	updated, err := s.store.GetTruckByID(ctx, a.ID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("fetch updated truck earnings: %w", err))
	}
	return connect.NewResponse(&pb.UpdateTruckEarningsResponse{Item: truckEarningsToProto(updated)}), nil
}

func (s *EarningsServer) DeleteTruckEarnings(ctx context.Context, req *connect.Request[pb.DeleteTruckEarningsRequest]) (*connect.Response[pb.DeleteTruckEarningsResponse], error) {
	if err := s.store.DeleteTruck(ctx, int(req.Msg.Id)); err != nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("truck earnings %d not found", req.Msg.Id))
	}
	s.audit.Log(ctx, "truck_earnings_adj", int(req.Msg.Id), "DELETE", nil, nil)
	return connect.NewResponse(&pb.DeleteTruckEarningsResponse{Success: true}), nil
}

// --- Earnings converters ---

func driverEarningsToProto(a *models.DriverEarningsAdj) *pb.DriverEarningsAdj {
	name := a.EmployeeName
	adjDate := a.AdjDate.Format(time.RFC3339)
	desc := a.Description
	adjType := a.AdjType
	amount := a.Amount
	return &pb.DriverEarningsAdj{
		Id:           int32(a.ID),
		EmployeeId:   int32(a.EmployeeID),
		EmployeeName: &name,
		AdjDate:      &adjDate,
		Description:  &desc,
		AdjType:      &adjType,
		Amount:       &amount,
		Reference:    sp(a.Reference),
		CreatedAt:    a.CreatedAt.Format(time.RFC3339),
		UpdatedAt:    a.UpdatedAt.Format(time.RFC3339),
	}
}

func truckEarningsToProto(a *models.TruckEarningsAdj) *pb.TruckEarningsAdj {
	num := a.TruckNumber
	adjDate := a.AdjDate.Format(time.RFC3339)
	desc := a.Description
	adjType := a.AdjType
	amount := a.Amount
	return &pb.TruckEarningsAdj{
		Id:          int32(a.ID),
		TruckId:     int32(a.TruckID),
		TruckNumber: &num,
		AdjDate:     &adjDate,
		Description: &desc,
		AdjType:     &adjType,
		Amount:      &amount,
		Reference:   sp(a.Reference),
		CreatedAt:   a.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   a.UpdatedAt.Format(time.RFC3339),
	}
}

func protoToDriverEarningsFilter(msg *pb.ListDriverEarningsRequest) models.EarningsAdjFilter {
	f := models.EarningsAdjFilter{}
	if msg.Pagination != nil {
		f.Page = int(msg.Pagination.Page)
		f.PageSize = int(msg.Pagination.PageSize)
	}
	if msg.EmployeeId != nil {
		f.EntityID = int(*msg.EmployeeId)
	}
	if msg.DateFrom != nil {
		f.DateFrom = *msg.DateFrom
	}
	if msg.DateTo != nil {
		f.DateTo = *msg.DateTo
	}
	return f
}

func protoToTruckEarningsFilter(msg *pb.ListTruckEarningsRequest) models.EarningsAdjFilter {
	f := models.EarningsAdjFilter{}
	if msg.Pagination != nil {
		f.Page = int(msg.Pagination.Page)
		f.PageSize = int(msg.Pagination.PageSize)
	}
	if msg.TruckId != nil {
		f.EntityID = int(*msg.TruckId)
	}
	if msg.DateFrom != nil {
		f.DateFrom = *msg.DateFrom
	}
	if msg.DateTo != nil {
		f.DateTo = *msg.DateTo
	}
	return f
}

func derefStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func derefTime(s *string) time.Time {
	t := parseDate(s)
	if t == nil {
		return time.Time{}
	}
	return *t
}

func createDriverEarningsReqToModel(msg *pb.CreateDriverEarningsRequest) *models.DriverEarningsAdj {
	return &models.DriverEarningsAdj{
		EmployeeID:  int(msg.EmployeeId),
		AdjDate:     derefTime(msg.AdjDate),
		Description: derefStr(msg.Description),
		AdjType:     derefStr(msg.AdjType),
		Amount:      derefStr(msg.Amount),
		Reference:   sp(msg.Reference),
	}
}

func updateDriverEarningsReqToModel(msg *pb.UpdateDriverEarningsRequest) *models.DriverEarningsAdj {
	return &models.DriverEarningsAdj{
		ID:          int(msg.Id),
		EmployeeID:  int(msg.EmployeeId),
		AdjDate:     derefTime(msg.AdjDate),
		Description: derefStr(msg.Description),
		AdjType:     derefStr(msg.AdjType),
		Amount:      derefStr(msg.Amount),
		Reference:   sp(msg.Reference),
	}
}

func createTruckEarningsReqToModel(msg *pb.CreateTruckEarningsRequest) *models.TruckEarningsAdj {
	return &models.TruckEarningsAdj{
		TruckID:     int(msg.TruckId),
		AdjDate:     derefTime(msg.AdjDate),
		Description: derefStr(msg.Description),
		AdjType:     derefStr(msg.AdjType),
		Amount:      derefStr(msg.Amount),
		Reference:   sp(msg.Reference),
	}
}

func updateTruckEarningsReqToModel(msg *pb.UpdateTruckEarningsRequest) *models.TruckEarningsAdj {
	return &models.TruckEarningsAdj{
		ID:          int(msg.Id),
		TruckID:     int(msg.TruckId),
		AdjDate:     derefTime(msg.AdjDate),
		Description: derefStr(msg.Description),
		AdjType:     derefStr(msg.AdjType),
		Amount:      derefStr(msg.Amount),
		Reference:   sp(msg.Reference),
	}
}
