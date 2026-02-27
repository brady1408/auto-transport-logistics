package migration

import (
	"database/sql"
	"strings"
	"time"
)

// IDMap stores old MSSQL PK → new PostgreSQL PK.
type IDMap map[int]int

func ns(v sql.NullString) *string {
	if !v.Valid {
		return nil
	}
	s := strings.TrimRight(v.String, "\x00")
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	return &s
}

func nns(v sql.NullString) string {
	if !v.Valid {
		return ""
	}
	return strings.TrimSpace(strings.TrimRight(v.String, "\x00"))
}

func nt(v sql.NullTime) *time.Time {
	if !v.Valid || v.Time.IsZero() {
		return nil
	}
	t := v.Time
	return &t
}

func ni(v sql.NullInt64) *int {
	if !v.Valid || v.Int64 == 0 {
		return nil
	}
	i := int(v.Int64)
	return &i
}

func nint(v sql.NullInt64) int {
	if !v.Valid {
		return 0
	}
	return int(v.Int64)
}

func nd(v sql.NullFloat64) *float64 {
	if !v.Valid || v.Float64 == 0 {
		return nil
	}
	return &v.Float64
}

func nb(v sql.NullInt64) bool { return v.Valid && v.Int64 != 0 }

func lookupFK(m IDMap, v sql.NullInt64) *int {
	if !v.Valid || v.Int64 == 0 {
		return nil
	}
	if newID, ok := m[int(v.Int64)]; ok {
		return &newID
	}
	return nil
}

func tableExists(src *sql.DB, table string) bool {
	var count int
	err := src.QueryRow("SELECT COUNT(*) FROM INFORMATION_SCHEMA.TABLES WHERE TABLE_NAME = @p1", table).Scan(&count)
	return err == nil && count > 0
}
