package connectrpc

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	"github.com/brady1408/auto-transport-logistics/internal/audit"
	pb "github.com/brady1408/auto-transport-logistics/internal/gen/atlinks/v1"
	"github.com/brady1408/auto-transport-logistics/internal/gen/atlinks/v1/atlinkspbconnect"
	"github.com/brady1408/auto-transport-logistics/internal/store"
)

// Store interfaces for dependency injection.

type lookupStoreI interface {
	List(ctx context.Context) ([]store.LookupItem, error)
	GetByID(ctx context.Context, id int) (*store.LookupItem, error)
	Create(ctx context.Context, code, description string) (*store.LookupItem, error)
	Update(ctx context.Context, id int, code, description string) error
	Delete(ctx context.Context, id int) error
}

type termsStoreI interface {
	List(ctx context.Context) ([]store.TermItem, error)
	Create(ctx context.Context, term, description string, days *int) (*store.TermItem, error)
	Update(ctx context.Context, id int, term, description string, days *int) error
	Delete(ctx context.Context, id int) error
}

type taxCodeStoreI interface {
	List(ctx context.Context) ([]store.TaxCodeItem, error)
	Create(ctx context.Context, code, description string, rate *string) (*store.TaxCodeItem, error)
	Update(ctx context.Context, id int, code, description string, rate *string) error
	Delete(ctx context.Context, id int) error
}

type itemStoreI interface {
	List(ctx context.Context) ([]store.ItemRecord, error)
	Create(ctx context.Context, item, description string, defaultAmount, calcType *string) (*store.ItemRecord, error)
	Update(ctx context.Context, id int, item, description string, defaultAmount, calcType *string) error
	Delete(ctx context.Context, id int) error
}

// LookupServer implements the LookupService Connect-RPC handler.
type LookupServer struct {
	atlinkspbconnect.UnimplementedLookupServiceHandler
	lookups  map[string]lookupStoreI
	terms    termsStoreI
	taxCodes taxCodeStoreI
	items    itemStoreI
	audit    *audit.Service
}

// NewLookupServer creates a new LookupServer.
func NewLookupServer(lookups map[string]*store.LookupStore, terms termsStoreI, taxCodes taxCodeStoreI, items itemStoreI, audit *audit.Service) *LookupServer {
	m := make(map[string]lookupStoreI, len(lookups))
	for k, v := range lookups {
		m[k] = v
	}
	return &LookupServer{
		lookups:  m,
		terms:    terms,
		taxCodes: taxCodes,
		items:    items,
		audit:    audit,
	}
}

// --- Generic lookup helpers ---

func (s *LookupServer) listLookup(ctx context.Context, table string) (*connect.Response[pb.ListLookupResponse], error) {
	st, ok := s.lookups[table]
	if !ok {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("unknown lookup table: %s", table))
	}
	items, err := st.List(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("list %s: %w", table, err))
	}
	pbItems := make([]*pb.LookupItem, len(items))
	for i, item := range items {
		pbItems[i] = &pb.LookupItem{Id: int32(item.ID), Code: item.Code, Description: item.Description}
	}
	return connect.NewResponse(&pb.ListLookupResponse{Items: pbItems}), nil
}

func (s *LookupServer) createLookup(ctx context.Context, table string, req *pb.CreateLookupRequest) (*connect.Response[pb.CreateLookupResponse], error) {
	st, ok := s.lookups[table]
	if !ok {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("unknown lookup table: %s", table))
	}
	item, err := st.Create(ctx, req.Code, req.Description)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("create %s: %w", table, err))
	}
	s.audit.Log(ctx, table, item.ID, "INSERT", nil, item)
	return connect.NewResponse(&pb.CreateLookupResponse{
		Item: &pb.LookupItem{Id: int32(item.ID), Code: item.Code, Description: item.Description},
	}), nil
}

func (s *LookupServer) updateLookup(ctx context.Context, table string, req *pb.UpdateLookupRequest) (*connect.Response[pb.UpdateLookupResponse], error) {
	st, ok := s.lookups[table]
	if !ok {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("unknown lookup table: %s", table))
	}
	if err := st.Update(ctx, int(req.Id), req.Code, req.Description); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("update %s: %w", table, err))
	}
	s.audit.Log(ctx, table, int(req.Id), "UPDATE", nil, nil)
	return connect.NewResponse(&pb.UpdateLookupResponse{Success: true}), nil
}

func (s *LookupServer) deleteLookup(ctx context.Context, table string, req *pb.DeleteLookupRequest) (*connect.Response[pb.DeleteLookupResponse], error) {
	st, ok := s.lookups[table]
	if !ok {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("unknown lookup table: %s", table))
	}
	if err := st.Delete(ctx, int(req.Id)); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("delete %s: %w", table, err))
	}
	s.audit.Log(ctx, table, int(req.Id), "DELETE", nil, nil)
	return connect.NewResponse(&pb.DeleteLookupResponse{Success: true}), nil
}

// --- Dispatch Codes ---

func (s *LookupServer) ListDispatchCodes(ctx context.Context, req *connect.Request[pb.ListLookupRequest]) (*connect.Response[pb.ListLookupResponse], error) {
	return s.listLookup(ctx, "dispatch_codes")
}

func (s *LookupServer) CreateDispatchCode(ctx context.Context, req *connect.Request[pb.CreateLookupRequest]) (*connect.Response[pb.CreateLookupResponse], error) {
	return s.createLookup(ctx, "dispatch_codes", req.Msg)
}

func (s *LookupServer) UpdateDispatchCode(ctx context.Context, req *connect.Request[pb.UpdateLookupRequest]) (*connect.Response[pb.UpdateLookupResponse], error) {
	return s.updateLookup(ctx, "dispatch_codes", req.Msg)
}

func (s *LookupServer) DeleteDispatchCode(ctx context.Context, req *connect.Request[pb.DeleteLookupRequest]) (*connect.Response[pb.DeleteLookupResponse], error) {
	return s.deleteLookup(ctx, "dispatch_codes", req.Msg)
}

// --- Equipment Types ---

func (s *LookupServer) ListEquipmentTypes(ctx context.Context, req *connect.Request[pb.ListLookupRequest]) (*connect.Response[pb.ListLookupResponse], error) {
	return s.listLookup(ctx, "equipment_types")
}

func (s *LookupServer) CreateEquipmentType(ctx context.Context, req *connect.Request[pb.CreateLookupRequest]) (*connect.Response[pb.CreateLookupResponse], error) {
	return s.createLookup(ctx, "equipment_types", req.Msg)
}

func (s *LookupServer) UpdateEquipmentType(ctx context.Context, req *connect.Request[pb.UpdateLookupRequest]) (*connect.Response[pb.UpdateLookupResponse], error) {
	return s.updateLookup(ctx, "equipment_types", req.Msg)
}

func (s *LookupServer) DeleteEquipmentType(ctx context.Context, req *connect.Request[pb.DeleteLookupRequest]) (*connect.Response[pb.DeleteLookupResponse], error) {
	return s.deleteLookup(ctx, "equipment_types", req.Msg)
}

// --- Hold Codes ---

func (s *LookupServer) ListHoldCodes(ctx context.Context, req *connect.Request[pb.ListLookupRequest]) (*connect.Response[pb.ListLookupResponse], error) {
	return s.listLookup(ctx, "hold_codes")
}

func (s *LookupServer) CreateHoldCode(ctx context.Context, req *connect.Request[pb.CreateLookupRequest]) (*connect.Response[pb.CreateLookupResponse], error) {
	return s.createLookup(ctx, "hold_codes", req.Msg)
}

func (s *LookupServer) UpdateHoldCode(ctx context.Context, req *connect.Request[pb.UpdateLookupRequest]) (*connect.Response[pb.UpdateLookupResponse], error) {
	return s.updateLookup(ctx, "hold_codes", req.Msg)
}

func (s *LookupServer) DeleteHoldCode(ctx context.Context, req *connect.Request[pb.DeleteLookupRequest]) (*connect.Response[pb.DeleteLookupResponse], error) {
	return s.deleteLookup(ctx, "hold_codes", req.Msg)
}

// --- Declination Codes ---

func (s *LookupServer) ListDeclinationCodes(ctx context.Context, req *connect.Request[pb.ListLookupRequest]) (*connect.Response[pb.ListLookupResponse], error) {
	return s.listLookup(ctx, "declination_codes")
}

func (s *LookupServer) CreateDeclinationCode(ctx context.Context, req *connect.Request[pb.CreateLookupRequest]) (*connect.Response[pb.CreateLookupResponse], error) {
	return s.createLookup(ctx, "declination_codes", req.Msg)
}

func (s *LookupServer) UpdateDeclinationCode(ctx context.Context, req *connect.Request[pb.UpdateLookupRequest]) (*connect.Response[pb.UpdateLookupResponse], error) {
	return s.updateLookup(ctx, "declination_codes", req.Msg)
}

func (s *LookupServer) DeleteDeclinationCode(ctx context.Context, req *connect.Request[pb.DeleteLookupRequest]) (*connect.Response[pb.DeleteLookupResponse], error) {
	return s.deleteLookup(ctx, "declination_codes", req.Msg)
}

// --- Regions ---

func (s *LookupServer) ListRegions(ctx context.Context, req *connect.Request[pb.ListLookupRequest]) (*connect.Response[pb.ListLookupResponse], error) {
	return s.listLookup(ctx, "regions")
}

func (s *LookupServer) CreateRegion(ctx context.Context, req *connect.Request[pb.CreateLookupRequest]) (*connect.Response[pb.CreateLookupResponse], error) {
	return s.createLookup(ctx, "regions", req.Msg)
}

func (s *LookupServer) UpdateRegion(ctx context.Context, req *connect.Request[pb.UpdateLookupRequest]) (*connect.Response[pb.UpdateLookupResponse], error) {
	return s.updateLookup(ctx, "regions", req.Msg)
}

func (s *LookupServer) DeleteRegion(ctx context.Context, req *connect.Request[pb.DeleteLookupRequest]) (*connect.Response[pb.DeleteLookupResponse], error) {
	return s.deleteLookup(ctx, "regions", req.Msg)
}

// --- Damage Areas ---

func (s *LookupServer) ListDamageAreas(ctx context.Context, req *connect.Request[pb.ListLookupRequest]) (*connect.Response[pb.ListLookupResponse], error) {
	return s.listLookup(ctx, "damage_areas")
}

func (s *LookupServer) CreateDamageArea(ctx context.Context, req *connect.Request[pb.CreateLookupRequest]) (*connect.Response[pb.CreateLookupResponse], error) {
	return s.createLookup(ctx, "damage_areas", req.Msg)
}

func (s *LookupServer) UpdateDamageArea(ctx context.Context, req *connect.Request[pb.UpdateLookupRequest]) (*connect.Response[pb.UpdateLookupResponse], error) {
	return s.updateLookup(ctx, "damage_areas", req.Msg)
}

func (s *LookupServer) DeleteDamageArea(ctx context.Context, req *connect.Request[pb.DeleteLookupRequest]) (*connect.Response[pb.DeleteLookupResponse], error) {
	return s.deleteLookup(ctx, "damage_areas", req.Msg)
}

// --- Damage Types ---

func (s *LookupServer) ListDamageTypes(ctx context.Context, req *connect.Request[pb.ListLookupRequest]) (*connect.Response[pb.ListLookupResponse], error) {
	return s.listLookup(ctx, "damage_types")
}

func (s *LookupServer) CreateDamageType(ctx context.Context, req *connect.Request[pb.CreateLookupRequest]) (*connect.Response[pb.CreateLookupResponse], error) {
	return s.createLookup(ctx, "damage_types", req.Msg)
}

func (s *LookupServer) UpdateDamageType(ctx context.Context, req *connect.Request[pb.UpdateLookupRequest]) (*connect.Response[pb.UpdateLookupResponse], error) {
	return s.updateLookup(ctx, "damage_types", req.Msg)
}

func (s *LookupServer) DeleteDamageType(ctx context.Context, req *connect.Request[pb.DeleteLookupRequest]) (*connect.Response[pb.DeleteLookupResponse], error) {
	return s.deleteLookup(ctx, "damage_types", req.Msg)
}

// --- Damage Severities ---

func (s *LookupServer) ListDamageSeverities(ctx context.Context, req *connect.Request[pb.ListLookupRequest]) (*connect.Response[pb.ListLookupResponse], error) {
	return s.listLookup(ctx, "damage_severities")
}

func (s *LookupServer) CreateDamageSeverity(ctx context.Context, req *connect.Request[pb.CreateLookupRequest]) (*connect.Response[pb.CreateLookupResponse], error) {
	return s.createLookup(ctx, "damage_severities", req.Msg)
}

func (s *LookupServer) UpdateDamageSeverity(ctx context.Context, req *connect.Request[pb.UpdateLookupRequest]) (*connect.Response[pb.UpdateLookupResponse], error) {
	return s.updateLookup(ctx, "damage_severities", req.Msg)
}

func (s *LookupServer) DeleteDamageSeverity(ctx context.Context, req *connect.Request[pb.DeleteLookupRequest]) (*connect.Response[pb.DeleteLookupResponse], error) {
	return s.deleteLookup(ctx, "damage_severities", req.Msg)
}

// --- Field Codes 1 ---

func (s *LookupServer) ListFieldCodes1(ctx context.Context, req *connect.Request[pb.ListLookupRequest]) (*connect.Response[pb.ListLookupResponse], error) {
	return s.listLookup(ctx, "field_codes_1")
}

func (s *LookupServer) CreateFieldCode1(ctx context.Context, req *connect.Request[pb.CreateLookupRequest]) (*connect.Response[pb.CreateLookupResponse], error) {
	return s.createLookup(ctx, "field_codes_1", req.Msg)
}

func (s *LookupServer) UpdateFieldCode1(ctx context.Context, req *connect.Request[pb.UpdateLookupRequest]) (*connect.Response[pb.UpdateLookupResponse], error) {
	return s.updateLookup(ctx, "field_codes_1", req.Msg)
}

func (s *LookupServer) DeleteFieldCode1(ctx context.Context, req *connect.Request[pb.DeleteLookupRequest]) (*connect.Response[pb.DeleteLookupResponse], error) {
	return s.deleteLookup(ctx, "field_codes_1", req.Msg)
}

// --- Field Codes 2 ---

func (s *LookupServer) ListFieldCodes2(ctx context.Context, req *connect.Request[pb.ListLookupRequest]) (*connect.Response[pb.ListLookupResponse], error) {
	return s.listLookup(ctx, "field_codes_2")
}

func (s *LookupServer) CreateFieldCode2(ctx context.Context, req *connect.Request[pb.CreateLookupRequest]) (*connect.Response[pb.CreateLookupResponse], error) {
	return s.createLookup(ctx, "field_codes_2", req.Msg)
}

func (s *LookupServer) UpdateFieldCode2(ctx context.Context, req *connect.Request[pb.UpdateLookupRequest]) (*connect.Response[pb.UpdateLookupResponse], error) {
	return s.updateLookup(ctx, "field_codes_2", req.Msg)
}

func (s *LookupServer) DeleteFieldCode2(ctx context.Context, req *connect.Request[pb.DeleteLookupRequest]) (*connect.Response[pb.DeleteLookupResponse], error) {
	return s.deleteLookup(ctx, "field_codes_2", req.Msg)
}

// --- Field Codes 3 ---

func (s *LookupServer) ListFieldCodes3(ctx context.Context, req *connect.Request[pb.ListLookupRequest]) (*connect.Response[pb.ListLookupResponse], error) {
	return s.listLookup(ctx, "field_codes_3")
}

func (s *LookupServer) CreateFieldCode3(ctx context.Context, req *connect.Request[pb.CreateLookupRequest]) (*connect.Response[pb.CreateLookupResponse], error) {
	return s.createLookup(ctx, "field_codes_3", req.Msg)
}

func (s *LookupServer) UpdateFieldCode3(ctx context.Context, req *connect.Request[pb.UpdateLookupRequest]) (*connect.Response[pb.UpdateLookupResponse], error) {
	return s.updateLookup(ctx, "field_codes_3", req.Msg)
}

func (s *LookupServer) DeleteFieldCode3(ctx context.Context, req *connect.Request[pb.DeleteLookupRequest]) (*connect.Response[pb.DeleteLookupResponse], error) {
	return s.deleteLookup(ctx, "field_codes_3", req.Msg)
}

// --- Field Codes 4 ---

func (s *LookupServer) ListFieldCodes4(ctx context.Context, req *connect.Request[pb.ListLookupRequest]) (*connect.Response[pb.ListLookupResponse], error) {
	return s.listLookup(ctx, "field_codes_4")
}

func (s *LookupServer) CreateFieldCode4(ctx context.Context, req *connect.Request[pb.CreateLookupRequest]) (*connect.Response[pb.CreateLookupResponse], error) {
	return s.createLookup(ctx, "field_codes_4", req.Msg)
}

func (s *LookupServer) UpdateFieldCode4(ctx context.Context, req *connect.Request[pb.UpdateLookupRequest]) (*connect.Response[pb.UpdateLookupResponse], error) {
	return s.updateLookup(ctx, "field_codes_4", req.Msg)
}

func (s *LookupServer) DeleteFieldCode4(ctx context.Context, req *connect.Request[pb.DeleteLookupRequest]) (*connect.Response[pb.DeleteLookupResponse], error) {
	return s.deleteLookup(ctx, "field_codes_4", req.Msg)
}

// --- Field Codes 5 ---

func (s *LookupServer) ListFieldCodes5(ctx context.Context, req *connect.Request[pb.ListLookupRequest]) (*connect.Response[pb.ListLookupResponse], error) {
	return s.listLookup(ctx, "field_codes_5")
}

func (s *LookupServer) CreateFieldCode5(ctx context.Context, req *connect.Request[pb.CreateLookupRequest]) (*connect.Response[pb.CreateLookupResponse], error) {
	return s.createLookup(ctx, "field_codes_5", req.Msg)
}

func (s *LookupServer) UpdateFieldCode5(ctx context.Context, req *connect.Request[pb.UpdateLookupRequest]) (*connect.Response[pb.UpdateLookupResponse], error) {
	return s.updateLookup(ctx, "field_codes_5", req.Msg)
}

func (s *LookupServer) DeleteFieldCode5(ctx context.Context, req *connect.Request[pb.DeleteLookupRequest]) (*connect.Response[pb.DeleteLookupResponse], error) {
	return s.deleteLookup(ctx, "field_codes_5", req.Msg)
}

// --- Terms ---

func (s *LookupServer) ListTerms(ctx context.Context, req *connect.Request[pb.ListLookupRequest]) (*connect.Response[pb.ListTermsResponse], error) {
	items, err := s.terms.List(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("list terms: %w", err))
	}
	pbItems := make([]*pb.TermItem, len(items))
	for i, item := range items {
		pbItems[i] = &pb.TermItem{
			Id:          int32(item.ID),
			Term:        item.Term,
			Description: item.Description,
			Days:        ip(item.Days),
		}
	}
	return connect.NewResponse(&pb.ListTermsResponse{Items: pbItems}), nil
}

func (s *LookupServer) CreateTerm(ctx context.Context, req *connect.Request[pb.CreateTermRequest]) (*connect.Response[pb.CreateTermResponse], error) {
	item, err := s.terms.Create(ctx, req.Msg.Term, req.Msg.Description, i32p(req.Msg.Days))
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("create term: %w", err))
	}
	s.audit.Log(ctx, "terms", item.ID, "INSERT", nil, item)
	return connect.NewResponse(&pb.CreateTermResponse{
		Item: &pb.TermItem{
			Id:          int32(item.ID),
			Term:        item.Term,
			Description: item.Description,
			Days:        ip(item.Days),
		},
	}), nil
}

func (s *LookupServer) UpdateTerm(ctx context.Context, req *connect.Request[pb.UpdateTermRequest]) (*connect.Response[pb.UpdateTermResponse], error) {
	if err := s.terms.Update(ctx, int(req.Msg.Id), req.Msg.Term, req.Msg.Description, i32p(req.Msg.Days)); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("update term: %w", err))
	}
	s.audit.Log(ctx, "terms", int(req.Msg.Id), "UPDATE", nil, nil)
	return connect.NewResponse(&pb.UpdateTermResponse{Success: true}), nil
}

func (s *LookupServer) DeleteTerm(ctx context.Context, req *connect.Request[pb.DeleteLookupRequest]) (*connect.Response[pb.DeleteLookupResponse], error) {
	if err := s.terms.Delete(ctx, int(req.Msg.Id)); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("delete term: %w", err))
	}
	s.audit.Log(ctx, "terms", int(req.Msg.Id), "DELETE", nil, nil)
	return connect.NewResponse(&pb.DeleteLookupResponse{Success: true}), nil
}

// --- Tax Codes ---

func (s *LookupServer) ListTaxCodes(ctx context.Context, req *connect.Request[pb.ListLookupRequest]) (*connect.Response[pb.ListTaxCodesResponse], error) {
	items, err := s.taxCodes.List(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("list tax_codes: %w", err))
	}
	pbItems := make([]*pb.TaxCodeItem, len(items))
	for i, item := range items {
		pbItems[i] = &pb.TaxCodeItem{
			Id:          int32(item.ID),
			Code:        item.Code,
			Description: item.Description,
			Rate:        item.Rate,
		}
	}
	return connect.NewResponse(&pb.ListTaxCodesResponse{Items: pbItems}), nil
}

func (s *LookupServer) CreateTaxCode(ctx context.Context, req *connect.Request[pb.CreateTaxCodeRequest]) (*connect.Response[pb.CreateTaxCodeResponse], error) {
	item, err := s.taxCodes.Create(ctx, req.Msg.Code, req.Msg.Description, req.Msg.Rate)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("create tax_code: %w", err))
	}
	s.audit.Log(ctx, "tax_codes", item.ID, "INSERT", nil, item)
	return connect.NewResponse(&pb.CreateTaxCodeResponse{
		Item: &pb.TaxCodeItem{
			Id:          int32(item.ID),
			Code:        item.Code,
			Description: item.Description,
			Rate:        item.Rate,
		},
	}), nil
}

func (s *LookupServer) UpdateTaxCode(ctx context.Context, req *connect.Request[pb.UpdateTaxCodeRequest]) (*connect.Response[pb.UpdateTaxCodeResponse], error) {
	if err := s.taxCodes.Update(ctx, int(req.Msg.Id), req.Msg.Code, req.Msg.Description, req.Msg.Rate); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("update tax_code: %w", err))
	}
	s.audit.Log(ctx, "tax_codes", int(req.Msg.Id), "UPDATE", nil, nil)
	return connect.NewResponse(&pb.UpdateTaxCodeResponse{Success: true}), nil
}

func (s *LookupServer) DeleteTaxCode(ctx context.Context, req *connect.Request[pb.DeleteLookupRequest]) (*connect.Response[pb.DeleteLookupResponse], error) {
	if err := s.taxCodes.Delete(ctx, int(req.Msg.Id)); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("delete tax_code: %w", err))
	}
	s.audit.Log(ctx, "tax_codes", int(req.Msg.Id), "DELETE", nil, nil)
	return connect.NewResponse(&pb.DeleteLookupResponse{Success: true}), nil
}

// --- Items ---

func (s *LookupServer) ListItems(ctx context.Context, req *connect.Request[pb.ListLookupRequest]) (*connect.Response[pb.ListItemsResponse], error) {
	items, err := s.items.List(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("list items: %w", err))
	}
	pbItems := make([]*pb.ItemRecord, len(items))
	for i, item := range items {
		pbItems[i] = &pb.ItemRecord{
			Id:            int32(item.ID),
			Item:          item.Item,
			Description:   item.Description,
			DefaultAmount: item.DefaultAmount,
			CalcType:      item.CalcType,
		}
	}
	return connect.NewResponse(&pb.ListItemsResponse{Items: pbItems}), nil
}

func (s *LookupServer) CreateItem(ctx context.Context, req *connect.Request[pb.CreateItemRequest]) (*connect.Response[pb.CreateItemResponse], error) {
	item, err := s.items.Create(ctx, req.Msg.Item, req.Msg.Description, req.Msg.DefaultAmount, req.Msg.CalcType)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("create item: %w", err))
	}
	s.audit.Log(ctx, "items", item.ID, "INSERT", nil, item)
	return connect.NewResponse(&pb.CreateItemResponse{
		Item: &pb.ItemRecord{
			Id:            int32(item.ID),
			Item:          item.Item,
			Description:   item.Description,
			DefaultAmount: item.DefaultAmount,
			CalcType:      item.CalcType,
		},
	}), nil
}

func (s *LookupServer) UpdateItem(ctx context.Context, req *connect.Request[pb.UpdateItemRequest]) (*connect.Response[pb.UpdateItemResponse], error) {
	if err := s.items.Update(ctx, int(req.Msg.Id), req.Msg.Item, req.Msg.Description, req.Msg.DefaultAmount, req.Msg.CalcType); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("update item: %w", err))
	}
	s.audit.Log(ctx, "items", int(req.Msg.Id), "UPDATE", nil, nil)
	return connect.NewResponse(&pb.UpdateItemResponse{Success: true}), nil
}

func (s *LookupServer) DeleteItem(ctx context.Context, req *connect.Request[pb.DeleteLookupRequest]) (*connect.Response[pb.DeleteLookupResponse], error) {
	if err := s.items.Delete(ctx, int(req.Msg.Id)); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("delete item: %w", err))
	}
	s.audit.Log(ctx, "items", int(req.Msg.Id), "DELETE", nil, nil)
	return connect.NewResponse(&pb.DeleteLookupResponse{Success: true}), nil
}
