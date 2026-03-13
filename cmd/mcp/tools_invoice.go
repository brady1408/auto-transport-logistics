package main

import (
	"context"
	"fmt"
	"strings"

	"connectrpc.com/connect"
	"github.com/mark3labs/mcp-go/mcp"

	pb "github.com/brady1408/auto-transport-logistics/internal/gen/atlinks/v1"
)

func registerInvoiceTools(register toolRegister, client *atlClient) {
	register(mcp.NewTool("list_invoices",
		mcp.WithDescription("List invoices with optional filters. Returns paginated results."),
		mcp.WithString("search", mcp.Description("Search by invoice number or customer")),
		mcp.WithNumber("customer_id", mcp.Description("Filter by customer ID")),
		mcp.WithString("status", mcp.Description("Filter by status")),
		mcp.WithString("date_from", mcp.Description("Filter from date (YYYY-MM-DD)")),
		mcp.WithString("date_to", mcp.Description("Filter to date (YYYY-MM-DD)")),
		mcp.WithNumber("page", mcp.Description("Page number (default: 1)")),
		mcp.WithNumber("page_size", mcp.Description("Items per page (default: 25)")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := argMap(req)
		resp, err := client.invoices.ListInvoices(ctx, connect.NewRequest(&pb.ListInvoicesRequest{
			Pagination: &pb.PaginationRequest{
				Page:     int32(argInt(args, "page", 1)),
				PageSize: int32(argInt(args, "page_size", 25)),
			},
			Search:     argStrPtr(args, "search"),
			CustomerId: argI32Ptr(args, "customer_id"),
			Status:     argStrPtr(args, "status"),
			DateFrom:   argStrPtr(args, "date_from"),
			DateTo:     argStrPtr(args, "date_to"),
		}))
		if err != nil {
			return mcp.NewToolResultError(connectErr(err)), nil
		}
		return mcp.NewToolResultText(formatInvoiceList(resp.Msg)), nil
	})

	register(mcp.NewTool("get_invoice",
		mcp.WithDescription("Get a single invoice by ID."),
		mcp.WithNumber("id", mcp.Required(), mcp.Description("Invoice ID")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := argMap(req)
		resp, err := client.invoices.GetInvoice(ctx, connect.NewRequest(&pb.GetInvoiceRequest{
			Id: int32(argInt(args, "id", 0)),
		}))
		if err != nil {
			return mcp.NewToolResultError(connectErr(err)), nil
		}
		return mcp.NewToolResultText(formatInvoice(resp.Msg.Invoice)), nil
	})

	register(mcp.NewTool("create_invoice",
		mcp.WithDescription("Create a new invoice."),
		mcp.WithNumber("customer_id", mcp.Description("Customer ID")),
		mcp.WithNumber("order_id", mcp.Description("Order ID")),
		mcp.WithString("invoice_date", mcp.Description("Invoice date (YYYY-MM-DD)")),
		mcp.WithString("due_date", mcp.Description("Due date (YYYY-MM-DD)")),
		mcp.WithString("terms", mcp.Description("Payment terms")),
		mcp.WithString("tax_code", mcp.Description("Tax code")),
		mcp.WithString("comments", mcp.Description("Comments")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := argMap(req)
		resp, err := client.invoices.CreateInvoice(ctx, connect.NewRequest(&pb.CreateInvoiceRequest{
			CustomerId:  argI32Ptr(args, "customer_id"),
			OrderId:     argI32Ptr(args, "order_id"),
			InvoiceDate: argStrPtr(args, "invoice_date"),
			DueDate:     argStrPtr(args, "due_date"),
			Terms:       argStrPtr(args, "terms"),
			TaxCode:     argStrPtr(args, "tax_code"),
			Comments:    argStrPtr(args, "comments"),
		}))
		if err != nil {
			return mcp.NewToolResultError(connectErr(err)), nil
		}
		return mcp.NewToolResultText(fmt.Sprintf("Created invoice #%d: %s", resp.Msg.Invoice.Id, resp.Msg.Invoice.InvoiceNumber)), nil
	})

	register(mcp.NewTool("update_invoice",
		mcp.WithDescription("Update an existing invoice."),
		mcp.WithNumber("id", mcp.Required(), mcp.Description("Invoice ID")),
		mcp.WithNumber("customer_id", mcp.Description("Customer ID")),
		mcp.WithString("invoice_date", mcp.Description("Invoice date (YYYY-MM-DD)")),
		mcp.WithString("due_date", mcp.Description("Due date (YYYY-MM-DD)")),
		mcp.WithString("terms", mcp.Description("Payment terms")),
		mcp.WithString("tax_code", mcp.Description("Tax code")),
		mcp.WithString("subtotal", mcp.Description("Subtotal")),
		mcp.WithString("tax", mcp.Description("Tax amount")),
		mcp.WithString("total_amount", mcp.Description("Total amount")),
		mcp.WithString("comments", mcp.Description("Comments")),
		mcp.WithString("status", mcp.Description("Status")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := argMap(req)
		resp, err := client.invoices.UpdateInvoice(ctx, connect.NewRequest(&pb.UpdateInvoiceRequest{
			Id:          int32(argInt(args, "id", 0)),
			CustomerId:  argI32Ptr(args, "customer_id"),
			InvoiceDate: argStrPtr(args, "invoice_date"),
			DueDate:     argStrPtr(args, "due_date"),
			Terms:       argStrPtr(args, "terms"),
			TaxCode:     argStrPtr(args, "tax_code"),
			Subtotal:    argStrPtr(args, "subtotal"),
			Tax:         argStrPtr(args, "tax"),
			TotalAmount: argStrPtr(args, "total_amount"),
			Comments:    argStrPtr(args, "comments"),
			Status:      argStrPtr(args, "status"),
		}))
		if err != nil {
			return mcp.NewToolResultError(connectErr(err)), nil
		}
		return mcp.NewToolResultText(fmt.Sprintf("Updated invoice #%d: %s", resp.Msg.Invoice.Id, resp.Msg.Invoice.InvoiceNumber)), nil
	})

	register(mcp.NewTool("delete_invoice",
		mcp.WithDescription("Delete an invoice by ID."),
		mcp.WithNumber("id", mcp.Required(), mcp.Description("Invoice ID")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := argMap(req)
		_, err := client.invoices.DeleteInvoice(ctx, connect.NewRequest(&pb.DeleteInvoiceRequest{
			Id: int32(argInt(args, "id", 0)),
		}))
		if err != nil {
			return mcp.NewToolResultError(connectErr(err)), nil
		}
		return mcp.NewToolResultText("Invoice deleted successfully."), nil
	})

	register(mcp.NewTool("list_invoice_details",
		mcp.WithDescription("List line items for an invoice."),
		mcp.WithNumber("invoice_id", mcp.Required(), mcp.Description("Invoice ID")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := argMap(req)
		resp, err := client.invoices.ListInvoiceDetails(ctx, connect.NewRequest(&pb.ListInvoiceDetailsRequest{
			InvoiceId: int32(argInt(args, "invoice_id", 0)),
		}))
		if err != nil {
			return mcp.NewToolResultError(connectErr(err)), nil
		}
		if len(resp.Msg.Details) == 0 {
			return mcp.NewToolResultText("No invoice details found."), nil
		}
		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("Invoice details (%d items):\n\n", len(resp.Msg.Details)))
		for _, d := range resp.Msg.Details {
			sb.WriteString(fmt.Sprintf("  #%d VIN: %s %s %s %s — $%s\n",
				d.Id, deref(d.Vin), deref(d.Year), deref(d.Make), deref(d.Model), deref(d.Amount)))
		}
		return mcp.NewToolResultText(sb.String()), nil
	})

	register(mcp.NewTool("generate_invoice",
		mcp.WithDescription("Generate an invoice from an order."),
		mcp.WithNumber("order_id", mcp.Required(), mcp.Description("Order ID")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := argMap(req)
		resp, err := client.invoices.GenerateInvoice(ctx, connect.NewRequest(&pb.GenerateInvoiceRequest{
			OrderId: int32(argInt(args, "order_id", 0)),
		}))
		if err != nil {
			return mcp.NewToolResultError(connectErr(err)), nil
		}
		return mcp.NewToolResultText(fmt.Sprintf("Generated invoice #%d: %s", resp.Msg.Invoice.Id, resp.Msg.Invoice.InvoiceNumber)), nil
	})

	register(mcp.NewTool("void_invoice",
		mcp.WithDescription("Void an invoice."),
		mcp.WithNumber("invoice_id", mcp.Required(), mcp.Description("Invoice ID")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := argMap(req)
		_, err := client.invoices.VoidInvoice(ctx, connect.NewRequest(&pb.VoidInvoiceRequest{
			InvoiceId: int32(argInt(args, "invoice_id", 0)),
		}))
		if err != nil {
			return mcp.NewToolResultError(connectErr(err)), nil
		}
		return mcp.NewToolResultText("Invoice voided successfully."), nil
	})

	register(mcp.NewTool("post_invoices",
		mcp.WithDescription("Post invoices within a date range."),
		mcp.WithString("date_from", mcp.Required(), mcp.Description("Start date (YYYY-MM-DD)")),
		mcp.WithString("date_to", mcp.Required(), mcp.Description("End date (YYYY-MM-DD)")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := argMap(req)
		resp, err := client.invoices.PostInvoices(ctx, connect.NewRequest(&pb.PostInvoicesRequest{
			DateFrom: argStr(args, "date_from"),
			DateTo:   argStr(args, "date_to"),
		}))
		if err != nil {
			return mcp.NewToolResultError(connectErr(err)), nil
		}
		return mcp.NewToolResultText(fmt.Sprintf("Posted %d invoices.", resp.Msg.Count)), nil
	})
}

func formatInvoiceList(resp *pb.ListInvoicesResponse) string {
	if len(resp.Invoices) == 0 {
		return "No invoices found."
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Found %d invoices (page %d, %d per page):\n\n",
		resp.Pagination.TotalCount, resp.Pagination.Page, resp.Pagination.PageSize))
	for _, inv := range resp.Invoices {
		sb.WriteString(fmt.Sprintf("  #%d %s", inv.Id, inv.InvoiceNumber))
		if inv.CustomerName != nil {
			sb.WriteString(fmt.Sprintf(" — %s", *inv.CustomerName))
		}
		if inv.TotalAmount != nil {
			sb.WriteString(fmt.Sprintf(" $%s", *inv.TotalAmount))
		}
		if inv.Balance != nil {
			sb.WriteString(fmt.Sprintf(" (bal: $%s)", *inv.Balance))
		}
		if inv.Status != nil {
			sb.WriteString(fmt.Sprintf(" [%s]", *inv.Status))
		}
		sb.WriteString("\n")
	}
	return sb.String()
}

func formatInvoice(inv *pb.Invoice) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Invoice #%d: %s\n", inv.Id, inv.InvoiceNumber))
	if inv.CustomerName != nil {
		sb.WriteString(fmt.Sprintf("  Customer: %s\n", *inv.CustomerName))
	}
	if inv.InvoiceDate != nil {
		sb.WriteString(fmt.Sprintf("  Date: %s\n", *inv.InvoiceDate))
	}
	if inv.DueDate != nil {
		sb.WriteString(fmt.Sprintf("  Due: %s\n", *inv.DueDate))
	}
	if inv.TotalAmount != nil {
		sb.WriteString(fmt.Sprintf("  Total: $%s\n", *inv.TotalAmount))
	}
	if inv.AmountPaid != nil {
		sb.WriteString(fmt.Sprintf("  Paid: $%s\n", *inv.AmountPaid))
	}
	if inv.Balance != nil {
		sb.WriteString(fmt.Sprintf("  Balance: $%s\n", *inv.Balance))
	}
	if inv.Status != nil {
		sb.WriteString(fmt.Sprintf("  Status: %s\n", *inv.Status))
	}
	sb.WriteString(fmt.Sprintf("  Active: %v\n", inv.Active))
	return sb.String()
}
