package handler

import (
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestBindMaintenanceLogForm(t *testing.T) {
	form := url.Values{}
	form.Set("maintenance_date", "2026-07-09")
	form.Set("type_code", "OIL")
	form.Set("mileage", "125000")
	form.Set("cost", "89.50")
	form.Set("notes", "Full synthetic")

	r := httptest.NewRequest("POST", "/global/trucks/1/maintenance", strings.NewReader(form.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	m, dateOK := bindMaintenanceLogForm(r)
	if !dateOK {
		t.Fatal("expected dateOK to be true")
	}
	if got := m.MaintenanceDate.Format("2006-01-02"); got != "2026-07-09" {
		t.Errorf("MaintenanceDate = %q, want 2026-07-09", got)
	}
	if m.TypeCode == nil || *m.TypeCode != "OIL" {
		t.Errorf("TypeCode = %v, want OIL", m.TypeCode)
	}
	if m.Mileage == nil || *m.Mileage != 125000 {
		t.Errorf("Mileage = %v, want 125000", m.Mileage)
	}
	if m.Cost == nil || *m.Cost != "89.50" {
		t.Errorf("Cost = %v, want 89.50", m.Cost)
	}
	if m.Notes == nil || *m.Notes != "Full synthetic" {
		t.Errorf("Notes = %v, want Full synthetic", m.Notes)
	}
}

func TestBindMaintenanceLogFormMissingDate(t *testing.T) {
	tests := []struct {
		name string
		date string
	}{
		{"empty", ""},
		{"garbage", "not-a-date"},
		{"wrong format", "07/09/2026"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			form := url.Values{}
			if tt.date != "" {
				form.Set("maintenance_date", tt.date)
			}
			form.Set("notes", "no date")

			r := httptest.NewRequest("POST", "/global/trucks/1/maintenance", strings.NewReader(form.Encode()))
			r.Header.Set("Content-Type", "application/x-www-form-urlencoded")

			m, dateOK := bindMaintenanceLogForm(r)
			if dateOK {
				t.Error("expected dateOK to be false")
			}
			if !m.MaintenanceDate.IsZero() {
				t.Errorf("MaintenanceDate = %v, want zero", m.MaintenanceDate)
			}
		})
	}
}

func TestBindMaintenanceLogFormOptionalFieldsEmpty(t *testing.T) {
	form := url.Values{}
	form.Set("maintenance_date", "2026-01-15")

	r := httptest.NewRequest("POST", "/global/trucks/1/maintenance", strings.NewReader(form.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	m, dateOK := bindMaintenanceLogForm(r)
	if !dateOK {
		t.Fatal("expected dateOK to be true")
	}
	if m.TypeCode != nil {
		t.Errorf("TypeCode = %v, want nil", m.TypeCode)
	}
	if m.Mileage != nil {
		t.Errorf("Mileage = %v, want nil", m.Mileage)
	}
	if m.Cost != nil {
		t.Errorf("Cost = %v, want nil", m.Cost)
	}
	if m.Notes != nil {
		t.Errorf("Notes = %v, want nil", m.Notes)
	}
}
