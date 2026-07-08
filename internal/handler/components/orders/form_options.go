package orders

import "github.com/brady1408/auto-transport-logistics/internal/store"

// FormOptions carries the dropdown option lists for the order form.
type FormOptions struct {
	CalcTypes     []string
	FuelCalcTypes []string
	TaxCodes      []store.TaxCodeItem
}

func containsValue(values []string, v string) bool {
	for _, s := range values {
		if s == v {
			return true
		}
	}
	return false
}

func taxCodesContain(items []store.TaxCodeItem, code string) bool {
	for _, t := range items {
		if t.Code == code {
			return true
		}
	}
	return false
}

func taxCodeLabel(t store.TaxCodeItem) string {
	if t.Description != "" {
		return t.Code + " - " + t.Description
	}
	return t.Code
}
