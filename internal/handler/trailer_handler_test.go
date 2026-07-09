package handler

import (
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestBindTrailerForm(t *testing.T) {
	form := url.Values{
		"trailer_number":    {"  T101  "},
		"make":              {"Cottrell"},
		"model":             {"CX-09"},
		"year":              {"2019"},
		"serial_number":     {"1C9SA4839KM123456"},
		"type_code":         {"TRLR"},
		"manufacture_date":  {"2019-03-15"},
		"license":           {"ABC1234"},
		"license_exp":       {"2027-01-31"},
		"safety_inspection": {"2026-06-30"},
		"tare_weight":       {"14500"},
		"capacity":          {"9"},
		"length_ft":         {"53"},
		"width_ft":          {"8.5"},
		"height_ft":         {"13.5"},
		"purchased_from":    {"Cottrell Inc"},
		"purchase_date":     {"2019-04-01"},
		"cost":              {"85000.00"},
		"comments":          {"Rebuilt ramps 2024"},
		"active":            {"on"},
	}
	r := httptest.NewRequest("POST", "/global/trailers", strings.NewReader(form.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	got := bindTrailerForm(r)

	if got.TrailerNumber != "T101" {
		t.Errorf("TrailerNumber = %q, want T101", got.TrailerNumber)
	}
	if got.Make == nil || *got.Make != "Cottrell" {
		t.Errorf("Make = %v, want Cottrell", got.Make)
	}
	if got.TypeCode == nil || *got.TypeCode != "TRLR" {
		t.Errorf("TypeCode = %v, want TRLR", got.TypeCode)
	}
	if got.TareWeight == nil || *got.TareWeight != 14500 {
		t.Errorf("TareWeight = %v, want 14500", got.TareWeight)
	}
	if got.Capacity == nil || *got.Capacity != 9 {
		t.Errorf("Capacity = %v, want 9", got.Capacity)
	}
	if got.LengthFt == nil || *got.LengthFt != "53" {
		t.Errorf("LengthFt = %v, want 53", got.LengthFt)
	}
	if got.Cost == nil || *got.Cost != "85000.00" {
		t.Errorf("Cost = %v, want 85000.00", got.Cost)
	}
	if !got.Active {
		t.Error("Active = false, want true")
	}

	wantLicenseExp := time.Date(2027, 1, 31, 0, 0, 0, 0, time.UTC)
	if got.LicenseExp == nil || !got.LicenseExp.Equal(wantLicenseExp) {
		t.Errorf("LicenseExp = %v, want %v", got.LicenseExp, wantLicenseExp)
	}
	wantInspection := time.Date(2026, 6, 30, 0, 0, 0, 0, time.UTC)
	if got.SafetyInspection == nil || !got.SafetyInspection.Equal(wantInspection) {
		t.Errorf("SafetyInspection = %v, want %v", got.SafetyInspection, wantInspection)
	}
}

func TestBindTrailerFormEmpty(t *testing.T) {
	form := url.Values{
		"trailer_number": {""},
		"tare_weight":    {"abc"},
		"license_exp":    {"not-a-date"},
	}
	r := httptest.NewRequest("POST", "/global/trailers", strings.NewReader(form.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	got := bindTrailerForm(r)

	if got.TrailerNumber != "" {
		t.Errorf("TrailerNumber = %q, want empty", got.TrailerNumber)
	}
	if got.Make != nil {
		t.Errorf("Make = %v, want nil", got.Make)
	}
	if got.TareWeight != nil {
		t.Errorf("TareWeight = %v, want nil for non-numeric input", got.TareWeight)
	}
	if got.LicenseExp != nil {
		t.Errorf("LicenseExp = %v, want nil for invalid date", got.LicenseExp)
	}
	if got.Active {
		t.Error("Active = true, want false when unchecked")
	}
}
