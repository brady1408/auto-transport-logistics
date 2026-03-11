package main

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

type toolRegister = func(mcp.Tool, func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error))

func registerAllTools(s *server.MCPServer, client *atlClient) {
	register := toolRegister(func(tool mcp.Tool, handler func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error)) {
		s.AddTool(tool, handler)
	})

	registerCustomerTools(register, client)
	registerOrderTools(register, client)
	registerVehicleTools(register, client)
	registerFeedbackTools(register, client)
	registerEmployeeTools(register, client)
	registerTruckTools(register, client)
	registerVendorTools(register, client)
	registerZoneTools(register, client)
	registerChargeTools(register, client)
	registerDamageTools(register, client)
	registerDamageClaimTools(register, client)
	registerCreditMemoTools(register, client)
	registerTripTools(register, client)
	registerInvoiceTools(register, client)
	registerPaymentTools(register, client)
	registerAPTools(register, client)
	registerEarningsTools(register, client)
	registerLookupTools(register, client)
}

// --- Argument helpers ---

func argMap(req mcp.CallToolRequest) map[string]any {
	return req.GetArguments()
}

func argStr(args map[string]any, key string) string {
	if v, ok := args[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func argStrPtr(args map[string]any, key string) *string {
	if v, ok := args[key]; ok {
		if s, ok := v.(string); ok && s != "" {
			return &s
		}
	}
	return nil
}

func argInt(args map[string]any, key string, dflt int) int {
	if v, ok := args[key]; ok {
		switch n := v.(type) {
		case float64:
			return int(n)
		case int:
			return n
		}
	}
	return dflt
}

func argI32Ptr(args map[string]any, key string) *int32 {
	if v, ok := args[key]; ok {
		switch n := v.(type) {
		case float64:
			i := int32(n)
			return &i
		case int:
			i := int32(n)
			return &i
		}
	}
	return nil
}

func argBool(args map[string]any, key string) bool {
	if v, ok := args[key]; ok {
		if b, ok := v.(bool); ok {
			return b
		}
	}
	return false
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
