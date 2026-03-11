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

type orderStore interface {
	List(ctx context.Context, f models.OrderFilter) (*models.OrderListResult, error)
	GetByID(ctx context.Context, id int) (*models.Order, error)
	Create(ctx context.Context, o *models.Order) error
	Update(ctx context.Context, o *models.Order) error
	Delete(ctx context.Context, id int) error
	NextOrderNumber(ctx context.Context) (string, error)
}

type OrderServer struct {
	atlinkspbconnect.UnimplementedOrderServiceHandler
	store orderStore
	audit *audit.Service
}

func NewOrderServer(store orderStore, audit *audit.Service) *OrderServer {
	return &OrderServer{store: store, audit: audit}
}

func (s *OrderServer) ListOrders(ctx context.Context, req *connect.Request[pb.ListOrdersRequest]) (*connect.Response[pb.ListOrdersResponse], error) {
	filter := protoToOrderFilter(req.Msg)
	result, err := s.store.List(ctx, filter)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("list orders: %w", err))
	}

	orders := make([]*pb.Order, len(result.Items))
	for i := range result.Items {
		orders[i] = orderToProto(&result.Items[i])
	}

	return connect.NewResponse(&pb.ListOrdersResponse{
		Orders: orders,
		Pagination: &pb.PaginationResponse{
			TotalCount: int32(result.TotalCount),
			Page:       int32(result.Page),
			PageSize:   int32(result.PageSize),
		},
	}), nil
}

func (s *OrderServer) GetOrder(ctx context.Context, req *connect.Request[pb.GetOrderRequest]) (*connect.Response[pb.GetOrderResponse], error) {
	o, err := s.store.GetByID(ctx, int(req.Msg.Id))
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("order %d not found", req.Msg.Id))
	}
	return connect.NewResponse(&pb.GetOrderResponse{Order: orderToProto(o)}), nil
}

func (s *OrderServer) CreateOrder(ctx context.Context, req *connect.Request[pb.CreateOrderRequest]) (*connect.Response[pb.CreateOrderResponse], error) {
	o := createOrderReqToModel(req.Msg)

	// Auto-generate order number if not provided
	if o.OrderNumber == "" {
		num, err := s.store.NextOrderNumber(ctx)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("generate order number: %w", err))
		}
		o.OrderNumber = num
	}

	if err := s.store.Create(ctx, o); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("create order: %w", err))
	}

	s.audit.Log(ctx, "orders", o.ID, "INSERT", nil, o)

	created, err := s.store.GetByID(ctx, o.ID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("fetch created order: %w", err))
	}
	return connect.NewResponse(&pb.CreateOrderResponse{Order: orderToProto(created)}), nil
}

func (s *OrderServer) UpdateOrder(ctx context.Context, req *connect.Request[pb.UpdateOrderRequest]) (*connect.Response[pb.UpdateOrderResponse], error) {
	old, err := s.store.GetByID(ctx, int(req.Msg.Id))
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("order %d not found", req.Msg.Id))
	}

	o := updateOrderReqToModel(req.Msg)
	o.OrderNumber = old.OrderNumber // order number is immutable
	if err := s.store.Update(ctx, o); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("update order: %w", err))
	}

	s.audit.Log(ctx, "orders", o.ID, "UPDATE", old, o)

	updated, err := s.store.GetByID(ctx, o.ID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("fetch updated order: %w", err))
	}
	return connect.NewResponse(&pb.UpdateOrderResponse{Order: orderToProto(updated)}), nil
}

func (s *OrderServer) DeleteOrder(ctx context.Context, req *connect.Request[pb.DeleteOrderRequest]) (*connect.Response[pb.DeleteOrderResponse], error) {
	if err := s.store.Delete(ctx, int(req.Msg.Id)); err != nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("order %d not found", req.Msg.Id))
	}
	s.audit.Log(ctx, "orders", int(req.Msg.Id), "DELETE", nil, nil)
	return connect.NewResponse(&pb.DeleteOrderResponse{Success: true}), nil
}
