package handler

import (
	"bytes"
	"testing"
)

func TestParseTemplates(t *testing.T) {
	tmpl, err := ParseTemplates()
	if err != nil {
		t.Fatalf("ParseTemplates() error: %v", err)
	}

	// Verify page templates exist
	pages := []string{
		"dashboard.html",
		"company_form.html",
		"customer_list.html",
		"customer_form.html",
		"employee_list.html",
		"employee_form.html",
		"truck_list.html",
		"truck_form.html",
		"zone_list.html",
		"zone_pricing_list.html",
		"lookup_list.html",
		"terms_list.html",
		"tax_codes_list.html",
		"items_list.html",
	}

	for _, name := range pages {
		if _, ok := tmpl.pages[name]; !ok {
			t.Errorf("page template %q not found", name)
		}
	}

	// Verify standalone/partial templates exist
	partials := []string{
		"login.html",
		"customer_table",
		"employee_table",
		"truck_table",
		"zone_table",
		"zone_pricing_table",
		"lookup_table",
		"terms_table",
		"tax_codes_table",
		"items_table",
	}

	for _, name := range partials {
		if tmpl.partials.Lookup(name) == nil {
			t.Errorf("partial/standalone template %q not found", name)
		}
	}

	// Verify a page can render (basic smoke test)
	var buf bytes.Buffer
	err = tmpl.RenderTemplate(&buf, "login.html", map[string]any{})
	if err != nil {
		t.Errorf("RenderTemplate(login.html) error: %v", err)
	}
	if !bytes.Contains(buf.Bytes(), []byte("ATLinks")) {
		t.Error("login.html output missing 'ATLinks'")
	}
}
