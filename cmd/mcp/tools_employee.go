package main

import (
	"context"
	"fmt"
	"strings"

	"connectrpc.com/connect"
	"github.com/mark3labs/mcp-go/mcp"

	pb "github.com/brady1408/atlinks/internal/gen/atlinks/v1"
)

func registerEmployeeTools(register toolRegister, client *atlClient) {
	register(mcp.NewTool("list_employees",
		mcp.WithDescription("List employees with optional filters. Returns paginated results."),
		mcp.WithString("search", mcp.Description("Search by name or employee number")),
		mcp.WithString("active", mcp.Description("Filter: 'active', 'inactive', or 'all' (default: all)")),
		mcp.WithString("is_driver", mcp.Description("Filter: 'true', 'false', or omit for all")),
		mcp.WithNumber("page", mcp.Description("Page number (default: 1)")),
		mcp.WithNumber("page_size", mcp.Description("Items per page (default: 25)")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := argMap(req)
		resp, err := client.employees.ListEmployees(ctx, connect.NewRequest(&pb.ListEmployeesRequest{
			Pagination: &pb.PaginationRequest{
				Page:     int32(argInt(args, "page", 1)),
				PageSize: int32(argInt(args, "page_size", 25)),
			},
			Search:   argStrPtr(args, "search"),
			Active:   argStrPtr(args, "active"),
			IsDriver: argStrPtr(args, "is_driver"),
		}))
		if err != nil {
			return mcp.NewToolResultError(connectErr(err)), nil
		}
		return mcp.NewToolResultText(formatEmployeeList(resp.Msg)), nil
	})

	register(mcp.NewTool("get_employee",
		mcp.WithDescription("Get a single employee by ID."),
		mcp.WithNumber("id", mcp.Required(), mcp.Description("Employee ID")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := argMap(req)
		resp, err := client.employees.GetEmployee(ctx, connect.NewRequest(&pb.GetEmployeeRequest{
			Id: int32(argInt(args, "id", 0)),
		}))
		if err != nil {
			return mcp.NewToolResultError(connectErr(err)), nil
		}
		return mcp.NewToolResultText(formatEmployee(resp.Msg.Employee)), nil
	})

	register(mcp.NewTool("create_employee",
		mcp.WithDescription("Create a new employee. Name is required."),
		mcp.WithString("name", mcp.Required(), mcp.Description("Employee name")),
		mcp.WithString("address", mcp.Description("Street address")),
		mcp.WithString("city", mcp.Description("City")),
		mcp.WithString("state", mcp.Description("State")),
		mcp.WithString("zip", mcp.Description("ZIP code")),
		mcp.WithString("phone", mcp.Description("Phone number")),
		mcp.WithString("rate", mcp.Description("Pay rate")),
		mcp.WithString("emp_id_number", mcp.Description("Employee ID number")),
		mcp.WithBoolean("active", mcp.Description("Active status (default: false)")),
		mcp.WithBoolean("is_driver", mcp.Description("Is a driver")),
		mcp.WithBoolean("is_sales", mcp.Description("Is a salesperson")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := argMap(req)
		resp, err := client.employees.CreateEmployee(ctx, connect.NewRequest(&pb.CreateEmployeeRequest{
			Name:        argStr(args, "name"),
			Address:     argStrPtr(args, "address"),
			City:        argStrPtr(args, "city"),
			State:       argStrPtr(args, "state"),
			Zip:         argStrPtr(args, "zip"),
			Phone:       argStrPtr(args, "phone"),
			Rate:        argStrPtr(args, "rate"),
			EmpIdNumber: argStrPtr(args, "emp_id_number"),
			Active:      argBool(args, "active"),
			IsDriver:    argBool(args, "is_driver"),
			IsSales:     argBool(args, "is_sales"),
		}))
		if err != nil {
			return mcp.NewToolResultError(connectErr(err)), nil
		}
		return mcp.NewToolResultText(fmt.Sprintf("Created employee #%d: %s", resp.Msg.Employee.Id, resp.Msg.Employee.Name)), nil
	})

	register(mcp.NewTool("update_employee",
		mcp.WithDescription("Update an existing employee. ID and name are required."),
		mcp.WithNumber("id", mcp.Required(), mcp.Description("Employee ID")),
		mcp.WithString("name", mcp.Required(), mcp.Description("Employee name")),
		mcp.WithString("address", mcp.Description("Street address")),
		mcp.WithString("city", mcp.Description("City")),
		mcp.WithString("state", mcp.Description("State")),
		mcp.WithString("zip", mcp.Description("ZIP code")),
		mcp.WithString("phone", mcp.Description("Phone number")),
		mcp.WithString("rate", mcp.Description("Pay rate")),
		mcp.WithString("emp_id_number", mcp.Description("Employee ID number")),
		mcp.WithBoolean("active", mcp.Description("Active status")),
		mcp.WithBoolean("is_driver", mcp.Description("Is a driver")),
		mcp.WithBoolean("is_sales", mcp.Description("Is a salesperson")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := argMap(req)
		resp, err := client.employees.UpdateEmployee(ctx, connect.NewRequest(&pb.UpdateEmployeeRequest{
			Id:          int32(argInt(args, "id", 0)),
			Name:        argStr(args, "name"),
			Address:     argStrPtr(args, "address"),
			City:        argStrPtr(args, "city"),
			State:       argStrPtr(args, "state"),
			Zip:         argStrPtr(args, "zip"),
			Phone:       argStrPtr(args, "phone"),
			Rate:        argStrPtr(args, "rate"),
			EmpIdNumber: argStrPtr(args, "emp_id_number"),
			Active:      argBool(args, "active"),
			IsDriver:    argBool(args, "is_driver"),
			IsSales:     argBool(args, "is_sales"),
		}))
		if err != nil {
			return mcp.NewToolResultError(connectErr(err)), nil
		}
		return mcp.NewToolResultText(fmt.Sprintf("Updated employee #%d: %s", resp.Msg.Employee.Id, resp.Msg.Employee.Name)), nil
	})

	register(mcp.NewTool("delete_employee",
		mcp.WithDescription("Delete (soft-delete) an employee by ID."),
		mcp.WithNumber("id", mcp.Required(), mcp.Description("Employee ID")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := argMap(req)
		_, err := client.employees.DeleteEmployee(ctx, connect.NewRequest(&pb.DeleteEmployeeRequest{
			Id: int32(argInt(args, "id", 0)),
		}))
		if err != nil {
			return mcp.NewToolResultError(connectErr(err)), nil
		}
		return mcp.NewToolResultText("Employee deleted successfully."), nil
	})
}

func formatEmployeeList(resp *pb.ListEmployeesResponse) string {
	if len(resp.Employees) == 0 {
		return "No employees found."
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Found %d employees (page %d, %d per page):\n\n",
		resp.Pagination.TotalCount, resp.Pagination.Page, resp.Pagination.PageSize))
	for _, e := range resp.Employees {
		sb.WriteString(fmt.Sprintf("  #%d %s", e.Id, e.Name))
		if e.EmpIdNumber != nil {
			sb.WriteString(fmt.Sprintf(" (%s)", *e.EmpIdNumber))
		}
		if !e.Active {
			sb.WriteString(" [INACTIVE]")
		}
		if e.IsDriver {
			sb.WriteString(" [DRIVER]")
		}
		sb.WriteString("\n")
	}
	return sb.String()
}

func formatEmployee(e *pb.Employee) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Employee #%d: %s\n", e.Id, e.Name))
	if e.EmpIdNumber != nil {
		sb.WriteString(fmt.Sprintf("  Employee Number: %s\n", *e.EmpIdNumber))
	}
	if e.Address != nil {
		sb.WriteString(fmt.Sprintf("  Address: %s\n", *e.Address))
	}
	if e.City != nil || e.State != nil || e.Zip != nil {
		sb.WriteString(fmt.Sprintf("  City/State/Zip: %s, %s %s\n", deref(e.City), deref(e.State), deref(e.Zip)))
	}
	if e.Phone != nil {
		sb.WriteString(fmt.Sprintf("  Phone: %s\n", *e.Phone))
	}
	if e.Rate != nil {
		sb.WriteString(fmt.Sprintf("  Rate: %s\n", *e.Rate))
	}
	sb.WriteString(fmt.Sprintf("  Active: %v | Driver: %v | Sales: %v\n", e.Active, e.IsDriver, e.IsSales))
	return sb.String()
}
