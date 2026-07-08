package components

import "testing"

func TestMoney(t *testing.T) {
	ptr := func(s string) *string { return &s }

	tests := []struct {
		name string
		in   *string
		want string
	}{
		{"nil", nil, ""},
		{"empty", ptr(""), ""},
		{"whitespace", ptr("  "), ""},
		{"four decimals", ptr("780.0000"), "780.00"},
		{"two decimals", ptr("780.00"), "780.00"},
		{"no decimals", ptr("780"), "780.00"},
		{"thousands", ptr("1234.5"), "1,234.50"},
		{"millions", ptr("1234567.891"), "1,234,567.89"},
		{"dollar prefix", ptr("$1,234.50"), "1,234.50"},
		{"negative", ptr("-42.5"), "-42.50"},
		{"non-numeric passthrough", ptr("N/A"), "N/A"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Money(tt.in); got != tt.want {
				t.Errorf("Money(%v) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}
