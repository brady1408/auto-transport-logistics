package main

import (
	"context"
	"encoding/json"
	"fmt"

	"connectrpc.com/connect"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	pb "github.com/brady1408/auto-transport-logistics/internal/gen/atlinks/v1"
)

func registerResources(s *server.MCPServer, client *atlClient) {
	// Lookup tables as resources — small reference data useful for AI context
	lookups := []struct {
		name string
		desc string
		list func(context.Context, *connect.Request[pb.ListLookupRequest]) (*connect.Response[pb.ListLookupResponse], error)
	}{
		{"dispatch_codes", "Dispatch status codes", client.lookups.ListDispatchCodes},
		{"equipment_types", "Equipment/trailer types", client.lookups.ListEquipmentTypes},
		{"hold_codes", "Hold reason codes", client.lookups.ListHoldCodes},
		{"declination_codes", "Declination reason codes", client.lookups.ListDeclinationCodes},
		{"regions", "Geographic regions", client.lookups.ListRegions},
		{"damage_areas", "Vehicle damage area codes", client.lookups.ListDamageAreas},
		{"damage_types", "Vehicle damage type codes", client.lookups.ListDamageTypes},
		{"damage_severities", "Vehicle damage severity codes", client.lookups.ListDamageSeverities},
		{"field_codes_1", "Custom field codes 1", client.lookups.ListFieldCodes1},
		{"field_codes_2", "Custom field codes 2", client.lookups.ListFieldCodes2},
		{"field_codes_3", "Custom field codes 3", client.lookups.ListFieldCodes3},
		{"field_codes_4", "Custom field codes 4", client.lookups.ListFieldCodes4},
		{"field_codes_5", "Custom field codes 5", client.lookups.ListFieldCodes5},
	}

	for _, l := range lookups {
		s.AddResource(
			mcp.NewResource(
				"atlinks://lookups/"+l.name,
				l.name,
				mcp.WithResourceDescription(l.desc),
				mcp.WithMIMEType("application/json"),
			),
			func(ctx context.Context, req mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
				resp, err := l.list(ctx, connect.NewRequest(&pb.ListLookupRequest{}))
				if err != nil {
					return nil, fmt.Errorf("fetch %s: %w", l.name, err)
				}
				data, _ := json.MarshalIndent(resp.Msg.Items, "", "  ")
				return []mcp.ResourceContents{
					mcp.TextResourceContents{
						URI:      "atlinks://lookups/" + l.name,
						MIMEType: "application/json",
						Text:     string(data),
					},
				}, nil
			},
		)
	}

	// Terms
	s.AddResource(
		mcp.NewResource("atlinks://lookups/terms", "terms",
			mcp.WithResourceDescription("Payment terms"),
			mcp.WithMIMEType("application/json"),
		),
		func(ctx context.Context, req mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
			resp, err := client.lookups.ListTerms(ctx, connect.NewRequest(&pb.ListLookupRequest{}))
			if err != nil {
				return nil, fmt.Errorf("fetch terms: %w", err)
			}
			data, _ := json.MarshalIndent(resp.Msg.Items, "", "  ")
			return []mcp.ResourceContents{
				mcp.TextResourceContents{
					URI:      "atlinks://lookups/terms",
					MIMEType: "application/json",
					Text:     string(data),
				},
			}, nil
		},
	)

	// Tax codes
	s.AddResource(
		mcp.NewResource("atlinks://lookups/tax_codes", "tax_codes",
			mcp.WithResourceDescription("Tax codes with rates"),
			mcp.WithMIMEType("application/json"),
		),
		func(ctx context.Context, req mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
			resp, err := client.lookups.ListTaxCodes(ctx, connect.NewRequest(&pb.ListLookupRequest{}))
			if err != nil {
				return nil, fmt.Errorf("fetch tax_codes: %w", err)
			}
			data, _ := json.MarshalIndent(resp.Msg.Items, "", "  ")
			return []mcp.ResourceContents{
				mcp.TextResourceContents{
					URI:      "atlinks://lookups/tax_codes",
					MIMEType: "application/json",
					Text:     string(data),
				},
			}, nil
		},
	)

	// Items (billing items)
	s.AddResource(
		mcp.NewResource("atlinks://lookups/items", "items",
			mcp.WithResourceDescription("Billing item codes with default amounts"),
			mcp.WithMIMEType("application/json"),
		),
		func(ctx context.Context, req mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
			resp, err := client.lookups.ListItems(ctx, connect.NewRequest(&pb.ListLookupRequest{}))
			if err != nil {
				return nil, fmt.Errorf("fetch items: %w", err)
			}
			data, _ := json.MarshalIndent(resp.Msg.Items, "", "  ")
			return []mcp.ResourceContents{
				mcp.TextResourceContents{
					URI:      "atlinks://lookups/items",
					MIMEType: "application/json",
					Text:     string(data),
				},
			}, nil
		},
	)
}
