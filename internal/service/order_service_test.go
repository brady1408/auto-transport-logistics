package service

import "testing"

func TestIsForwardTransition(t *testing.T) {
	tests := []struct {
		from, to string
		want     bool
	}{
		{"Waiting", "Scheduled", true},
		{"Scheduled", "Loaded", true},
		{"Loaded", "Delivered", true},
		{"Delivered", "Confirmed", true},
		{"Waiting", "Confirmed", true},
		{"Confirmed", "Waiting", false},
		{"Delivered", "Loaded", false},
		{"Scheduled", "Waiting", false},
		{"Waiting", "Waiting", false},
		{"Confirmed", "Confirmed", false},
		{"", "Waiting", false}, // empty string maps to 0, same as Waiting
		{"Unknown", "Waiting", false},
	}
	for _, tt := range tests {
		t.Run(tt.from+"->"+tt.to, func(t *testing.T) {
			got := isForwardTransition(tt.from, tt.to)
			if got != tt.want {
				t.Errorf("isForwardTransition(%q, %q) = %v, want %v", tt.from, tt.to, got, tt.want)
			}
		})
	}
}

func TestValidTransitions(t *testing.T) {
	tests := []struct {
		from, to string
		valid    bool
	}{
		{"Waiting", "Scheduled", true},
		{"Waiting", "Loaded", false},
		{"Scheduled", "Loaded", true},
		{"Scheduled", "Waiting", true},
		{"Scheduled", "Delivered", false},
		{"Loaded", "Delivered", true},
		{"Loaded", "Scheduled", true},
		{"Delivered", "Confirmed", true},
		{"Delivered", "Loaded", true},
		{"Confirmed", "Delivered", true},
		{"Confirmed", "Waiting", false},
	}
	for _, tt := range tests {
		t.Run(tt.from+"->"+tt.to, func(t *testing.T) {
			allowed := validTransitions[tt.from]
			found := false
			for _, s := range allowed {
				if s == tt.to {
					found = true
					break
				}
			}
			if found != tt.valid {
				t.Errorf("validTransitions[%q] contains %q = %v, want %v", tt.from, tt.to, found, tt.valid)
			}
		})
	}
}
