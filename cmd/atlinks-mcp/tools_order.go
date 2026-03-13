package main

import (
	"context"
	"fmt"
	"strings"

	"connectrpc.com/connect"
	"github.com/mark3labs/mcp-go/mcp"

	pb "github.com/brady1408/auto-transport-logistics/internal/gen/atlinks/v1"
)

func registerOrderTools(register toolRegister, client *atlClient) {
	register(mcp.NewTool("list_orders",
		mcp.WithDescription("List orders with optional filters. Returns paginated results."),
		mcp.WithString("search", mcp.Description("Search by order number or customer name")),
		mcp.WithString("origin_zone", mcp.Description("Filter by origin zone")),
		mcp.WithString("destination_zone", mcp.Description("Filter by destination zone")),
		mcp.WithString("dispatch_code", mcp.Description("Filter by dispatch code")),
		mcp.WithString("active", mcp.Description("Filter: 'active', 'inactive', or '' (default: all)")),
		mcp.WithString("status", mcp.Description("Filter: 'uninvoiced_delivered' for orders with delivered vehicles not yet invoiced")),
		mcp.WithString("date_from", mcp.Description("Filter orders created on or after this date (YYYY-MM-DD)")),
		mcp.WithString("date_to", mcp.Description("Filter orders created on or before this date (YYYY-MM-DD)")),
		mcp.WithNumber("page", mcp.Description("Page number (default: 1)")),
		mcp.WithNumber("page_size", mcp.Description("Items per page (default: 25)")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := argMap(req)
		resp, err := client.orders.ListOrders(ctx, connect.NewRequest(&pb.ListOrdersRequest{
			Pagination: &pb.PaginationRequest{
				Page:     int32(argInt(args, "page", 1)),
				PageSize: int32(argInt(args, "page_size", 25)),
			},
			Search:       argStrPtr(args, "search"),
			OriginZone:      argStrPtr(args, "origin_zone"),
			DestinationZone: argStrPtr(args, "destination_zone"),
			DispatchCode:    argStrPtr(args, "dispatch_code"),
			Active:       argStrPtr(args, "active"),
			Status:       argStrPtr(args, "status"),
			DateFrom:     argStrPtr(args, "date_from"),
			DateTo:       argStrPtr(args, "date_to"),
		}))
		if err != nil {
			return mcp.NewToolResultError(connectErr(err)), nil
		}
		return mcp.NewToolResultText(formatOrderList(resp.Msg)), nil
	})

	register(mcp.NewTool("get_order",
		mcp.WithDescription("Get a single order by ID, including status counts."),
		mcp.WithNumber("id", mcp.Required(), mcp.Description("Order ID")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := argMap(req)
		resp, err := client.orders.GetOrder(ctx, connect.NewRequest(&pb.GetOrderRequest{
			Id: int32(argInt(args, "id", 0)),
		}))
		if err != nil {
			return mcp.NewToolResultError(connectErr(err)), nil
		}
		return mcp.NewToolResultText(formatOrder(resp.Msg.Order)), nil
	})

	register(mcp.NewTool("create_order",
		mcp.WithDescription("Create a new order. Order number is auto-generated if not provided. Set active=true for new orders."),
		mcp.WithString("order_number", mcp.Description("Order number (auto-generated if empty)")),
		mcp.WithBoolean("active", mcp.Description("Whether order is active (default: true)"), mcp.DefaultBool(true)),
		mcp.WithString("origin_zone", mcp.Description("Origin zone")),
		mcp.WithString("destination_zone", mcp.Description("Destination zone")),
		mcp.WithString("dispatch_code", mcp.Description("Dispatch code")),
		mcp.WithNumber("bill_customer_id", mcp.Description("Bill-to customer ID")),
		mcp.WithString("bill_customer_name", mcp.Description("Bill-to customer name")),
		mcp.WithNumber("load_customer_id", mcp.Description("Pickup/load customer ID")),
		mcp.WithString("load_customer_name", mcp.Description("Pickup customer name")),
		mcp.WithNumber("drop_customer_id", mcp.Description("Drop/delivery customer ID")),
		mcp.WithString("drop_customer_name", mcp.Description("Drop customer name")),
		mcp.WithString("comments", mcp.Description("Order comments")),
		mcp.WithString("pu_instructions", mcp.Description("Pickup instructions")),
		mcp.WithString("do_instructions", mcp.Description("Delivery instructions")),
		mcp.WithString("est_pickup_date", mcp.Description("Estimated pickup date (YYYY-MM-DD)")),
		mcp.WithString("est_deliver_date", mcp.Description("Estimated delivery date (YYYY-MM-DD)")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := argMap(req)
		active := true
		if v, ok := args["active"]; ok {
			if b, ok := v.(bool); ok {
				active = b
			}
		}
		resp, err := client.orders.CreateOrder(ctx, connect.NewRequest(&pb.CreateOrderRequest{
			OrderNumber:      argStrPtr(args, "order_number"),
			Active:           active,
			OriginZone:       argStrPtr(args, "origin_zone"),
			DestinationZone:  argStrPtr(args, "destination_zone"),
			DispatchCode:     argStrPtr(args, "dispatch_code"),
			BillCustomerId:   argI32Ptr(args, "bill_customer_id"),
			BillCustomerName: argStrPtr(args, "bill_customer_name"),
			LoadCustomerId:   argI32Ptr(args, "load_customer_id"),
			LoadCustomerName: argStrPtr(args, "load_customer_name"),
			DropCustomerId:   argI32Ptr(args, "drop_customer_id"),
			DropCustomerName: argStrPtr(args, "drop_customer_name"),
			Comments:         argStrPtr(args, "comments"),
			PuInstructions:   argStrPtr(args, "pu_instructions"),
			DoInstructions:   argStrPtr(args, "do_instructions"),
			EstPickupDate:    argStrPtr(args, "est_pickup_date"),
			EstDeliverDate:   argStrPtr(args, "est_deliver_date"),
		}))
		if err != nil {
			return mcp.NewToolResultError(connectErr(err)), nil
		}
		return mcp.NewToolResultText(fmt.Sprintf("Created order #%d (number: %s)", resp.Msg.Order.Id, resp.Msg.Order.OrderNumber)), nil
	})

	register(mcp.NewTool("update_order",
		mcp.WithDescription("Update an existing order. ID is required."),
		mcp.WithNumber("id", mcp.Required(), mcp.Description("Order ID")),
		mcp.WithBoolean("active", mcp.Description("Whether order is active")),
		mcp.WithString("origin_zone", mcp.Description("Origin zone")),
		mcp.WithString("destination_zone", mcp.Description("Destination zone")),
		mcp.WithString("dispatch_code", mcp.Description("Dispatch code")),
		mcp.WithNumber("bill_customer_id", mcp.Description("Bill-to customer ID")),
		mcp.WithString("bill_customer_name", mcp.Description("Bill-to customer name")),
		mcp.WithNumber("load_customer_id", mcp.Description("Pickup/load customer ID")),
		mcp.WithString("load_customer_name", mcp.Description("Pickup customer name")),
		mcp.WithNumber("drop_customer_id", mcp.Description("Drop/delivery customer ID")),
		mcp.WithString("drop_customer_name", mcp.Description("Drop customer name")),
		mcp.WithString("comments", mcp.Description("Order comments")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := argMap(req)
		resp, err := client.orders.UpdateOrder(ctx, connect.NewRequest(&pb.UpdateOrderRequest{
			Id:               int32(argInt(args, "id", 0)),
			Active:           argBool(args, "active"),
			OriginZone:       argStrPtr(args, "origin_zone"),
			DestinationZone:  argStrPtr(args, "destination_zone"),
			DispatchCode:     argStrPtr(args, "dispatch_code"),
			BillCustomerId:   argI32Ptr(args, "bill_customer_id"),
			BillCustomerName: argStrPtr(args, "bill_customer_name"),
			LoadCustomerId:   argI32Ptr(args, "load_customer_id"),
			LoadCustomerName: argStrPtr(args, "load_customer_name"),
			DropCustomerId:   argI32Ptr(args, "drop_customer_id"),
			DropCustomerName: argStrPtr(args, "drop_customer_name"),
			Comments:         argStrPtr(args, "comments"),
		}))
		if err != nil {
			return mcp.NewToolResultError(connectErr(err)), nil
		}
		return mcp.NewToolResultText(fmt.Sprintf("Updated order #%d (number: %s)", resp.Msg.Order.Id, resp.Msg.Order.OrderNumber)), nil
	})

	register(mcp.NewTool("delete_order",
		mcp.WithDescription("Delete (soft-delete) an order by ID."),
		mcp.WithNumber("id", mcp.Required(), mcp.Description("Order ID")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := argMap(req)
		_, err := client.orders.DeleteOrder(ctx, connect.NewRequest(&pb.DeleteOrderRequest{
			Id: int32(argInt(args, "id", 0)),
		}))
		if err != nil {
			return mcp.NewToolResultError(connectErr(err)), nil
		}
		return mcp.NewToolResultText("Order deleted successfully."), nil
	})
}

func formatOrderList(resp *pb.ListOrdersResponse) string {
	if len(resp.Orders) == 0 {
		return "No orders found."
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Found %d orders (page %d, %d per page):\n\n",
		resp.Pagination.TotalCount, resp.Pagination.Page, resp.Pagination.PageSize))
	for _, o := range resp.Orders {
		sb.WriteString(fmt.Sprintf("  #%d Order %s", o.Id, o.OrderNumber))
		if o.BillCustomerName != nil {
			sb.WriteString(fmt.Sprintf(" — Bill: %s", *o.BillCustomerName))
		}
		sb.WriteString(fmt.Sprintf(" | Vehicles: %d (W:%d S:%d L:%d D:%d C:%d)",
			o.VehicleCount, o.WaitingCount, o.ScheduledCount, o.LoadedCount, o.DeliveredCount, o.ConfirmedCount))
		if !o.Active {
			sb.WriteString(" [INACTIVE]")
		}
		sb.WriteString("\n")
	}
	return sb.String()
}

func formatOrder(o *pb.Order) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Order #%d — %s\n", o.Id, o.OrderNumber))
	sb.WriteString(fmt.Sprintf("  Active: %v\n", o.Active))
	if o.OriginZone != nil {
		sb.WriteString(fmt.Sprintf("  Origin Zone: %s\n", *o.OriginZone))
	}
	if o.DestinationZone != nil {
		sb.WriteString(fmt.Sprintf("  Destination Zone: %s\n", *o.DestinationZone))
	}
	if o.DispatchCode != nil {
		sb.WriteString(fmt.Sprintf("  Dispatch: %s\n", *o.DispatchCode))
	}
	if o.BillCustomerName != nil {
		sb.WriteString(fmt.Sprintf("  Bill To: %s\n", *o.BillCustomerName))
	}
	if o.LoadCustomerName != nil {
		sb.WriteString(fmt.Sprintf("  Pickup: %s\n", *o.LoadCustomerName))
	}
	if o.DropCustomerName != nil {
		sb.WriteString(fmt.Sprintf("  Delivery: %s\n", *o.DropCustomerName))
	}
	sb.WriteString(fmt.Sprintf("  Vehicles: %d total | Waiting: %d | Scheduled: %d | Loaded: %d | Delivered: %d | Confirmed: %d | Invoiced: %d\n",
		o.VehicleCount, o.WaitingCount, o.ScheduledCount, o.LoadedCount, o.DeliveredCount, o.ConfirmedCount, o.InvoicedCount))
	if o.TotalCharge != nil {
		sb.WriteString(fmt.Sprintf("  Total Charge: $%s\n", *o.TotalCharge))
	}
	return sb.String()
}
