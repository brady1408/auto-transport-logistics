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

type apStore interface {
	List(ctx context.Context, f models.APFilter) (*models.APListResult, error)
	GetByID(ctx context.Context, id int) (*models.AccountsPayable, error)
	Create(ctx context.Context, ap *models.AccountsPayable) error
	Update(ctx context.Context, ap *models.AccountsPayable) error
	Delete(ctx context.Context, id int) error
}

type APServer struct {
	atlinkspbconnect.UnimplementedAPServiceHandler
	store apStore
	audit *audit.Service
}

func NewAPServer(store apStore, audit *audit.Service) *APServer {
	return &APServer{store: store, audit: audit}
}

func (s *APServer) ListAP(ctx context.Context, req *connect.Request[pb.ListAPRequest]) (*connect.Response[pb.ListAPResponse], error) {
	filter := protoToAPFilter(req.Msg)
	result, err := s.store.List(ctx, filter)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("list accounts payable: %w", err))
	}

	items := make([]*pb.AccountsPayable, len(result.Items))
	for i := range result.Items {
		items[i] = apToProto(&result.Items[i])
	}

	return connect.NewResponse(&pb.ListAPResponse{
		Items: items,
		Pagination: &pb.PaginationResponse{
			TotalCount: int32(result.TotalCount),
			Page:       int32(result.Page),
			PageSize:   int32(result.PageSize),
		},
	}), nil
}

func (s *APServer) GetAP(ctx context.Context, req *connect.Request[pb.GetAPRequest]) (*connect.Response[pb.GetAPResponse], error) {
	ap, err := s.store.GetByID(ctx, int(req.Msg.Id))
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("accounts payable %d not found", req.Msg.Id))
	}
	return connect.NewResponse(&pb.GetAPResponse{Item: apToProto(ap)}), nil
}

func (s *APServer) CreateAP(ctx context.Context, req *connect.Request[pb.CreateAPRequest]) (*connect.Response[pb.CreateAPResponse], error) {
	ap := createAPReqToModel(req.Msg)
	if err := s.store.Create(ctx, ap); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("create accounts payable: %w", err))
	}

	s.audit.Log(ctx, "accounts_payable", ap.ID, "INSERT", nil, ap)

	return connect.NewResponse(&pb.CreateAPResponse{Item: apToProto(ap)}), nil
}

func (s *APServer) UpdateAP(ctx context.Context, req *connect.Request[pb.UpdateAPRequest]) (*connect.Response[pb.UpdateAPResponse], error) {
	old, err := s.store.GetByID(ctx, int(req.Msg.Id))
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("accounts payable %d not found", req.Msg.Id))
	}

	ap := updateAPReqToModel(req.Msg)
	if err := s.store.Update(ctx, ap); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("update accounts payable: %w", err))
	}

	s.audit.Log(ctx, "accounts_payable", ap.ID, "UPDATE", old, ap)

	updated, err := s.store.GetByID(ctx, ap.ID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("fetch updated accounts payable: %w", err))
	}
	return connect.NewResponse(&pb.UpdateAPResponse{Item: apToProto(updated)}), nil
}

func (s *APServer) DeleteAP(ctx context.Context, req *connect.Request[pb.DeleteAPRequest]) (*connect.Response[pb.DeleteAPResponse], error) {
	if err := s.store.Delete(ctx, int(req.Msg.Id)); err != nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("accounts payable %d not found", req.Msg.Id))
	}
	s.audit.Log(ctx, "accounts_payable", int(req.Msg.Id), "DELETE", nil, nil)
	return connect.NewResponse(&pb.DeleteAPResponse{Success: true}), nil
}

// --- AP converters ---

func apToProto(ap *models.AccountsPayable) *pb.AccountsPayable {
	return &pb.AccountsPayable{
		Id:          int32(ap.ID),
		TripId:      ip(ap.TripID),
		EmployeeId:  ip(ap.EmployeeID),
		TruckId:     ip(ap.TruckID),
		VendorName:  sp(ap.VendorName),
		PayableDate: timeStr(ap.PayableDate),
		Amount:      sp(ap.Amount),
		PaidAmount:  sp(ap.PaidAmount),
		Status:      sp(ap.Status),
		Description: sp(ap.Description),
		CheckNumber: sp(ap.CheckNumber),
		CheckDate:   timeStr(ap.CheckDate),
		Comments:    sp(ap.Comments),
		CreatedAt:   ap.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   ap.UpdatedAt.Format(time.RFC3339),
	}
}

func protoToAPFilter(msg *pb.ListAPRequest) models.APFilter {
	f := models.APFilter{}
	if msg.Pagination != nil {
		f.Page = int(msg.Pagination.Page)
		f.PageSize = int(msg.Pagination.PageSize)
	}
	if msg.Search != nil {
		f.Search = *msg.Search
	}
	if msg.Status != nil {
		f.Status = *msg.Status
	}
	if msg.EmployeeId != nil {
		f.EmployeeID = fmt.Sprintf("%d", *msg.EmployeeId)
	}
	if msg.TruckId != nil {
		f.TruckID = fmt.Sprintf("%d", *msg.TruckId)
	}
	return f
}

func createAPReqToModel(msg *pb.CreateAPRequest) *models.AccountsPayable {
	return &models.AccountsPayable{
		TripID:      i32p(msg.TripId),
		EmployeeID:  i32p(msg.EmployeeId),
		TruckID:     i32p(msg.TruckId),
		VendorName:  sp(msg.VendorName),
		PayableDate: parseDate(msg.PayableDate),
		Amount:      sp(msg.Amount),
		Status:      sp(msg.Status),
		Description: sp(msg.Description),
		Comments:    sp(msg.Comments),
	}
}

func updateAPReqToModel(msg *pb.UpdateAPRequest) *models.AccountsPayable {
	return &models.AccountsPayable{
		ID:          int(msg.Id),
		TripID:      i32p(msg.TripId),
		EmployeeID:  i32p(msg.EmployeeId),
		TruckID:     i32p(msg.TruckId),
		VendorName:  sp(msg.VendorName),
		PayableDate: parseDate(msg.PayableDate),
		Amount:      sp(msg.Amount),
		PaidAmount:  sp(msg.PaidAmount),
		Status:      sp(msg.Status),
		Description: sp(msg.Description),
		CheckNumber: sp(msg.CheckNumber),
		CheckDate:   parseDate(msg.CheckDate),
		Comments:    sp(msg.Comments),
	}
}
