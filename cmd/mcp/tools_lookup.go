package main

import (
	"context"
	"fmt"
	"strings"

	"connectrpc.com/connect"
	"github.com/mark3labs/mcp-go/mcp"

	pb "github.com/brady1408/atlinks/internal/gen/atlinks/v1"
	"github.com/brady1408/atlinks/internal/gen/atlinks/v1/atlinkspbconnect"
)

func registerLookupTools(register toolRegister, client *atlClient) {
	// Generic lookups — 13 tables × 4 tools = 52 tools
	type lookupDef struct {
		singular string
		plural   string
		list     func(context.Context, *connect.Request[pb.ListLookupRequest]) (*connect.Response[pb.ListLookupResponse], error)
		create   func(context.Context, *connect.Request[pb.CreateLookupRequest]) (*connect.Response[pb.CreateLookupResponse], error)
		update   func(context.Context, *connect.Request[pb.UpdateLookupRequest]) (*connect.Response[pb.UpdateLookupResponse], error)
		del      func(context.Context, *connect.Request[pb.DeleteLookupRequest]) (*connect.Response[pb.DeleteLookupResponse], error)
	}

	lookups := []lookupDef{
		{"dispatch_code", "dispatch_codes", client.lookups.ListDispatchCodes, client.lookups.CreateDispatchCode, client.lookups.UpdateDispatchCode, client.lookups.DeleteDispatchCode},
		{"equipment_type", "equipment_types", client.lookups.ListEquipmentTypes, client.lookups.CreateEquipmentType, client.lookups.UpdateEquipmentType, client.lookups.DeleteEquipmentType},
		{"hold_code", "hold_codes", client.lookups.ListHoldCodes, client.lookups.CreateHoldCode, client.lookups.UpdateHoldCode, client.lookups.DeleteHoldCode},
		{"declination_code", "declination_codes", client.lookups.ListDeclinationCodes, client.lookups.CreateDeclinationCode, client.lookups.UpdateDeclinationCode, client.lookups.DeleteDeclinationCode},
		{"region", "regions", client.lookups.ListRegions, client.lookups.CreateRegion, client.lookups.UpdateRegion, client.lookups.DeleteRegion},
		{"damage_area", "damage_areas", client.lookups.ListDamageAreas, client.lookups.CreateDamageArea, client.lookups.UpdateDamageArea, client.lookups.DeleteDamageArea},
		{"damage_type", "damage_types", client.lookups.ListDamageTypes, client.lookups.CreateDamageType, client.lookups.UpdateDamageType, client.lookups.DeleteDamageType},
		{"damage_severity", "damage_severities", client.lookups.ListDamageSeverities, client.lookups.CreateDamageSeverity, client.lookups.UpdateDamageSeverity, client.lookups.DeleteDamageSeverity},
		{"field_code_1", "field_codes_1", client.lookups.ListFieldCodes1, client.lookups.CreateFieldCode1, client.lookups.UpdateFieldCode1, client.lookups.DeleteFieldCode1},
		{"field_code_2", "field_codes_2", client.lookups.ListFieldCodes2, client.lookups.CreateFieldCode2, client.lookups.UpdateFieldCode2, client.lookups.DeleteFieldCode2},
		{"field_code_3", "field_codes_3", client.lookups.ListFieldCodes3, client.lookups.CreateFieldCode3, client.lookups.UpdateFieldCode3, client.lookups.DeleteFieldCode3},
		{"field_code_4", "field_codes_4", client.lookups.ListFieldCodes4, client.lookups.CreateFieldCode4, client.lookups.UpdateFieldCode4, client.lookups.DeleteFieldCode4},
		{"field_code_5", "field_codes_5", client.lookups.ListFieldCodes5, client.lookups.CreateFieldCode5, client.lookups.UpdateFieldCode5, client.lookups.DeleteFieldCode5},
	}

	for _, l := range lookups {
		registerGenericLookup(register, l)
	}

	// --- Terms (4 tools) ---
	registerTermsTools(register, client.lookups)

	// --- Tax Codes (4 tools) ---
	registerTaxCodeTools(register, client.lookups)

	// --- Items (4 tools) ---
	registerItemTools(register, client.lookups)
}

func registerGenericLookup(register toolRegister, l struct {
	singular string
	plural   string
	list     func(context.Context, *connect.Request[pb.ListLookupRequest]) (*connect.Response[pb.ListLookupResponse], error)
	create   func(context.Context, *connect.Request[pb.CreateLookupRequest]) (*connect.Response[pb.CreateLookupResponse], error)
	update   func(context.Context, *connect.Request[pb.UpdateLookupRequest]) (*connect.Response[pb.UpdateLookupResponse], error)
	del      func(context.Context, *connect.Request[pb.DeleteLookupRequest]) (*connect.Response[pb.DeleteLookupResponse], error)
}) {
	register(mcp.NewTool("list_"+l.plural,
		mcp.WithDescription(fmt.Sprintf("List all %s.", strings.ReplaceAll(l.plural, "_", " "))),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		resp, err := l.list(ctx, connect.NewRequest(&pb.ListLookupRequest{}))
		if err != nil {
			return mcp.NewToolResultError(connectErr(err)), nil
		}
		return mcp.NewToolResultText(formatLookupItems(resp.Msg.Items, l.plural)), nil
	})

	register(mcp.NewTool("create_"+l.singular,
		mcp.WithDescription(fmt.Sprintf("Create a %s.", strings.ReplaceAll(l.singular, "_", " "))),
		mcp.WithString("code", mcp.Required(), mcp.Description("Code")),
		mcp.WithString("description", mcp.Description("Description")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := argMap(req)
		resp, err := l.create(ctx, connect.NewRequest(&pb.CreateLookupRequest{
			Code:        argStr(args, "code"),
			Description: argStr(args, "description"),
		}))
		if err != nil {
			return mcp.NewToolResultError(connectErr(err)), nil
		}
		return mcp.NewToolResultText(fmt.Sprintf("Created %s #%d: %s", l.singular, resp.Msg.Item.Id, resp.Msg.Item.Code)), nil
	})

	register(mcp.NewTool("update_"+l.singular,
		mcp.WithDescription(fmt.Sprintf("Update a %s.", strings.ReplaceAll(l.singular, "_", " "))),
		mcp.WithNumber("id", mcp.Required(), mcp.Description("ID")),
		mcp.WithString("code", mcp.Required(), mcp.Description("Code")),
		mcp.WithString("description", mcp.Description("Description")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := argMap(req)
		_, err := l.update(ctx, connect.NewRequest(&pb.UpdateLookupRequest{
			Id:          int32(argInt(args, "id", 0)),
			Code:        argStr(args, "code"),
			Description: argStr(args, "description"),
		}))
		if err != nil {
			return mcp.NewToolResultError(connectErr(err)), nil
		}
		return mcp.NewToolResultText(fmt.Sprintf("Updated %s.", l.singular)), nil
	})

	register(mcp.NewTool("delete_"+l.singular,
		mcp.WithDescription(fmt.Sprintf("Delete a %s.", strings.ReplaceAll(l.singular, "_", " "))),
		mcp.WithNumber("id", mcp.Required(), mcp.Description("ID")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := argMap(req)
		_, err := l.del(ctx, connect.NewRequest(&pb.DeleteLookupRequest{
			Id: int32(argInt(args, "id", 0)),
		}))
		if err != nil {
			return mcp.NewToolResultError(connectErr(err)), nil
		}
		return mcp.NewToolResultText(fmt.Sprintf("Deleted %s.", l.singular)), nil
	})
}

func registerTermsTools(register toolRegister, lookups atlinkspbconnect.LookupServiceClient) {
	register(mcp.NewTool("list_terms",
		mcp.WithDescription("List all payment terms."),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		resp, err := lookups.ListTerms(ctx, connect.NewRequest(&pb.ListLookupRequest{}))
		if err != nil {
			return mcp.NewToolResultError(connectErr(err)), nil
		}
		if len(resp.Msg.Items) == 0 {
			return mcp.NewToolResultText("No terms found."), nil
		}
		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("Terms (%d):\n\n", len(resp.Msg.Items)))
		for _, t := range resp.Msg.Items {
			sb.WriteString(fmt.Sprintf("  #%d %s — %s", t.Id, t.Term, t.Description))
			if t.Days != nil {
				sb.WriteString(fmt.Sprintf(" (%d days)", *t.Days))
			}
			sb.WriteString("\n")
		}
		return mcp.NewToolResultText(sb.String()), nil
	})

	register(mcp.NewTool("create_term",
		mcp.WithDescription("Create a payment term."),
		mcp.WithString("term", mcp.Required(), mcp.Description("Term code")),
		mcp.WithString("description", mcp.Description("Description")),
		mcp.WithNumber("days", mcp.Description("Number of days")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := argMap(req)
		resp, err := lookups.CreateTerm(ctx, connect.NewRequest(&pb.CreateTermRequest{
			Term:        argStr(args, "term"),
			Description: argStr(args, "description"),
			Days:        argI32Ptr(args, "days"),
		}))
		if err != nil {
			return mcp.NewToolResultError(connectErr(err)), nil
		}
		return mcp.NewToolResultText(fmt.Sprintf("Created term #%d: %s", resp.Msg.Item.Id, resp.Msg.Item.Term)), nil
	})

	register(mcp.NewTool("update_term",
		mcp.WithDescription("Update a payment term."),
		mcp.WithNumber("id", mcp.Required(), mcp.Description("Term ID")),
		mcp.WithString("term", mcp.Required(), mcp.Description("Term code")),
		mcp.WithString("description", mcp.Description("Description")),
		mcp.WithNumber("days", mcp.Description("Number of days")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := argMap(req)
		_, err := lookups.UpdateTerm(ctx, connect.NewRequest(&pb.UpdateTermRequest{
			Id:          int32(argInt(args, "id", 0)),
			Term:        argStr(args, "term"),
			Description: argStr(args, "description"),
			Days:        argI32Ptr(args, "days"),
		}))
		if err != nil {
			return mcp.NewToolResultError(connectErr(err)), nil
		}
		return mcp.NewToolResultText("Term updated."), nil
	})

	register(mcp.NewTool("delete_term",
		mcp.WithDescription("Delete a payment term."),
		mcp.WithNumber("id", mcp.Required(), mcp.Description("Term ID")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := argMap(req)
		_, err := lookups.DeleteTerm(ctx, connect.NewRequest(&pb.DeleteLookupRequest{
			Id: int32(argInt(args, "id", 0)),
		}))
		if err != nil {
			return mcp.NewToolResultError(connectErr(err)), nil
		}
		return mcp.NewToolResultText("Term deleted."), nil
	})
}

func registerTaxCodeTools(register toolRegister, lookups atlinkspbconnect.LookupServiceClient) {
	register(mcp.NewTool("list_tax_codes",
		mcp.WithDescription("List all tax codes."),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		resp, err := lookups.ListTaxCodes(ctx, connect.NewRequest(&pb.ListLookupRequest{}))
		if err != nil {
			return mcp.NewToolResultError(connectErr(err)), nil
		}
		if len(resp.Msg.Items) == 0 {
			return mcp.NewToolResultText("No tax codes found."), nil
		}
		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("Tax codes (%d):\n\n", len(resp.Msg.Items)))
		for _, t := range resp.Msg.Items {
			sb.WriteString(fmt.Sprintf("  #%d %s — %s", t.Id, t.Code, t.Description))
			if t.Rate != nil {
				sb.WriteString(fmt.Sprintf(" (rate: %s)", *t.Rate))
			}
			sb.WriteString("\n")
		}
		return mcp.NewToolResultText(sb.String()), nil
	})

	register(mcp.NewTool("create_tax_code",
		mcp.WithDescription("Create a tax code."),
		mcp.WithString("code", mcp.Required(), mcp.Description("Tax code")),
		mcp.WithString("description", mcp.Description("Description")),
		mcp.WithString("rate", mcp.Description("Tax rate")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := argMap(req)
		resp, err := lookups.CreateTaxCode(ctx, connect.NewRequest(&pb.CreateTaxCodeRequest{
			Code:        argStr(args, "code"),
			Description: argStr(args, "description"),
			Rate:        argStrPtr(args, "rate"),
		}))
		if err != nil {
			return mcp.NewToolResultError(connectErr(err)), nil
		}
		return mcp.NewToolResultText(fmt.Sprintf("Created tax code #%d: %s", resp.Msg.Item.Id, resp.Msg.Item.Code)), nil
	})

	register(mcp.NewTool("update_tax_code",
		mcp.WithDescription("Update a tax code."),
		mcp.WithNumber("id", mcp.Required(), mcp.Description("Tax code ID")),
		mcp.WithString("code", mcp.Required(), mcp.Description("Tax code")),
		mcp.WithString("description", mcp.Description("Description")),
		mcp.WithString("rate", mcp.Description("Tax rate")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := argMap(req)
		_, err := lookups.UpdateTaxCode(ctx, connect.NewRequest(&pb.UpdateTaxCodeRequest{
			Id:          int32(argInt(args, "id", 0)),
			Code:        argStr(args, "code"),
			Description: argStr(args, "description"),
			Rate:        argStrPtr(args, "rate"),
		}))
		if err != nil {
			return mcp.NewToolResultError(connectErr(err)), nil
		}
		return mcp.NewToolResultText("Tax code updated."), nil
	})

	register(mcp.NewTool("delete_tax_code",
		mcp.WithDescription("Delete a tax code."),
		mcp.WithNumber("id", mcp.Required(), mcp.Description("Tax code ID")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := argMap(req)
		_, err := lookups.DeleteTaxCode(ctx, connect.NewRequest(&pb.DeleteLookupRequest{
			Id: int32(argInt(args, "id", 0)),
		}))
		if err != nil {
			return mcp.NewToolResultError(connectErr(err)), nil
		}
		return mcp.NewToolResultText("Tax code deleted."), nil
	})
}

func registerItemTools(register toolRegister, lookups atlinkspbconnect.LookupServiceClient) {
	register(mcp.NewTool("list_items",
		mcp.WithDescription("List all item codes."),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		resp, err := lookups.ListItems(ctx, connect.NewRequest(&pb.ListLookupRequest{}))
		if err != nil {
			return mcp.NewToolResultError(connectErr(err)), nil
		}
		if len(resp.Msg.Items) == 0 {
			return mcp.NewToolResultText("No items found."), nil
		}
		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("Items (%d):\n\n", len(resp.Msg.Items)))
		for _, i := range resp.Msg.Items {
			sb.WriteString(fmt.Sprintf("  #%d %s — %s", i.Id, i.Item, i.Description))
			if i.DefaultAmount != nil {
				sb.WriteString(fmt.Sprintf(" ($%s)", *i.DefaultAmount))
			}
			if i.CalcType != nil {
				sb.WriteString(fmt.Sprintf(" [%s]", *i.CalcType))
			}
			sb.WriteString("\n")
		}
		return mcp.NewToolResultText(sb.String()), nil
	})

	register(mcp.NewTool("create_item",
		mcp.WithDescription("Create an item code."),
		mcp.WithString("item", mcp.Required(), mcp.Description("Item code")),
		mcp.WithString("description", mcp.Description("Description")),
		mcp.WithString("default_amount", mcp.Description("Default amount")),
		mcp.WithString("calc_type", mcp.Description("Calculation type")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := argMap(req)
		resp, err := lookups.CreateItem(ctx, connect.NewRequest(&pb.CreateItemRequest{
			Item:          argStr(args, "item"),
			Description:   argStr(args, "description"),
			DefaultAmount: argStrPtr(args, "default_amount"),
			CalcType:      argStrPtr(args, "calc_type"),
		}))
		if err != nil {
			return mcp.NewToolResultError(connectErr(err)), nil
		}
		return mcp.NewToolResultText(fmt.Sprintf("Created item #%d: %s", resp.Msg.Item.Id, resp.Msg.Item.Item)), nil
	})

	register(mcp.NewTool("update_item",
		mcp.WithDescription("Update an item code."),
		mcp.WithNumber("id", mcp.Required(), mcp.Description("Item ID")),
		mcp.WithString("item", mcp.Required(), mcp.Description("Item code")),
		mcp.WithString("description", mcp.Description("Description")),
		mcp.WithString("default_amount", mcp.Description("Default amount")),
		mcp.WithString("calc_type", mcp.Description("Calculation type")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := argMap(req)
		_, err := lookups.UpdateItem(ctx, connect.NewRequest(&pb.UpdateItemRequest{
			Id:            int32(argInt(args, "id", 0)),
			Item:          argStr(args, "item"),
			Description:   argStr(args, "description"),
			DefaultAmount: argStrPtr(args, "default_amount"),
			CalcType:      argStrPtr(args, "calc_type"),
		}))
		if err != nil {
			return mcp.NewToolResultError(connectErr(err)), nil
		}
		return mcp.NewToolResultText("Item updated."), nil
	})

	register(mcp.NewTool("delete_item",
		mcp.WithDescription("Delete an item code."),
		mcp.WithNumber("id", mcp.Required(), mcp.Description("Item ID")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := argMap(req)
		_, err := lookups.DeleteItem(ctx, connect.NewRequest(&pb.DeleteLookupRequest{
			Id: int32(argInt(args, "id", 0)),
		}))
		if err != nil {
			return mcp.NewToolResultError(connectErr(err)), nil
		}
		return mcp.NewToolResultText("Item deleted."), nil
	})
}

func formatLookupItems(items []*pb.LookupItem, label string) string {
	if len(items) == 0 {
		return fmt.Sprintf("No %s found.", strings.ReplaceAll(label, "_", " "))
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("%s (%d):\n\n", strings.ReplaceAll(label, "_", " "), len(items)))
	for _, item := range items {
		sb.WriteString(fmt.Sprintf("  #%d %s — %s\n", item.Id, item.Code, item.Description))
	}
	return sb.String()
}
