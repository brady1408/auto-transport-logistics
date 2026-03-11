package main

import (
	"context"
	"fmt"
	"strings"

	"connectrpc.com/connect"
	"github.com/mark3labs/mcp-go/mcp"

	pb "github.com/brady1408/atlinks/internal/gen/atlinks/v1"
)

func registerVehicleTools(register toolRegister, client *atlClient) {
	register(mcp.NewTool("list_vehicles",
		mcp.WithDescription("List all vehicles on an order."),
		mcp.WithNumber("order_id", mcp.Required(), mcp.Description("Order ID")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := argMap(req)
		resp, err := client.vehicles.ListVehicles(ctx, connect.NewRequest(&pb.ListVehiclesRequest{
			OrderId: int32(argInt(args, "order_id", 0)),
		}))
		if err != nil {
			return mcp.NewToolResultError(connectErr(err)), nil
		}
		return mcp.NewToolResultText(formatVehicleList(resp.Msg)), nil
	})

	register(mcp.NewTool("get_vehicle",
		mcp.WithDescription("Get a single vehicle by ID."),
		mcp.WithNumber("id", mcp.Required(), mcp.Description("Vehicle ID")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := argMap(req)
		resp, err := client.vehicles.GetVehicle(ctx, connect.NewRequest(&pb.GetVehicleRequest{
			Id: int32(argInt(args, "id", 0)),
		}))
		if err != nil {
			return mcp.NewToolResultError(connectErr(err)), nil
		}
		return mcp.NewToolResultText(formatVehicle(resp.Msg.Vehicle)), nil
	})

	register(mcp.NewTool("create_vehicle",
		mcp.WithDescription("Create a vehicle on an order. Status defaults to 'Waiting'."),
		mcp.WithNumber("order_id", mcp.Required(), mcp.Description("Order ID to add vehicle to")),
		mcp.WithString("vin", mcp.Description("Vehicle Identification Number")),
		mcp.WithString("year", mcp.Description("Model year")),
		mcp.WithString("make", mcp.Description("Manufacturer (e.g. Toyota)")),
		mcp.WithString("model", mcp.Description("Model name (e.g. Camry)")),
		mcp.WithString("color", mcp.Description("Vehicle color")),
		mcp.WithBoolean("active", mcp.Description("Active (default: true)"), mcp.DefaultBool(true)),
		mcp.WithBoolean("operable", mcp.Description("Is vehicle operable"), mcp.DefaultBool(true)),
		mcp.WithBoolean("run_drive", mcp.Description("Can vehicle run/drive"), mcp.DefaultBool(true)),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := argMap(req)
		active := true
		if v, ok := args["active"]; ok {
			if b, ok := v.(bool); ok {
				active = b
			}
		}
		operable := true
		if v, ok := args["operable"]; ok {
			if b, ok := v.(bool); ok {
				operable = b
			}
		}
		runDrive := true
		if v, ok := args["run_drive"]; ok {
			if b, ok := v.(bool); ok {
				runDrive = b
			}
		}
		resp, err := client.vehicles.CreateVehicle(ctx, connect.NewRequest(&pb.CreateVehicleRequest{
			OrderId:  int32(argInt(args, "order_id", 0)),
			Active:   active,
			Vin:      argStrPtr(args, "vin"),
			Year:     argStrPtr(args, "year"),
			Make:     argStrPtr(args, "make"),
			Model:    argStrPtr(args, "model"),
			Color:    argStrPtr(args, "color"),
			Operable: operable,
			RunDrive: runDrive,
		}))
		if err != nil {
			return mcp.NewToolResultError(connectErr(err)), nil
		}
		v := resp.Msg.Vehicle
		return mcp.NewToolResultText(fmt.Sprintf("Created vehicle #%d on order %d (VIN: %s, Status: %s)",
			v.Id, v.OrderId, deref(v.Vin), v.Status)), nil
	})

	register(mcp.NewTool("update_vehicle",
		mcp.WithDescription("Update vehicle details. Does not change status — use update_vehicle_status for that."),
		mcp.WithNumber("id", mcp.Required(), mcp.Description("Vehicle ID")),
		mcp.WithString("vin", mcp.Description("Vehicle Identification Number")),
		mcp.WithString("year", mcp.Description("Model year")),
		mcp.WithString("make", mcp.Description("Manufacturer")),
		mcp.WithString("model", mcp.Description("Model name")),
		mcp.WithString("color", mcp.Description("Vehicle color")),
		mcp.WithBoolean("active", mcp.Description("Active")),
		mcp.WithBoolean("operable", mcp.Description("Is vehicle operable")),
		mcp.WithBoolean("run_drive", mcp.Description("Can vehicle run/drive")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := argMap(req)
		resp, err := client.vehicles.UpdateVehicle(ctx, connect.NewRequest(&pb.UpdateVehicleRequest{
			Id:       int32(argInt(args, "id", 0)),
			Active:   argBool(args, "active"),
			Vin:      argStrPtr(args, "vin"),
			Year:     argStrPtr(args, "year"),
			Make:     argStrPtr(args, "make"),
			Model:    argStrPtr(args, "model"),
			Color:    argStrPtr(args, "color"),
			Operable: argBool(args, "operable"),
			RunDrive: argBool(args, "run_drive"),
		}))
		if err != nil {
			return mcp.NewToolResultError(connectErr(err)), nil
		}
		return mcp.NewToolResultText(fmt.Sprintf("Updated vehicle #%d", resp.Msg.Vehicle.Id)), nil
	})

	register(mcp.NewTool("delete_vehicle",
		mcp.WithDescription("Delete (soft-delete) a vehicle and update order counts."),
		mcp.WithNumber("id", mcp.Required(), mcp.Description("Vehicle ID")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := argMap(req)
		_, err := client.vehicles.DeleteVehicle(ctx, connect.NewRequest(&pb.DeleteVehicleRequest{
			Id: int32(argInt(args, "id", 0)),
		}))
		if err != nil {
			return mcp.NewToolResultError(connectErr(err)), nil
		}
		return mcp.NewToolResultText("Vehicle deleted successfully."), nil
	})

	register(mcp.NewTool("update_vehicle_status",
		mcp.WithDescription("Transition a vehicle's status. Valid transitions: Waiting→Scheduled→Loaded→Delivered→Confirmed (and reverse). Updates order counts atomically."),
		mcp.WithNumber("id", mcp.Required(), mcp.Description("Vehicle ID")),
		mcp.WithString("status", mcp.Required(), mcp.Description("New status: Waiting, Scheduled, Loaded, Delivered, or Confirmed")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := argMap(req)
		resp, err := client.vehicles.UpdateVehicleStatus(ctx, connect.NewRequest(&pb.UpdateVehicleStatusRequest{
			Id:     int32(argInt(args, "id", 0)),
			Status: argStr(args, "status"),
		}))
		if err != nil {
			return mcp.NewToolResultError(connectErr(err)), nil
		}
		v := resp.Msg.Vehicle
		return mcp.NewToolResultText(fmt.Sprintf("Vehicle #%d status updated to: %s", v.Id, v.Status)), nil
	})
}

func formatVehicleList(resp *pb.ListVehiclesResponse) string {
	if len(resp.Vehicles) == 0 {
		return "No vehicles found on this order."
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("%d vehicles:\n\n", len(resp.Vehicles)))
	for _, v := range resp.Vehicles {
		sb.WriteString(fmt.Sprintf("  #%d %s %s %s — VIN: %s — Status: %s",
			v.Id, deref(v.Year), deref(v.Make), deref(v.Model), deref(v.Vin), v.Status))
		if v.TripId != nil {
			sb.WriteString(fmt.Sprintf(" (Trip #%d)", *v.TripId))
		}
		sb.WriteString("\n")
	}
	return sb.String()
}

func formatVehicle(v *pb.Vehicle) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Vehicle #%d (Order #%d)\n", v.Id, v.OrderId))
	sb.WriteString(fmt.Sprintf("  VIN: %s\n", deref(v.Vin)))
	sb.WriteString(fmt.Sprintf("  Year/Make/Model: %s %s %s\n", deref(v.Year), deref(v.Make), deref(v.Model)))
	if v.Color != nil {
		sb.WriteString(fmt.Sprintf("  Color: %s\n", *v.Color))
	}
	sb.WriteString(fmt.Sprintf("  Status: %s\n", v.Status))
	sb.WriteString(fmt.Sprintf("  Active: %v | Operable: %v | Run/Drive: %v\n", v.Active, v.Operable, v.RunDrive))
	if v.TripId != nil {
		sb.WriteString(fmt.Sprintf("  Trip: #%d\n", *v.TripId))
	}
	if v.TotalCharge != nil {
		sb.WriteString(fmt.Sprintf("  Total Charge: $%s\n", *v.TotalCharge))
	}
	return sb.String()
}
