package service

import "testing"

func TestParseCents(t *testing.T) {
	tests := []struct {
		input string
		want  int64
	}{
		{"123.45", 12345},
		{"0.01", 1},
		{"0.1", 10},
		{"0.10", 10},
		{".99", 99},
		{"1", 100},
		{"1234", 123400},
		{"0", 0},
		{"0.00", 0},
		{"-5.50", -550},
		{"-0.01", -1},
		{"", 0},
		{"  ", 0},
		{"abc", 0},
		{"  123.45  ", 12345},
		{"99.999", 10000}, // rounds to 100.00
		{"99.995", 10000}, // rounds to 100.00
		{"99.994", 9999},  // rounds to 99.99
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := parseCents(tt.input)
			if got != tt.want {
				t.Errorf("parseCents(%q) = %d, want %d", tt.input, got, tt.want)
			}
		})
	}
}

func TestParseCentsPtr(t *testing.T) {
	if got := parseCentsPtr(nil); got != 0 {
		t.Errorf("parseCentsPtr(nil) = %d, want 0", got)
	}

	s := "50.25"
	if got := parseCentsPtr(&s); got != 5025 {
		t.Errorf("parseCentsPtr(%q) = %d, want 5025", s, got)
	}

	empty := ""
	if got := parseCentsPtr(&empty); got != 0 {
		t.Errorf("parseCentsPtr(%q) = %d, want 0", empty, got)
	}
}

func TestCentsToStr(t *testing.T) {
	tests := []struct {
		input int64
		want  string
	}{
		{12345, "123.45"},
		{100, "1.00"},
		{1, "0.01"},
		{10, "0.10"},
		{0, "0.00"},
		{-550, "-5.50"},
		{-1, "-0.01"},
		{123400, "1234.00"},
	}
	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := centsToStr(tt.input)
			if got != tt.want {
				t.Errorf("centsToStr(%d) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
