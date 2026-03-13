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

type invoiceStoreI interface {
	List(ctx context.Context, f models.InvoiceFilter) (*models.InvoiceListResult, error)
	GetByID(ctx context.Context, id int) (*models.Invoice, error)
	Create(ctx context.Context, inv *models.Invoice) error
	Update(ctx context.Context, inv *models.Invoice) error
	Delete(ctx context.Context, id int) error
	NextInvoiceNumber(ctx context.Context) (string, error)
	PostByDateRange(ctx context.Context, dateFrom, dateTo, username string) (int, error)
}

type invoiceDetailStoreI interface {
	ListByInvoice(ctx context.Context, invoiceID int) ([]models.InvoiceDetail, error)
}

type InvoiceServer struct {
	atlinkspbconnect.UnimplementedInvoiceServiceHandler
	store      invoiceStoreI
	details    invoiceDetailStoreI
	invoiceSvc *service.InvoiceService
	audit      *audit.Service
}

func NewInvoiceServer(store invoiceStoreI, details invoiceDetailStoreI, svc *service.InvoiceService, audit *audit.Service) *InvoiceServer {
	return &InvoiceServer{store: store, details: details, invoiceSvc: svc, audit: audit}
}

func (s *InvoiceServer) ListInvoices(ctx context.Context, req *connect.Request[pb.ListInvoicesRequest]) (*connect.Response[pb.ListInvoicesResponse], error) {
	filter := protoToInvoiceFilter(req.Msg)
	result, err := s.store.List(ctx, filter)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("list invoices: %w", err))
	}

	invoices := make([]*pb.Invoice, len(result.Items))
	for i := range result.Items {
		invoices[i] = invoiceToProto(&result.Items[i])
	}

	return connect.NewResponse(&pb.ListInvoicesResponse{
		Invoices: invoices,
		Pagination: &pb.PaginationResponse{
			TotalCount: int32(result.TotalCount),
			Page:       int32(result.Page),
			PageSize:   int32(result.PageSize),
		},
	}), nil
}

func (s *InvoiceServer) GetInvoice(ctx context.Context, req *connect.Request[pb.GetInvoiceRequest]) (*connect.Response[pb.GetInvoiceResponse], error) {
	inv, err := s.store.GetByID(ctx, int(req.Msg.Id))
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("invoice %d not found", req.Msg.Id))
	}
	return connect.NewResponse(&pb.GetInvoiceResponse{Invoice: invoiceToProto(inv)}), nil
}

func (s *InvoiceServer) CreateInvoice(ctx context.Context, req *connect.Request[pb.CreateInvoiceRequest]) (*connect.Response[pb.CreateInvoiceResponse], error) {
	invNum, err := s.store.NextInvoiceNumber(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("next invoice number: %w", err))
	}

	inv := createInvoiceReqToModel(req.Msg)
	inv.InvoiceNumber = invNum
	inv.Active = true

	if err := s.store.Create(ctx, inv); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("create invoice: %w", err))
	}

	s.audit.Log(ctx, "invoices", inv.ID, "INSERT", nil, inv)

	return connect.NewResponse(&pb.CreateInvoiceResponse{Invoice: invoiceToProto(inv)}), nil
}

func (s *InvoiceServer) UpdateInvoice(ctx context.Context, req *connect.Request[pb.UpdateInvoiceRequest]) (*connect.Response[pb.UpdateInvoiceResponse], error) {
	old, err := s.store.GetByID(ctx, int(req.Msg.Id))
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("invoice %d not found", req.Msg.Id))
	}

	inv := updateInvoiceReqToModel(req.Msg)
	if err := s.store.Update(ctx, inv); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("update invoice: %w", err))
	}

	s.audit.Log(ctx, "invoices", inv.ID, "UPDATE", old, inv)

	updated, err := s.store.GetByID(ctx, inv.ID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("fetch updated invoice: %w", err))
	}
	return connect.NewResponse(&pb.UpdateInvoiceResponse{Invoice: invoiceToProto(updated)}), nil
}

func (s *InvoiceServer) DeleteInvoice(ctx context.Context, req *connect.Request[pb.DeleteInvoiceRequest]) (*connect.Response[pb.DeleteInvoiceResponse], error) {
	if err := s.store.Delete(ctx, int(req.Msg.Id)); err != nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("invoice %d not found", req.Msg.Id))
	}
	s.audit.Log(ctx, "invoices", int(req.Msg.Id), "DELETE", nil, nil)
	return connect.NewResponse(&pb.DeleteInvoiceResponse{Success: true}), nil
}

func (s *InvoiceServer) ListInvoiceDetails(ctx context.Context, req *connect.Request[pb.ListInvoiceDetailsRequest]) (*connect.Response[pb.ListInvoiceDetailsResponse], error) {
	details, err := s.details.ListByInvoice(ctx, int(req.Msg.InvoiceId))
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("list invoice details: %w", err))
	}

	pbDetails := make([]*pb.InvoiceDetail, len(details))
	for i := range details {
		pbDetails[i] = invoiceDetailToProto(&details[i])
	}

	return connect.NewResponse(&pb.ListInvoiceDetailsResponse{Details: pbDetails}), nil
}

func (s *InvoiceServer) GenerateInvoice(ctx context.Context, req *connect.Request[pb.GenerateInvoiceRequest]) (*connect.Response[pb.GenerateInvoiceResponse], error) {
	inv, err := s.invoiceSvc.GenerateFromOrder(ctx, int(req.Msg.OrderId))
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("generate invoice: %w", err))
	}

	s.audit.Log(ctx, "invoices", inv.ID, "INSERT", nil, inv)

	return connect.NewResponse(&pb.GenerateInvoiceResponse{Invoice: invoiceToProto(inv)}), nil
}

func (s *InvoiceServer) VoidInvoice(ctx context.Context, req *connect.Request[pb.VoidInvoiceRequest]) (*connect.Response[pb.VoidInvoiceResponse], error) {
	if err := s.invoiceSvc.VoidInvoice(ctx, int(req.Msg.InvoiceId)); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("void invoice: %w", err))
	}

	s.audit.Log(ctx, "invoices", int(req.Msg.InvoiceId), "UPDATE", nil, nil)

	return connect.NewResponse(&pb.VoidInvoiceResponse{Success: true}), nil
}

func (s *InvoiceServer) PostInvoices(ctx context.Context, req *connect.Request[pb.PostInvoicesRequest]) (*connect.Response[pb.PostInvoicesResponse], error) {
	if req.Msg.DateFrom == "" || req.Msg.DateTo == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("date_from and date_to are required"))
	}

	username := ""
	if user, ok := auth.GetUser(ctx); ok {
		username = user.Username
	}

	count, err := s.store.PostByDateRange(ctx, req.Msg.DateFrom, req.Msg.DateTo, username)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("post invoices: %w", err))
	}

	return connect.NewResponse(&pb.PostInvoicesResponse{Count: int32(count)}), nil
}

// --- Invoice converters ---

func invoiceToProto(inv *models.Invoice) *pb.Invoice {
	return &pb.Invoice{
		Id:             int32(inv.ID),
		InvoiceNumber:  inv.InvoiceNumber,
		Active:         inv.Active,
		CustomerId:     ip(inv.CustomerID),
		CustomerNumber: sp(inv.CustomerNumber),
		CustomerName:   sp(inv.CustomerName),
		OrderId:        ip(inv.OrderID),
		OrderNumber:    sp(inv.OrderNumber),
		InvoiceDate:    timeStr(inv.InvoiceDate),
		DueDate:        timeStr(inv.DueDate),
		Terms:          sp(inv.Terms),
		TaxCode:        sp(inv.TaxCode),
		Subtotal:       sp(inv.Subtotal),
		Tax:            sp(inv.Tax),
		TotalAmount:    sp(inv.TotalAmount),
		AmountPaid:     sp(inv.AmountPaid),
		Balance:        sp(inv.Balance),
		Status:         sp(inv.Status),
		Comments:       sp(inv.Comments),
		BillToAddress:  sp(inv.BillToAddress),
		BillToAddress2: sp(inv.BillToAddress2),
		BillToCity:     sp(inv.BillToCity),
		BillToState:    sp(inv.BillToState),
		BillToZip:      sp(inv.BillToZip),
		PostedAt:       timeStr(inv.PostedAt),
		PostedBy:       sp(inv.PostedBy),
		CreatedAt:      inv.CreatedAt.Format(time.RFC3339),
		UpdatedAt:      inv.UpdatedAt.Format(time.RFC3339),
	}
}

func invoiceDetailToProto(d *models.InvoiceDetail) *pb.InvoiceDetail {
	return &pb.InvoiceDetail{
		Id:          int32(d.ID),
		InvoiceId:   int32(d.InvoiceID),
		OrderId:     ip(d.OrderID),
		VehicleId:   ip(d.VehicleID),
		Vin:         sp(d.VIN),
		Year:        sp(d.Year),
		Make:        sp(d.Make),
		Model:       sp(d.Model),
		Description: sp(d.Description),
		Qty:         ip(d.Qty),
		Rate:        sp(d.Rate),
		Amount:      sp(d.Amount),
		Taxable:     d.Taxable,
		ItemCode:    sp(d.ItemCode),
		CreatedAt:   d.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   d.UpdatedAt.Format(time.RFC3339),
	}
}

func protoToInvoiceFilter(msg *pb.ListInvoicesRequest) models.InvoiceFilter {
	f := models.InvoiceFilter{}
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
	if msg.DateFrom != nil {
		f.DateFrom = *msg.DateFrom
	}
	if msg.DateTo != nil {
		f.DateTo = *msg.DateTo
	}
	return f
}

func createInvoiceReqToModel(msg *pb.CreateInvoiceRequest) *models.Invoice {
	return &models.Invoice{
		CustomerID:  i32p(msg.CustomerId),
		OrderID:     i32p(msg.OrderId),
		InvoiceDate: parseDate(msg.InvoiceDate),
		DueDate:     parseDate(msg.DueDate),
		Terms:       sp(msg.Terms),
		TaxCode:     sp(msg.TaxCode),
		Comments:    sp(msg.Comments),
	}
}

func updateInvoiceReqToModel(msg *pb.UpdateInvoiceRequest) *models.Invoice {
	return &models.Invoice{
		ID:          int(msg.Id),
		CustomerID:  i32p(msg.CustomerId),
		OrderID:     i32p(msg.OrderId),
		InvoiceDate: parseDate(msg.InvoiceDate),
		DueDate:     parseDate(msg.DueDate),
		Terms:       sp(msg.Terms),
		TaxCode:     sp(msg.TaxCode),
		Subtotal:    sp(msg.Subtotal),
		Tax:         sp(msg.Tax),
		TotalAmount: sp(msg.TotalAmount),
		Comments:    sp(msg.Comments),
		Status:      sp(msg.Status),
	}
}
