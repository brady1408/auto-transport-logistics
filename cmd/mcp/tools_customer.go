package main

import (
	"context"
	"fmt"
	"strings"

	"connectrpc.com/connect"
	"github.com/mark3labs/mcp-go/mcp"

	pb "github.com/brady1408/atlinks/internal/gen/atlinks/v1"
)

func registerCustomerTools(register toolRegister, client *atlClient) {
	register(mcp.NewTool("list_customers",
		mcp.WithDescription("List customers with optional filters. Returns paginated results."),
		mcp.WithString("search", mcp.Description("Search by name or number")),
		mcp.WithString("type", mcp.Description("Filter by customer type")),
		mcp.WithString("zone", mcp.Description("Filter by zone")),
		mcp.WithString("active", mcp.Description("Filter: 'active', 'inactive', or 'all' (default: all)")),
		mcp.WithNumber("page", mcp.Description("Page number (default: 1)")),
		mcp.WithNumber("page_size", mcp.Description("Items per page (default: 25)")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := argMap(req)
		resp, err := client.customers.ListCustomers(ctx, connect.NewRequest(&pb.ListCustomersRequest{
			Pagination: &pb.PaginationRequest{
				Page:     int32(argInt(args, "page", 1)),
				PageSize: int32(argInt(args, "page_size", 25)),
			},
			Search: argStrPtr(args, "search"),
			Type:   argStrPtr(args, "type"),
			Zone:   argStrPtr(args, "zone"),
			Active: argStrPtr(args, "active"),
		}))
		if err != nil {
			return mcp.NewToolResultError(connectErr(err)), nil
		}
		return mcp.NewToolResultText(formatCustomerList(resp.Msg)), nil
	})

	register(mcp.NewTool("get_customer",
		mcp.WithDescription("Get a single customer by ID."),
		mcp.WithNumber("id", mcp.Required(), mcp.Description("Customer ID")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := argMap(req)
		resp, err := client.customers.GetCustomer(ctx, connect.NewRequest(&pb.GetCustomerRequest{
			Id: int32(argInt(args, "id", 0)),
		}))
		if err != nil {
			return mcp.NewToolResultError(connectErr(err)), nil
		}
		return mcp.NewToolResultText(formatCustomer(resp.Msg.Customer)), nil
	})

	register(mcp.NewTool("create_customer",
		mcp.WithDescription("Create a new customer. Name is required."),
		mcp.WithString("name", mcp.Required(), mcp.Description("Customer name")),
		mcp.WithString("number", mcp.Description("Customer number/code")),
		mcp.WithString("address", mcp.Description("Street address")),
		mcp.WithString("city", mcp.Description("City")),
		mcp.WithString("state", mcp.Description("State")),
		mcp.WithString("zip", mcp.Description("ZIP code")),
		mcp.WithString("phone", mcp.Description("Phone number")),
		mcp.WithString("contact", mcp.Description("Contact person")),
		mcp.WithString("type", mcp.Description("Customer type")),
		mcp.WithString("zone", mcp.Description("Zone")),
		mcp.WithBoolean("cod", mcp.Description("Cash on delivery")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := argMap(req)
		resp, err := client.customers.CreateCustomer(ctx, connect.NewRequest(&pb.CreateCustomerRequest{
			Name:    argStr(args, "name"),
			Number:  argStrPtr(args, "number"),
			Address: argStrPtr(args, "address"),
			City:    argStrPtr(args, "city"),
			State:   argStrPtr(args, "state"),
			Zip:     argStrPtr(args, "zip"),
			Phone:   argStrPtr(args, "phone"),
			Contact: argStrPtr(args, "contact"),
			Type:    argStrPtr(args, "type"),
			Zone:    argStrPtr(args, "zone"),
			Cod:     argBool(args, "cod"),
		}))
		if err != nil {
			return mcp.NewToolResultError(connectErr(err)), nil
		}
		return mcp.NewToolResultText(fmt.Sprintf("Created customer #%d: %s", resp.Msg.Customer.Id, resp.Msg.Customer.Name)), nil
	})

	register(mcp.NewTool("update_customer",
		mcp.WithDescription("Update an existing customer. ID and name are required."),
		mcp.WithNumber("id", mcp.Required(), mcp.Description("Customer ID")),
		mcp.WithString("name", mcp.Required(), mcp.Description("Customer name")),
		mcp.WithString("number", mcp.Description("Customer number/code")),
		mcp.WithString("address", mcp.Description("Street address")),
		mcp.WithString("city", mcp.Description("City")),
		mcp.WithString("state", mcp.Description("State")),
		mcp.WithString("zip", mcp.Description("ZIP code")),
		mcp.WithString("phone", mcp.Description("Phone number")),
		mcp.WithString("contact", mcp.Description("Contact person")),
		mcp.WithString("type", mcp.Description("Customer type")),
		mcp.WithString("zone", mcp.Description("Zone")),
		mcp.WithBoolean("cod", mcp.Description("Cash on delivery")),
		mcp.WithBoolean("inactive", mcp.Description("Mark as inactive")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := argMap(req)
		resp, err := client.customers.UpdateCustomer(ctx, connect.NewRequest(&pb.UpdateCustomerRequest{
			Id:       int32(argInt(args, "id", 0)),
			Name:     argStr(args, "name"),
			Number:   argStrPtr(args, "number"),
			Address:  argStrPtr(args, "address"),
			City:     argStrPtr(args, "city"),
			State:    argStrPtr(args, "state"),
			Zip:      argStrPtr(args, "zip"),
			Phone:    argStrPtr(args, "phone"),
			Contact:  argStrPtr(args, "contact"),
			Type:     argStrPtr(args, "type"),
			Zone:     argStrPtr(args, "zone"),
			Cod:      argBool(args, "cod"),
			Inactive: argBool(args, "inactive"),
		}))
		if err != nil {
			return mcp.NewToolResultError(connectErr(err)), nil
		}
		return mcp.NewToolResultText(fmt.Sprintf("Updated customer #%d: %s", resp.Msg.Customer.Id, resp.Msg.Customer.Name)), nil
	})

	register(mcp.NewTool("delete_customer",
		mcp.WithDescription("Delete (soft-delete) a customer by ID."),
		mcp.WithNumber("id", mcp.Required(), mcp.Description("Customer ID")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := argMap(req)
		_, err := client.customers.DeleteCustomer(ctx, connect.NewRequest(&pb.DeleteCustomerRequest{
			Id: int32(argInt(args, "id", 0)),
		}))
		if err != nil {
			return mcp.NewToolResultError(connectErr(err)), nil
		}
		return mcp.NewToolResultText("Customer deleted successfully."), nil
	})
}

func formatCustomerList(resp *pb.ListCustomersResponse) string {
	if len(resp.Customers) == 0 {
		return "No customers found."
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Found %d customers (page %d, %d per page):\n\n",
		resp.Pagination.TotalCount, resp.Pagination.Page, resp.Pagination.PageSize))
	for _, c := range resp.Customers {
		sb.WriteString(fmt.Sprintf("  #%d %s", c.Id, c.Name))
		if c.Number != nil {
			sb.WriteString(fmt.Sprintf(" (%s)", *c.Number))
		}
		if c.City != nil || c.State != nil {
			sb.WriteString(fmt.Sprintf(" — %s, %s", deref(c.City), deref(c.State)))
		}
		if c.Inactive {
			sb.WriteString(" [INACTIVE]")
		}
		sb.WriteString("\n")
	}
	return sb.String()
}

func formatCustomer(c *pb.Customer) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Customer #%d: %s\n", c.Id, c.Name))
	if c.Number != nil {
		sb.WriteString(fmt.Sprintf("  Number: %s\n", *c.Number))
	}
	if c.Address != nil {
		sb.WriteString(fmt.Sprintf("  Address: %s\n", *c.Address))
	}
	if c.City != nil || c.State != nil || c.Zip != nil {
		sb.WriteString(fmt.Sprintf("  City/State/Zip: %s, %s %s\n", deref(c.City), deref(c.State), deref(c.Zip)))
	}
	if c.Phone != nil {
		sb.WriteString(fmt.Sprintf("  Phone: %s\n", *c.Phone))
	}
	if c.Contact != nil {
		sb.WriteString(fmt.Sprintf("  Contact: %s\n", *c.Contact))
	}
	if c.Type != nil {
		sb.WriteString(fmt.Sprintf("  Type: %s\n", *c.Type))
	}
	if c.Zone != nil {
		sb.WriteString(fmt.Sprintf("  Zone: %s\n", *c.Zone))
	}
	sb.WriteString(fmt.Sprintf("  COD: %v | Inactive: %v\n", c.Cod, c.Inactive))
	return sb.String()
}
