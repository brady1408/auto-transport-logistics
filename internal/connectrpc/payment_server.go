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
	"github.com/brady1408/auto-transport-logistics/internal/service"
)

type paymentStoreI interface {
	List(ctx context.Context, f models.PaymentFilter) (*models.PaymentListResult, error)
	GetByID(ctx context.Context, id int) (*models.Payment, error)
	Create(ctx context.Context, p *models.Payment) error
	Update(ctx context.Context, p *models.Payment) error
	Delete(ctx context.Context, id int) error
	PostByDateRange(ctx context.Context, dateFrom, dateTo, username string) (int, error)
}

type paymentDetailStoreI interface {
	ListByPayment(ctx context.Context, paymentID int) ([]models.PaymentDetail, error)
}

type PaymentServer struct {
	atlinkspbconnect.UnimplementedPaymentServiceHandler
	store      paymentStoreI
	details    paymentDetailStoreI
	paymentSvc *service.PaymentService
	audit      *audit.Service
}

func NewPaymentServer(store paymentStoreI, details paymentDetailStoreI, svc *service.PaymentService, audit *audit.Service) *PaymentServer {
	return &PaymentServer{store: store, details: details, paymentSvc: svc, audit: audit}
}

func (s *PaymentServer) ListPayments(ctx context.Context, req *connect.Request[pb.ListPaymentsRequest]) (*connect.Response[pb.ListPaymentsResponse], error) {
	filter := protoToPaymentFilter(req.Msg)
	result, err := s.store.List(ctx, filter)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("list payments: %w", err))
	}

	payments := make([]*pb.Payment, len(result.Items))
	for i := range result.Items {
		payments[i] = paymentToProto(&result.Items[i])
	}

	return connect.NewResponse(&pb.ListPaymentsResponse{
		Payments: payments,
		Pagination: &pb.PaginationResponse{
			TotalCount: int32(result.TotalCount),
			Page:       int32(result.Page),
			PageSize:   int32(result.PageSize),
		},
	}), nil
}

func (s *PaymentServer) GetPayment(ctx context.Context, req *connect.Request[pb.GetPaymentRequest]) (*connect.Response[pb.GetPaymentResponse], error) {
	p, err := s.store.GetByID(ctx, int(req.Msg.Id))
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("payment %d not found", req.Msg.Id))
	}
	return connect.NewResponse(&pb.GetPaymentResponse{Payment: paymentToProto(p)}), nil
}

func (s *PaymentServer) CreatePayment(ctx context.Context, req *connect.Request[pb.CreatePaymentRequest]) (*connect.Response[pb.CreatePaymentResponse], error) {
	p := createPaymentReqToModel(req.Msg)
	if err := s.store.Create(ctx, p); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("create payment: %w", err))
	}

	s.audit.Log(ctx, "payments", p.ID, "INSERT", nil, p)

	return connect.NewResponse(&pb.CreatePaymentResponse{Payment: paymentToProto(p)}), nil
}

func (s *PaymentServer) UpdatePayment(ctx context.Context, req *connect.Request[pb.UpdatePaymentRequest]) (*connect.Response[pb.UpdatePaymentResponse], error) {
	old, err := s.store.GetByID(ctx, int(req.Msg.Id))
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("payment %d not found", req.Msg.Id))
	}

	p := updatePaymentReqToModel(req.Msg)
	if err := s.store.Update(ctx, p); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("update payment: %w", err))
	}

	s.audit.Log(ctx, "payments", p.ID, "UPDATE", old, p)

	updated, err := s.store.GetByID(ctx, p.ID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("fetch updated payment: %w", err))
	}
	return connect.NewResponse(&pb.UpdatePaymentResponse{Payment: paymentToProto(updated)}), nil
}

func (s *PaymentServer) DeletePayment(ctx context.Context, req *connect.Request[pb.DeletePaymentRequest]) (*connect.Response[pb.DeletePaymentResponse], error) {
	if err := s.store.Delete(ctx, int(req.Msg.Id)); err != nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("payment %d not found", req.Msg.Id))
	}
	s.audit.Log(ctx, "payments", int(req.Msg.Id), "DELETE", nil, nil)
	return connect.NewResponse(&pb.DeletePaymentResponse{Success: true}), nil
}

func (s *PaymentServer) ListPaymentDetails(ctx context.Context, req *connect.Request[pb.ListPaymentDetailsRequest]) (*connect.Response[pb.ListPaymentDetailsResponse], error) {
	details, err := s.details.ListByPayment(ctx, int(req.Msg.PaymentId))
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("list payment details: %w", err))
	}

	pbDetails := make([]*pb.PaymentDetail, len(details))
	for i := range details {
		pbDetails[i] = paymentDetailToProto(&details[i])
	}

	return connect.NewResponse(&pb.ListPaymentDetailsResponse{Details: pbDetails}), nil
}

func (s *PaymentServer) ApplyPayment(ctx context.Context, req *connect.Request[pb.ApplyPaymentRequest]) (*connect.Response[pb.ApplyPaymentResponse], error) {
	if req.Msg.Amount == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("amount is required"))
	}

	discount := ""
	if req.Msg.Discount != nil {
		discount = *req.Msg.Discount
	}

	if err := s.paymentSvc.ApplyPayment(ctx, int(req.Msg.PaymentId), int(req.Msg.InvoiceId), req.Msg.Amount, discount); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("apply payment: %w", err))
	}

	return connect.NewResponse(&pb.ApplyPaymentResponse{Success: true}), nil
}

func (s *PaymentServer) UnapplyPayment(ctx context.Context, req *connect.Request[pb.UnapplyPaymentRequest]) (*connect.Response[pb.UnapplyPaymentResponse], error) {
	if err := s.paymentSvc.UnapplyPayment(ctx, int(req.Msg.PaymentDetailId)); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("unapply payment: %w", err))
	}

	return connect.NewResponse(&pb.UnapplyPaymentResponse{Success: true}), nil
}

func (s *PaymentServer) PostPayments(ctx context.Context, req *connect.Request[pb.PostPaymentsRequest]) (*connect.Response[pb.PostPaymentsResponse], error) {
	if req.Msg.DateFrom == "" || req.Msg.DateTo == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("date_from and date_to are required"))
	}

	username := ""
	if user, ok := auth.GetUser(ctx); ok {
		username = user.Username
	}

	count, err := s.store.PostByDateRange(ctx, req.Msg.DateFrom, req.Msg.DateTo, username)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("post payments: %w", err))
	}

	return connect.NewResponse(&pb.PostPaymentsResponse{Count: int32(count)}), nil
}

// --- Payment converters ---

func paymentToProto(p *models.Payment) *pb.Payment {
	return &pb.Payment{
		Id:              int32(p.ID),
		CustomerId:      ip(p.CustomerID),
		CustomerNumber:  sp(p.CustomerNumber),
		CustomerName:    sp(p.CustomerName),
		PaymentDate:     timeStr(p.PaymentDate),
		CheckNumber:     sp(p.CheckNumber),
		Amount:          sp(p.Amount),
		AppliedAmount:   sp(p.AppliedAmount),
		UnappliedAmount: sp(p.UnappliedAmount),
		PaymentMethod:   sp(p.PaymentMethod),
		Comments:        sp(p.Comments),
		CreatedBy:       sp(p.CreatedBy),
		PostedAt:        timeStr(p.PostedAt),
		PostedBy:        sp(p.PostedBy),
		CreatedAt:       p.CreatedAt.Format(time.RFC3339),
		UpdatedAt:       p.UpdatedAt.Format(time.RFC3339),
	}
}

func paymentDetailToProto(d *models.PaymentDetail) *pb.PaymentDetail {
	return &pb.PaymentDetail{
		Id:             int32(d.ID),
		PaymentId:      int32(d.PaymentID),
		InvoiceId:      ip(d.InvoiceID),
		InvoiceNumber:  sp(d.InvoiceNumber),
		Amount:         sp(d.Amount),
		DiscountAmount: sp(d.DiscountAmount),
		CreatedAt:      d.CreatedAt.Format(time.RFC3339),
		UpdatedAt:      d.UpdatedAt.Format(time.RFC3339),
	}
}

func protoToPaymentFilter(msg *pb.ListPaymentsRequest) models.PaymentFilter {
	f := models.PaymentFilter{}
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
	if msg.DateFrom != nil {
		f.DateFrom = *msg.DateFrom
	}
	if msg.DateTo != nil {
		f.DateTo = *msg.DateTo
	}
	return f
}

func createPaymentReqToModel(msg *pb.CreatePaymentRequest) *models.Payment {
	return &models.Payment{
		CustomerID:    i32p(msg.CustomerId),
		PaymentDate:   parseDate(msg.PaymentDate),
		CheckNumber:   sp(msg.CheckNumber),
		Amount:        sp(msg.Amount),
		PaymentMethod: sp(msg.PaymentMethod),
		Comments:      sp(msg.Comments),
	}
}

func updatePaymentReqToModel(msg *pb.UpdatePaymentRequest) *models.Payment {
	return &models.Payment{
		ID:            int(msg.Id),
		CustomerID:    i32p(msg.CustomerId),
		PaymentDate:   parseDate(msg.PaymentDate),
		CheckNumber:   sp(msg.CheckNumber),
		Amount:        sp(msg.Amount),
		PaymentMethod: sp(msg.PaymentMethod),
		Comments:      sp(msg.Comments),
	}
}
