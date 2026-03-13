package main

import (
	"context"
	"fmt"
	"strings"

	"connectrpc.com/connect"
	"github.com/mark3labs/mcp-go/mcp"

	pb "github.com/brady1408/auto-transport-logistics/internal/gen/atlinks/v1"
)

func registerVendorTools(register toolRegister, client *atlClient) {
	register(mcp.NewTool("list_vendors",
		mcp.WithDescription("List vendors with optional search. Returns paginated results."),
		mcp.WithString("search", mcp.Description("Search by name")),
		mcp.WithNumber("page", mcp.Description("Page number (default: 1)")),
		mcp.WithNumber("page_size", mcp.Description("Items per page (default: 25)")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := argMap(req)
		resp, err := client.vendors.ListVendors(ctx, connect.NewRequest(&pb.ListVendorsRequest{
			Pagination: &pb.PaginationRequest{
				Page:     int32(argInt(args, "page", 1)),
				PageSize: int32(argInt(args, "page_size", 25)),
			},
			Search: argStrPtr(args, "search"),
		}))
		if err != nil {
			return mcp.NewToolResultError(connectErr(err)), nil
		}
		return mcp.NewToolResultText(formatVendorList(resp.Msg)), nil
	})

	register(mcp.NewTool("get_vendor",
		mcp.WithDescription("Get a single vendor by ID."),
		mcp.WithNumber("id", mcp.Required(), mcp.Description("Vendor ID")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := argMap(req)
		resp, err := client.vendors.GetVendor(ctx, connect.NewRequest(&pb.GetVendorRequest{
			Id: int32(argInt(args, "id", 0)),
		}))
		if err != nil {
			return mcp.NewToolResultError(connectErr(err)), nil
		}
		return mcp.NewToolResultText(formatVendor(resp.Msg.Vendor)), nil
	})

	register(mcp.NewTool("create_vendor",
		mcp.WithDescription("Create a new vendor. Name is required."),
		mcp.WithString("name", mcp.Required(), mcp.Description("Vendor name")),
		mcp.WithString("address", mcp.Description("Street address")),
		mcp.WithString("city", mcp.Description("City")),
		mcp.WithString("state", mcp.Description("State")),
		mcp.WithString("zip", mcp.Description("ZIP code")),
		mcp.WithString("phone", mcp.Description("Phone number")),
		mcp.WithString("fax", mcp.Description("Fax number")),
		mcp.WithString("contact", mcp.Description("Contact person")),
		mcp.WithString("terms", mcp.Description("Payment terms")),
		mcp.WithString("tax_id", mcp.Description("Tax ID")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := argMap(req)
		resp, err := client.vendors.CreateVendor(ctx, connect.NewRequest(&pb.CreateVendorRequest{
			Name:    argStr(args, "name"),
			Address: argStrPtr(args, "address"),
			City:    argStrPtr(args, "city"),
			State:   argStrPtr(args, "state"),
			Zip:     argStrPtr(args, "zip"),
			Phone:   argStrPtr(args, "phone"),
			Fax:     argStrPtr(args, "fax"),
			Contact: argStrPtr(args, "contact"),
			Terms:   argStrPtr(args, "terms"),
			TaxId:   argStrPtr(args, "tax_id"),
		}))
		if err != nil {
			return mcp.NewToolResultError(connectErr(err)), nil
		}
		return mcp.NewToolResultText(fmt.Sprintf("Created vendor #%d: %s", resp.Msg.Vendor.Id, resp.Msg.Vendor.Name)), nil
	})

	register(mcp.NewTool("update_vendor",
		mcp.WithDescription("Update an existing vendor. ID and name are required."),
		mcp.WithNumber("id", mcp.Required(), mcp.Description("Vendor ID")),
		mcp.WithString("name", mcp.Required(), mcp.Description("Vendor name")),
		mcp.WithString("address", mcp.Description("Street address")),
		mcp.WithString("city", mcp.Description("City")),
		mcp.WithString("state", mcp.Description("State")),
		mcp.WithString("zip", mcp.Description("ZIP code")),
		mcp.WithString("phone", mcp.Description("Phone number")),
		mcp.WithString("fax", mcp.Description("Fax number")),
		mcp.WithString("contact", mcp.Description("Contact person")),
		mcp.WithString("terms", mcp.Description("Payment terms")),
		mcp.WithString("tax_id", mcp.Description("Tax ID")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := argMap(req)
		resp, err := client.vendors.UpdateVendor(ctx, connect.NewRequest(&pb.UpdateVendorRequest{
			Id:      int32(argInt(args, "id", 0)),
			Name:    argStr(args, "name"),
			Address: argStrPtr(args, "address"),
			City:    argStrPtr(args, "city"),
			State:   argStrPtr(args, "state"),
			Zip:     argStrPtr(args, "zip"),
			Phone:   argStrPtr(args, "phone"),
			Fax:     argStrPtr(args, "fax"),
			Contact: argStrPtr(args, "contact"),
			Terms:   argStrPtr(args, "terms"),
			TaxId:   argStrPtr(args, "tax_id"),
		}))
		if err != nil {
			return mcp.NewToolResultError(connectErr(err)), nil
		}
		return mcp.NewToolResultText(fmt.Sprintf("Updated vendor #%d: %s", resp.Msg.Vendor.Id, resp.Msg.Vendor.Name)), nil
	})

	register(mcp.NewTool("delete_vendor",
		mcp.WithDescription("Delete a vendor by ID."),
		mcp.WithNumber("id", mcp.Required(), mcp.Description("Vendor ID")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := argMap(req)
		_, err := client.vendors.DeleteVendor(ctx, connect.NewRequest(&pb.DeleteVendorRequest{
			Id: int32(argInt(args, "id", 0)),
		}))
		if err != nil {
			return mcp.NewToolResultError(connectErr(err)), nil
		}
		return mcp.NewToolResultText("Vendor deleted successfully."), nil
	})
}

func formatVendorList(resp *pb.ListVendorsResponse) string {
	if len(resp.Vendors) == 0 {
		return "No vendors found."
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Found %d vendors (page %d, %d per page):\n\n",
		resp.Pagination.TotalCount, resp.Pagination.Page, resp.Pagination.PageSize))
	for _, v := range resp.Vendors {
		sb.WriteString(fmt.Sprintf("  #%d %s", v.Id, v.Name))
		if v.City != nil || v.State != nil {
			sb.WriteString(fmt.Sprintf(" — %s, %s", deref(v.City), deref(v.State)))
		}
		if v.Phone != nil {
			sb.WriteString(fmt.Sprintf(" | %s", *v.Phone))
		}
		sb.WriteString("\n")
	}
	return sb.String()
}

func formatVendor(v *pb.Vendor) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Vendor #%d: %s\n", v.Id, v.Name))
	if v.Address != nil {
		sb.WriteString(fmt.Sprintf("  Address: %s\n", *v.Address))
	}
	if v.City != nil || v.State != nil || v.Zip != nil {
		sb.WriteString(fmt.Sprintf("  City/State/Zip: %s, %s %s\n", deref(v.City), deref(v.State), deref(v.Zip)))
	}
	if v.Phone != nil {
		sb.WriteString(fmt.Sprintf("  Phone: %s\n", *v.Phone))
	}
	if v.Contact != nil {
		sb.WriteString(fmt.Sprintf("  Contact: %s\n", *v.Contact))
	}
	if v.Terms != nil {
		sb.WriteString(fmt.Sprintf("  Terms: %s\n", *v.Terms))
	}
	return sb.String()
}
