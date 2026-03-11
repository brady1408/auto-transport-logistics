package main

import (
	"context"
	"fmt"
	"strings"

	"connectrpc.com/connect"
	"github.com/mark3labs/mcp-go/mcp"

	pb "github.com/brady1408/atlinks/internal/gen/atlinks/v1"
)

func registerDamageTools(register toolRegister, client *atlClient) {
	// ── Damage CRUD ────────────────────────────────────────────────────

	register(mcp.NewTool("list_damages_by_vehicle",
		mcp.WithDescription("List damages for a vehicle. Returns damage_area, damage_type, severity, description, and claim_status."),
		mcp.WithNumber("vehicle_id", mcp.Required(), mcp.Description("Vehicle ID")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := argMap(req)
		resp, err := client.damages.ListDamagesByVehicle(ctx, connect.NewRequest(&pb.ListDamagesByVehicleRequest{
			VehicleId: int32(argInt(args, "vehicle_id", 0)),
		}))
		if err != nil {
			return mcp.NewToolResultError(connectErr(err)), nil
		}
		return mcp.NewToolResultText(formatDamageList(resp.Msg)), nil
	})

	register(mcp.NewTool("list_damages_by_trip",
		mcp.WithDescription("List damages for a trip. Returns damage_area, damage_type, severity, description, and claim_status."),
		mcp.WithNumber("trip_id", mcp.Required(), mcp.Description("Trip ID")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := argMap(req)
		resp, err := client.damages.ListDamagesByTrip(ctx, connect.NewRequest(&pb.ListDamagesByTripRequest{
			TripId: int32(argInt(args, "trip_id", 0)),
		}))
		if err != nil {
			return mcp.NewToolResultError(connectErr(err)), nil
		}
		return mcp.NewToolResultText(formatDamageList(resp.Msg)), nil
	})

	register(mcp.NewTool("get_damage",
		mcp.WithDescription("Get a single damage record by ID."),
		mcp.WithNumber("id", mcp.Required(), mcp.Description("Damage ID")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := argMap(req)
		resp, err := client.damages.GetDamage(ctx, connect.NewRequest(&pb.GetDamageRequest{
			Id: int32(argInt(args, "id", 0)),
		}))
		if err != nil {
			return mcp.NewToolResultError(connectErr(err)), nil
		}
		return mcp.NewToolResultText(formatDamage(resp.Msg.Damage)), nil
	})

	register(mcp.NewTool("create_damage",
		mcp.WithDescription("Create a new damage record. order_id and vehicle_id are required."),
		mcp.WithNumber("order_id", mcp.Required(), mcp.Description("Order ID")),
		mcp.WithNumber("vehicle_id", mcp.Required(), mcp.Description("Vehicle ID")),
		mcp.WithNumber("trip_id", mcp.Description("Trip ID")),
		mcp.WithString("vin", mcp.Description("VIN")),
		mcp.WithString("damage_area", mcp.Description("Damage area")),
		mcp.WithString("damage_type", mcp.Description("Damage type")),
		mcp.WithString("damage_severity", mcp.Description("Damage severity")),
		mcp.WithString("description", mcp.Description("Description")),
		mcp.WithString("inspection_point", mcp.Description("Inspection point")),
		mcp.WithString("inspected_by", mcp.Description("Inspected by")),
		mcp.WithString("inspection_date", mcp.Description("Inspection date")),
		mcp.WithString("claim_amount", mcp.Description("Claim amount")),
		mcp.WithString("claim_status", mcp.Description("Claim status")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := argMap(req)
		resp, err := client.damages.CreateDamage(ctx, connect.NewRequest(&pb.CreateDamageRequest{
			OrderId:         int32(argInt(args, "order_id", 0)),
			VehicleId:       int32(argInt(args, "vehicle_id", 0)),
			TripId:          argI32Ptr(args, "trip_id"),
			Vin:             argStrPtr(args, "vin"),
			DamageArea:      argStrPtr(args, "damage_area"),
			DamageType:      argStrPtr(args, "damage_type"),
			DamageSeverity:  argStrPtr(args, "damage_severity"),
			Description:     argStrPtr(args, "description"),
			InspectionPoint: argStrPtr(args, "inspection_point"),
			InspectedBy:     argStrPtr(args, "inspected_by"),
			InspectionDate:  argStrPtr(args, "inspection_date"),
			ClaimAmount:     argStrPtr(args, "claim_amount"),
			ClaimStatus:     argStrPtr(args, "claim_status"),
		}))
		if err != nil {
			return mcp.NewToolResultError(connectErr(err)), nil
		}
		return mcp.NewToolResultText(fmt.Sprintf("Created damage #%d on vehicle %d", resp.Msg.Damage.Id, resp.Msg.Damage.VehicleId)), nil
	})

	register(mcp.NewTool("update_damage",
		mcp.WithDescription("Update an existing damage record. ID, order_id, and vehicle_id are required."),
		mcp.WithNumber("id", mcp.Required(), mcp.Description("Damage ID")),
		mcp.WithNumber("order_id", mcp.Required(), mcp.Description("Order ID")),
		mcp.WithNumber("vehicle_id", mcp.Required(), mcp.Description("Vehicle ID")),
		mcp.WithNumber("trip_id", mcp.Description("Trip ID")),
		mcp.WithString("vin", mcp.Description("VIN")),
		mcp.WithString("damage_area", mcp.Description("Damage area")),
		mcp.WithString("damage_type", mcp.Description("Damage type")),
		mcp.WithString("damage_severity", mcp.Description("Damage severity")),
		mcp.WithString("description", mcp.Description("Description")),
		mcp.WithString("inspection_point", mcp.Description("Inspection point")),
		mcp.WithString("inspected_by", mcp.Description("Inspected by")),
		mcp.WithString("inspection_date", mcp.Description("Inspection date")),
		mcp.WithString("claim_amount", mcp.Description("Claim amount")),
		mcp.WithString("claim_status", mcp.Description("Claim status")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := argMap(req)
		resp, err := client.damages.UpdateDamage(ctx, connect.NewRequest(&pb.UpdateDamageRequest{
			Id:              int32(argInt(args, "id", 0)),
			OrderId:         int32(argInt(args, "order_id", 0)),
			VehicleId:       int32(argInt(args, "vehicle_id", 0)),
			TripId:          argI32Ptr(args, "trip_id"),
			Vin:             argStrPtr(args, "vin"),
			DamageArea:      argStrPtr(args, "damage_area"),
			DamageType:      argStrPtr(args, "damage_type"),
			DamageSeverity:  argStrPtr(args, "damage_severity"),
			Description:     argStrPtr(args, "description"),
			InspectionPoint: argStrPtr(args, "inspection_point"),
			InspectedBy:     argStrPtr(args, "inspected_by"),
			InspectionDate:  argStrPtr(args, "inspection_date"),
			ClaimAmount:     argStrPtr(args, "claim_amount"),
			ClaimStatus:     argStrPtr(args, "claim_status"),
		}))
		if err != nil {
			return mcp.NewToolResultError(connectErr(err)), nil
		}
		return mcp.NewToolResultText(fmt.Sprintf("Updated damage #%d on vehicle %d", resp.Msg.Damage.Id, resp.Msg.Damage.VehicleId)), nil
	})

	register(mcp.NewTool("delete_damage",
		mcp.WithDescription("Delete a damage record by ID."),
		mcp.WithNumber("id", mcp.Required(), mcp.Description("Damage ID")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := argMap(req)
		_, err := client.damages.DeleteDamage(ctx, connect.NewRequest(&pb.DeleteDamageRequest{
			Id: int32(argInt(args, "id", 0)),
		}))
		if err != nil {
			return mcp.NewToolResultError(connectErr(err)), nil
		}
		return mcp.NewToolResultText("Damage deleted successfully."), nil
	})

	// ── Vehicle Notes ──────────────────────────────────────────────────

	register(mcp.NewTool("list_notes_by_vehicle",
		mcp.WithDescription("List notes for a vehicle. Returns note_date, description, comment, and created_by."),
		mcp.WithNumber("vehicle_id", mcp.Required(), mcp.Description("Vehicle ID")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := argMap(req)
		resp, err := client.damages.ListNotesByVehicle(ctx, connect.NewRequest(&pb.ListNotesByVehicleRequest{
			VehicleId: int32(argInt(args, "vehicle_id", 0)),
		}))
		if err != nil {
			return mcp.NewToolResultError(connectErr(err)), nil
		}
		return mcp.NewToolResultText(formatNoteList(resp.Msg)), nil
	})

	register(mcp.NewTool("create_note",
		mcp.WithDescription("Create a new vehicle note. vehicle_id is required."),
		mcp.WithNumber("vehicle_id", mcp.Required(), mcp.Description("Vehicle ID")),
		mcp.WithString("note_date", mcp.Description("Note date")),
		mcp.WithString("description", mcp.Description("Description")),
		mcp.WithString("comment", mcp.Description("Comment")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := argMap(req)
		resp, err := client.damages.CreateNote(ctx, connect.NewRequest(&pb.CreateNoteRequest{
			VehicleId:   int32(argInt(args, "vehicle_id", 0)),
			NoteDate:    argStrPtr(args, "note_date"),
			Description: argStrPtr(args, "description"),
			Comment:     argStrPtr(args, "comment"),
		}))
		if err != nil {
			return mcp.NewToolResultError(connectErr(err)), nil
		}
		return mcp.NewToolResultText(fmt.Sprintf("Created note #%d on vehicle %d", resp.Msg.Note.Id, resp.Msg.Note.VehicleId)), nil
	})

	register(mcp.NewTool("delete_note",
		mcp.WithDescription("Delete a vehicle note by ID."),
		mcp.WithNumber("id", mcp.Required(), mcp.Description("Note ID")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := argMap(req)
		_, err := client.damages.DeleteNote(ctx, connect.NewRequest(&pb.DeleteNoteRequest{
			Id: int32(argInt(args, "id", 0)),
		}))
		if err != nil {
			return mcp.NewToolResultError(connectErr(err)), nil
		}
		return mcp.NewToolResultText("Note deleted successfully."), nil
	})
}

func formatDamageList(resp *pb.ListDamagesResponse) string {
	if len(resp.Damages) == 0 {
		return "No damages found."
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Found %d damages:\n\n", len(resp.Damages)))
	for _, d := range resp.Damages {
		sb.WriteString(fmt.Sprintf("  #%d", d.Id))
		if d.DamageArea != nil {
			sb.WriteString(fmt.Sprintf(" %s", *d.DamageArea))
		}
		if d.DamageType != nil {
			sb.WriteString(fmt.Sprintf(" / %s", *d.DamageType))
		}
		if d.DamageSeverity != nil {
			sb.WriteString(fmt.Sprintf(" [%s]", *d.DamageSeverity))
		}
		if d.Description != nil {
			sb.WriteString(fmt.Sprintf(" — %s", *d.Description))
		}
		if d.ClaimStatus != nil {
			sb.WriteString(fmt.Sprintf(" (claim: %s)", *d.ClaimStatus))
		}
		sb.WriteString("\n")
	}
	return sb.String()
}

func formatDamage(d *pb.VehicleDamage) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Damage #%d (Order %d, Vehicle %d)\n", d.Id, d.OrderId, d.VehicleId))
	if d.TripId != nil {
		sb.WriteString(fmt.Sprintf("  Trip ID: %d\n", *d.TripId))
	}
	if d.Vin != nil {
		sb.WriteString(fmt.Sprintf("  VIN: %s\n", *d.Vin))
	}
	if d.DamageArea != nil {
		sb.WriteString(fmt.Sprintf("  Damage Area: %s\n", *d.DamageArea))
	}
	if d.DamageType != nil {
		sb.WriteString(fmt.Sprintf("  Damage Type: %s\n", *d.DamageType))
	}
	if d.DamageSeverity != nil {
		sb.WriteString(fmt.Sprintf("  Severity: %s\n", *d.DamageSeverity))
	}
	if d.Description != nil {
		sb.WriteString(fmt.Sprintf("  Description: %s\n", *d.Description))
	}
	if d.InspectionPoint != nil {
		sb.WriteString(fmt.Sprintf("  Inspection Point: %s\n", *d.InspectionPoint))
	}
	if d.InspectedBy != nil {
		sb.WriteString(fmt.Sprintf("  Inspected By: %s\n", *d.InspectedBy))
	}
	if d.InspectionDate != nil {
		sb.WriteString(fmt.Sprintf("  Inspection Date: %s\n", *d.InspectionDate))
	}
	if d.ClaimAmount != nil {
		sb.WriteString(fmt.Sprintf("  Claim Amount: $%s\n", *d.ClaimAmount))
	}
	if d.ClaimStatus != nil {
		sb.WriteString(fmt.Sprintf("  Claim Status: %s\n", *d.ClaimStatus))
	}
	return sb.String()
}

func formatNoteList(resp *pb.ListNotesResponse) string {
	if len(resp.Notes) == 0 {
		return "No notes found."
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Found %d notes:\n\n", len(resp.Notes)))
	for _, n := range resp.Notes {
		sb.WriteString(fmt.Sprintf("  #%d", n.Id))
		if n.NoteDate != nil {
			sb.WriteString(fmt.Sprintf(" %s", *n.NoteDate))
		}
		if n.Description != nil {
			sb.WriteString(fmt.Sprintf(" — %s", *n.Description))
		}
		if n.Comment != nil {
			sb.WriteString(fmt.Sprintf(" (%s)", *n.Comment))
		}
		if n.CreatedBy != nil {
			sb.WriteString(fmt.Sprintf(" by %s", *n.CreatedBy))
		}
		sb.WriteString("\n")
	}
	return sb.String()
}
