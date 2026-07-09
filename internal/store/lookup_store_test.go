package store

import "testing"

func TestNewLookupStoreAllowlist(t *testing.T) {
	// Valid tables should succeed (pool can be nil for this test since
	// we're only testing the allowlist check, not database ops)
	valid := []string{
		"dispatch_codes", "equipment_types", "maintenance_types", "hold_codes",
		"declination_codes", "regions", "damage_areas",
		"damage_types", "damage_severities",
		"field_codes_1", "field_codes_2", "field_codes_3",
		"field_codes_4", "field_codes_5",
	}
	for _, name := range valid {
		s, err := NewLookupStore(nil, name)
		if err != nil {
			t.Errorf("NewLookupStore(%q) unexpected error: %v", name, err)
		}
		if s == nil {
			t.Errorf("NewLookupStore(%q) returned nil", name)
		}
	}

	// Invalid tables should fail
	invalid := []string{
		"users", "customers", "orders",
		"", "DROP TABLE users", "dispatch_codes; --",
	}
	for _, name := range invalid {
		_, err := NewLookupStore(nil, name)
		if err == nil {
			t.Errorf("NewLookupStore(%q) should have returned error", name)
		}
	}
}

func TestLookupStoreTableName(t *testing.T) {
	s, _ := NewLookupStore(nil, "dispatch_codes")
	if s.TableName() != "dispatch_codes" {
		t.Errorf("TableName() = %q, want dispatch_codes", s.TableName())
	}
}

func TestLookupStoreCodeColumn(t *testing.T) {
	tests := []struct {
		table string
		want  string
	}{
		{"dispatch_codes", "code"},
		{"equipment_types", "type_code"},
		{"maintenance_types", "code"},
		{"regions", "region"},
		{"hold_codes", "code"},
		{"damage_areas", "code"},
	}
	for _, tt := range tests {
		s, _ := NewLookupStore(nil, tt.table)
		if got := s.codeColumn(); got != tt.want {
			t.Errorf("codeColumn(%q) = %q, want %q", tt.table, got, tt.want)
		}
	}
}
