package main

import (
	"context"
	"fmt"
	"strings"

	"connectrpc.com/connect"
	"github.com/mark3labs/mcp-go/mcp"

	pb "github.com/brady1408/auto-transport-logistics/internal/gen/atlinks/v1"
)

func registerCreditMemoTools(register toolRegister, client *atlClient) {
	register(mcp.NewTool("list_credit_memos",
		mcp.WithDescription("List credit memos with optional filters. Returns paginated results."),
		mcp.WithString("search", mcp.Description("Search by credit number or customer")),
		mcp.WithNumber("customer_id", mcp.Description("Filter by customer ID")),
		mcp.WithString("status", mcp.Description("Filter by status")),
		mcp.WithNumber("page", mcp.Description("Page number (default: 1)")),
		mcp.WithNumber("page_size", mcp.Description("Items per page (default: 25)")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := argMap(req)
		resp, err := client.creditMemos.ListCreditMemos(ctx, connect.NewRequest(&pb.ListCreditMemosRequest{
			Pagination: &pb.PaginationRequest{
				Page:     int32(argInt(args, "page", 1)),
				PageSize: int32(argInt(args, "page_size", 25)),
			},
			Search:     argStrPtr(args, "search"),
			CustomerId: argI32Ptr(args, "customer_id"),
			Status:     argStrPtr(args, "status"),
		}))
		if err != nil {
			return mcp.NewToolResultError(connectErr(err)), nil
		}
		return mcp.NewToolResultText(formatCreditMemoList(resp.Msg)), nil
	})

	register(mcp.NewTool("get_credit_memo",
		mcp.WithDescription("Get a single credit memo by ID."),
		mcp.WithNumber("id", mcp.Required(), mcp.Description("Credit memo ID")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := argMap(req)
		resp, err := client.creditMemos.GetCreditMemo(ctx, connect.NewRequest(&pb.GetCreditMemoRequest{
			Id: int32(argInt(args, "id", 0)),
		}))
		if err != nil {
			return mcp.NewToolResultError(connectErr(err)), nil
		}
		return mcp.NewToolResultText(formatCreditMemo(resp.Msg.CreditMemo)), nil
	})

	register(mcp.NewTool("create_credit_memo",
		mcp.WithDescription("Create a new credit memo."),
		mcp.WithNumber("customer_id", mcp.Description("Customer ID")),
		mcp.WithNumber("invoice_id", mcp.Description("Invoice ID")),
		mcp.WithString("credit_date", mcp.Description("Credit date (YYYY-MM-DD)")),
		mcp.WithString("amount", mcp.Description("Amount")),
		mcp.WithString("reason", mcp.Description("Reason")),
		mcp.WithString("status", mcp.Description("Status")),
		mcp.WithString("comments", mcp.Description("Comments")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := argMap(req)
		resp, err := client.creditMemos.CreateCreditMemo(ctx, connect.NewRequest(&pb.CreateCreditMemoRequest{
			CustomerId: argI32Ptr(args, "customer_id"),
			InvoiceId:  argI32Ptr(args, "invoice_id"),
			CreditDate: argStrPtr(args, "credit_date"),
			Amount:     argStrPtr(args, "amount"),
			Reason:     argStrPtr(args, "reason"),
			Status:     argStrPtr(args, "status"),
			Comments:   argStrPtr(args, "comments"),
		}))
		if err != nil {
			return mcp.NewToolResultError(connectErr(err)), nil
		}
		return mcp.NewToolResultText(fmt.Sprintf("Created credit memo #%d: %s", resp.Msg.CreditMemo.Id, resp.Msg.CreditMemo.CreditNumber)), nil
	})

	register(mcp.NewTool("update_credit_memo",
		mcp.WithDescription("Update an existing credit memo."),
		mcp.WithNumber("id", mcp.Required(), mcp.Description("Credit memo ID")),
		mcp.WithNumber("customer_id", mcp.Description("Customer ID")),
		mcp.WithNumber("invoice_id", mcp.Description("Invoice ID")),
		mcp.WithString("credit_date", mcp.Description("Credit date (YYYY-MM-DD)")),
		mcp.WithString("amount", mcp.Description("Amount")),
		mcp.WithString("reason", mcp.Description("Reason")),
		mcp.WithString("status", mcp.Description("Status")),
		mcp.WithString("comments", mcp.Description("Comments")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := argMap(req)
		resp, err := client.creditMemos.UpdateCreditMemo(ctx, connect.NewRequest(&pb.UpdateCreditMemoRequest{
			Id:         int32(argInt(args, "id", 0)),
			CustomerId: argI32Ptr(args, "customer_id"),
			InvoiceId:  argI32Ptr(args, "invoice_id"),
			CreditDate: argStrPtr(args, "credit_date"),
			Amount:     argStrPtr(args, "amount"),
			Reason:     argStrPtr(args, "reason"),
			Status:     argStrPtr(args, "status"),
			Comments:   argStrPtr(args, "comments"),
		}))
		if err != nil {
			return mcp.NewToolResultError(connectErr(err)), nil
		}
		return mcp.NewToolResultText(fmt.Sprintf("Updated credit memo #%d: %s", resp.Msg.CreditMemo.Id, resp.Msg.CreditMemo.CreditNumber)), nil
	})

	register(mcp.NewTool("delete_credit_memo",
		mcp.WithDescription("Delete a credit memo by ID."),
		mcp.WithNumber("id", mcp.Required(), mcp.Description("Credit memo ID")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := argMap(req)
		_, err := client.creditMemos.DeleteCreditMemo(ctx, connect.NewRequest(&pb.DeleteCreditMemoRequest{
			Id: int32(argInt(args, "id", 0)),
		}))
		if err != nil {
			return mcp.NewToolResultError(connectErr(err)), nil
		}
		return mcp.NewToolResultText("Credit memo deleted successfully."), nil
	})
}

func formatCreditMemoList(resp *pb.ListCreditMemosResponse) string {
	if len(resp.CreditMemos) == 0 {
		return "No credit memos found."
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Found %d credit memos (page %d, %d per page):\n\n",
		resp.Pagination.TotalCount, resp.Pagination.Page, resp.Pagination.PageSize))
	for _, c := range resp.CreditMemos {
		sb.WriteString(fmt.Sprintf("  #%d %s", c.Id, c.CreditNumber))
		if c.CustomerName != nil {
			sb.WriteString(fmt.Sprintf(" — %s", *c.CustomerName))
		}
		if c.Amount != nil {
			sb.WriteString(fmt.Sprintf(" $%s", *c.Amount))
		}
		if c.Status != nil {
			sb.WriteString(fmt.Sprintf(" [%s]", *c.Status))
		}
		sb.WriteString("\n")
	}
	return sb.String()
}

func formatCreditMemo(c *pb.CreditMemo) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Credit Memo #%d: %s\n", c.Id, c.CreditNumber))
	if c.CustomerName != nil {
		sb.WriteString(fmt.Sprintf("  Customer: %s\n", *c.CustomerName))
	}
	if c.CreditDate != nil {
		sb.WriteString(fmt.Sprintf("  Date: %s\n", *c.CreditDate))
	}
	if c.Amount != nil {
		sb.WriteString(fmt.Sprintf("  Amount: $%s\n", *c.Amount))
	}
	if c.Reason != nil {
		sb.WriteString(fmt.Sprintf("  Reason: %s\n", *c.Reason))
	}
	if c.Status != nil {
		sb.WriteString(fmt.Sprintf("  Status: %s\n", *c.Status))
	}
	if c.Comments != nil {
		sb.WriteString(fmt.Sprintf("  Comments: %s\n", *c.Comments))
	}
	return sb.String()
}
