package main

import (
	"context"
	"fmt"
	"strings"

	"connectrpc.com/connect"
	"github.com/mark3labs/mcp-go/mcp"

	pb "github.com/brady1408/atlinks/internal/gen/atlinks/v1"
)

func registerDamageClaimTools(register toolRegister, client *atlClient) {
	register(mcp.NewTool("list_damage_claims",
		mcp.WithDescription("List damage claims with optional filters. Returns paginated results."),
		mcp.WithString("search", mcp.Description("Search by claim number or VIN")),
		mcp.WithString("status", mcp.Description("Filter by status")),
		mcp.WithNumber("page", mcp.Description("Page number (default: 1)")),
		mcp.WithNumber("page_size", mcp.Description("Items per page (default: 25)")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := argMap(req)
		resp, err := client.damageClaims.ListDamageClaims(ctx, connect.NewRequest(&pb.ListDamageClaimsRequest{
			Pagination: &pb.PaginationRequest{
				Page:     int32(argInt(args, "page", 1)),
				PageSize: int32(argInt(args, "page_size", 25)),
			},
			Search: argStrPtr(args, "search"),
			Status: argStrPtr(args, "status"),
		}))
		if err != nil {
			return mcp.NewToolResultError(connectErr(err)), nil
		}
		return mcp.NewToolResultText(formatDamageClaimList(resp.Msg)), nil
	})

	register(mcp.NewTool("get_damage_claim",
		mcp.WithDescription("Get a single damage claim by ID."),
		mcp.WithNumber("id", mcp.Required(), mcp.Description("Damage claim ID")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := argMap(req)
		resp, err := client.damageClaims.GetDamageClaim(ctx, connect.NewRequest(&pb.GetDamageClaimRequest{
			Id: int32(argInt(args, "id", 0)),
		}))
		if err != nil {
			return mcp.NewToolResultError(connectErr(err)), nil
		}
		return mcp.NewToolResultText(formatDamageClaim(resp.Msg.Claim)), nil
	})

	register(mcp.NewTool("create_damage_claim",
		mcp.WithDescription("Create a new damage claim."),
		mcp.WithNumber("order_id", mcp.Description("Order ID")),
		mcp.WithNumber("vehicle_id", mcp.Description("Vehicle ID")),
		mcp.WithNumber("trip_id", mcp.Description("Trip ID")),
		mcp.WithString("vin", mcp.Description("Vehicle VIN")),
		mcp.WithString("claim_date", mcp.Description("Claim date (YYYY-MM-DD)")),
		mcp.WithString("claim_amount", mcp.Description("Claim amount")),
		mcp.WithString("status", mcp.Description("Claim status")),
		mcp.WithString("description", mcp.Description("Description of damage")),
		mcp.WithString("insurance_claim", mcp.Description("Insurance claim flag")),
		mcp.WithString("insurance_claim_number", mcp.Description("Insurance claim number")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := argMap(req)
		resp, err := client.damageClaims.CreateDamageClaim(ctx, connect.NewRequest(&pb.CreateDamageClaimRequest{
			OrderId:              argI32Ptr(args, "order_id"),
			VehicleId:            argI32Ptr(args, "vehicle_id"),
			TripId:               argI32Ptr(args, "trip_id"),
			Vin:                  argStrPtr(args, "vin"),
			ClaimDate:            argStrPtr(args, "claim_date"),
			ClaimAmount:          argStrPtr(args, "claim_amount"),
			Status:               argStrPtr(args, "status"),
			Description:          argStrPtr(args, "description"),
			InsuranceClaim:       argStrPtr(args, "insurance_claim"),
			InsuranceClaimNumber: argStrPtr(args, "insurance_claim_number"),
		}))
		if err != nil {
			return mcp.NewToolResultError(connectErr(err)), nil
		}
		return mcp.NewToolResultText(fmt.Sprintf("Created damage claim #%d: %s", resp.Msg.Claim.Id, resp.Msg.Claim.ClaimNumber)), nil
	})

	register(mcp.NewTool("update_damage_claim",
		mcp.WithDescription("Update an existing damage claim. ID is required."),
		mcp.WithNumber("id", mcp.Required(), mcp.Description("Damage claim ID")),
		mcp.WithNumber("order_id", mcp.Description("Order ID")),
		mcp.WithNumber("vehicle_id", mcp.Description("Vehicle ID")),
		mcp.WithNumber("trip_id", mcp.Description("Trip ID")),
		mcp.WithString("vin", mcp.Description("Vehicle VIN")),
		mcp.WithString("claim_date", mcp.Description("Claim date (YYYY-MM-DD)")),
		mcp.WithString("claim_amount", mcp.Description("Claim amount")),
		mcp.WithString("paid_amount", mcp.Description("Paid amount")),
		mcp.WithString("status", mcp.Description("Claim status")),
		mcp.WithString("description", mcp.Description("Description of damage")),
		mcp.WithString("insurance_claim", mcp.Description("Insurance claim flag")),
		mcp.WithString("insurance_claim_number", mcp.Description("Insurance claim number")),
		mcp.WithString("resolution", mcp.Description("Resolution details")),
		mcp.WithString("resolved_date", mcp.Description("Resolved date (YYYY-MM-DD)")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := argMap(req)
		resp, err := client.damageClaims.UpdateDamageClaim(ctx, connect.NewRequest(&pb.UpdateDamageClaimRequest{
			Id:                   int32(argInt(args, "id", 0)),
			OrderId:              argI32Ptr(args, "order_id"),
			VehicleId:            argI32Ptr(args, "vehicle_id"),
			TripId:               argI32Ptr(args, "trip_id"),
			Vin:                  argStrPtr(args, "vin"),
			ClaimDate:            argStrPtr(args, "claim_date"),
			ClaimAmount:          argStrPtr(args, "claim_amount"),
			PaidAmount:           argStrPtr(args, "paid_amount"),
			Status:               argStrPtr(args, "status"),
			Description:          argStrPtr(args, "description"),
			InsuranceClaim:       argStrPtr(args, "insurance_claim"),
			InsuranceClaimNumber: argStrPtr(args, "insurance_claim_number"),
			Resolution:           argStrPtr(args, "resolution"),
			ResolvedDate:         argStrPtr(args, "resolved_date"),
		}))
		if err != nil {
			return mcp.NewToolResultError(connectErr(err)), nil
		}
		return mcp.NewToolResultText(fmt.Sprintf("Updated damage claim #%d: %s", resp.Msg.Claim.Id, resp.Msg.Claim.ClaimNumber)), nil
	})

	register(mcp.NewTool("delete_damage_claim",
		mcp.WithDescription("Delete a damage claim by ID."),
		mcp.WithNumber("id", mcp.Required(), mcp.Description("Damage claim ID")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := argMap(req)
		_, err := client.damageClaims.DeleteDamageClaim(ctx, connect.NewRequest(&pb.DeleteDamageClaimRequest{
			Id: int32(argInt(args, "id", 0)),
		}))
		if err != nil {
			return mcp.NewToolResultError(connectErr(err)), nil
		}
		return mcp.NewToolResultText("Damage claim deleted successfully."), nil
	})
}

func formatDamageClaimList(resp *pb.ListDamageClaimsResponse) string {
	if len(resp.Claims) == 0 {
		return "No damage claims found."
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Found %d damage claims (page %d, %d per page):\n\n",
		resp.Pagination.TotalCount, resp.Pagination.Page, resp.Pagination.PageSize))
	for _, c := range resp.Claims {
		sb.WriteString(fmt.Sprintf("  #%d %s", c.Id, c.ClaimNumber))
		if c.Vin != nil {
			sb.WriteString(fmt.Sprintf(" VIN:%s", *c.Vin))
		}
		if c.ClaimDate != nil {
			sb.WriteString(fmt.Sprintf(" %s", *c.ClaimDate))
		}
		if c.Status != nil {
			sb.WriteString(fmt.Sprintf(" [%s]", *c.Status))
		}
		if c.ClaimAmount != nil {
			sb.WriteString(fmt.Sprintf(" $%s", *c.ClaimAmount))
		}
		if c.PaidAmount != nil {
			sb.WriteString(fmt.Sprintf(" (paid: $%s)", *c.PaidAmount))
		}
		sb.WriteString("\n")
	}
	return sb.String()
}

func formatDamageClaim(c *pb.DamageClaim) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Damage Claim #%d: %s\n", c.Id, c.ClaimNumber))
	if c.Vin != nil {
		sb.WriteString(fmt.Sprintf("  VIN: %s\n", *c.Vin))
	}
	if c.OrderId != nil {
		sb.WriteString(fmt.Sprintf("  Order ID: %d\n", *c.OrderId))
	}
	if c.VehicleId != nil {
		sb.WriteString(fmt.Sprintf("  Vehicle ID: %d\n", *c.VehicleId))
	}
	if c.TripId != nil {
		sb.WriteString(fmt.Sprintf("  Trip ID: %d\n", *c.TripId))
	}
	if c.ClaimDate != nil {
		sb.WriteString(fmt.Sprintf("  Claim Date: %s\n", *c.ClaimDate))
	}
	if c.ClaimAmount != nil {
		sb.WriteString(fmt.Sprintf("  Claim Amount: $%s\n", *c.ClaimAmount))
	}
	if c.PaidAmount != nil {
		sb.WriteString(fmt.Sprintf("  Paid Amount: $%s\n", *c.PaidAmount))
	}
	if c.Status != nil {
		sb.WriteString(fmt.Sprintf("  Status: %s\n", *c.Status))
	}
	if c.Description != nil {
		sb.WriteString(fmt.Sprintf("  Description: %s\n", *c.Description))
	}
	if c.InsuranceClaim != nil {
		sb.WriteString(fmt.Sprintf("  Insurance Claim: %s\n", *c.InsuranceClaim))
	}
	if c.InsuranceClaimNumber != nil {
		sb.WriteString(fmt.Sprintf("  Insurance Claim #: %s\n", *c.InsuranceClaimNumber))
	}
	if c.Resolution != nil {
		sb.WriteString(fmt.Sprintf("  Resolution: %s\n", *c.Resolution))
	}
	if c.ResolvedDate != nil {
		sb.WriteString(fmt.Sprintf("  Resolved Date: %s\n", *c.ResolvedDate))
	}
	return sb.String()
}
