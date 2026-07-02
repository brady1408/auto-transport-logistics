package customers

import (
	"strings"

	"github.com/brady1408/auto-transport-logistics/internal/handler/components"
	"github.com/brady1408/auto-transport-logistics/internal/models"
)

// searchResultPayload is the shape consumed by the atlinksSelectCustomer JS
// helper when a typeahead row is clicked. Fields mirror the read-only detail
// inputs the widget populates.
type searchResultPayload struct {
	ID       int    `json:"id"`
	Number   string `json:"number"`
	Name     string `json:"name"`
	Address  string `json:"address"`
	Address2 string `json:"address2"`
	City     string `json:"city"`
	State    string `json:"state"`
	Zip      string `json:"zip"`
	Contact  string `json:"contact"`
	Phone    string `json:"phone"`
	Zone     string `json:"zone"`
}

// searchResultData builds the JS payload embedded in each dropdown row.
func searchResultData(c models.Customer) searchResultPayload {
	return searchResultPayload{
		ID:       c.ID,
		Number:   components.Deref(c.Number),
		Name:     c.Name,
		Address:  components.Deref(c.Address),
		Address2: components.Deref(c.Address2),
		City:     components.Deref(c.City),
		State:    components.Deref(c.State),
		Zip:      components.Deref(c.Zip),
		Contact:  components.Deref(c.Contact),
		Phone:    components.Deref(c.Phone),
		Zone:     components.Deref(c.Zone),
	}
}

// searchResultMeta builds the secondary line shown under a customer's name,
// e.g. "C001 · Greer, SC". Empty parts are omitted.
func searchResultMeta(c models.Customer) string {
	var parts []string
	if num := components.Deref(c.Number); num != "" {
		parts = append(parts, num)
	}
	city := components.Deref(c.City)
	state := components.Deref(c.State)
	switch {
	case city != "" && state != "":
		parts = append(parts, city+", "+state)
	case city != "":
		parts = append(parts, city)
	case state != "":
		parts = append(parts, state)
	}
	return strings.Join(parts, " · ")
}
