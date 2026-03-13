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

type vendorStore interface {
	List(ctx context.Context, f models.VendorFilter) (*models.VendorListResult, error)
	GetByID(ctx context.Context, id int) (*models.Vendor, error)
	Create(ctx context.Context, v *models.Vendor) error
	Update(ctx context.Context, v *models.Vendor) error
	Delete(ctx context.Context, id int) error
}

type VendorServer struct {
	atlinkspbconnect.UnimplementedVendorServiceHandler
	store vendorStore
	audit *audit.Service
}

func NewVendorServer(store vendorStore, audit *audit.Service) *VendorServer {
	return &VendorServer{store: store, audit: audit}
}

func (s *VendorServer) ListVendors(ctx context.Context, req *connect.Request[pb.ListVendorsRequest]) (*connect.Response[pb.ListVendorsResponse], error) {
	filter := protoToVendorFilter(req.Msg)
	result, err := s.store.List(ctx, filter)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("list vendors: %w", err))
	}

	vendors := make([]*pb.Vendor, len(result.Items))
	for i := range result.Items {
		vendors[i] = vendorToProto(&result.Items[i])
	}

	return connect.NewResponse(&pb.ListVendorsResponse{
		Vendors: vendors,
		Pagination: &pb.PaginationResponse{
			TotalCount: int32(result.TotalCount),
			Page:       int32(result.Page),
			PageSize:   int32(result.PageSize),
		},
	}), nil
}

func (s *VendorServer) GetVendor(ctx context.Context, req *connect.Request[pb.GetVendorRequest]) (*connect.Response[pb.GetVendorResponse], error) {
	v, err := s.store.GetByID(ctx, int(req.Msg.Id))
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("vendor %d not found", req.Msg.Id))
	}
	return connect.NewResponse(&pb.GetVendorResponse{Vendor: vendorToProto(v)}), nil
}

func (s *VendorServer) CreateVendor(ctx context.Context, req *connect.Request[pb.CreateVendorRequest]) (*connect.Response[pb.CreateVendorResponse], error) {
	if req.Msg.Name == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("name is required"))
	}

	v := createVendorReqToModel(req.Msg)
	if err := s.store.Create(ctx, v); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("create vendor: %w", err))
	}

	s.audit.Log(ctx, "vendors", v.ID, "INSERT", nil, v)

	return connect.NewResponse(&pb.CreateVendorResponse{Vendor: vendorToProto(v)}), nil
}

func (s *VendorServer) UpdateVendor(ctx context.Context, req *connect.Request[pb.UpdateVendorRequest]) (*connect.Response[pb.UpdateVendorResponse], error) {
	if req.Msg.Name == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("name is required"))
	}

	old, err := s.store.GetByID(ctx, int(req.Msg.Id))
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("vendor %d not found", req.Msg.Id))
	}

	v := updateVendorReqToModel(req.Msg)
	if err := s.store.Update(ctx, v); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("update vendor: %w", err))
	}

	s.audit.Log(ctx, "vendors", v.ID, "UPDATE", old, v)

	updated, err := s.store.GetByID(ctx, v.ID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("fetch updated vendor: %w", err))
	}
	return connect.NewResponse(&pb.UpdateVendorResponse{Vendor: vendorToProto(updated)}), nil
}

func (s *VendorServer) DeleteVendor(ctx context.Context, req *connect.Request[pb.DeleteVendorRequest]) (*connect.Response[pb.DeleteVendorResponse], error) {
	if err := s.store.Delete(ctx, int(req.Msg.Id)); err != nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("vendor %d not found", req.Msg.Id))
	}
	s.audit.Log(ctx, "vendors", int(req.Msg.Id), "DELETE", nil, nil)
	return connect.NewResponse(&pb.DeleteVendorResponse{Success: true}), nil
}

// --- Vendor converters ---

func vendorToProto(v *models.Vendor) *pb.Vendor {
	return &pb.Vendor{
		Id:        int32(v.ID),
		Name:      v.Name,
		Address:   sp(v.Address),
		Address2:  sp(v.Address2),
		City:      sp(v.City),
		State:     sp(v.State),
		Zip:       sp(v.Zip),
		Phone:     sp(v.Phone),
		Fax:       sp(v.Fax),
		Contact:   sp(v.Contact),
		Terms:     sp(v.Terms),
		TaxId:     sp(v.TaxID),
		CreatedAt: v.CreatedAt.Format(time.RFC3339),
		UpdatedAt: v.UpdatedAt.Format(time.RFC3339),
	}
}

func protoToVendorFilter(msg *pb.ListVendorsRequest) models.VendorFilter {
	f := models.VendorFilter{}
	if msg.Pagination != nil {
		f.Page = int(msg.Pagination.Page)
		f.PageSize = int(msg.Pagination.PageSize)
	}
	if msg.Search != nil {
		f.Search = *msg.Search
	}
	return f
}

func createVendorReqToModel(msg *pb.CreateVendorRequest) *models.Vendor {
	return &models.Vendor{
		Name:    msg.Name,
		Address: sp(msg.Address),
		Address2: sp(msg.Address2),
		City:    sp(msg.City),
		State:   sp(msg.State),
		Zip:     sp(msg.Zip),
		Phone:   sp(msg.Phone),
		Fax:     sp(msg.Fax),
		Contact: sp(msg.Contact),
		Terms:   sp(msg.Terms),
		TaxID:   sp(msg.TaxId),
	}
}

func updateVendorReqToModel(msg *pb.UpdateVendorRequest) *models.Vendor {
	return &models.Vendor{
		ID:       int(msg.Id),
		Name:     msg.Name,
		Address:  sp(msg.Address),
		Address2: sp(msg.Address2),
		City:     sp(msg.City),
		State:    sp(msg.State),
		Zip:      sp(msg.Zip),
		Phone:    sp(msg.Phone),
		Fax:      sp(msg.Fax),
		Contact:  sp(msg.Contact),
		Terms:    sp(msg.Terms),
		TaxID:    sp(msg.TaxId),
	}
}
