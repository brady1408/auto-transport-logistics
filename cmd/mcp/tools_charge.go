package main

import (
	"context"
	"fmt"
	"strings"

	"connectrpc.com/connect"
	"github.com/mark3labs/mcp-go/mcp"

	pb "github.com/brady1408/auto-transport-logistics/internal/gen/atlinks/v1"
)

func registerChargeTools(register toolRegister, client *atlClient) {
	register(mcp.NewTool("list_charges",
		mcp.WithDescription("List charges for an order. Returns description, amount, item_code, qty, rate, and calc_type."),
		mcp.WithNumber("order_id", mcp.Required(), mcp.Description("Order ID")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := argMap(req)
		resp, err := client.charges.ListCharges(ctx, connect.NewRequest(&pb.ListChargesRequest{
			OrderId: int32(argInt(args, "order_id", 0)),
		}))
		if err != nil {
			return mcp.NewToolResultError(connectErr(err)), nil
		}
		return mcp.NewToolResultText(formatChargeList(resp.Msg)), nil
	})

	register(mcp.NewTool("get_charge",
		mcp.WithDescription("Get a single charge by ID."),
		mcp.WithNumber("id", mcp.Required(), mcp.Description("Charge ID")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := argMap(req)
		resp, err := client.charges.GetCharge(ctx, connect.NewRequest(&pb.GetChargeRequest{
			Id: int32(argInt(args, "id", 0)),
		}))
		if err != nil {
			return mcp.NewToolResultError(connectErr(err)), nil
		}
		return mcp.NewToolResultText(formatCharge(resp.Msg.Charge)), nil
	})

	register(mcp.NewTool("create_charge",
		mcp.WithDescription("Create a new charge on an order. order_id is required."),
		mcp.WithNumber("order_id", mcp.Required(), mcp.Description("Order ID")),
		mcp.WithNumber("vehicle_id", mcp.Description("Vehicle ID")),
		mcp.WithNumber("trip_id", mcp.Description("Trip ID")),
		mcp.WithString("description", mcp.Description("Charge description")),
		mcp.WithString("amount", mcp.Description("Charge amount")),
		mcp.WithString("item_code", mcp.Description("Item code")),
		mcp.WithNumber("qty", mcp.Description("Quantity")),
		mcp.WithString("rate", mcp.Description("Rate")),
		mcp.WithString("calc_type", mcp.Description("Calculation type")),
		mcp.WithBoolean("taxable", mcp.Description("Taxable")),
		mcp.WithBoolean("billable", mcp.Description("Billable")),
		mcp.WithBoolean("ap_payable", mcp.Description("AP payable")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := argMap(req)
		resp, err := client.charges.CreateCharge(ctx, connect.NewRequest(&pb.CreateChargeRequest{
			OrderId:     int32(argInt(args, "order_id", 0)),
			VehicleId:   argI32Ptr(args, "vehicle_id"),
			TripId:      argI32Ptr(args, "trip_id"),
			Description: argStrPtr(args, "description"),
			Amount:      argStrPtr(args, "amount"),
			ItemCode:    argStrPtr(args, "item_code"),
			Qty:         argI32Ptr(args, "qty"),
			Rate:        argStrPtr(args, "rate"),
			CalcType:    argStrPtr(args, "calc_type"),
			Taxable:     argBool(args, "taxable"),
			Billable:    argBool(args, "billable"),
			ApPayable:   argBool(args, "ap_payable"),
		}))
		if err != nil {
			return mcp.NewToolResultError(connectErr(err)), nil
		}
		return mcp.NewToolResultText(fmt.Sprintf("Created charge #%d on order %d", resp.Msg.Charge.Id, resp.Msg.Charge.OrderId)), nil
	})

	register(mcp.NewTool("update_charge",
		mcp.WithDescription("Update an existing charge. ID and order_id are required."),
		mcp.WithNumber("id", mcp.Required(), mcp.Description("Charge ID")),
		mcp.WithNumber("order_id", mcp.Required(), mcp.Description("Order ID")),
		mcp.WithNumber("vehicle_id", mcp.Description("Vehicle ID")),
		mcp.WithNumber("trip_id", mcp.Description("Trip ID")),
		mcp.WithString("description", mcp.Description("Charge description")),
		mcp.WithString("amount", mcp.Description("Charge amount")),
		mcp.WithString("item_code", mcp.Description("Item code")),
		mcp.WithNumber("qty", mcp.Description("Quantity")),
		mcp.WithString("rate", mcp.Description("Rate")),
		mcp.WithString("calc_type", mcp.Description("Calculation type")),
		mcp.WithBoolean("taxable", mcp.Description("Taxable")),
		mcp.WithBoolean("billable", mcp.Description("Billable")),
		mcp.WithBoolean("ap_payable", mcp.Description("AP payable")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := argMap(req)
		resp, err := client.charges.UpdateCharge(ctx, connect.NewRequest(&pb.UpdateChargeRequest{
			Id:          int32(argInt(args, "id", 0)),
			OrderId:     int32(argInt(args, "order_id", 0)),
			VehicleId:   argI32Ptr(args, "vehicle_id"),
			TripId:      argI32Ptr(args, "trip_id"),
			Description: argStrPtr(args, "description"),
			Amount:      argStrPtr(args, "amount"),
			ItemCode:    argStrPtr(args, "item_code"),
			Qty:         argI32Ptr(args, "qty"),
			Rate:        argStrPtr(args, "rate"),
			CalcType:    argStrPtr(args, "calc_type"),
			Taxable:     argBool(args, "taxable"),
			Billable:    argBool(args, "billable"),
			ApPayable:   argBool(args, "ap_payable"),
		}))
		if err != nil {
			return mcp.NewToolResultError(connectErr(err)), nil
		}
		return mcp.NewToolResultText(fmt.Sprintf("Updated charge #%d on order %d", resp.Msg.Charge.Id, resp.Msg.Charge.OrderId)), nil
	})

	register(mcp.NewTool("delete_charge",
		mcp.WithDescription("Delete a charge by ID."),
		mcp.WithNumber("id", mcp.Required(), mcp.Description("Charge ID")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := argMap(req)
		_, err := client.charges.DeleteCharge(ctx, connect.NewRequest(&pb.DeleteChargeRequest{
			Id: int32(argInt(args, "id", 0)),
		}))
		if err != nil {
			return mcp.NewToolResultError(connectErr(err)), nil
		}
		return mcp.NewToolResultText("Charge deleted successfully."), nil
	})
}

func formatChargeList(resp *pb.ListChargesResponse) string {
	if len(resp.Charges) == 0 {
		return "No charges found."
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Found %d charges:\n\n", len(resp.Charges)))
	for _, c := range resp.Charges {
		sb.WriteString(fmt.Sprintf("  #%d", c.Id))
		if c.Description != nil {
			sb.WriteString(fmt.Sprintf(" %s", *c.Description))
		}
		if c.Amount != nil {
			sb.WriteString(fmt.Sprintf(" $%s", *c.Amount))
		}
		if c.ItemCode != nil {
			sb.WriteString(fmt.Sprintf(" [%s]", *c.ItemCode))
		}
		if c.Qty != nil {
			sb.WriteString(fmt.Sprintf(" qty:%d", *c.Qty))
		}
		if c.Rate != nil {
			sb.WriteString(fmt.Sprintf(" rate:%s", *c.Rate))
		}
		if c.CalcType != nil {
			sb.WriteString(fmt.Sprintf(" calc:%s", *c.CalcType))
		}
		sb.WriteString("\n")
	}
	return sb.String()
}

func formatCharge(c *pb.OrderCharge) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Charge #%d (Order %d)\n", c.Id, c.OrderId))
	if c.VehicleId != nil {
		sb.WriteString(fmt.Sprintf("  Vehicle ID: %d\n", *c.VehicleId))
	}
	if c.TripId != nil {
		sb.WriteString(fmt.Sprintf("  Trip ID: %d\n", *c.TripId))
	}
	if c.Description != nil {
		sb.WriteString(fmt.Sprintf("  Description: %s\n", *c.Description))
	}
	if c.Amount != nil {
		sb.WriteString(fmt.Sprintf("  Amount: $%s\n", *c.Amount))
	}
	if c.ItemCode != nil {
		sb.WriteString(fmt.Sprintf("  Item Code: %s\n", *c.ItemCode))
	}
	if c.Qty != nil {
		sb.WriteString(fmt.Sprintf("  Qty: %d\n", *c.Qty))
	}
	if c.Rate != nil {
		sb.WriteString(fmt.Sprintf("  Rate: %s\n", *c.Rate))
	}
	if c.CalcType != nil {
		sb.WriteString(fmt.Sprintf("  Calc Type: %s\n", *c.CalcType))
	}
	sb.WriteString(fmt.Sprintf("  Taxable: %v | Billable: %v | AP Payable: %v\n", c.Taxable, c.Billable, c.ApPayable))
	return sb.String()
}
