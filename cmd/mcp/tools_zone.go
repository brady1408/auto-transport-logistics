package main

import (
	"context"
	"fmt"
	"strings"

	"connectrpc.com/connect"
	"github.com/mark3labs/mcp-go/mcp"

	pb "github.com/brady1408/atlinks/internal/gen/atlinks/v1"
)

func registerZoneTools(register toolRegister, client *atlClient) {
	// ── Zone CRUD ──────────────────────────────────────────────────────

	register(mcp.NewTool("list_zones",
		mcp.WithDescription("List all zones. Returns zone, description, and region."),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		resp, err := client.zones.ListZones(ctx, connect.NewRequest(&pb.ListZonesRequest{}))
		if err != nil {
			return mcp.NewToolResultError(connectErr(err)), nil
		}
		return mcp.NewToolResultText(formatZoneList(resp.Msg)), nil
	})

	register(mcp.NewTool("get_zone",
		mcp.WithDescription("Get a single zone by ID."),
		mcp.WithNumber("id", mcp.Required(), mcp.Description("Zone ID")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := argMap(req)
		resp, err := client.zones.GetZone(ctx, connect.NewRequest(&pb.GetZoneRequest{
			Id: int32(argInt(args, "id", 0)),
		}))
		if err != nil {
			return mcp.NewToolResultError(connectErr(err)), nil
		}
		return mcp.NewToolResultText(formatZone(resp.Msg.Zone)), nil
	})

	register(mcp.NewTool("create_zone",
		mcp.WithDescription("Create a new zone. Zone code is required."),
		mcp.WithString("zone", mcp.Required(), mcp.Description("Zone code")),
		mcp.WithString("description", mcp.Description("Zone description")),
		mcp.WithString("region", mcp.Description("Region")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := argMap(req)
		resp, err := client.zones.CreateZone(ctx, connect.NewRequest(&pb.CreateZoneRequest{
			Zone:        argStr(args, "zone"),
			Description: argStrPtr(args, "description"),
			Region:      argStrPtr(args, "region"),
		}))
		if err != nil {
			return mcp.NewToolResultError(connectErr(err)), nil
		}
		return mcp.NewToolResultText(fmt.Sprintf("Created zone #%d: %s", resp.Msg.Zone.Id, resp.Msg.Zone.Zone)), nil
	})

	register(mcp.NewTool("update_zone",
		mcp.WithDescription("Update an existing zone. ID and zone code are required."),
		mcp.WithNumber("id", mcp.Required(), mcp.Description("Zone ID")),
		mcp.WithString("zone", mcp.Required(), mcp.Description("Zone code")),
		mcp.WithString("description", mcp.Description("Zone description")),
		mcp.WithString("region", mcp.Description("Region")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := argMap(req)
		resp, err := client.zones.UpdateZone(ctx, connect.NewRequest(&pb.UpdateZoneRequest{
			Id:          int32(argInt(args, "id", 0)),
			Zone:        argStr(args, "zone"),
			Description: argStrPtr(args, "description"),
			Region:      argStrPtr(args, "region"),
		}))
		if err != nil {
			return mcp.NewToolResultError(connectErr(err)), nil
		}
		return mcp.NewToolResultText(fmt.Sprintf("Updated zone #%d: %s", resp.Msg.Zone.Id, resp.Msg.Zone.Zone)), nil
	})

	register(mcp.NewTool("delete_zone",
		mcp.WithDescription("Delete a zone by ID."),
		mcp.WithNumber("id", mcp.Required(), mcp.Description("Zone ID")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := argMap(req)
		_, err := client.zones.DeleteZone(ctx, connect.NewRequest(&pb.DeleteZoneRequest{
			Id: int32(argInt(args, "id", 0)),
		}))
		if err != nil {
			return mcp.NewToolResultError(connectErr(err)), nil
		}
		return mcp.NewToolResultText("Zone deleted successfully."), nil
	})

	// ── ZonePricing CRUD ───────────────────────────────────────────────

	register(mcp.NewTool("list_zone_pricing",
		mcp.WithDescription("List all zone pricing entries. Returns zone_a, zone_b, description, amount, and miles."),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		resp, err := client.zones.ListZonePricing(ctx, connect.NewRequest(&pb.ListZonePricingRequest{}))
		if err != nil {
			return mcp.NewToolResultError(connectErr(err)), nil
		}
		return mcp.NewToolResultText(formatZonePricingList(resp.Msg)), nil
	})

	register(mcp.NewTool("get_zone_pricing",
		mcp.WithDescription("Get a single zone pricing entry by ID."),
		mcp.WithNumber("id", mcp.Required(), mcp.Description("Zone pricing ID")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := argMap(req)
		resp, err := client.zones.GetZonePricing(ctx, connect.NewRequest(&pb.GetZonePricingRequest{
			Id: int32(argInt(args, "id", 0)),
		}))
		if err != nil {
			return mcp.NewToolResultError(connectErr(err)), nil
		}
		return mcp.NewToolResultText(formatZonePricing(resp.Msg.Item)), nil
	})

	register(mcp.NewTool("create_zone_pricing",
		mcp.WithDescription("Create a new zone pricing entry. zone_a and zone_b are required."),
		mcp.WithString("zone_a", mcp.Required(), mcp.Description("Origin zone code")),
		mcp.WithString("zone_b", mcp.Required(), mcp.Description("Destination zone code")),
		mcp.WithString("description", mcp.Description("Description")),
		mcp.WithString("amount", mcp.Description("Price amount")),
		mcp.WithNumber("miles", mcp.Description("Miles between zones")),
		mcp.WithNumber("transport_days", mcp.Description("Transit time in days")),
		mcp.WithString("ship_to", mcp.Description("Ship-to location")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := argMap(req)
		resp, err := client.zones.CreateZonePricing(ctx, connect.NewRequest(&pb.CreateZonePricingRequest{
			ZoneA:         argStr(args, "zone_a"),
			ZoneB:         argStr(args, "zone_b"),
			Description:   argStrPtr(args, "description"),
			Amount:        argStrPtr(args, "amount"),
			Miles:         argI32Ptr(args, "miles"),
			TransportDays: argI32Ptr(args, "transport_days"),
			ShipTo:        argStrPtr(args, "ship_to"),
		}))
		if err != nil {
			return mcp.NewToolResultError(connectErr(err)), nil
		}
		return mcp.NewToolResultText(fmt.Sprintf("Created zone pricing #%d: %s → %s", resp.Msg.Item.Id, resp.Msg.Item.ZoneA, resp.Msg.Item.ZoneB)), nil
	})

	register(mcp.NewTool("update_zone_pricing",
		mcp.WithDescription("Update an existing zone pricing entry. ID, zone_a, and zone_b are required."),
		mcp.WithNumber("id", mcp.Required(), mcp.Description("Zone pricing ID")),
		mcp.WithString("zone_a", mcp.Required(), mcp.Description("Origin zone code")),
		mcp.WithString("zone_b", mcp.Required(), mcp.Description("Destination zone code")),
		mcp.WithString("description", mcp.Description("Description")),
		mcp.WithString("amount", mcp.Description("Price amount")),
		mcp.WithNumber("miles", mcp.Description("Miles between zones")),
		mcp.WithNumber("transport_days", mcp.Description("Transit time in days")),
		mcp.WithString("ship_to", mcp.Description("Ship-to location")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := argMap(req)
		resp, err := client.zones.UpdateZonePricing(ctx, connect.NewRequest(&pb.UpdateZonePricingRequest{
			Id:            int32(argInt(args, "id", 0)),
			ZoneA:         argStr(args, "zone_a"),
			ZoneB:         argStr(args, "zone_b"),
			Description:   argStrPtr(args, "description"),
			Amount:        argStrPtr(args, "amount"),
			Miles:         argI32Ptr(args, "miles"),
			TransportDays: argI32Ptr(args, "transport_days"),
			ShipTo:        argStrPtr(args, "ship_to"),
		}))
		if err != nil {
			return mcp.NewToolResultError(connectErr(err)), nil
		}
		return mcp.NewToolResultText(fmt.Sprintf("Updated zone pricing #%d: %s → %s", resp.Msg.Item.Id, resp.Msg.Item.ZoneA, resp.Msg.Item.ZoneB)), nil
	})

	register(mcp.NewTool("delete_zone_pricing",
		mcp.WithDescription("Delete a zone pricing entry by ID."),
		mcp.WithNumber("id", mcp.Required(), mcp.Description("Zone pricing ID")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := argMap(req)
		_, err := client.zones.DeleteZonePricing(ctx, connect.NewRequest(&pb.DeleteZonePricingRequest{
			Id: int32(argInt(args, "id", 0)),
		}))
		if err != nil {
			return mcp.NewToolResultError(connectErr(err)), nil
		}
		return mcp.NewToolResultText("Zone pricing deleted successfully."), nil
	})
}

func formatZoneList(resp *pb.ListZonesResponse) string {
	if len(resp.Zones) == 0 {
		return "No zones found."
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Found %d zones:\n\n", len(resp.Zones)))
	for _, z := range resp.Zones {
		sb.WriteString(fmt.Sprintf("  #%d %s", z.Id, z.Zone))
		if z.Description != nil {
			sb.WriteString(fmt.Sprintf(" — %s", *z.Description))
		}
		if z.Region != nil {
			sb.WriteString(fmt.Sprintf(" [%s]", *z.Region))
		}
		sb.WriteString("\n")
	}
	return sb.String()
}

func formatZone(z *pb.Zone) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Zone #%d: %s\n", z.Id, z.Zone))
	if z.Description != nil {
		sb.WriteString(fmt.Sprintf("  Description: %s\n", *z.Description))
	}
	if z.Region != nil {
		sb.WriteString(fmt.Sprintf("  Region: %s\n", *z.Region))
	}
	return sb.String()
}

func formatZonePricingList(resp *pb.ListZonePricingResponse) string {
	if len(resp.Items) == 0 {
		return "No zone pricing entries found."
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Found %d zone pricing entries:\n\n", len(resp.Items)))
	for _, p := range resp.Items {
		sb.WriteString(fmt.Sprintf("  #%d %s → %s", p.Id, p.ZoneA, p.ZoneB))
		if p.Description != nil {
			sb.WriteString(fmt.Sprintf(" — %s", *p.Description))
		}
		if p.Amount != nil {
			sb.WriteString(fmt.Sprintf(" $%s", *p.Amount))
		}
		if p.Miles != nil {
			sb.WriteString(fmt.Sprintf(" %d mi", *p.Miles))
		}
		sb.WriteString("\n")
	}
	return sb.String()
}

func formatZonePricing(p *pb.ZonePricing) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Zone Pricing #%d: %s → %s\n", p.Id, p.ZoneA, p.ZoneB))
	if p.Description != nil {
		sb.WriteString(fmt.Sprintf("  Description: %s\n", *p.Description))
	}
	if p.Amount != nil {
		sb.WriteString(fmt.Sprintf("  Amount: $%s\n", *p.Amount))
	}
	if p.Miles != nil {
		sb.WriteString(fmt.Sprintf("  Miles: %d\n", *p.Miles))
	}
	if p.TransportDays != nil {
		sb.WriteString(fmt.Sprintf("  Transport Days: %d\n", *p.TransportDays))
	}
	if p.ShipTo != nil {
		sb.WriteString(fmt.Sprintf("  Ship To: %s\n", *p.ShipTo))
	}
	return sb.String()
}
