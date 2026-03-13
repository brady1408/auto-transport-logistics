package connectrpc

import (
	"context"
	"fmt"
	"time"

	"connectrpc.com/connect"
	"github.com/brady1408/auto-transport-logistics/internal/audit"
	"github.com/brady1408/auto-transport-logistics/internal/auth"
	pb "github.com/brady1408/auto-transport-logistics/internal/gen/atlinks/v1"
	"github.com/brady1408/auto-transport-logistics/internal/gen/atlinks/v1/atlinkspbconnect"
	"github.com/brady1408/auto-transport-logistics/internal/models"
)

type creditMemoStore interface {
	List(ctx context.Context, f models.CreditMemoFilter) (*models.CreditMemoListResult, error)
	GetByID(ctx context.Context, id int) (*models.CreditMemo, error)
	Create(ctx context.Context, cm *models.CreditMemo) error
	Update(ctx context.Context, cm *models.CreditMemo) error
	Delete(ctx context.Context, id int) error
	NextCreditNumber(ctx context.Context) (string, error)
}

type CreditMemoServer struct {
	atlinkspbconnect.UnimplementedCreditMemoServiceHandler
	store creditMemoStore
	audit *audit.Service
}

func NewCreditMemoServer(store creditMemoStore, audit *audit.Service) *CreditMemoServer {
	return &CreditMemoServer{store: store, audit: audit}
}

func (s *CreditMemoServer) ListCreditMemos(ctx context.Context, req *connect.Request[pb.ListCreditMemosRequest]) (*connect.Response[pb.ListCreditMemosResponse], error) {
	filter := protoToCreditMemoFilter(req.Msg)
	result, err := s.store.List(ctx, filter)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("list credit memos: %w", err))
	}

	memos := make([]*pb.CreditMemo, len(result.Items))
	for i := range result.Items {
		memos[i] = creditMemoToProto(&result.Items[i])
	}

	return connect.NewResponse(&pb.ListCreditMemosResponse{
		CreditMemos: memos,
		Pagination: &pb.PaginationResponse{
			TotalCount: int32(result.TotalCount),
			Page:       int32(result.Page),
			PageSize:   int32(result.PageSize),
		},
	}), nil
}

func (s *CreditMemoServer) GetCreditMemo(ctx context.Context, req *connect.Request[pb.GetCreditMemoRequest]) (*connect.Response[pb.GetCreditMemoResponse], error) {
	cm, err := s.store.GetByID(ctx, int(req.Msg.Id))
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("credit memo %d not found", req.Msg.Id))
	}
	return connect.NewResponse(&pb.GetCreditMemoResponse{CreditMemo: creditMemoToProto(cm)}), nil
}

func (s *CreditMemoServer) CreateCreditMemo(ctx context.Context, req *connect.Request[pb.CreateCreditMemoRequest]) (*connect.Response[pb.CreateCreditMemoResponse], error) {
	num, err := s.store.NextCreditNumber(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("generate credit number: %w", err))
	}

	cm := createCreditMemoReqToModel(req.Msg)
	cm.CreditNumber = num

	// Populate created_by from auth context if available
	if user, ok := auth.GetUser(ctx); ok {
		cm.CreatedBy = &user.Username
	}

	if err := s.store.Create(ctx, cm); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("create credit memo: %w", err))
	}

	s.audit.Log(ctx, "credit_memos", cm.ID, "INSERT", nil, cm)

	created, err := s.store.GetByID(ctx, cm.ID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("fetch created credit memo: %w", err))
	}
	return connect.NewResponse(&pb.CreateCreditMemoResponse{CreditMemo: creditMemoToProto(created)}), nil
}

func (s *CreditMemoServer) UpdateCreditMemo(ctx context.Context, req *connect.Request[pb.UpdateCreditMemoRequest]) (*connect.Response[pb.UpdateCreditMemoResponse], error) {
	old, err := s.store.GetByID(ctx, int(req.Msg.Id))
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("credit memo %d not found", req.Msg.Id))
	}

	cm := updateCreditMemoReqToModel(req.Msg)
	cm.CreditNumber = old.CreditNumber // credit number is immutable
	if err := s.store.Update(ctx, cm); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("update credit memo: %w", err))
	}

	s.audit.Log(ctx, "credit_memos", cm.ID, "UPDATE", old, cm)

	updated, err := s.store.GetByID(ctx, cm.ID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("fetch updated credit memo: %w", err))
	}
	return connect.NewResponse(&pb.UpdateCreditMemoResponse{CreditMemo: creditMemoToProto(updated)}), nil
}

func (s *CreditMemoServer) DeleteCreditMemo(ctx context.Context, req *connect.Request[pb.DeleteCreditMemoRequest]) (*connect.Response[pb.DeleteCreditMemoResponse], error) {
	if err := s.store.Delete(ctx, int(req.Msg.Id)); err != nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("credit memo %d not found", req.Msg.Id))
	}
	s.audit.Log(ctx, "credit_memos", int(req.Msg.Id), "DELETE", nil, nil)
	return connect.NewResponse(&pb.DeleteCreditMemoResponse{Success: true}), nil
}

func creditMemoToProto(cm *models.CreditMemo) *pb.CreditMemo {
	return &pb.CreditMemo{
		Id:             int32(cm.ID),
		CreditNumber:   cm.CreditNumber,
		CustomerId:     ip(cm.CustomerID),
		CustomerNumber: sp(cm.CustomerNumber),
		CustomerName:   sp(cm.CustomerName),
		InvoiceId:      ip(cm.InvoiceID),
		InvoiceNumber:  sp(cm.InvoiceNumber),
		CreditDate:     timeStr(cm.CreditDate),
		Amount:         sp(cm.Amount),
		Reason:         sp(cm.Reason),
		Status:         sp(cm.Status),
		CreatedBy:      sp(cm.CreatedBy),
		Comments:       sp(cm.Comments),
		CreatedAt:      cm.CreatedAt.Format(time.RFC3339),
		UpdatedAt:      cm.UpdatedAt.Format(time.RFC3339),
	}
}

func protoToCreditMemoFilter(msg *pb.ListCreditMemosRequest) models.CreditMemoFilter {
	f := models.CreditMemoFilter{}
	if msg.Pagination != nil {
		f.Page = int(msg.Pagination.Page)
		f.PageSize = int(msg.Pagination.PageSize)
	}
	if msg.Search != nil {
		f.Search = *msg.Search
	}
	if msg.CustomerId != nil {
		f.CustomerID = fmt.Sprintf("%d", *msg.CustomerId)
	}
	if msg.Status != nil {
		f.Status = *msg.Status
	}
	return f
}

func createCreditMemoReqToModel(msg *pb.CreateCreditMemoRequest) *models.CreditMemo {
	return &models.CreditMemo{
		CustomerID: i32p(msg.CustomerId),
		InvoiceID:  i32p(msg.InvoiceId),
		CreditDate: parseDate(msg.CreditDate),
		Amount:     sp(msg.Amount),
		Reason:     sp(msg.Reason),
		Status:     sp(msg.Status),
		Comments:   sp(msg.Comments),
	}
}

func updateCreditMemoReqToModel(msg *pb.UpdateCreditMemoRequest) *models.CreditMemo {
	return &models.CreditMemo{
		ID:         int(msg.Id),
		CustomerID: i32p(msg.CustomerId),
		InvoiceID:  i32p(msg.InvoiceId),
		CreditDate: parseDate(msg.CreditDate),
		Amount:     sp(msg.Amount),
		Reason:     sp(msg.Reason),
		Status:     sp(msg.Status),
		Comments:   sp(msg.Comments),
	}
}
