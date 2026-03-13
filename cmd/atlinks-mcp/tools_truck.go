package main

import (
	"context"
	"fmt"
	"strings"

	"connectrpc.com/connect"
	"github.com/mark3labs/mcp-go/mcp"

	pb "github.com/brady1408/auto-transport-logistics/internal/gen/atlinks/v1"
)

func registerTruckTools(register toolRegister, client *atlClient) {
	register(mcp.NewTool("list_trucks",
		mcp.WithDescription("List trucks with optional filters. Returns paginated results."),
		mcp.WithString("search", mcp.Description("Search by truck number or make/model")),
		mcp.WithString("active", mcp.Description("Filter: 'active', 'inactive', or 'all' (default: all)")),
		mcp.WithString("leased_truck", mcp.Description("Filter: 'true', 'false', or omit for all")),
		mcp.WithString("class", mcp.Description("Filter by truck class")),
		mcp.WithNumber("page", mcp.Description("Page number (default: 1)")),
		mcp.WithNumber("page_size", mcp.Description("Items per page (default: 25)")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := argMap(req)
		resp, err := client.trucks.ListTrucks(ctx, connect.NewRequest(&pb.ListTrucksRequest{
			Pagination: &pb.PaginationRequest{
				Page:     int32(argInt(args, "page", 1)),
				PageSize: int32(argInt(args, "page_size", 25)),
			},
			Search:      argStrPtr(args, "search"),
			Active:      argStrPtr(args, "active"),
			LeasedTruck: argStrPtr(args, "leased_truck"),
			Class:       argStrPtr(args, "class"),
		}))
		if err != nil {
			return mcp.NewToolResultError(connectErr(err)), nil
		}
		return mcp.NewToolResultText(formatTruckList(resp.Msg)), nil
	})

	register(mcp.NewTool("get_truck",
		mcp.WithDescription("Get a single truck by ID."),
		mcp.WithNumber("id", mcp.Required(), mcp.Description("Truck ID")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := argMap(req)
		resp, err := client.trucks.GetTruck(ctx, connect.NewRequest(&pb.GetTruckRequest{
			Id: int32(argInt(args, "id", 0)),
		}))
		if err != nil {
			return mcp.NewToolResultError(connectErr(err)), nil
		}
		return mcp.NewToolResultText(formatTruck(resp.Msg.Truck)), nil
	})

	register(mcp.NewTool("create_truck",
		mcp.WithDescription("Create a new truck. Truck number is required."),
		mcp.WithString("truck_number", mcp.Required(), mcp.Description("Truck number")),
		mcp.WithString("truck_make", mcp.Description("Make")),
		mcp.WithString("truck_model", mcp.Description("Model")),
		mcp.WithString("truck_year", mcp.Description("Year")),
		mcp.WithString("trailer_number", mcp.Description("Trailer number")),
		mcp.WithString("class", mcp.Description("Truck class")),
		mcp.WithBoolean("active", mcp.Description("Active status (default: false)")),
		mcp.WithBoolean("leased_truck", mcp.Description("Is a leased truck")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := argMap(req)
		resp, err := client.trucks.CreateTruck(ctx, connect.NewRequest(&pb.CreateTruckRequest{
			TruckNumber:   argStr(args, "truck_number"),
			TruckMake:     argStrPtr(args, "truck_make"),
			TruckModel:    argStrPtr(args, "truck_model"),
			TruckYear:     argStrPtr(args, "truck_year"),
			TrailerNumber: argStrPtr(args, "trailer_number"),
			Class:         argStrPtr(args, "class"),
			Active:        argBool(args, "active"),
			LeasedTruck:   argBool(args, "leased_truck"),
		}))
		if err != nil {
			return mcp.NewToolResultError(connectErr(err)), nil
		}
		return mcp.NewToolResultText(fmt.Sprintf("Created truck #%d: %s", resp.Msg.Truck.Id, resp.Msg.Truck.TruckNumber)), nil
	})

	register(mcp.NewTool("update_truck",
		mcp.WithDescription("Update an existing truck. ID and truck number are required."),
		mcp.WithNumber("id", mcp.Required(), mcp.Description("Truck ID")),
		mcp.WithString("truck_number", mcp.Required(), mcp.Description("Truck number")),
		mcp.WithString("truck_make", mcp.Description("Make")),
		mcp.WithString("truck_model", mcp.Description("Model")),
		mcp.WithString("truck_year", mcp.Description("Year")),
		mcp.WithString("trailer_number", mcp.Description("Trailer number")),
		mcp.WithString("class", mcp.Description("Truck class")),
		mcp.WithBoolean("active", mcp.Description("Active status")),
		mcp.WithBoolean("leased_truck", mcp.Description("Is a leased truck")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := argMap(req)
		resp, err := client.trucks.UpdateTruck(ctx, connect.NewRequest(&pb.UpdateTruckRequest{
			Id:            int32(argInt(args, "id", 0)),
			TruckNumber:   argStr(args, "truck_number"),
			TruckMake:     argStrPtr(args, "truck_make"),
			TruckModel:    argStrPtr(args, "truck_model"),
			TruckYear:     argStrPtr(args, "truck_year"),
			TrailerNumber: argStrPtr(args, "trailer_number"),
			Class:         argStrPtr(args, "class"),
			Active:        argBool(args, "active"),
			LeasedTruck:   argBool(args, "leased_truck"),
		}))
		if err != nil {
			return mcp.NewToolResultError(connectErr(err)), nil
		}
		return mcp.NewToolResultText(fmt.Sprintf("Updated truck #%d: %s", resp.Msg.Truck.Id, resp.Msg.Truck.TruckNumber)), nil
	})

	register(mcp.NewTool("delete_truck",
		mcp.WithDescription("Delete (soft-delete) a truck by ID."),
		mcp.WithNumber("id", mcp.Required(), mcp.Description("Truck ID")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := argMap(req)
		_, err := client.trucks.DeleteTruck(ctx, connect.NewRequest(&pb.DeleteTruckRequest{
			Id: int32(argInt(args, "id", 0)),
		}))
		if err != nil {
			return mcp.NewToolResultError(connectErr(err)), nil
		}
		return mcp.NewToolResultText("Truck deleted successfully."), nil
	})
}

func formatTruckList(resp *pb.ListTrucksResponse) string {
	if len(resp.Trucks) == 0 {
		return "No trucks found."
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Found %d trucks (page %d, %d per page):\n\n",
		resp.Pagination.TotalCount, resp.Pagination.Page, resp.Pagination.PageSize))
	for _, t := range resp.Trucks {
		sb.WriteString(fmt.Sprintf("  #%d %s", t.Id, t.TruckNumber))
		parts := []string{}
		if t.TruckMake != nil {
			parts = append(parts, *t.TruckMake)
		}
		if t.TruckModel != nil {
			parts = append(parts, *t.TruckModel)
		}
		if t.TruckYear != nil {
			parts = append(parts, *t.TruckYear)
		}
		if len(parts) > 0 {
			sb.WriteString(fmt.Sprintf(" (%s)", strings.Join(parts, " ")))
		}
		if !t.Active {
			sb.WriteString(" [INACTIVE]")
		}
		if t.Driver1 != nil {
			sb.WriteString(fmt.Sprintf(" Driver: %s", *t.Driver1))
		}
		sb.WriteString("\n")
	}
	return sb.String()
}

func formatTruck(t *pb.Truck) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Truck #%d: %s\n", t.Id, t.TruckNumber))
	if t.TruckMake != nil || t.TruckModel != nil || t.TruckYear != nil {
		sb.WriteString(fmt.Sprintf("  Make/Model/Year: %s %s %s\n", deref(t.TruckMake), deref(t.TruckModel), deref(t.TruckYear)))
	}
	if t.TrailerNumber != nil {
		sb.WriteString(fmt.Sprintf("  Trailer: %s\n", *t.TrailerNumber))
	}
	if t.Class != nil {
		sb.WriteString(fmt.Sprintf("  Class: %s\n", *t.Class))
	}
	if t.Driver1 != nil {
		sb.WriteString(fmt.Sprintf("  Driver 1: %s\n", *t.Driver1))
	}
	if t.Driver2 != nil {
		sb.WriteString(fmt.Sprintf("  Driver 2: %s\n", *t.Driver2))
	}
	if t.TruckRate != nil {
		sb.WriteString(fmt.Sprintf("  Rate: %s\n", *t.TruckRate))
	}
	sb.WriteString(fmt.Sprintf("  Active: %v | Leased: %v\n", t.Active, t.LeasedTruck))
	return sb.String()
}
