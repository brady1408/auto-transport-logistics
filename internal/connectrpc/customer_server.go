package connectrpc

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	"github.com/brady1408/atlinks/internal/audit"
	pb "github.com/brady1408/atlinks/internal/gen/atlinks/v1"
	"github.com/brady1408/atlinks/internal/gen/atlinks/v1/atlinkspbconnect"
	"github.com/brady1408/atlinks/internal/models"
)

type customerStore interface {
	List(ctx context.Context, f models.CustomerFilter) (*models.CustomerListResult, error)
	GetByID(ctx context.Context, id int) (*models.Customer, error)
	Create(ctx context.Context, c *models.Customer) error
	Update(ctx context.Context, c *models.Customer) error
	Delete(ctx context.Context, id int) error
}

type CustomerServer struct {
	atlinkspbconnect.UnimplementedCustomerServiceHandler
	store customerStore
	audit *audit.Service
}

func NewCustomerServer(store customerStore, audit *audit.Service) *CustomerServer {
	return &CustomerServer{store: store, audit: audit}
}

func (s *CustomerServer) ListCustomers(ctx context.Context, req *connect.Request[pb.ListCustomersRequest]) (*connect.Response[pb.ListCustomersResponse], error) {
	filter := protoToCustomerFilter(req.Msg)
	result, err := s.store.List(ctx, filter)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("list customers: %w", err))
	}

	customers := make([]*pb.Customer, len(result.Items))
	for i := range result.Items {
		customers[i] = customerToProto(&result.Items[i])
	}

	return connect.NewResponse(&pb.ListCustomersResponse{
		Customers: customers,
		Pagination: &pb.PaginationResponse{
			TotalCount: int32(result.TotalCount),
			Page:       int32(result.Page),
			PageSize:   int32(result.PageSize),
		},
	}), nil
}

func (s *CustomerServer) GetCustomer(ctx context.Context, req *connect.Request[pb.GetCustomerRequest]) (*connect.Response[pb.GetCustomerResponse], error) {
	c, err := s.store.GetByID(ctx, int(req.Msg.Id))
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("customer %d not found", req.Msg.Id))
	}
	return connect.NewResponse(&pb.GetCustomerResponse{Customer: customerToProto(c)}), nil
}

func (s *CustomerServer) CreateCustomer(ctx context.Context, req *connect.Request[pb.CreateCustomerRequest]) (*connect.Response[pb.CreateCustomerResponse], error) {
	if req.Msg.Name == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("name is required"))
	}

	c := createCustomerReqToModel(req.Msg)
	if err := s.store.Create(ctx, c); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("create customer: %w", err))
	}

	s.audit.Log(ctx, "customers", c.ID, "INSERT", nil, c)

	return connect.NewResponse(&pb.CreateCustomerResponse{Customer: customerToProto(c)}), nil
}

func (s *CustomerServer) UpdateCustomer(ctx context.Context, req *connect.Request[pb.UpdateCustomerRequest]) (*connect.Response[pb.UpdateCustomerResponse], error) {
	if req.Msg.Name == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("name is required"))
	}

	old, err := s.store.GetByID(ctx, int(req.Msg.Id))
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("customer %d not found", req.Msg.Id))
	}

	c := updateCustomerReqToModel(req.Msg)
	if err := s.store.Update(ctx, c); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("update customer: %w", err))
	}

	s.audit.Log(ctx, "customers", c.ID, "UPDATE", old, c)

	updated, err := s.store.GetByID(ctx, c.ID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("fetch updated customer: %w", err))
	}
	return connect.NewResponse(&pb.UpdateCustomerResponse{Customer: customerToProto(updated)}), nil
}

func (s *CustomerServer) DeleteCustomer(ctx context.Context, req *connect.Request[pb.DeleteCustomerRequest]) (*connect.Response[pb.DeleteCustomerResponse], error) {
	if err := s.store.Delete(ctx, int(req.Msg.Id)); err != nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("customer %d not found", req.Msg.Id))
	}
	s.audit.Log(ctx, "customers", int(req.Msg.Id), "DELETE", nil, nil)
	return connect.NewResponse(&pb.DeleteCustomerResponse{Success: true}), nil
}
