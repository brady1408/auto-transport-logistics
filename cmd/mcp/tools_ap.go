package main

import (
	"context"
	"fmt"
	"strings"

	"connectrpc.com/connect"
	"github.com/mark3labs/mcp-go/mcp"

	pb "github.com/brady1408/atlinks/internal/gen/atlinks/v1"
)

func registerAPTools(register toolRegister, client *atlClient) {
	register(mcp.NewTool("list_ap",
		mcp.WithDescription("List accounts payable with optional filters. Returns paginated results."),
		mcp.WithString("search", mcp.Description("Search by vendor or description")),
		mcp.WithString("status", mcp.Description("Filter by status")),
		mcp.WithNumber("employee_id", mcp.Description("Filter by employee ID")),
		mcp.WithNumber("truck_id", mcp.Description("Filter by truck ID")),
		mcp.WithNumber("page", mcp.Description("Page number (default: 1)")),
		mcp.WithNumber("page_size", mcp.Description("Items per page (default: 25)")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := argMap(req)
		resp, err := client.ap.ListAP(ctx, connect.NewRequest(&pb.ListAPRequest{
			Pagination: &pb.PaginationRequest{
				Page:     int32(argInt(args, "page", 1)),
				PageSize: int32(argInt(args, "page_size", 25)),
			},
			Search:     argStrPtr(args, "search"),
			Status:     argStrPtr(args, "status"),
			EmployeeId: argI32Ptr(args, "employee_id"),
			TruckId:    argI32Ptr(args, "truck_id"),
		}))
		if err != nil {
			return mcp.NewToolResultError(connectErr(err)), nil
		}
		return mcp.NewToolResultText(formatAPList(resp.Msg)), nil
	})

	register(mcp.NewTool("get_ap",
		mcp.WithDescription("Get a single accounts payable record by ID."),
		mcp.WithNumber("id", mcp.Required(), mcp.Description("AP ID")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := argMap(req)
		resp, err := client.ap.GetAP(ctx, connect.NewRequest(&pb.GetAPRequest{
			Id: int32(argInt(args, "id", 0)),
		}))
		if err != nil {
			return mcp.NewToolResultError(connectErr(err)), nil
		}
		return mcp.NewToolResultText(formatAP(resp.Msg.Item)), nil
	})

	register(mcp.NewTool("create_ap",
		mcp.WithDescription("Create a new accounts payable record."),
		mcp.WithNumber("trip_id", mcp.Description("Trip ID")),
		mcp.WithNumber("employee_id", mcp.Description("Employee ID")),
		mcp.WithNumber("truck_id", mcp.Description("Truck ID")),
		mcp.WithString("vendor_name", mcp.Description("Vendor name")),
		mcp.WithString("payable_date", mcp.Description("Payable date (YYYY-MM-DD)")),
		mcp.WithString("amount", mcp.Description("Amount")),
		mcp.WithString("status", mcp.Description("Status")),
		mcp.WithString("description", mcp.Description("Description")),
		mcp.WithString("comments", mcp.Description("Comments")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := argMap(req)
		resp, err := client.ap.CreateAP(ctx, connect.NewRequest(&pb.CreateAPRequest{
			TripId:      argI32Ptr(args, "trip_id"),
			EmployeeId:  argI32Ptr(args, "employee_id"),
			TruckId:     argI32Ptr(args, "truck_id"),
			VendorName:  argStrPtr(args, "vendor_name"),
			PayableDate: argStrPtr(args, "payable_date"),
			Amount:      argStrPtr(args, "amount"),
			Status:      argStrPtr(args, "status"),
			Description: argStrPtr(args, "description"),
			Comments:    argStrPtr(args, "comments"),
		}))
		if err != nil {
			return mcp.NewToolResultError(connectErr(err)), nil
		}
		return mcp.NewToolResultText(fmt.Sprintf("Created AP record #%d", resp.Msg.Item.Id)), nil
	})

	register(mcp.NewTool("update_ap",
		mcp.WithDescription("Update an accounts payable record."),
		mcp.WithNumber("id", mcp.Required(), mcp.Description("AP ID")),
		mcp.WithNumber("trip_id", mcp.Description("Trip ID")),
		mcp.WithNumber("employee_id", mcp.Description("Employee ID")),
		mcp.WithNumber("truck_id", mcp.Description("Truck ID")),
		mcp.WithString("vendor_name", mcp.Description("Vendor name")),
		mcp.WithString("payable_date", mcp.Description("Payable date (YYYY-MM-DD)")),
		mcp.WithString("amount", mcp.Description("Amount")),
		mcp.WithString("paid_amount", mcp.Description("Paid amount")),
		mcp.WithString("status", mcp.Description("Status")),
		mcp.WithString("description", mcp.Description("Description")),
		mcp.WithString("check_number", mcp.Description("Check number")),
		mcp.WithString("check_date", mcp.Description("Check date (YYYY-MM-DD)")),
		mcp.WithString("comments", mcp.Description("Comments")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := argMap(req)
		resp, err := client.ap.UpdateAP(ctx, connect.NewRequest(&pb.UpdateAPRequest{
			Id:          int32(argInt(args, "id", 0)),
			TripId:      argI32Ptr(args, "trip_id"),
			EmployeeId:  argI32Ptr(args, "employee_id"),
			TruckId:     argI32Ptr(args, "truck_id"),
			VendorName:  argStrPtr(args, "vendor_name"),
			PayableDate: argStrPtr(args, "payable_date"),
			Amount:      argStrPtr(args, "amount"),
			PaidAmount:  argStrPtr(args, "paid_amount"),
			Status:      argStrPtr(args, "status"),
			Description: argStrPtr(args, "description"),
			CheckNumber: argStrPtr(args, "check_number"),
			CheckDate:   argStrPtr(args, "check_date"),
			Comments:    argStrPtr(args, "comments"),
		}))
		if err != nil {
			return mcp.NewToolResultError(connectErr(err)), nil
		}
		return mcp.NewToolResultText(fmt.Sprintf("Updated AP record #%d", resp.Msg.Item.Id)), nil
	})

	register(mcp.NewTool("delete_ap",
		mcp.WithDescription("Delete an accounts payable record."),
		mcp.WithNumber("id", mcp.Required(), mcp.Description("AP ID")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := argMap(req)
		_, err := client.ap.DeleteAP(ctx, connect.NewRequest(&pb.DeleteAPRequest{
			Id: int32(argInt(args, "id", 0)),
		}))
		if err != nil {
			return mcp.NewToolResultError(connectErr(err)), nil
		}
		return mcp.NewToolResultText("AP record deleted successfully."), nil
	})
}

func formatAPList(resp *pb.ListAPResponse) string {
	if len(resp.Items) == 0 {
		return "No AP records found."
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Found %d AP records (page %d, %d per page):\n\n",
		resp.Pagination.TotalCount, resp.Pagination.Page, resp.Pagination.PageSize))
	for _, a := range resp.Items {
		sb.WriteString(fmt.Sprintf("  #%d", a.Id))
		if a.VendorName != nil {
			sb.WriteString(fmt.Sprintf(" %s", *a.VendorName))
		}
		if a.Amount != nil {
			sb.WriteString(fmt.Sprintf(" $%s", *a.Amount))
		}
		if a.Status != nil {
			sb.WriteString(fmt.Sprintf(" [%s]", *a.Status))
		}
		if a.PayableDate != nil {
			sb.WriteString(fmt.Sprintf(" %s", *a.PayableDate))
		}
		sb.WriteString("\n")
	}
	return sb.String()
}

func formatAP(a *pb.AccountsPayable) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("AP Record #%d\n", a.Id))
	if a.VendorName != nil {
		sb.WriteString(fmt.Sprintf("  Vendor: %s\n", *a.VendorName))
	}
	if a.PayableDate != nil {
		sb.WriteString(fmt.Sprintf("  Date: %s\n", *a.PayableDate))
	}
	if a.Amount != nil {
		sb.WriteString(fmt.Sprintf("  Amount: $%s\n", *a.Amount))
	}
	if a.PaidAmount != nil {
		sb.WriteString(fmt.Sprintf("  Paid: $%s\n", *a.PaidAmount))
	}
	if a.Status != nil {
		sb.WriteString(fmt.Sprintf("  Status: %s\n", *a.Status))
	}
	if a.Description != nil {
		sb.WriteString(fmt.Sprintf("  Description: %s\n", *a.Description))
	}
	return sb.String()
}
