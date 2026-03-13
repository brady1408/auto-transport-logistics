package main

import (
	"context"
	"fmt"
	"strings"

	"connectrpc.com/connect"
	"github.com/mark3labs/mcp-go/mcp"

	pb "github.com/brady1408/auto-transport-logistics/internal/gen/atlinks/v1"
)

func registerPaymentTools(register toolRegister, client *atlClient) {
	register(mcp.NewTool("list_payments",
		mcp.WithDescription("List payments with optional filters. Returns paginated results."),
		mcp.WithString("search", mcp.Description("Search by check number or customer")),
		mcp.WithNumber("customer_id", mcp.Description("Filter by customer ID")),
		mcp.WithString("date_from", mcp.Description("Filter from date (YYYY-MM-DD)")),
		mcp.WithString("date_to", mcp.Description("Filter to date (YYYY-MM-DD)")),
		mcp.WithNumber("page", mcp.Description("Page number (default: 1)")),
		mcp.WithNumber("page_size", mcp.Description("Items per page (default: 25)")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := argMap(req)
		resp, err := client.payments.ListPayments(ctx, connect.NewRequest(&pb.ListPaymentsRequest{
			Pagination: &pb.PaginationRequest{
				Page:     int32(argInt(args, "page", 1)),
				PageSize: int32(argInt(args, "page_size", 25)),
			},
			Search:     argStrPtr(args, "search"),
			CustomerId: argI32Ptr(args, "customer_id"),
			DateFrom:   argStrPtr(args, "date_from"),
			DateTo:     argStrPtr(args, "date_to"),
		}))
		if err != nil {
			return mcp.NewToolResultError(connectErr(err)), nil
		}
		return mcp.NewToolResultText(formatPaymentList(resp.Msg)), nil
	})

	register(mcp.NewTool("get_payment",
		mcp.WithDescription("Get a single payment by ID."),
		mcp.WithNumber("id", mcp.Required(), mcp.Description("Payment ID")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := argMap(req)
		resp, err := client.payments.GetPayment(ctx, connect.NewRequest(&pb.GetPaymentRequest{
			Id: int32(argInt(args, "id", 0)),
		}))
		if err != nil {
			return mcp.NewToolResultError(connectErr(err)), nil
		}
		return mcp.NewToolResultText(formatPayment(resp.Msg.Payment)), nil
	})

	register(mcp.NewTool("create_payment",
		mcp.WithDescription("Create a new payment."),
		mcp.WithNumber("customer_id", mcp.Description("Customer ID")),
		mcp.WithString("payment_date", mcp.Description("Payment date (YYYY-MM-DD)")),
		mcp.WithString("check_number", mcp.Description("Check number")),
		mcp.WithString("amount", mcp.Description("Amount")),
		mcp.WithString("payment_method", mcp.Description("Payment method")),
		mcp.WithString("comments", mcp.Description("Comments")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := argMap(req)
		resp, err := client.payments.CreatePayment(ctx, connect.NewRequest(&pb.CreatePaymentRequest{
			CustomerId:    argI32Ptr(args, "customer_id"),
			PaymentDate:   argStrPtr(args, "payment_date"),
			CheckNumber:   argStrPtr(args, "check_number"),
			Amount:        argStrPtr(args, "amount"),
			PaymentMethod: argStrPtr(args, "payment_method"),
			Comments:      argStrPtr(args, "comments"),
		}))
		if err != nil {
			return mcp.NewToolResultError(connectErr(err)), nil
		}
		return mcp.NewToolResultText(fmt.Sprintf("Created payment #%d", resp.Msg.Payment.Id)), nil
	})

	register(mcp.NewTool("update_payment",
		mcp.WithDescription("Update an existing payment."),
		mcp.WithNumber("id", mcp.Required(), mcp.Description("Payment ID")),
		mcp.WithNumber("customer_id", mcp.Description("Customer ID")),
		mcp.WithString("payment_date", mcp.Description("Payment date (YYYY-MM-DD)")),
		mcp.WithString("check_number", mcp.Description("Check number")),
		mcp.WithString("amount", mcp.Description("Amount")),
		mcp.WithString("payment_method", mcp.Description("Payment method")),
		mcp.WithString("comments", mcp.Description("Comments")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := argMap(req)
		resp, err := client.payments.UpdatePayment(ctx, connect.NewRequest(&pb.UpdatePaymentRequest{
			Id:            int32(argInt(args, "id", 0)),
			CustomerId:    argI32Ptr(args, "customer_id"),
			PaymentDate:   argStrPtr(args, "payment_date"),
			CheckNumber:   argStrPtr(args, "check_number"),
			Amount:        argStrPtr(args, "amount"),
			PaymentMethod: argStrPtr(args, "payment_method"),
			Comments:      argStrPtr(args, "comments"),
		}))
		if err != nil {
			return mcp.NewToolResultError(connectErr(err)), nil
		}
		return mcp.NewToolResultText(fmt.Sprintf("Updated payment #%d", resp.Msg.Payment.Id)), nil
	})

	register(mcp.NewTool("delete_payment",
		mcp.WithDescription("Delete a payment by ID."),
		mcp.WithNumber("id", mcp.Required(), mcp.Description("Payment ID")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := argMap(req)
		_, err := client.payments.DeletePayment(ctx, connect.NewRequest(&pb.DeletePaymentRequest{
			Id: int32(argInt(args, "id", 0)),
		}))
		if err != nil {
			return mcp.NewToolResultError(connectErr(err)), nil
		}
		return mcp.NewToolResultText("Payment deleted successfully."), nil
	})

	register(mcp.NewTool("list_payment_details",
		mcp.WithDescription("List payment application details."),
		mcp.WithNumber("payment_id", mcp.Required(), mcp.Description("Payment ID")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := argMap(req)
		resp, err := client.payments.ListPaymentDetails(ctx, connect.NewRequest(&pb.ListPaymentDetailsRequest{
			PaymentId: int32(argInt(args, "payment_id", 0)),
		}))
		if err != nil {
			return mcp.NewToolResultError(connectErr(err)), nil
		}
		if len(resp.Msg.Details) == 0 {
			return mcp.NewToolResultText("No payment details found."), nil
		}
		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("Payment details (%d):\n\n", len(resp.Msg.Details)))
		for _, d := range resp.Msg.Details {
			sb.WriteString(fmt.Sprintf("  #%d Invoice: %s Amount: $%s",
				d.Id, deref(d.InvoiceNumber), deref(d.Amount)))
			if d.DiscountAmount != nil {
				sb.WriteString(fmt.Sprintf(" Discount: $%s", *d.DiscountAmount))
			}
			sb.WriteString("\n")
		}
		return mcp.NewToolResultText(sb.String()), nil
	})

	register(mcp.NewTool("apply_payment",
		mcp.WithDescription("Apply a payment to an invoice."),
		mcp.WithNumber("payment_id", mcp.Required(), mcp.Description("Payment ID")),
		mcp.WithNumber("invoice_id", mcp.Required(), mcp.Description("Invoice ID")),
		mcp.WithString("amount", mcp.Required(), mcp.Description("Amount to apply")),
		mcp.WithString("discount", mcp.Description("Discount amount")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := argMap(req)
		_, err := client.payments.ApplyPayment(ctx, connect.NewRequest(&pb.ApplyPaymentRequest{
			PaymentId: int32(argInt(args, "payment_id", 0)),
			InvoiceId: int32(argInt(args, "invoice_id", 0)),
			Amount:    argStr(args, "amount"),
			Discount:  argStrPtr(args, "discount"),
		}))
		if err != nil {
			return mcp.NewToolResultError(connectErr(err)), nil
		}
		return mcp.NewToolResultText("Payment applied successfully."), nil
	})

	register(mcp.NewTool("unapply_payment",
		mcp.WithDescription("Unapply a payment detail."),
		mcp.WithNumber("payment_detail_id", mcp.Required(), mcp.Description("Payment detail ID")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := argMap(req)
		_, err := client.payments.UnapplyPayment(ctx, connect.NewRequest(&pb.UnapplyPaymentRequest{
			PaymentDetailId: int32(argInt(args, "payment_detail_id", 0)),
		}))
		if err != nil {
			return mcp.NewToolResultError(connectErr(err)), nil
		}
		return mcp.NewToolResultText("Payment unapplied successfully."), nil
	})

	register(mcp.NewTool("post_payments",
		mcp.WithDescription("Post payments within a date range."),
		mcp.WithString("date_from", mcp.Required(), mcp.Description("Start date (YYYY-MM-DD)")),
		mcp.WithString("date_to", mcp.Required(), mcp.Description("End date (YYYY-MM-DD)")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := argMap(req)
		resp, err := client.payments.PostPayments(ctx, connect.NewRequest(&pb.PostPaymentsRequest{
			DateFrom: argStr(args, "date_from"),
			DateTo:   argStr(args, "date_to"),
		}))
		if err != nil {
			return mcp.NewToolResultError(connectErr(err)), nil
		}
		return mcp.NewToolResultText(fmt.Sprintf("Posted %d payments.", resp.Msg.Count)), nil
	})
}

func formatPaymentList(resp *pb.ListPaymentsResponse) string {
	if len(resp.Payments) == 0 {
		return "No payments found."
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Found %d payments (page %d, %d per page):\n\n",
		resp.Pagination.TotalCount, resp.Pagination.Page, resp.Pagination.PageSize))
	for _, p := range resp.Payments {
		sb.WriteString(fmt.Sprintf("  #%d", p.Id))
		if p.CustomerName != nil {
			sb.WriteString(fmt.Sprintf(" %s", *p.CustomerName))
		}
		if p.Amount != nil {
			sb.WriteString(fmt.Sprintf(" $%s", *p.Amount))
		}
		if p.UnappliedAmount != nil {
			sb.WriteString(fmt.Sprintf(" (unapplied: $%s)", *p.UnappliedAmount))
		}
		if p.PaymentDate != nil {
			sb.WriteString(fmt.Sprintf(" %s", *p.PaymentDate))
		}
		sb.WriteString("\n")
	}
	return sb.String()
}

func formatPayment(p *pb.Payment) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Payment #%d\n", p.Id))
	if p.CustomerName != nil {
		sb.WriteString(fmt.Sprintf("  Customer: %s\n", *p.CustomerName))
	}
	if p.PaymentDate != nil {
		sb.WriteString(fmt.Sprintf("  Date: %s\n", *p.PaymentDate))
	}
	if p.CheckNumber != nil {
		sb.WriteString(fmt.Sprintf("  Check #: %s\n", *p.CheckNumber))
	}
	if p.Amount != nil {
		sb.WriteString(fmt.Sprintf("  Amount: $%s\n", *p.Amount))
	}
	if p.AppliedAmount != nil {
		sb.WriteString(fmt.Sprintf("  Applied: $%s\n", *p.AppliedAmount))
	}
	if p.UnappliedAmount != nil {
		sb.WriteString(fmt.Sprintf("  Unapplied: $%s\n", *p.UnappliedAmount))
	}
	if p.PaymentMethod != nil {
		sb.WriteString(fmt.Sprintf("  Method: %s\n", *p.PaymentMethod))
	}
	return sb.String()
}
