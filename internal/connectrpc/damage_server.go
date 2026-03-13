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

type damageStore interface {
	ListByVehicle(ctx context.Context, vehicleID int) ([]models.VehicleDamage, error)
	ListByTrip(ctx context.Context, tripID int) ([]models.VehicleDamage, error)
	GetByID(ctx context.Context, id int) (*models.VehicleDamage, error)
	Create(ctx context.Context, d *models.VehicleDamage) error
	Update(ctx context.Context, d *models.VehicleDamage) error
	Delete(ctx context.Context, id int) error
}

type noteStore interface {
	ListByVehicle(ctx context.Context, vehicleID int) ([]models.VehicleNote, error)
	Create(ctx context.Context, n *models.VehicleNote) error
	Delete(ctx context.Context, id int) error
}

type DamageServer struct {
	atlinkspbconnect.UnimplementedDamageServiceHandler
	damages damageStore
	notes   noteStore
	audit   *audit.Service
}

func NewDamageServer(damages damageStore, notes noteStore, audit *audit.Service) *DamageServer {
	return &DamageServer{damages: damages, notes: notes, audit: audit}
}

func (s *DamageServer) ListDamagesByVehicle(ctx context.Context, req *connect.Request[pb.ListDamagesByVehicleRequest]) (*connect.Response[pb.ListDamagesResponse], error) {
	items, err := s.damages.ListByVehicle(ctx, int(req.Msg.VehicleId))
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("list damages by vehicle: %w", err))
	}

	damages := make([]*pb.VehicleDamage, len(items))
	for i := range items {
		damages[i] = damageToProto(&items[i])
	}

	return connect.NewResponse(&pb.ListDamagesResponse{Damages: damages}), nil
}

func (s *DamageServer) ListDamagesByTrip(ctx context.Context, req *connect.Request[pb.ListDamagesByTripRequest]) (*connect.Response[pb.ListDamagesResponse], error) {
	items, err := s.damages.ListByTrip(ctx, int(req.Msg.TripId))
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("list damages by trip: %w", err))
	}

	damages := make([]*pb.VehicleDamage, len(items))
	for i := range items {
		damages[i] = damageToProto(&items[i])
	}

	return connect.NewResponse(&pb.ListDamagesResponse{Damages: damages}), nil
}

func (s *DamageServer) GetDamage(ctx context.Context, req *connect.Request[pb.GetDamageRequest]) (*connect.Response[pb.GetDamageResponse], error) {
	d, err := s.damages.GetByID(ctx, int(req.Msg.Id))
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("damage %d not found", req.Msg.Id))
	}
	return connect.NewResponse(&pb.GetDamageResponse{Damage: damageToProto(d)}), nil
}

func (s *DamageServer) CreateDamage(ctx context.Context, req *connect.Request[pb.CreateDamageRequest]) (*connect.Response[pb.CreateDamageResponse], error) {
	if req.Msg.OrderId == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("order_id is required"))
	}
	if req.Msg.VehicleId == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("vehicle_id is required"))
	}

	d := createDamageReqToModel(req.Msg)
	if err := s.damages.Create(ctx, d); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("create damage: %w", err))
	}

	s.audit.Log(ctx, "vehicle_damages", d.ID, "INSERT", nil, d)

	return connect.NewResponse(&pb.CreateDamageResponse{Damage: damageToProto(d)}), nil
}

func (s *DamageServer) UpdateDamage(ctx context.Context, req *connect.Request[pb.UpdateDamageRequest]) (*connect.Response[pb.UpdateDamageResponse], error) {
	old, err := s.damages.GetByID(ctx, int(req.Msg.Id))
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("damage %d not found", req.Msg.Id))
	}

	d := updateDamageReqToModel(req.Msg)
	if err := s.damages.Update(ctx, d); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("update damage: %w", err))
	}

	s.audit.Log(ctx, "vehicle_damages", d.ID, "UPDATE", old, d)

	updated, err := s.damages.GetByID(ctx, d.ID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("fetch updated damage: %w", err))
	}
	return connect.NewResponse(&pb.UpdateDamageResponse{Damage: damageToProto(updated)}), nil
}

func (s *DamageServer) DeleteDamage(ctx context.Context, req *connect.Request[pb.DeleteDamageRequest]) (*connect.Response[pb.DeleteDamageResponse], error) {
	if err := s.damages.Delete(ctx, int(req.Msg.Id)); err != nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("damage %d not found", req.Msg.Id))
	}
	s.audit.Log(ctx, "vehicle_damages", int(req.Msg.Id), "DELETE", nil, nil)
	return connect.NewResponse(&pb.DeleteDamageResponse{Success: true}), nil
}

func (s *DamageServer) ListNotesByVehicle(ctx context.Context, req *connect.Request[pb.ListNotesByVehicleRequest]) (*connect.Response[pb.ListNotesResponse], error) {
	items, err := s.notes.ListByVehicle(ctx, int(req.Msg.VehicleId))
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("list notes: %w", err))
	}

	notes := make([]*pb.VehicleNote, len(items))
	for i := range items {
		notes[i] = noteToProto(&items[i])
	}

	return connect.NewResponse(&pb.ListNotesResponse{Notes: notes}), nil
}

func (s *DamageServer) CreateNote(ctx context.Context, req *connect.Request[pb.CreateNoteRequest]) (*connect.Response[pb.CreateNoteResponse], error) {
	if req.Msg.VehicleId == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("vehicle_id is required"))
	}

	n := createNoteReqToModel(req.Msg)

	// Populate created_by from auth context if available
	if user, ok := auth.GetUser(ctx); ok {
		n.CreatedBy = &user.Username
	}

	if err := s.notes.Create(ctx, n); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("create note: %w", err))
	}

	s.audit.Log(ctx, "vehicle_notes", n.ID, "INSERT", nil, n)

	return connect.NewResponse(&pb.CreateNoteResponse{Note: noteToProto(n)}), nil
}

func (s *DamageServer) DeleteNote(ctx context.Context, req *connect.Request[pb.DeleteNoteRequest]) (*connect.Response[pb.DeleteNoteResponse], error) {
	if err := s.notes.Delete(ctx, int(req.Msg.Id)); err != nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("note %d not found", req.Msg.Id))
	}
	s.audit.Log(ctx, "vehicle_notes", int(req.Msg.Id), "DELETE", nil, nil)
	return connect.NewResponse(&pb.DeleteNoteResponse{Success: true}), nil
}

func damageToProto(d *models.VehicleDamage) *pb.VehicleDamage {
	var orderID, vehicleID int32
	if d.OrderID != nil {
		orderID = int32(*d.OrderID)
	}
	if d.VehicleID != nil {
		vehicleID = int32(*d.VehicleID)
	}
	return &pb.VehicleDamage{
		Id:              int32(d.ID),
		OrderId:         orderID,
		VehicleId:       vehicleID,
		TripId:          ip(d.TripID),
		Vin:             sp(d.VIN),
		DamageArea:      sp(d.DamageArea),
		DamageType:      sp(d.DamageType),
		DamageSeverity:  sp(d.DamageSeverity),
		Description:     sp(d.Description),
		InspectionPoint: sp(d.InspectionPoint),
		InspectedBy:     sp(d.InspectedBy),
		InspectionDate:  timeStr(d.InspectionDate),
		ClaimAmount:     sp(d.ClaimAmount),
		ClaimStatus:     sp(d.ClaimStatus),
		CreatedAt:       d.CreatedAt.Format(time.RFC3339),
		UpdatedAt:       d.UpdatedAt.Format(time.RFC3339),
	}
}

func noteToProto(n *models.VehicleNote) *pb.VehicleNote {
	return &pb.VehicleNote{
		Id:          int32(n.ID),
		VehicleId:   int32(n.VehicleID),
		NoteDate:    timeStr(n.NoteDate),
		Description: sp(n.Description),
		Comment:     sp(n.Comment),
		CreatedBy:   sp(n.CreatedBy),
		CreatedAt:   n.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   n.UpdatedAt.Format(time.RFC3339),
	}
}

func createDamageReqToModel(msg *pb.CreateDamageRequest) *models.VehicleDamage {
	orderID := int(msg.OrderId)
	vehicleID := int(msg.VehicleId)
	return &models.VehicleDamage{
		OrderID:         &orderID,
		VehicleID:       &vehicleID,
		TripID:          i32p(msg.TripId),
		VIN:             sp(msg.Vin),
		DamageArea:      sp(msg.DamageArea),
		DamageType:      sp(msg.DamageType),
		DamageSeverity:  sp(msg.DamageSeverity),
		Description:     sp(msg.Description),
		InspectionPoint: sp(msg.InspectionPoint),
		InspectedBy:     sp(msg.InspectedBy),
		InspectionDate:  parseDate(msg.InspectionDate),
		ClaimAmount:     sp(msg.ClaimAmount),
		ClaimStatus:     sp(msg.ClaimStatus),
	}
}

func updateDamageReqToModel(msg *pb.UpdateDamageRequest) *models.VehicleDamage {
	orderID := int(msg.OrderId)
	vehicleID := int(msg.VehicleId)
	return &models.VehicleDamage{
		ID:              int(msg.Id),
		OrderID:         &orderID,
		VehicleID:       &vehicleID,
		TripID:          i32p(msg.TripId),
		VIN:             sp(msg.Vin),
		DamageArea:      sp(msg.DamageArea),
		DamageType:      sp(msg.DamageType),
		DamageSeverity:  sp(msg.DamageSeverity),
		Description:     sp(msg.Description),
		InspectionPoint: sp(msg.InspectionPoint),
		InspectedBy:     sp(msg.InspectedBy),
		InspectionDate:  parseDate(msg.InspectionDate),
		ClaimAmount:     sp(msg.ClaimAmount),
		ClaimStatus:     sp(msg.ClaimStatus),
	}
}

func createNoteReqToModel(msg *pb.CreateNoteRequest) *models.VehicleNote {
	return &models.VehicleNote{
		VehicleID:   int(msg.VehicleId),
		NoteDate:    parseDate(msg.NoteDate),
		Description: sp(msg.Description),
		Comment:     sp(msg.Comment),
	}
}
