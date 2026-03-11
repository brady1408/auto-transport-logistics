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
	"github.com/brady1408/atlinks/internal/service"
)

type tripStoreI interface {
	List(ctx context.Context, f models.TripFilter) (*models.TripListResult, error)
	GetByID(ctx context.Context, id int) (*models.Trip, error)
	Create(ctx context.Context, t *models.Trip) error
	Update(ctx context.Context, t *models.Trip) error
	Delete(ctx context.Context, id int) error
	NextLoadNumber(ctx context.Context) (string, error)
}

type loadDetailStoreI interface {
	ListByTrip(ctx context.Context, tripID int) ([]models.LoadDetail, error)
}

type tripFuelStoreI interface {
	ListByTrip(ctx context.Context, tripID int) ([]models.TripFuel, error)
	GetByID(ctx context.Context, id int) (*models.TripFuel, error)
	Create(ctx context.Context, f *models.TripFuel) error
	Update(ctx context.Context, f *models.TripFuel) error
	Delete(ctx context.Context, id int) error
}

type tripExpenseStoreI interface {
	ListByTrip(ctx context.Context, tripID int) ([]models.TripExpense, error)
	GetByID(ctx context.Context, id int) (*models.TripExpense, error)
	Create(ctx context.Context, e *models.TripExpense) error
	Update(ctx context.Context, e *models.TripExpense) error
	Delete(ctx context.Context, id int) error
}

type tripRouteStoreI interface {
	ListByTrip(ctx context.Context, tripID int) ([]models.TripRoute, error)
	GetByID(ctx context.Context, id int) (*models.TripRoute, error)
	Create(ctx context.Context, r *models.TripRoute) error
	Update(ctx context.Context, r *models.TripRoute) error
	Delete(ctx context.Context, id int) error
}

type TripServer struct {
	atlinkspbconnect.UnimplementedTripServiceHandler
	trips    tripStoreI
	loads    loadDetailStoreI
	fuel     tripFuelStoreI
	expenses tripExpenseStoreI
	routes   tripRouteStoreI
	tripSvc  *service.TripService
	audit    *audit.Service
}

func NewTripServer(trips tripStoreI, loads loadDetailStoreI, fuel tripFuelStoreI, expenses tripExpenseStoreI, routes tripRouteStoreI, tripSvc *service.TripService, audit *audit.Service) *TripServer {
	return &TripServer{
		trips:    trips,
		loads:    loads,
		fuel:     fuel,
		expenses: expenses,
		routes:   routes,
		tripSvc:  tripSvc,
		audit:    audit,
	}
}

// --- Trip CRUD ---

func (s *TripServer) ListTrips(ctx context.Context, req *connect.Request[pb.ListTripsRequest]) (*connect.Response[pb.ListTripsResponse], error) {
	filter := protoToTripFilter(req.Msg)
	result, err := s.trips.List(ctx, filter)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("list trips: %w", err))
	}

	trips := make([]*pb.Trip, len(result.Items))
	for i := range result.Items {
		trips[i] = tripToProto(&result.Items[i])
	}

	return connect.NewResponse(&pb.ListTripsResponse{
		Trips: trips,
		Pagination: &pb.PaginationResponse{
			TotalCount: int32(result.TotalCount),
			Page:       int32(result.Page),
			PageSize:   int32(result.PageSize),
		},
	}), nil
}

func (s *TripServer) GetTrip(ctx context.Context, req *connect.Request[pb.GetTripRequest]) (*connect.Response[pb.GetTripResponse], error) {
	t, err := s.trips.GetByID(ctx, int(req.Msg.Id))
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("trip %d not found", req.Msg.Id))
	}
	return connect.NewResponse(&pb.GetTripResponse{Trip: tripToProto(t)}), nil
}

func (s *TripServer) CreateTrip(ctx context.Context, req *connect.Request[pb.CreateTripRequest]) (*connect.Response[pb.CreateTripResponse], error) {
	t := createTripReqToModel(req.Msg)

	num, err := s.trips.NextLoadNumber(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("generate load number: %w", err))
	}
	t.LoadNumber = num

	if err := s.trips.Create(ctx, t); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("create trip: %w", err))
	}

	s.audit.Log(ctx, "trips", t.ID, "INSERT", nil, t)

	created, err := s.trips.GetByID(ctx, t.ID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("fetch created trip: %w", err))
	}
	return connect.NewResponse(&pb.CreateTripResponse{Trip: tripToProto(created)}), nil
}

func (s *TripServer) UpdateTrip(ctx context.Context, req *connect.Request[pb.UpdateTripRequest]) (*connect.Response[pb.UpdateTripResponse], error) {
	old, err := s.trips.GetByID(ctx, int(req.Msg.Id))
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("trip %d not found", req.Msg.Id))
	}

	t := updateTripReqToModel(req.Msg)
	t.LoadNumber = old.LoadNumber // load number is immutable
	if err := s.trips.Update(ctx, t); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("update trip: %w", err))
	}

	s.audit.Log(ctx, "trips", t.ID, "UPDATE", old, t)

	updated, err := s.trips.GetByID(ctx, t.ID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("fetch updated trip: %w", err))
	}
	return connect.NewResponse(&pb.UpdateTripResponse{Trip: tripToProto(updated)}), nil
}

func (s *TripServer) DeleteTrip(ctx context.Context, req *connect.Request[pb.DeleteTripRequest]) (*connect.Response[pb.DeleteTripResponse], error) {
	if err := s.trips.Delete(ctx, int(req.Msg.Id)); err != nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("trip %d not found", req.Msg.Id))
	}
	s.audit.Log(ctx, "trips", int(req.Msg.Id), "DELETE", nil, nil)
	return connect.NewResponse(&pb.DeleteTripResponse{Success: true}), nil
}

// --- Load Details ---

func (s *TripServer) ListLoadDetails(ctx context.Context, req *connect.Request[pb.ListLoadDetailsRequest]) (*connect.Response[pb.ListLoadDetailsResponse], error) {
	items, err := s.loads.ListByTrip(ctx, int(req.Msg.TripId))
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("list load details: %w", err))
	}

	details := make([]*pb.LoadDetail, len(items))
	for i := range items {
		details[i] = loadDetailToProto(&items[i])
	}

	return connect.NewResponse(&pb.ListLoadDetailsResponse{Details: details}), nil
}

func (s *TripServer) AssignVehicleToTrip(ctx context.Context, req *connect.Request[pb.AssignVehicleToTripRequest]) (*connect.Response[pb.AssignVehicleToTripResponse], error) {
	bayNumber := ""
	if req.Msg.BayNumber != nil {
		bayNumber = *req.Msg.BayNumber
	}

	if err := s.tripSvc.AssignVehicleToTrip(ctx, int(req.Msg.TripId), int(req.Msg.VehicleId), bayNumber); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("assign vehicle to trip: %w", err))
	}

	s.audit.Log(ctx, "load_details", 0, "INSERT", nil, map[string]interface{}{
		"trip_id":    req.Msg.TripId,
		"vehicle_id": req.Msg.VehicleId,
		"bay_number": bayNumber,
	})

	return connect.NewResponse(&pb.AssignVehicleToTripResponse{Success: true}), nil
}

func (s *TripServer) UnassignVehicle(ctx context.Context, req *connect.Request[pb.UnassignVehicleRequest]) (*connect.Response[pb.UnassignVehicleResponse], error) {
	if err := s.tripSvc.UnassignVehicle(ctx, int(req.Msg.LoadDetailId)); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("unassign vehicle: %w", err))
	}

	s.audit.Log(ctx, "load_details", int(req.Msg.LoadDetailId), "DELETE", nil, nil)

	return connect.NewResponse(&pb.UnassignVehicleResponse{Success: true}), nil
}

func (s *TripServer) AssignAllFromOrder(ctx context.Context, req *connect.Request[pb.AssignAllFromOrderRequest]) (*connect.Response[pb.AssignAllFromOrderResponse], error) {
	count, err := s.tripSvc.AssignAllFromOrder(ctx, int(req.Msg.TripId), int(req.Msg.OrderId))
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("assign all from order: %w", err))
	}

	s.audit.Log(ctx, "load_details", 0, "INSERT", nil, map[string]interface{}{
		"trip_id":  req.Msg.TripId,
		"order_id": req.Msg.OrderId,
		"count":    count,
	})

	return connect.NewResponse(&pb.AssignAllFromOrderResponse{Count: int32(count)}), nil
}

// --- Trip Fuel ---

func (s *TripServer) ListTripFuel(ctx context.Context, req *connect.Request[pb.ListTripFuelRequest]) (*connect.Response[pb.ListTripFuelResponse], error) {
	items, err := s.fuel.ListByTrip(ctx, int(req.Msg.TripId))
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("list trip fuel: %w", err))
	}

	result := make([]*pb.TripFuel, len(items))
	for i := range items {
		result[i] = tripFuelToProto(&items[i])
	}

	return connect.NewResponse(&pb.ListTripFuelResponse{Items: result}), nil
}

func (s *TripServer) CreateTripFuel(ctx context.Context, req *connect.Request[pb.CreateTripFuelRequest]) (*connect.Response[pb.CreateTripFuelResponse], error) {
	f := createTripFuelReqToModel(req.Msg)
	if err := s.fuel.Create(ctx, f); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("create trip fuel: %w", err))
	}

	s.audit.Log(ctx, "trip_fuel", f.ID, "INSERT", nil, f)

	created, err := s.fuel.GetByID(ctx, f.ID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("fetch created trip fuel: %w", err))
	}
	return connect.NewResponse(&pb.CreateTripFuelResponse{Item: tripFuelToProto(created)}), nil
}

func (s *TripServer) UpdateTripFuel(ctx context.Context, req *connect.Request[pb.UpdateTripFuelRequest]) (*connect.Response[pb.UpdateTripFuelResponse], error) {
	old, err := s.fuel.GetByID(ctx, int(req.Msg.Id))
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("trip fuel %d not found", req.Msg.Id))
	}

	f := updateTripFuelReqToModel(req.Msg)
	if err := s.fuel.Update(ctx, f); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("update trip fuel: %w", err))
	}

	s.audit.Log(ctx, "trip_fuel", f.ID, "UPDATE", old, f)

	updated, err := s.fuel.GetByID(ctx, f.ID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("fetch updated trip fuel: %w", err))
	}
	return connect.NewResponse(&pb.UpdateTripFuelResponse{Item: tripFuelToProto(updated)}), nil
}

func (s *TripServer) DeleteTripFuel(ctx context.Context, req *connect.Request[pb.DeleteTripFuelRequest]) (*connect.Response[pb.DeleteTripFuelResponse], error) {
	if err := s.fuel.Delete(ctx, int(req.Msg.Id)); err != nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("trip fuel %d not found", req.Msg.Id))
	}
	s.audit.Log(ctx, "trip_fuel", int(req.Msg.Id), "DELETE", nil, nil)
	return connect.NewResponse(&pb.DeleteTripFuelResponse{Success: true}), nil
}

// --- Trip Expenses ---

func (s *TripServer) ListTripExpenses(ctx context.Context, req *connect.Request[pb.ListTripExpensesRequest]) (*connect.Response[pb.ListTripExpensesResponse], error) {
	items, err := s.expenses.ListByTrip(ctx, int(req.Msg.TripId))
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("list trip expenses: %w", err))
	}

	result := make([]*pb.TripExpense, len(items))
	for i := range items {
		result[i] = tripExpenseToProto(&items[i])
	}

	return connect.NewResponse(&pb.ListTripExpensesResponse{Items: result}), nil
}

func (s *TripServer) CreateTripExpense(ctx context.Context, req *connect.Request[pb.CreateTripExpenseRequest]) (*connect.Response[pb.CreateTripExpenseResponse], error) {
	e := createTripExpenseReqToModel(req.Msg)
	if err := s.expenses.Create(ctx, e); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("create trip expense: %w", err))
	}

	s.audit.Log(ctx, "trip_expenses", e.ID, "INSERT", nil, e)

	created, err := s.expenses.GetByID(ctx, e.ID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("fetch created trip expense: %w", err))
	}
	return connect.NewResponse(&pb.CreateTripExpenseResponse{Item: tripExpenseToProto(created)}), nil
}

func (s *TripServer) UpdateTripExpense(ctx context.Context, req *connect.Request[pb.UpdateTripExpenseRequest]) (*connect.Response[pb.UpdateTripExpenseResponse], error) {
	old, err := s.expenses.GetByID(ctx, int(req.Msg.Id))
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("trip expense %d not found", req.Msg.Id))
	}

	e := updateTripExpenseReqToModel(req.Msg)
	if err := s.expenses.Update(ctx, e); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("update trip expense: %w", err))
	}

	s.audit.Log(ctx, "trip_expenses", e.ID, "UPDATE", old, e)

	updated, err := s.expenses.GetByID(ctx, e.ID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("fetch updated trip expense: %w", err))
	}
	return connect.NewResponse(&pb.UpdateTripExpenseResponse{Item: tripExpenseToProto(updated)}), nil
}

func (s *TripServer) DeleteTripExpense(ctx context.Context, req *connect.Request[pb.DeleteTripExpenseRequest]) (*connect.Response[pb.DeleteTripExpenseResponse], error) {
	if err := s.expenses.Delete(ctx, int(req.Msg.Id)); err != nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("trip expense %d not found", req.Msg.Id))
	}
	s.audit.Log(ctx, "trip_expenses", int(req.Msg.Id), "DELETE", nil, nil)
	return connect.NewResponse(&pb.DeleteTripExpenseResponse{Success: true}), nil
}

// --- Trip Routes ---

func (s *TripServer) ListTripRoutes(ctx context.Context, req *connect.Request[pb.ListTripRoutesRequest]) (*connect.Response[pb.ListTripRoutesResponse], error) {
	items, err := s.routes.ListByTrip(ctx, int(req.Msg.TripId))
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("list trip routes: %w", err))
	}

	result := make([]*pb.TripRoute, len(items))
	for i := range items {
		result[i] = tripRouteToProto(&items[i])
	}

	return connect.NewResponse(&pb.ListTripRoutesResponse{Items: result}), nil
}

func (s *TripServer) CreateTripRoute(ctx context.Context, req *connect.Request[pb.CreateTripRouteRequest]) (*connect.Response[pb.CreateTripRouteResponse], error) {
	r := createTripRouteReqToModel(req.Msg)
	if err := s.routes.Create(ctx, r); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("create trip route: %w", err))
	}

	s.audit.Log(ctx, "trip_routes", r.ID, "INSERT", nil, r)

	created, err := s.routes.GetByID(ctx, r.ID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("fetch created trip route: %w", err))
	}
	return connect.NewResponse(&pb.CreateTripRouteResponse{Item: tripRouteToProto(created)}), nil
}

func (s *TripServer) UpdateTripRoute(ctx context.Context, req *connect.Request[pb.UpdateTripRouteRequest]) (*connect.Response[pb.UpdateTripRouteResponse], error) {
	old, err := s.routes.GetByID(ctx, int(req.Msg.Id))
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("trip route %d not found", req.Msg.Id))
	}

	r := updateTripRouteReqToModel(req.Msg)
	if err := s.routes.Update(ctx, r); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("update trip route: %w", err))
	}

	s.audit.Log(ctx, "trip_routes", r.ID, "UPDATE", old, r)

	updated, err := s.routes.GetByID(ctx, r.ID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("fetch updated trip route: %w", err))
	}
	return connect.NewResponse(&pb.UpdateTripRouteResponse{Item: tripRouteToProto(updated)}), nil
}

func (s *TripServer) DeleteTripRoute(ctx context.Context, req *connect.Request[pb.DeleteTripRouteRequest]) (*connect.Response[pb.DeleteTripRouteResponse], error) {
	if err := s.routes.Delete(ctx, int(req.Msg.Id)); err != nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("trip route %d not found", req.Msg.Id))
	}
	s.audit.Log(ctx, "trip_routes", int(req.Msg.Id), "DELETE", nil, nil)
	return connect.NewResponse(&pb.DeleteTripRouteResponse{Success: true}), nil
}

// --- Converters ---

func tripToProto(t *models.Trip) *pb.Trip {
	return &pb.Trip{
		Id:                int32(t.ID),
		LoadNumber:        t.LoadNumber,
		Active:            t.Active,
		TruckNumber:       sp(t.TruckNumber),
		TruckId:           ip(t.TruckID),
		TrailerNumber:     sp(t.TrailerNumber),
		Driver:            sp(t.Driver),
		Driver1Id:         ip(t.Driver1ID),
		Driver2:           sp(t.Driver2),
		Driver2Id:         ip(t.Driver2ID),
		TripDate:          timeStr(t.TripDate),
		EstDeliverDate:    timeStr(t.EstDeliverDate),
		DeliverDate:       timeStr(t.DeliverDate),
		ArrivalDate:       timeStr(t.ArrivalDate),
		ReturnDate:        timeStr(t.ReturnDate),
		TotalMileage:      intToOptStr(t.TotalMileage),
		TotalFuelGallons:  sp(t.TotalFuelGallons),
		FuelAdvance:       sp(t.FuelAdvance),
		TripAdvance:       sp(t.TripAdvance),
		TollsAdvance:      sp(t.TollsAdvance),
		DriverRate:        sp(t.DriverRate),
		DriverCalcType:    sp(t.DriverCalcType),
		DriverAddRate:     sp(t.DriverAddRate),
		DriverAddCalcType: sp(t.DriverAddCalcType),
		TruckRate:         sp(t.TruckRate),
		TruckCalcType:     sp(t.TruckCalcType),
		Comments:          sp(t.Comments),
		Status:            sp(t.Status),
		EquipmentType:     sp(t.EquipmentType),
		Zone:              sp(t.Zone),
		CreatedAt:         t.CreatedAt.Format(time.RFC3339),
		UpdatedAt:         t.UpdatedAt.Format(time.RFC3339),
	}
}

func protoToTripFilter(msg *pb.ListTripsRequest) models.TripFilter {
	f := models.TripFilter{}
	if msg.Pagination != nil {
		f.Page = int(msg.Pagination.Page)
		f.PageSize = int(msg.Pagination.PageSize)
	}
	if msg.Search != nil {
		f.Search = *msg.Search
	}
	if msg.Active != nil {
		f.Active = *msg.Active
	}
	if msg.DateFrom != nil {
		f.DateFrom = *msg.DateFrom
	}
	if msg.DateTo != nil {
		f.DateTo = *msg.DateTo
	}
	if msg.TruckNumber != nil {
		f.TruckNumber = *msg.TruckNumber
	}
	return f
}

func createTripReqToModel(msg *pb.CreateTripRequest) *models.Trip {
	return &models.Trip{
		Active:         msg.Active,
		TruckNumber:    sp(msg.TruckNumber),
		TruckID:        i32p(msg.TruckId),
		TrailerNumber:  sp(msg.TrailerNumber),
		Driver:         sp(msg.Driver),
		Driver1ID:      i32p(msg.Driver1Id),
		Driver2:        sp(msg.Driver2),
		Driver2ID:      i32p(msg.Driver2Id),
		TripDate:       parseDate(msg.TripDate),
		EstDeliverDate: parseDate(msg.EstDeliverDate),
		DriverRate:     sp(msg.DriverRate),
		DriverCalcType: sp(msg.DriverCalcType),
		TruckRate:      sp(msg.TruckRate),
		TruckCalcType:  sp(msg.TruckCalcType),
		Comments:       sp(msg.Comments),
		EquipmentType:  sp(msg.EquipmentType),
		Zone:           sp(msg.Zone),
	}
}

func updateTripReqToModel(msg *pb.UpdateTripRequest) *models.Trip {
	return &models.Trip{
		ID:                int(msg.Id),
		Active:            msg.Active,
		TruckNumber:       sp(msg.TruckNumber),
		TruckID:           i32p(msg.TruckId),
		TrailerNumber:     sp(msg.TrailerNumber),
		Driver:            sp(msg.Driver),
		Driver1ID:         i32p(msg.Driver1Id),
		Driver2:           sp(msg.Driver2),
		Driver2ID:         i32p(msg.Driver2Id),
		TripDate:          parseDate(msg.TripDate),
		EstDeliverDate:    parseDate(msg.EstDeliverDate),
		DeliverDate:       parseDate(msg.DeliverDate),
		ArrivalDate:       parseDate(msg.ArrivalDate),
		ReturnDate:        parseDate(msg.ReturnDate),
		TotalMileage:      optStrToInt(msg.TotalMileage),
		TotalFuelGallons:  sp(msg.TotalFuelGallons),
		FuelAdvance:       sp(msg.FuelAdvance),
		TripAdvance:       sp(msg.TripAdvance),
		TollsAdvance:      sp(msg.TollsAdvance),
		DriverRate:        sp(msg.DriverRate),
		DriverCalcType:    sp(msg.DriverCalcType),
		DriverAddRate:     sp(msg.DriverAddRate),
		DriverAddCalcType: sp(msg.DriverAddCalcType),
		TruckRate:         sp(msg.TruckRate),
		TruckCalcType:     sp(msg.TruckCalcType),
		Comments:          sp(msg.Comments),
		Status:            sp(msg.Status),
		EquipmentType:     sp(msg.EquipmentType),
		Zone:              sp(msg.Zone),
	}
}

func loadDetailToProto(d *models.LoadDetail) *pb.LoadDetail {
	return &pb.LoadDetail{
		Id:            int32(d.ID),
		TripId:        int32(d.TripID),
		OrderId:       derefI32(d.OrderID),
		VehicleId:     derefI32(d.VehicleID),
		Vin:           sp(d.VIN),
		Year:          sp(d.Year),
		Make:          sp(d.Make),
		Model:         sp(d.Model),
		Color:         sp(d.Color),
		Weight:        ip(d.Weight),
		Category:      sp(d.Category),
		BayNumber:     sp(d.BayNumber),
		Status:        sp(d.Status),
		LoadedDate:    timeStr(d.LoadedDate),
		DeliveredDate: timeStr(d.DeliveredDate),
		CreatedAt:     d.CreatedAt.Format(time.RFC3339),
		UpdatedAt:     d.UpdatedAt.Format(time.RFC3339),
	}
}

func tripFuelToProto(f *models.TripFuel) *pb.TripFuel {
	var loadedMiles *string
	if f.LoadedMiles {
		s := "true"
		loadedMiles = &s
	}
	return &pb.TripFuel{
		Id:          int32(f.ID),
		TripId:      int32(f.TripID),
		LoadedMiles: loadedMiles,
		TruckNumber: sp(f.TruckNumber),
		State:       sp(f.State),
		Mileage:     intToOptStr(f.Mileage),
		Gallons:     sp(f.Gallons),
		CreatedAt:   f.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   f.UpdatedAt.Format(time.RFC3339),
	}
}

func createTripFuelReqToModel(msg *pb.CreateTripFuelRequest) *models.TripFuel {
	return &models.TripFuel{
		TripID:      int(msg.TripId),
		LoadedMiles: msg.LoadedMiles != nil && *msg.LoadedMiles == "true",
		TruckNumber: sp(msg.TruckNumber),
		State:       sp(msg.State),
		Mileage:     optStrToInt(msg.Mileage),
		Gallons:     sp(msg.Gallons),
	}
}

func updateTripFuelReqToModel(msg *pb.UpdateTripFuelRequest) *models.TripFuel {
	return &models.TripFuel{
		ID:          int(msg.Id),
		TripID:      int(msg.TripId),
		LoadedMiles: msg.LoadedMiles != nil && *msg.LoadedMiles == "true",
		TruckNumber: sp(msg.TruckNumber),
		State:       sp(msg.State),
		Mileage:     optStrToInt(msg.Mileage),
		Gallons:     sp(msg.Gallons),
	}
}

func tripExpenseToProto(e *models.TripExpense) *pb.TripExpense {
	return &pb.TripExpense{
		Id:          int32(e.ID),
		TripId:      int32(e.TripID),
		Description: sp(e.Description),
		Amount:      sp(e.Amount),
		ExpenseDate: timeStr(e.ExpenseDate),
		CreatedAt:   e.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   e.UpdatedAt.Format(time.RFC3339),
	}
}

func createTripExpenseReqToModel(msg *pb.CreateTripExpenseRequest) *models.TripExpense {
	return &models.TripExpense{
		TripID:      int(msg.TripId),
		Description: sp(msg.Description),
		Amount:      sp(msg.Amount),
		ExpenseDate: parseDate(msg.ExpenseDate),
	}
}

func updateTripExpenseReqToModel(msg *pb.UpdateTripExpenseRequest) *models.TripExpense {
	return &models.TripExpense{
		ID:          int(msg.Id),
		TripID:      int(msg.TripId),
		Description: sp(msg.Description),
		Amount:      sp(msg.Amount),
		ExpenseDate: parseDate(msg.ExpenseDate),
	}
}

func tripRouteToProto(r *models.TripRoute) *pb.TripRoute {
	return &pb.TripRoute{
		Id:           int32(r.ID),
		TripId:       int32(r.TripID),
		Sequence:     ip(r.Sequence),
		CustomerId:   ip(r.CustomerID),
		CustomerName: sp(r.CustomerName),
		City:         sp(r.City),
		State:        sp(r.State),
		StopType:     sp(r.StopType),
		Miles:        intToOptStr(r.Miles),
		EstArrival:   timeStr(r.EstArrival),
		CreatedAt:    r.CreatedAt.Format(time.RFC3339),
		UpdatedAt:    r.UpdatedAt.Format(time.RFC3339),
	}
}

func createTripRouteReqToModel(msg *pb.CreateTripRouteRequest) *models.TripRoute {
	return &models.TripRoute{
		TripID:       int(msg.TripId),
		Sequence:     i32p(msg.Sequence),
		CustomerID:   i32p(msg.CustomerId),
		CustomerName: sp(msg.CustomerName),
		City:         sp(msg.City),
		State:        sp(msg.State),
		StopType:     sp(msg.StopType),
		Miles:        optStrToInt(msg.Miles),
		EstArrival:   parseDate(msg.EstArrival),
	}
}

func updateTripRouteReqToModel(msg *pb.UpdateTripRouteRequest) *models.TripRoute {
	return &models.TripRoute{
		ID:           int(msg.Id),
		TripID:       int(msg.TripId),
		Sequence:     i32p(msg.Sequence),
		CustomerID:   i32p(msg.CustomerId),
		CustomerName: sp(msg.CustomerName),
		City:         sp(msg.City),
		State:        sp(msg.State),
		StopType:     sp(msg.StopType),
		Miles:        optStrToInt(msg.Miles),
		EstArrival:   parseDate(msg.EstArrival),
	}
}
