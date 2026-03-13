package main

import (
	"context"
	"fmt"
	"strings"

	"connectrpc.com/connect"
	"github.com/mark3labs/mcp-go/mcp"

	pb "github.com/brady1408/auto-transport-logistics/internal/gen/atlinks/v1"
)

func registerTripTools(register toolRegister, client *atlClient) {
	// --- Trip CRUD ---
	register(mcp.NewTool("list_trips",
		mcp.WithDescription("List trips with optional filters. Returns paginated results."),
		mcp.WithString("search", mcp.Description("Search by load number")),
		mcp.WithString("active", mcp.Description("Filter: 'active', 'inactive', or 'all'")),
		mcp.WithString("date_from", mcp.Description("Filter by trip date from (YYYY-MM-DD)")),
		mcp.WithString("date_to", mcp.Description("Filter by trip date to (YYYY-MM-DD)")),
		mcp.WithString("truck_number", mcp.Description("Filter by truck number")),
		mcp.WithNumber("page", mcp.Description("Page number (default: 1)")),
		mcp.WithNumber("page_size", mcp.Description("Items per page (default: 25)")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := argMap(req)
		resp, err := client.trips.ListTrips(ctx, connect.NewRequest(&pb.ListTripsRequest{
			Pagination: &pb.PaginationRequest{
				Page:     int32(argInt(args, "page", 1)),
				PageSize: int32(argInt(args, "page_size", 25)),
			},
			Search:      argStrPtr(args, "search"),
			Active:      argStrPtr(args, "active"),
			DateFrom:    argStrPtr(args, "date_from"),
			DateTo:      argStrPtr(args, "date_to"),
			TruckNumber: argStrPtr(args, "truck_number"),
		}))
		if err != nil {
			return mcp.NewToolResultError(connectErr(err)), nil
		}
		return mcp.NewToolResultText(formatTripList(resp.Msg)), nil
	})

	register(mcp.NewTool("get_trip",
		mcp.WithDescription("Get a single trip by ID."),
		mcp.WithNumber("id", mcp.Required(), mcp.Description("Trip ID")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := argMap(req)
		resp, err := client.trips.GetTrip(ctx, connect.NewRequest(&pb.GetTripRequest{
			Id: int32(argInt(args, "id", 0)),
		}))
		if err != nil {
			return mcp.NewToolResultError(connectErr(err)), nil
		}
		return mcp.NewToolResultText(formatTrip(resp.Msg.Trip)), nil
	})

	register(mcp.NewTool("create_trip",
		mcp.WithDescription("Create a new trip."),
		mcp.WithNumber("truck_id", mcp.Description("Truck ID")),
		mcp.WithNumber("driver1_id", mcp.Description("Driver 1 employee ID")),
		mcp.WithString("trip_date", mcp.Description("Trip date (YYYY-MM-DD)")),
		mcp.WithString("est_deliver_date", mcp.Description("Estimated delivery date")),
		mcp.WithString("equipment_type", mcp.Description("Equipment type")),
		mcp.WithString("zone", mcp.Description("Zone")),
		mcp.WithString("comments", mcp.Description("Comments")),
		mcp.WithBoolean("active", mcp.Description("Active (default: true)")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := argMap(req)
		resp, err := client.trips.CreateTrip(ctx, connect.NewRequest(&pb.CreateTripRequest{
			TruckId:        argI32Ptr(args, "truck_id"),
			Driver1Id:      argI32Ptr(args, "driver1_id"),
			TripDate:       argStrPtr(args, "trip_date"),
			EstDeliverDate: argStrPtr(args, "est_deliver_date"),
			EquipmentType:  argStrPtr(args, "equipment_type"),
			Zone:           argStrPtr(args, "zone"),
			Comments:       argStrPtr(args, "comments"),
			Active:         argBool(args, "active"),
		}))
		if err != nil {
			return mcp.NewToolResultError(connectErr(err)), nil
		}
		return mcp.NewToolResultText(fmt.Sprintf("Created trip #%d: %s", resp.Msg.Trip.Id, resp.Msg.Trip.LoadNumber)), nil
	})

	register(mcp.NewTool("update_trip",
		mcp.WithDescription("Update an existing trip."),
		mcp.WithNumber("id", mcp.Required(), mcp.Description("Trip ID")),
		mcp.WithNumber("truck_id", mcp.Description("Truck ID")),
		mcp.WithNumber("driver1_id", mcp.Description("Driver 1 employee ID")),
		mcp.WithString("trip_date", mcp.Description("Trip date (YYYY-MM-DD)")),
		mcp.WithString("est_deliver_date", mcp.Description("Estimated delivery date")),
		mcp.WithString("status", mcp.Description("Trip status")),
		mcp.WithString("equipment_type", mcp.Description("Equipment type")),
		mcp.WithString("zone", mcp.Description("Zone")),
		mcp.WithString("comments", mcp.Description("Comments")),
		mcp.WithBoolean("active", mcp.Description("Active")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := argMap(req)
		resp, err := client.trips.UpdateTrip(ctx, connect.NewRequest(&pb.UpdateTripRequest{
			Id:             int32(argInt(args, "id", 0)),
			TruckId:        argI32Ptr(args, "truck_id"),
			Driver1Id:      argI32Ptr(args, "driver1_id"),
			TripDate:       argStrPtr(args, "trip_date"),
			EstDeliverDate: argStrPtr(args, "est_deliver_date"),
			Status:         argStrPtr(args, "status"),
			EquipmentType:  argStrPtr(args, "equipment_type"),
			Zone:           argStrPtr(args, "zone"),
			Comments:       argStrPtr(args, "comments"),
			Active:         argBool(args, "active"),
		}))
		if err != nil {
			return mcp.NewToolResultError(connectErr(err)), nil
		}
		return mcp.NewToolResultText(fmt.Sprintf("Updated trip #%d: %s", resp.Msg.Trip.Id, resp.Msg.Trip.LoadNumber)), nil
	})

	register(mcp.NewTool("delete_trip",
		mcp.WithDescription("Delete a trip by ID."),
		mcp.WithNumber("id", mcp.Required(), mcp.Description("Trip ID")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := argMap(req)
		_, err := client.trips.DeleteTrip(ctx, connect.NewRequest(&pb.DeleteTripRequest{
			Id: int32(argInt(args, "id", 0)),
		}))
		if err != nil {
			return mcp.NewToolResultError(connectErr(err)), nil
		}
		return mcp.NewToolResultText("Trip deleted successfully."), nil
	})

	// --- Load Details ---
	register(mcp.NewTool("list_load_details",
		mcp.WithDescription("List load details (vehicles assigned) for a trip."),
		mcp.WithNumber("trip_id", mcp.Required(), mcp.Description("Trip ID")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := argMap(req)
		resp, err := client.trips.ListLoadDetails(ctx, connect.NewRequest(&pb.ListLoadDetailsRequest{
			TripId: int32(argInt(args, "trip_id", 0)),
		}))
		if err != nil {
			return mcp.NewToolResultError(connectErr(err)), nil
		}
		if len(resp.Msg.Details) == 0 {
			return mcp.NewToolResultText("No load details found."), nil
		}
		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("Load details (%d vehicles):\n\n", len(resp.Msg.Details)))
		for _, d := range resp.Msg.Details {
			sb.WriteString(fmt.Sprintf("  #%d VIN: %s %s %s %s", d.Id, deref(d.Vin), deref(d.Year), deref(d.Make), deref(d.Model)))
			if d.BayNumber != nil {
				sb.WriteString(fmt.Sprintf(" Bay: %s", *d.BayNumber))
			}
			if d.Status != nil {
				sb.WriteString(fmt.Sprintf(" [%s]", *d.Status))
			}
			sb.WriteString("\n")
		}
		return mcp.NewToolResultText(sb.String()), nil
	})

	register(mcp.NewTool("assign_vehicle_to_trip",
		mcp.WithDescription("Assign a vehicle to a trip."),
		mcp.WithNumber("trip_id", mcp.Required(), mcp.Description("Trip ID")),
		mcp.WithNumber("vehicle_id", mcp.Required(), mcp.Description("Vehicle ID")),
		mcp.WithString("bay_number", mcp.Description("Bay number")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := argMap(req)
		_, err := client.trips.AssignVehicleToTrip(ctx, connect.NewRequest(&pb.AssignVehicleToTripRequest{
			TripId:    int32(argInt(args, "trip_id", 0)),
			VehicleId: int32(argInt(args, "vehicle_id", 0)),
			BayNumber: argStrPtr(args, "bay_number"),
		}))
		if err != nil {
			return mcp.NewToolResultError(connectErr(err)), nil
		}
		return mcp.NewToolResultText("Vehicle assigned to trip successfully."), nil
	})

	register(mcp.NewTool("unassign_vehicle",
		mcp.WithDescription("Unassign a vehicle from a trip."),
		mcp.WithNumber("load_detail_id", mcp.Required(), mcp.Description("Load detail ID")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := argMap(req)
		_, err := client.trips.UnassignVehicle(ctx, connect.NewRequest(&pb.UnassignVehicleRequest{
			LoadDetailId: int32(argInt(args, "load_detail_id", 0)),
		}))
		if err != nil {
			return mcp.NewToolResultError(connectErr(err)), nil
		}
		return mcp.NewToolResultText("Vehicle unassigned successfully."), nil
	})

	register(mcp.NewTool("assign_all_from_order",
		mcp.WithDescription("Assign all vehicles from an order to a trip."),
		mcp.WithNumber("trip_id", mcp.Required(), mcp.Description("Trip ID")),
		mcp.WithNumber("order_id", mcp.Required(), mcp.Description("Order ID")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := argMap(req)
		resp, err := client.trips.AssignAllFromOrder(ctx, connect.NewRequest(&pb.AssignAllFromOrderRequest{
			TripId:  int32(argInt(args, "trip_id", 0)),
			OrderId: int32(argInt(args, "order_id", 0)),
		}))
		if err != nil {
			return mcp.NewToolResultError(connectErr(err)), nil
		}
		return mcp.NewToolResultText(fmt.Sprintf("Assigned %d vehicles from order to trip.", resp.Msg.Count)), nil
	})

	// --- Trip Fuel ---
	register(mcp.NewTool("list_trip_fuel",
		mcp.WithDescription("List fuel records for a trip."),
		mcp.WithNumber("trip_id", mcp.Required(), mcp.Description("Trip ID")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := argMap(req)
		resp, err := client.trips.ListTripFuel(ctx, connect.NewRequest(&pb.ListTripFuelRequest{
			TripId: int32(argInt(args, "trip_id", 0)),
		}))
		if err != nil {
			return mcp.NewToolResultError(connectErr(err)), nil
		}
		if len(resp.Msg.Items) == 0 {
			return mcp.NewToolResultText("No fuel records found."), nil
		}
		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("Fuel records (%d):\n\n", len(resp.Msg.Items)))
		for _, f := range resp.Msg.Items {
			sb.WriteString(fmt.Sprintf("  #%d State: %s Mileage: %s Gallons: %s\n",
				f.Id, deref(f.State), deref(f.Mileage), deref(f.Gallons)))
		}
		return mcp.NewToolResultText(sb.String()), nil
	})

	register(mcp.NewTool("create_trip_fuel",
		mcp.WithDescription("Create a fuel record for a trip."),
		mcp.WithNumber("trip_id", mcp.Required(), mcp.Description("Trip ID")),
		mcp.WithString("state", mcp.Description("State")),
		mcp.WithString("mileage", mcp.Description("Mileage")),
		mcp.WithString("gallons", mcp.Description("Gallons")),
		mcp.WithString("loaded_miles", mcp.Description("Loaded miles (true/false)")),
		mcp.WithString("truck_number", mcp.Description("Truck number")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := argMap(req)
		resp, err := client.trips.CreateTripFuel(ctx, connect.NewRequest(&pb.CreateTripFuelRequest{
			TripId:      int32(argInt(args, "trip_id", 0)),
			State:       argStrPtr(args, "state"),
			Mileage:     argStrPtr(args, "mileage"),
			Gallons:     argStrPtr(args, "gallons"),
			LoadedMiles: argStrPtr(args, "loaded_miles"),
			TruckNumber: argStrPtr(args, "truck_number"),
		}))
		if err != nil {
			return mcp.NewToolResultError(connectErr(err)), nil
		}
		return mcp.NewToolResultText(fmt.Sprintf("Created fuel record #%d.", resp.Msg.Item.Id)), nil
	})

	register(mcp.NewTool("update_trip_fuel",
		mcp.WithDescription("Update a fuel record."),
		mcp.WithNumber("id", mcp.Required(), mcp.Description("Fuel record ID")),
		mcp.WithNumber("trip_id", mcp.Required(), mcp.Description("Trip ID")),
		mcp.WithString("state", mcp.Description("State")),
		mcp.WithString("mileage", mcp.Description("Mileage")),
		mcp.WithString("gallons", mcp.Description("Gallons")),
		mcp.WithString("loaded_miles", mcp.Description("Loaded miles")),
		mcp.WithString("truck_number", mcp.Description("Truck number")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := argMap(req)
		_, err := client.trips.UpdateTripFuel(ctx, connect.NewRequest(&pb.UpdateTripFuelRequest{
			Id:          int32(argInt(args, "id", 0)),
			TripId:      int32(argInt(args, "trip_id", 0)),
			State:       argStrPtr(args, "state"),
			Mileage:     argStrPtr(args, "mileage"),
			Gallons:     argStrPtr(args, "gallons"),
			LoadedMiles: argStrPtr(args, "loaded_miles"),
			TruckNumber: argStrPtr(args, "truck_number"),
		}))
		if err != nil {
			return mcp.NewToolResultError(connectErr(err)), nil
		}
		return mcp.NewToolResultText("Fuel record updated."), nil
	})

	register(mcp.NewTool("delete_trip_fuel",
		mcp.WithDescription("Delete a fuel record."),
		mcp.WithNumber("id", mcp.Required(), mcp.Description("Fuel record ID")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := argMap(req)
		_, err := client.trips.DeleteTripFuel(ctx, connect.NewRequest(&pb.DeleteTripFuelRequest{
			Id: int32(argInt(args, "id", 0)),
		}))
		if err != nil {
			return mcp.NewToolResultError(connectErr(err)), nil
		}
		return mcp.NewToolResultText("Fuel record deleted."), nil
	})

	// --- Trip Expenses ---
	register(mcp.NewTool("list_trip_expenses",
		mcp.WithDescription("List expenses for a trip."),
		mcp.WithNumber("trip_id", mcp.Required(), mcp.Description("Trip ID")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := argMap(req)
		resp, err := client.trips.ListTripExpenses(ctx, connect.NewRequest(&pb.ListTripExpensesRequest{
			TripId: int32(argInt(args, "trip_id", 0)),
		}))
		if err != nil {
			return mcp.NewToolResultError(connectErr(err)), nil
		}
		if len(resp.Msg.Items) == 0 {
			return mcp.NewToolResultText("No expenses found."), nil
		}
		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("Expenses (%d):\n\n", len(resp.Msg.Items)))
		for _, e := range resp.Msg.Items {
			sb.WriteString(fmt.Sprintf("  #%d %s $%s (%s)\n",
				e.Id, deref(e.Description), deref(e.Amount), deref(e.ExpenseDate)))
		}
		return mcp.NewToolResultText(sb.String()), nil
	})

	register(mcp.NewTool("create_trip_expense",
		mcp.WithDescription("Create an expense for a trip."),
		mcp.WithNumber("trip_id", mcp.Required(), mcp.Description("Trip ID")),
		mcp.WithString("description", mcp.Description("Description")),
		mcp.WithString("amount", mcp.Description("Amount")),
		mcp.WithString("expense_date", mcp.Description("Expense date (YYYY-MM-DD)")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := argMap(req)
		resp, err := client.trips.CreateTripExpense(ctx, connect.NewRequest(&pb.CreateTripExpenseRequest{
			TripId:      int32(argInt(args, "trip_id", 0)),
			Description: argStrPtr(args, "description"),
			Amount:      argStrPtr(args, "amount"),
			ExpenseDate: argStrPtr(args, "expense_date"),
		}))
		if err != nil {
			return mcp.NewToolResultError(connectErr(err)), nil
		}
		return mcp.NewToolResultText(fmt.Sprintf("Created expense #%d.", resp.Msg.Item.Id)), nil
	})

	register(mcp.NewTool("update_trip_expense",
		mcp.WithDescription("Update a trip expense."),
		mcp.WithNumber("id", mcp.Required(), mcp.Description("Expense ID")),
		mcp.WithNumber("trip_id", mcp.Required(), mcp.Description("Trip ID")),
		mcp.WithString("description", mcp.Description("Description")),
		mcp.WithString("amount", mcp.Description("Amount")),
		mcp.WithString("expense_date", mcp.Description("Expense date (YYYY-MM-DD)")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := argMap(req)
		_, err := client.trips.UpdateTripExpense(ctx, connect.NewRequest(&pb.UpdateTripExpenseRequest{
			Id:          int32(argInt(args, "id", 0)),
			TripId:      int32(argInt(args, "trip_id", 0)),
			Description: argStrPtr(args, "description"),
			Amount:      argStrPtr(args, "amount"),
			ExpenseDate: argStrPtr(args, "expense_date"),
		}))
		if err != nil {
			return mcp.NewToolResultError(connectErr(err)), nil
		}
		return mcp.NewToolResultText("Expense updated."), nil
	})

	register(mcp.NewTool("delete_trip_expense",
		mcp.WithDescription("Delete a trip expense."),
		mcp.WithNumber("id", mcp.Required(), mcp.Description("Expense ID")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := argMap(req)
		_, err := client.trips.DeleteTripExpense(ctx, connect.NewRequest(&pb.DeleteTripExpenseRequest{
			Id: int32(argInt(args, "id", 0)),
		}))
		if err != nil {
			return mcp.NewToolResultError(connectErr(err)), nil
		}
		return mcp.NewToolResultText("Expense deleted."), nil
	})

	// --- Trip Routes ---
	register(mcp.NewTool("list_trip_routes",
		mcp.WithDescription("List route stops for a trip."),
		mcp.WithNumber("trip_id", mcp.Required(), mcp.Description("Trip ID")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := argMap(req)
		resp, err := client.trips.ListTripRoutes(ctx, connect.NewRequest(&pb.ListTripRoutesRequest{
			TripId: int32(argInt(args, "trip_id", 0)),
		}))
		if err != nil {
			return mcp.NewToolResultError(connectErr(err)), nil
		}
		if len(resp.Msg.Items) == 0 {
			return mcp.NewToolResultText("No route stops found."), nil
		}
		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("Route stops (%d):\n\n", len(resp.Msg.Items)))
		for _, r := range resp.Msg.Items {
			seq := ""
			if r.Sequence != nil {
				seq = fmt.Sprintf("%d. ", *r.Sequence)
			}
			sb.WriteString(fmt.Sprintf("  #%d %s%s %s, %s [%s] Miles: %s\n",
				r.Id, seq, deref(r.CustomerName), deref(r.City), deref(r.State), deref(r.StopType), deref(r.Miles)))
		}
		return mcp.NewToolResultText(sb.String()), nil
	})

	register(mcp.NewTool("create_trip_route",
		mcp.WithDescription("Create a route stop for a trip."),
		mcp.WithNumber("trip_id", mcp.Required(), mcp.Description("Trip ID")),
		mcp.WithNumber("sequence", mcp.Description("Stop sequence number")),
		mcp.WithNumber("customer_id", mcp.Description("Customer ID")),
		mcp.WithString("customer_name", mcp.Description("Customer name")),
		mcp.WithString("city", mcp.Description("City")),
		mcp.WithString("state", mcp.Description("State")),
		mcp.WithString("stop_type", mcp.Description("Stop type (pickup/delivery)")),
		mcp.WithString("miles", mcp.Description("Miles")),
		mcp.WithString("est_arrival", mcp.Description("Estimated arrival (YYYY-MM-DD)")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := argMap(req)
		resp, err := client.trips.CreateTripRoute(ctx, connect.NewRequest(&pb.CreateTripRouteRequest{
			TripId:       int32(argInt(args, "trip_id", 0)),
			Sequence:     argI32Ptr(args, "sequence"),
			CustomerId:   argI32Ptr(args, "customer_id"),
			CustomerName: argStrPtr(args, "customer_name"),
			City:         argStrPtr(args, "city"),
			State:        argStrPtr(args, "state"),
			StopType:     argStrPtr(args, "stop_type"),
			Miles:        argStrPtr(args, "miles"),
			EstArrival:   argStrPtr(args, "est_arrival"),
		}))
		if err != nil {
			return mcp.NewToolResultError(connectErr(err)), nil
		}
		return mcp.NewToolResultText(fmt.Sprintf("Created route stop #%d.", resp.Msg.Item.Id)), nil
	})

	register(mcp.NewTool("update_trip_route",
		mcp.WithDescription("Update a route stop."),
		mcp.WithNumber("id", mcp.Required(), mcp.Description("Route stop ID")),
		mcp.WithNumber("trip_id", mcp.Required(), mcp.Description("Trip ID")),
		mcp.WithNumber("sequence", mcp.Description("Stop sequence number")),
		mcp.WithNumber("customer_id", mcp.Description("Customer ID")),
		mcp.WithString("customer_name", mcp.Description("Customer name")),
		mcp.WithString("city", mcp.Description("City")),
		mcp.WithString("state", mcp.Description("State")),
		mcp.WithString("stop_type", mcp.Description("Stop type")),
		mcp.WithString("miles", mcp.Description("Miles")),
		mcp.WithString("est_arrival", mcp.Description("Estimated arrival (YYYY-MM-DD)")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := argMap(req)
		_, err := client.trips.UpdateTripRoute(ctx, connect.NewRequest(&pb.UpdateTripRouteRequest{
			Id:           int32(argInt(args, "id", 0)),
			TripId:       int32(argInt(args, "trip_id", 0)),
			Sequence:     argI32Ptr(args, "sequence"),
			CustomerId:   argI32Ptr(args, "customer_id"),
			CustomerName: argStrPtr(args, "customer_name"),
			City:         argStrPtr(args, "city"),
			State:        argStrPtr(args, "state"),
			StopType:     argStrPtr(args, "stop_type"),
			Miles:        argStrPtr(args, "miles"),
			EstArrival:   argStrPtr(args, "est_arrival"),
		}))
		if err != nil {
			return mcp.NewToolResultError(connectErr(err)), nil
		}
		return mcp.NewToolResultText("Route stop updated."), nil
	})

	register(mcp.NewTool("delete_trip_route",
		mcp.WithDescription("Delete a route stop."),
		mcp.WithNumber("id", mcp.Required(), mcp.Description("Route stop ID")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := argMap(req)
		_, err := client.trips.DeleteTripRoute(ctx, connect.NewRequest(&pb.DeleteTripRouteRequest{
			Id: int32(argInt(args, "id", 0)),
		}))
		if err != nil {
			return mcp.NewToolResultError(connectErr(err)), nil
		}
		return mcp.NewToolResultText("Route stop deleted."), nil
	})
}

func formatTripList(resp *pb.ListTripsResponse) string {
	if len(resp.Trips) == 0 {
		return "No trips found."
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Found %d trips (page %d, %d per page):\n\n",
		resp.Pagination.TotalCount, resp.Pagination.Page, resp.Pagination.PageSize))
	for _, t := range resp.Trips {
		sb.WriteString(fmt.Sprintf("  #%d %s", t.Id, t.LoadNumber))
		if t.TruckNumber != nil {
			sb.WriteString(fmt.Sprintf(" Truck: %s", *t.TruckNumber))
		}
		if t.Driver != nil {
			sb.WriteString(fmt.Sprintf(" Driver: %s", *t.Driver))
		}
		if t.TripDate != nil {
			sb.WriteString(fmt.Sprintf(" Date: %s", *t.TripDate))
		}
		if t.Status != nil {
			sb.WriteString(fmt.Sprintf(" [%s]", *t.Status))
		}
		if !t.Active {
			sb.WriteString(" [INACTIVE]")
		}
		sb.WriteString("\n")
	}
	return sb.String()
}

func formatTrip(t *pb.Trip) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Trip #%d: %s\n", t.Id, t.LoadNumber))
	sb.WriteString(fmt.Sprintf("  Active: %v\n", t.Active))
	if t.TruckNumber != nil {
		sb.WriteString(fmt.Sprintf("  Truck: %s\n", *t.TruckNumber))
	}
	if t.Driver != nil {
		sb.WriteString(fmt.Sprintf("  Driver: %s\n", *t.Driver))
	}
	if t.TripDate != nil {
		sb.WriteString(fmt.Sprintf("  Trip Date: %s\n", *t.TripDate))
	}
	if t.Status != nil {
		sb.WriteString(fmt.Sprintf("  Status: %s\n", *t.Status))
	}
	if t.Zone != nil {
		sb.WriteString(fmt.Sprintf("  Zone: %s\n", *t.Zone))
	}
	if t.Comments != nil {
		sb.WriteString(fmt.Sprintf("  Comments: %s\n", *t.Comments))
	}
	return sb.String()
}
