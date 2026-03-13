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

type zoneStore interface {
	List(ctx context.Context) ([]models.Zone, error)
	GetByID(ctx context.Context, id int) (*models.Zone, error)
	Create(ctx context.Context, z *models.Zone) error
	Update(ctx context.Context, z *models.Zone) error
	Delete(ctx context.Context, id int) error
}

type zonePricingStore interface {
	List(ctx context.Context) ([]models.ZonePricing, error)
	GetByID(ctx context.Context, id int) (*models.ZonePricing, error)
	Create(ctx context.Context, zp *models.ZonePricing) error
	Update(ctx context.Context, zp *models.ZonePricing) error
	Delete(ctx context.Context, id int) error
}

type ZoneServer struct {
	atlinkspbconnect.UnimplementedZoneServiceHandler
	zones   zoneStore
	pricing zonePricingStore
	audit   *audit.Service
}

func NewZoneServer(zones zoneStore, pricing zonePricingStore, audit *audit.Service) *ZoneServer {
	return &ZoneServer{zones: zones, pricing: pricing, audit: audit}
}

// --- Zone RPCs ---

func (s *ZoneServer) ListZones(ctx context.Context, req *connect.Request[pb.ListZonesRequest]) (*connect.Response[pb.ListZonesResponse], error) {
	items, err := s.zones.List(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("list zones: %w", err))
	}

	zones := make([]*pb.Zone, len(items))
	for i := range items {
		zones[i] = zoneToProto(&items[i])
	}

	return connect.NewResponse(&pb.ListZonesResponse{Zones: zones}), nil
}

func (s *ZoneServer) GetZone(ctx context.Context, req *connect.Request[pb.GetZoneRequest]) (*connect.Response[pb.GetZoneResponse], error) {
	z, err := s.zones.GetByID(ctx, int(req.Msg.Id))
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("zone %d not found", req.Msg.Id))
	}
	return connect.NewResponse(&pb.GetZoneResponse{Zone: zoneToProto(z)}), nil
}

func (s *ZoneServer) CreateZone(ctx context.Context, req *connect.Request[pb.CreateZoneRequest]) (*connect.Response[pb.CreateZoneResponse], error) {
	if req.Msg.Zone == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("zone is required"))
	}

	z := createZoneReqToModel(req.Msg)
	if err := s.zones.Create(ctx, z); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("create zone: %w", err))
	}

	s.audit.Log(ctx, "zones", z.ID, "INSERT", nil, z)

	return connect.NewResponse(&pb.CreateZoneResponse{Zone: zoneToProto(z)}), nil
}

func (s *ZoneServer) UpdateZone(ctx context.Context, req *connect.Request[pb.UpdateZoneRequest]) (*connect.Response[pb.UpdateZoneResponse], error) {
	if req.Msg.Zone == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("zone is required"))
	}

	old, err := s.zones.GetByID(ctx, int(req.Msg.Id))
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("zone %d not found", req.Msg.Id))
	}

	z := updateZoneReqToModel(req.Msg)
	if err := s.zones.Update(ctx, z); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("update zone: %w", err))
	}

	s.audit.Log(ctx, "zones", z.ID, "UPDATE", old, z)

	updated, err := s.zones.GetByID(ctx, z.ID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("fetch updated zone: %w", err))
	}
	return connect.NewResponse(&pb.UpdateZoneResponse{Zone: zoneToProto(updated)}), nil
}

func (s *ZoneServer) DeleteZone(ctx context.Context, req *connect.Request[pb.DeleteZoneRequest]) (*connect.Response[pb.DeleteZoneResponse], error) {
	if err := s.zones.Delete(ctx, int(req.Msg.Id)); err != nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("zone %d not found", req.Msg.Id))
	}
	s.audit.Log(ctx, "zones", int(req.Msg.Id), "DELETE", nil, nil)
	return connect.NewResponse(&pb.DeleteZoneResponse{Success: true}), nil
}

// --- ZonePricing RPCs ---

func (s *ZoneServer) ListZonePricing(ctx context.Context, req *connect.Request[pb.ListZonePricingRequest]) (*connect.Response[pb.ListZonePricingResponse], error) {
	items, err := s.pricing.List(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("list zone pricing: %w", err))
	}

	pricing := make([]*pb.ZonePricing, len(items))
	for i := range items {
		pricing[i] = zonePricingToProto(&items[i])
	}

	return connect.NewResponse(&pb.ListZonePricingResponse{Items: pricing}), nil
}

func (s *ZoneServer) GetZonePricing(ctx context.Context, req *connect.Request[pb.GetZonePricingRequest]) (*connect.Response[pb.GetZonePricingResponse], error) {
	zp, err := s.pricing.GetByID(ctx, int(req.Msg.Id))
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("zone pricing %d not found", req.Msg.Id))
	}
	return connect.NewResponse(&pb.GetZonePricingResponse{Item: zonePricingToProto(zp)}), nil
}

func (s *ZoneServer) CreateZonePricing(ctx context.Context, req *connect.Request[pb.CreateZonePricingRequest]) (*connect.Response[pb.CreateZonePricingResponse], error) {
	if req.Msg.ZoneA == "" || req.Msg.ZoneB == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("zone_a and zone_b are required"))
	}

	zp := createZonePricingReqToModel(req.Msg)
	if err := s.pricing.Create(ctx, zp); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("create zone pricing: %w", err))
	}

	s.audit.Log(ctx, "zone_pricing", zp.ID, "INSERT", nil, zp)

	return connect.NewResponse(&pb.CreateZonePricingResponse{Item: zonePricingToProto(zp)}), nil
}

func (s *ZoneServer) UpdateZonePricing(ctx context.Context, req *connect.Request[pb.UpdateZonePricingRequest]) (*connect.Response[pb.UpdateZonePricingResponse], error) {
	if req.Msg.ZoneA == "" || req.Msg.ZoneB == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("zone_a and zone_b are required"))
	}

	old, err := s.pricing.GetByID(ctx, int(req.Msg.Id))
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("zone pricing %d not found", req.Msg.Id))
	}

	zp := updateZonePricingReqToModel(req.Msg)
	if err := s.pricing.Update(ctx, zp); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("update zone pricing: %w", err))
	}

	s.audit.Log(ctx, "zone_pricing", zp.ID, "UPDATE", old, zp)

	updated, err := s.pricing.GetByID(ctx, zp.ID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("fetch updated zone pricing: %w", err))
	}
	return connect.NewResponse(&pb.UpdateZonePricingResponse{Item: zonePricingToProto(updated)}), nil
}

func (s *ZoneServer) DeleteZonePricing(ctx context.Context, req *connect.Request[pb.DeleteZonePricingRequest]) (*connect.Response[pb.DeleteZonePricingResponse], error) {
	if err := s.pricing.Delete(ctx, int(req.Msg.Id)); err != nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("zone pricing %d not found", req.Msg.Id))
	}
	s.audit.Log(ctx, "zone_pricing", int(req.Msg.Id), "DELETE", nil, nil)
	return connect.NewResponse(&pb.DeleteZonePricingResponse{Success: true}), nil
}

// --- Zone converters ---

func zoneToProto(z *models.Zone) *pb.Zone {
	return &pb.Zone{
		Id:          int32(z.ID),
		Zone:        z.Zone,
		Description: sp(z.Description),
		Region:      sp(z.Region),
		CreatedAt:   z.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   z.UpdatedAt.Format(time.RFC3339),
	}
}

func createZoneReqToModel(msg *pb.CreateZoneRequest) *models.Zone {
	return &models.Zone{
		Zone:        msg.Zone,
		Description: sp(msg.Description),
		Region:      sp(msg.Region),
	}
}

func updateZoneReqToModel(msg *pb.UpdateZoneRequest) *models.Zone {
	return &models.Zone{
		ID:          int(msg.Id),
		Zone:        msg.Zone,
		Description: sp(msg.Description),
		Region:      sp(msg.Region),
	}
}

// --- ZonePricing converters ---

func zonePricingToProto(zp *models.ZonePricing) *pb.ZonePricing {
	return &pb.ZonePricing{
		Id:            int32(zp.ID),
		ZoneA:         zp.ZoneA,
		ZoneB:         zp.ZoneB,
		Description:   sp(zp.Description),
		Amount:        sp(zp.Amount),
		Miles:         ip(zp.Miles),
		TransportDays: ip(zp.TransportDays),
		ShipTo:        sp(zp.ShipTo),
		CreatedAt:     zp.CreatedAt.Format(time.RFC3339),
		UpdatedAt:     zp.UpdatedAt.Format(time.RFC3339),
	}
}

func createZonePricingReqToModel(msg *pb.CreateZonePricingRequest) *models.ZonePricing {
	return &models.ZonePricing{
		ZoneA:         msg.ZoneA,
		ZoneB:         msg.ZoneB,
		Description:   sp(msg.Description),
		Amount:        sp(msg.Amount),
		Miles:         i32p(msg.Miles),
		TransportDays: i32p(msg.TransportDays),
		ShipTo:        sp(msg.ShipTo),
	}
}

func updateZonePricingReqToModel(msg *pb.UpdateZonePricingRequest) *models.ZonePricing {
	return &models.ZonePricing{
		ID:            int(msg.Id),
		ZoneA:         msg.ZoneA,
		ZoneB:         msg.ZoneB,
		Description:   sp(msg.Description),
		Amount:        sp(msg.Amount),
		Miles:         i32p(msg.Miles),
		TransportDays: i32p(msg.TransportDays),
		ShipTo:        sp(msg.ShipTo),
	}
}
