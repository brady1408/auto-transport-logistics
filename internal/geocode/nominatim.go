package geocode

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type nominatimResult struct {
	Lat string `json:"lat"`
	Lon string `json:"lon"`
}

// Geocode takes address components and returns lat/lng via OpenStreetMap Nominatim.
// Returns (nil, nil, nil) if no result found.
func Geocode(ctx context.Context, name, city, state, zip string) (*float64, *float64, error) {
	parts := []string{}
	if name != "" {
		parts = append(parts, name)
	}
	if city != "" {
		parts = append(parts, city)
	}
	if state != "" {
		parts = append(parts, state)
	}
	if zip != "" {
		parts = append(parts, zip)
	}
	if len(parts) == 0 {
		return nil, nil, nil
	}

	q := strings.Join(parts, ", ")
	u := fmt.Sprintf("https://nominatim.openstreetmap.org/search?%s",
		url.Values{
			"q":            {q},
			"format":       {"json"},
			"limit":        {"1"},
			"countrycodes": {"us"},
		}.Encode(),
	)

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", u, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("User-Agent", "ATLinks/1.0")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, nil, fmt.Errorf("nominatim request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, nil, fmt.Errorf("nominatim status %d", resp.StatusCode)
	}

	var results []nominatimResult
	if err := json.NewDecoder(resp.Body).Decode(&results); err != nil {
		return nil, nil, fmt.Errorf("decode response: %w", err)
	}
	if len(results) == 0 {
		return nil, nil, nil
	}

	var lat, lng float64
	if _, err := fmt.Sscanf(results[0].Lat, "%f", &lat); err != nil {
		return nil, nil, fmt.Errorf("parse lat: %w", err)
	}
	if _, err := fmt.Sscanf(results[0].Lon, "%f", &lng); err != nil {
		return nil, nil, fmt.Errorf("parse lng: %w", err)
	}

	return &lat, &lng, nil
}
