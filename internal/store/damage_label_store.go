package store

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// DamageLabelMaps holds code→description maps for the three damage lookup tables.
type DamageLabelMaps struct {
	Areas      map[string]string
	Types      map[string]string
	Severities map[string]string
}

// Area resolves a damage area code to its description, falling back to the raw code if not found.
func (m DamageLabelMaps) Area(code *string) string {
	if code == nil {
		return ""
	}
	if desc, ok := m.Areas[*code]; ok {
		return desc
	}
	return *code
}

// Type resolves a damage type code to its description, falling back to the raw code if not found.
func (m DamageLabelMaps) Type(code *string) string {
	if code == nil {
		return ""
	}
	if desc, ok := m.Types[*code]; ok {
		return desc
	}
	return *code
}

// Severity resolves a damage severity code to its description, falling back to the raw code if not found.
func (m DamageLabelMaps) Severity(code *string) string {
	if code == nil {
		return ""
	}
	if desc, ok := m.Severities[*code]; ok {
		return desc
	}
	return *code
}

// DamageLabelStore fetches human-readable labels for damage area, type, and severity codes.
type DamageLabelStore struct {
	areas      *LookupStore
	types      *LookupStore
	severities *LookupStore
}

func NewDamageLabelStore(pool *pgxpool.Pool) (*DamageLabelStore, error) {
	areas, err := NewLookupStore(pool, "damage_areas")
	if err != nil {
		return nil, err
	}
	types, err := NewLookupStore(pool, "damage_types")
	if err != nil {
		return nil, err
	}
	severities, err := NewLookupStore(pool, "damage_severities")
	if err != nil {
		return nil, err
	}
	return &DamageLabelStore{areas: areas, types: types, severities: severities}, nil
}

func (s *DamageLabelStore) Maps(ctx context.Context) (DamageLabelMaps, error) {
	areas, err := s.areas.CodeMap(ctx)
	if err != nil {
		return DamageLabelMaps{}, fmt.Errorf("damage area labels: %w", err)
	}
	types, err := s.types.CodeMap(ctx)
	if err != nil {
		return DamageLabelMaps{}, fmt.Errorf("damage type labels: %w", err)
	}
	severities, err := s.severities.CodeMap(ctx)
	if err != nil {
		return DamageLabelMaps{}, fmt.Errorf("damage severity labels: %w", err)
	}
	return DamageLabelMaps{Areas: areas, Types: types, Severities: severities}, nil
}
