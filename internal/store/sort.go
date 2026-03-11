package store

import "fmt"

// SortConfig defines allowed sort columns and the default sort for a list query.
type SortConfig struct {
	Allowed    map[string]string // user-facing key → SQL column name
	DefaultCol string            // default user-facing key
	DefaultDir string            // "ASC" or "DESC"
}

// ValidateSort returns a safe (userKey, sqlColumn, direction) triple.
// Invalid input falls back to defaults.
func ValidateSort(cfg SortConfig, sortBy, sortDir string) (string, string, string) {
	col, ok := cfg.Allowed[sortBy]
	if !ok {
		sortBy = cfg.DefaultCol
		col = cfg.Allowed[cfg.DefaultCol]
	}
	if sortDir != "ASC" && sortDir != "DESC" {
		sortDir = cfg.DefaultDir
	}
	return sortBy, col, sortDir
}

// OrderByClause returns an ORDER BY with NULLS LAST and a secondary id DESC for stable pagination.
func OrderByClause(col, dir string) string {
	return fmt.Sprintf("ORDER BY %s %s NULLS LAST, id DESC", col, dir)
}
