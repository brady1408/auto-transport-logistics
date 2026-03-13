package main

import (
	"context"
	"fmt"
	"strings"

	"connectrpc.com/connect"
	"github.com/mark3labs/mcp-go/mcp"

	pb "github.com/brady1408/auto-transport-logistics/internal/gen/atlinks/v1"
)

func registerEarningsTools(register toolRegister, client *atlClient) {
	// --- Driver Earnings ---
	register(mcp.NewTool("list_driver_earnings",
		mcp.WithDescription("List driver earnings adjustments."),
		mcp.WithNumber("employee_id", mcp.Description("Filter by employee ID")),
		mcp.WithString("date_from", mcp.Description("Filter from date (YYYY-MM-DD)")),
		mcp.WithString("date_to", mcp.Description("Filter to date (YYYY-MM-DD)")),
		mcp.WithNumber("page", mcp.Description("Page number (default: 1)")),
		mcp.WithNumber("page_size", mcp.Description("Items per page (default: 25)")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := argMap(req)
		resp, err := client.earnings.ListDriverEarnings(ctx, connect.NewRequest(&pb.ListDriverEarningsRequest{
			Pagination: &pb.PaginationRequest{
				Page:     int32(argInt(args, "page", 1)),
				PageSize: int32(argInt(args, "page_size", 25)),
			},
			EmployeeId: argI32Ptr(args, "employee_id"),
			DateFrom:   argStrPtr(args, "date_from"),
			DateTo:     argStrPtr(args, "date_to"),
		}))
		if err != nil {
			return mcp.NewToolResultError(connectErr(err)), nil
		}
		return mcp.NewToolResultText(formatDriverEarningsList(resp.Msg)), nil
	})

	register(mcp.NewTool("get_driver_earnings",
		mcp.WithDescription("Get a driver earnings adjustment by ID."),
		mcp.WithNumber("id", mcp.Required(), mcp.Description("Earnings adjustment ID")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := argMap(req)
		resp, err := client.earnings.GetDriverEarnings(ctx, connect.NewRequest(&pb.GetDriverEarningsRequest{
			Id: int32(argInt(args, "id", 0)),
		}))
		if err != nil {
			return mcp.NewToolResultError(connectErr(err)), nil
		}
		return mcp.NewToolResultText(formatDriverEarnings(resp.Msg.Item)), nil
	})

	register(mcp.NewTool("create_driver_earnings",
		mcp.WithDescription("Create a driver earnings adjustment."),
		mcp.WithNumber("employee_id", mcp.Required(), mcp.Description("Employee ID")),
		mcp.WithString("adj_date", mcp.Description("Adjustment date (YYYY-MM-DD)")),
		mcp.WithString("description", mcp.Description("Description")),
		mcp.WithString("adj_type", mcp.Description("Adjustment type")),
		mcp.WithString("amount", mcp.Description("Amount")),
		mcp.WithString("reference", mcp.Description("Reference")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := argMap(req)
		resp, err := client.earnings.CreateDriverEarnings(ctx, connect.NewRequest(&pb.CreateDriverEarningsRequest{
			EmployeeId:  int32(argInt(args, "employee_id", 0)),
			AdjDate:     argStrPtr(args, "adj_date"),
			Description: argStrPtr(args, "description"),
			AdjType:     argStrPtr(args, "adj_type"),
			Amount:      argStrPtr(args, "amount"),
			Reference:   argStrPtr(args, "reference"),
		}))
		if err != nil {
			return mcp.NewToolResultError(connectErr(err)), nil
		}
		return mcp.NewToolResultText(fmt.Sprintf("Created driver earnings adjustment #%d", resp.Msg.Item.Id)), nil
	})

	register(mcp.NewTool("update_driver_earnings",
		mcp.WithDescription("Update a driver earnings adjustment."),
		mcp.WithNumber("id", mcp.Required(), mcp.Description("Earnings adjustment ID")),
		mcp.WithNumber("employee_id", mcp.Required(), mcp.Description("Employee ID")),
		mcp.WithString("adj_date", mcp.Description("Adjustment date (YYYY-MM-DD)")),
		mcp.WithString("description", mcp.Description("Description")),
		mcp.WithString("adj_type", mcp.Description("Adjustment type")),
		mcp.WithString("amount", mcp.Description("Amount")),
		mcp.WithString("reference", mcp.Description("Reference")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := argMap(req)
		_, err := client.earnings.UpdateDriverEarnings(ctx, connect.NewRequest(&pb.UpdateDriverEarningsRequest{
			Id:          int32(argInt(args, "id", 0)),
			EmployeeId:  int32(argInt(args, "employee_id", 0)),
			AdjDate:     argStrPtr(args, "adj_date"),
			Description: argStrPtr(args, "description"),
			AdjType:     argStrPtr(args, "adj_type"),
			Amount:      argStrPtr(args, "amount"),
			Reference:   argStrPtr(args, "reference"),
		}))
		if err != nil {
			return mcp.NewToolResultError(connectErr(err)), nil
		}
		return mcp.NewToolResultText("Driver earnings adjustment updated."), nil
	})

	register(mcp.NewTool("delete_driver_earnings",
		mcp.WithDescription("Delete a driver earnings adjustment."),
		mcp.WithNumber("id", mcp.Required(), mcp.Description("Earnings adjustment ID")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := argMap(req)
		_, err := client.earnings.DeleteDriverEarnings(ctx, connect.NewRequest(&pb.DeleteDriverEarningsRequest{
			Id: int32(argInt(args, "id", 0)),
		}))
		if err != nil {
			return mcp.NewToolResultError(connectErr(err)), nil
		}
		return mcp.NewToolResultText("Driver earnings adjustment deleted."), nil
	})

	// --- Truck Earnings ---
	register(mcp.NewTool("list_truck_earnings",
		mcp.WithDescription("List truck earnings adjustments."),
		mcp.WithNumber("truck_id", mcp.Description("Filter by truck ID")),
		mcp.WithString("date_from", mcp.Description("Filter from date (YYYY-MM-DD)")),
		mcp.WithString("date_to", mcp.Description("Filter to date (YYYY-MM-DD)")),
		mcp.WithNumber("page", mcp.Description("Page number (default: 1)")),
		mcp.WithNumber("page_size", mcp.Description("Items per page (default: 25)")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := argMap(req)
		resp, err := client.earnings.ListTruckEarnings(ctx, connect.NewRequest(&pb.ListTruckEarningsRequest{
			Pagination: &pb.PaginationRequest{
				Page:     int32(argInt(args, "page", 1)),
				PageSize: int32(argInt(args, "page_size", 25)),
			},
			TruckId:  argI32Ptr(args, "truck_id"),
			DateFrom: argStrPtr(args, "date_from"),
			DateTo:   argStrPtr(args, "date_to"),
		}))
		if err != nil {
			return mcp.NewToolResultError(connectErr(err)), nil
		}
		return mcp.NewToolResultText(formatTruckEarningsList(resp.Msg)), nil
	})

	register(mcp.NewTool("get_truck_earnings",
		mcp.WithDescription("Get a truck earnings adjustment by ID."),
		mcp.WithNumber("id", mcp.Required(), mcp.Description("Earnings adjustment ID")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := argMap(req)
		resp, err := client.earnings.GetTruckEarnings(ctx, connect.NewRequest(&pb.GetTruckEarningsRequest{
			Id: int32(argInt(args, "id", 0)),
		}))
		if err != nil {
			return mcp.NewToolResultError(connectErr(err)), nil
		}
		return mcp.NewToolResultText(formatTruckEarnings(resp.Msg.Item)), nil
	})

	register(mcp.NewTool("create_truck_earnings",
		mcp.WithDescription("Create a truck earnings adjustment."),
		mcp.WithNumber("truck_id", mcp.Required(), mcp.Description("Truck ID")),
		mcp.WithString("adj_date", mcp.Description("Adjustment date (YYYY-MM-DD)")),
		mcp.WithString("description", mcp.Description("Description")),
		mcp.WithString("adj_type", mcp.Description("Adjustment type")),
		mcp.WithString("amount", mcp.Description("Amount")),
		mcp.WithString("reference", mcp.Description("Reference")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := argMap(req)
		resp, err := client.earnings.CreateTruckEarnings(ctx, connect.NewRequest(&pb.CreateTruckEarningsRequest{
			TruckId:     int32(argInt(args, "truck_id", 0)),
			AdjDate:     argStrPtr(args, "adj_date"),
			Description: argStrPtr(args, "description"),
			AdjType:     argStrPtr(args, "adj_type"),
			Amount:      argStrPtr(args, "amount"),
			Reference:   argStrPtr(args, "reference"),
		}))
		if err != nil {
			return mcp.NewToolResultError(connectErr(err)), nil
		}
		return mcp.NewToolResultText(fmt.Sprintf("Created truck earnings adjustment #%d", resp.Msg.Item.Id)), nil
	})

	register(mcp.NewTool("update_truck_earnings",
		mcp.WithDescription("Update a truck earnings adjustment."),
		mcp.WithNumber("id", mcp.Required(), mcp.Description("Earnings adjustment ID")),
		mcp.WithNumber("truck_id", mcp.Required(), mcp.Description("Truck ID")),
		mcp.WithString("adj_date", mcp.Description("Adjustment date (YYYY-MM-DD)")),
		mcp.WithString("description", mcp.Description("Description")),
		mcp.WithString("adj_type", mcp.Description("Adjustment type")),
		mcp.WithString("amount", mcp.Description("Amount")),
		mcp.WithString("reference", mcp.Description("Reference")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := argMap(req)
		_, err := client.earnings.UpdateTruckEarnings(ctx, connect.NewRequest(&pb.UpdateTruckEarningsRequest{
			Id:          int32(argInt(args, "id", 0)),
			TruckId:     int32(argInt(args, "truck_id", 0)),
			AdjDate:     argStrPtr(args, "adj_date"),
			Description: argStrPtr(args, "description"),
			AdjType:     argStrPtr(args, "adj_type"),
			Amount:      argStrPtr(args, "amount"),
			Reference:   argStrPtr(args, "reference"),
		}))
		if err != nil {
			return mcp.NewToolResultError(connectErr(err)), nil
		}
		return mcp.NewToolResultText("Truck earnings adjustment updated."), nil
	})

	register(mcp.NewTool("delete_truck_earnings",
		mcp.WithDescription("Delete a truck earnings adjustment."),
		mcp.WithNumber("id", mcp.Required(), mcp.Description("Earnings adjustment ID")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := argMap(req)
		_, err := client.earnings.DeleteTruckEarnings(ctx, connect.NewRequest(&pb.DeleteTruckEarningsRequest{
			Id: int32(argInt(args, "id", 0)),
		}))
		if err != nil {
			return mcp.NewToolResultError(connectErr(err)), nil
		}
		return mcp.NewToolResultText("Truck earnings adjustment deleted."), nil
	})
}

func formatDriverEarningsList(resp *pb.ListDriverEarningsResponse) string {
	if len(resp.Items) == 0 {
		return "No driver earnings adjustments found."
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Found %d driver earnings adjustments (page %d, %d per page):\n\n",
		resp.Pagination.TotalCount, resp.Pagination.Page, resp.Pagination.PageSize))
	for _, a := range resp.Items {
		sb.WriteString(fmt.Sprintf("  #%d", a.Id))
		if a.EmployeeName != nil {
			sb.WriteString(fmt.Sprintf(" %s", *a.EmployeeName))
		}
		if a.AdjType != nil {
			sb.WriteString(fmt.Sprintf(" [%s]", *a.AdjType))
		}
		if a.Amount != nil {
			sb.WriteString(fmt.Sprintf(" $%s", *a.Amount))
		}
		if a.AdjDate != nil {
			sb.WriteString(fmt.Sprintf(" %s", *a.AdjDate))
		}
		sb.WriteString("\n")
	}
	return sb.String()
}

func formatDriverEarnings(a *pb.DriverEarningsAdj) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Driver Earnings Adj #%d\n", a.Id))
	sb.WriteString(fmt.Sprintf("  Employee ID: %d\n", a.EmployeeId))
	if a.EmployeeName != nil {
		sb.WriteString(fmt.Sprintf("  Name: %s\n", *a.EmployeeName))
	}
	if a.AdjDate != nil {
		sb.WriteString(fmt.Sprintf("  Date: %s\n", *a.AdjDate))
	}
	if a.AdjType != nil {
		sb.WriteString(fmt.Sprintf("  Type: %s\n", *a.AdjType))
	}
	if a.Amount != nil {
		sb.WriteString(fmt.Sprintf("  Amount: $%s\n", *a.Amount))
	}
	if a.Description != nil {
		sb.WriteString(fmt.Sprintf("  Description: %s\n", *a.Description))
	}
	if a.Reference != nil {
		sb.WriteString(fmt.Sprintf("  Reference: %s\n", *a.Reference))
	}
	return sb.String()
}

func formatTruckEarningsList(resp *pb.ListTruckEarningsResponse) string {
	if len(resp.Items) == 0 {
		return "No truck earnings adjustments found."
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Found %d truck earnings adjustments (page %d, %d per page):\n\n",
		resp.Pagination.TotalCount, resp.Pagination.Page, resp.Pagination.PageSize))
	for _, a := range resp.Items {
		sb.WriteString(fmt.Sprintf("  #%d", a.Id))
		if a.TruckNumber != nil {
			sb.WriteString(fmt.Sprintf(" Truck %s", *a.TruckNumber))
		}
		if a.AdjType != nil {
			sb.WriteString(fmt.Sprintf(" [%s]", *a.AdjType))
		}
		if a.Amount != nil {
			sb.WriteString(fmt.Sprintf(" $%s", *a.Amount))
		}
		if a.AdjDate != nil {
			sb.WriteString(fmt.Sprintf(" %s", *a.AdjDate))
		}
		sb.WriteString("\n")
	}
	return sb.String()
}

func formatTruckEarnings(a *pb.TruckEarningsAdj) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Truck Earnings Adj #%d\n", a.Id))
	sb.WriteString(fmt.Sprintf("  Truck ID: %d\n", a.TruckId))
	if a.TruckNumber != nil {
		sb.WriteString(fmt.Sprintf("  Truck #: %s\n", *a.TruckNumber))
	}
	if a.AdjDate != nil {
		sb.WriteString(fmt.Sprintf("  Date: %s\n", *a.AdjDate))
	}
	if a.AdjType != nil {
		sb.WriteString(fmt.Sprintf("  Type: %s\n", *a.AdjType))
	}
	if a.Amount != nil {
		sb.WriteString(fmt.Sprintf("  Amount: $%s\n", *a.Amount))
	}
	if a.Description != nil {
		sb.WriteString(fmt.Sprintf("  Description: %s\n", *a.Description))
	}
	if a.Reference != nil {
		sb.WriteString(fmt.Sprintf("  Reference: %s\n", *a.Reference))
	}
	return sb.String()
}
