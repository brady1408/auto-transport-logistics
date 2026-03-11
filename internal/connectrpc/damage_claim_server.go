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

type damageClaimStore interface {
	List(ctx context.Context, f models.DamageClaimFilter) (*models.DamageClaimListResult, error)
	GetByID(ctx context.Context, id int) (*models.DamageClaim, error)
	Create(ctx context.Context, dc *models.DamageClaim) error
	Update(ctx context.Context, dc *models.DamageClaim) error
	Delete(ctx context.Context, id int) error
	NextClaimNumber(ctx context.Context) (string, error)
}

type DamageClaimServer struct {
	atlinkspbconnect.UnimplementedDamageClaimServiceHandler
	store damageClaimStore
	audit *audit.Service
}

func NewDamageClaimServer(store damageClaimStore, audit *audit.Service) *DamageClaimServer {
	return &DamageClaimServer{store: store, audit: audit}
}

func (s *DamageClaimServer) ListDamageClaims(ctx context.Context, req *connect.Request[pb.ListDamageClaimsRequest]) (*connect.Response[pb.ListDamageClaimsResponse], error) {
	filter := protoToDamageClaimFilter(req.Msg)
	result, err := s.store.List(ctx, filter)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("list damage claims: %w", err))
	}

	claims := make([]*pb.DamageClaim, len(result.Items))
	for i := range result.Items {
		claims[i] = damageClaimToProto(&result.Items[i])
	}

	return connect.NewResponse(&pb.ListDamageClaimsResponse{
		Claims: claims,
		Pagination: &pb.PaginationResponse{
			TotalCount: int32(result.TotalCount),
			Page:       int32(result.Page),
			PageSize:   int32(result.PageSize),
		},
	}), nil
}

func (s *DamageClaimServer) GetDamageClaim(ctx context.Context, req *connect.Request[pb.GetDamageClaimRequest]) (*connect.Response[pb.GetDamageClaimResponse], error) {
	dc, err := s.store.GetByID(ctx, int(req.Msg.Id))
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("damage claim %d not found", req.Msg.Id))
	}
	return connect.NewResponse(&pb.GetDamageClaimResponse{Claim: damageClaimToProto(dc)}), nil
}

func (s *DamageClaimServer) CreateDamageClaim(ctx context.Context, req *connect.Request[pb.CreateDamageClaimRequest]) (*connect.Response[pb.CreateDamageClaimResponse], error) {
	num, err := s.store.NextClaimNumber(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("generate claim number: %w", err))
	}

	dc := createDamageClaimReqToModel(req.Msg)
	dc.ClaimNumber = num

	if err := s.store.Create(ctx, dc); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("create damage claim: %w", err))
	}

	s.audit.Log(ctx, "damage_claims", dc.ID, "INSERT", nil, dc)

	created, err := s.store.GetByID(ctx, dc.ID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("fetch created damage claim: %w", err))
	}
	return connect.NewResponse(&pb.CreateDamageClaimResponse{Claim: damageClaimToProto(created)}), nil
}

func (s *DamageClaimServer) UpdateDamageClaim(ctx context.Context, req *connect.Request[pb.UpdateDamageClaimRequest]) (*connect.Response[pb.UpdateDamageClaimResponse], error) {
	old, err := s.store.GetByID(ctx, int(req.Msg.Id))
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("damage claim %d not found", req.Msg.Id))
	}

	dc := updateDamageClaimReqToModel(req.Msg)
	dc.ClaimNumber = old.ClaimNumber // claim number is immutable
	if err := s.store.Update(ctx, dc); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("update damage claim: %w", err))
	}

	s.audit.Log(ctx, "damage_claims", dc.ID, "UPDATE", old, dc)

	updated, err := s.store.GetByID(ctx, dc.ID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("fetch updated damage claim: %w", err))
	}
	return connect.NewResponse(&pb.UpdateDamageClaimResponse{Claim: damageClaimToProto(updated)}), nil
}

func (s *DamageClaimServer) DeleteDamageClaim(ctx context.Context, req *connect.Request[pb.DeleteDamageClaimRequest]) (*connect.Response[pb.DeleteDamageClaimResponse], error) {
	if err := s.store.Delete(ctx, int(req.Msg.Id)); err != nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("damage claim %d not found", req.Msg.Id))
	}
	s.audit.Log(ctx, "damage_claims", int(req.Msg.Id), "DELETE", nil, nil)
	return connect.NewResponse(&pb.DeleteDamageClaimResponse{Success: true}), nil
}

func damageClaimToProto(dc *models.DamageClaim) *pb.DamageClaim {
	return &pb.DamageClaim{
		Id:                   int32(dc.ID),
		ClaimNumber:          dc.ClaimNumber,
		OrderId:              ip(dc.OrderID),
		VehicleId:            ip(dc.VehicleID),
		TripId:               ip(dc.TripID),
		Vin:                  sp(dc.VIN),
		ClaimDate:            timeStr(dc.ClaimDate),
		ClaimAmount:          sp(dc.ClaimAmount),
		PaidAmount:           sp(dc.PaidAmount),
		Status:               sp(dc.Status),
		Description:          sp(dc.Description),
		InsuranceClaim:       boolToOptString(dc.InsuranceClaim),
		InsuranceClaimNumber: sp(dc.InsuranceClaimNumber),
		Resolution:           sp(dc.Resolution),
		ResolvedDate:         timeStr(dc.ResolvedDate),
		CreatedAt:            dc.CreatedAt.Format(time.RFC3339),
		UpdatedAt:            dc.UpdatedAt.Format(time.RFC3339),
	}
}

func protoToDamageClaimFilter(msg *pb.ListDamageClaimsRequest) models.DamageClaimFilter {
	f := models.DamageClaimFilter{}
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
	return f
}

func createDamageClaimReqToModel(msg *pb.CreateDamageClaimRequest) *models.DamageClaim {
	return &models.DamageClaim{
		OrderID:              i32p(msg.OrderId),
		VehicleID:            i32p(msg.VehicleId),
		TripID:               i32p(msg.TripId),
		VIN:                  sp(msg.Vin),
		ClaimDate:            parseDate(msg.ClaimDate),
		ClaimAmount:          sp(msg.ClaimAmount),
		Status:               sp(msg.Status),
		Description:          sp(msg.Description),
		InsuranceClaim:       optStringToBool(msg.InsuranceClaim),
		InsuranceClaimNumber: sp(msg.InsuranceClaimNumber),
	}
}

func updateDamageClaimReqToModel(msg *pb.UpdateDamageClaimRequest) *models.DamageClaim {
	return &models.DamageClaim{
		ID:                   int(msg.Id),
		OrderID:              i32p(msg.OrderId),
		VehicleID:            i32p(msg.VehicleId),
		TripID:               i32p(msg.TripId),
		VIN:                  sp(msg.Vin),
		ClaimDate:            parseDate(msg.ClaimDate),
		ClaimAmount:          sp(msg.ClaimAmount),
		PaidAmount:           sp(msg.PaidAmount),
		Status:               sp(msg.Status),
		Description:          sp(msg.Description),
		InsuranceClaim:       optStringToBool(msg.InsuranceClaim),
		InsuranceClaimNumber: sp(msg.InsuranceClaimNumber),
		Resolution:           sp(msg.Resolution),
		ResolvedDate:         parseDate(msg.ResolvedDate),
	}
}
