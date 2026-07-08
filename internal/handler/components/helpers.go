package components

import (
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/brady1408/auto-transport-logistics/internal/auth"
	"github.com/brady1408/auto-transport-logistics/internal/models"
)

// JSONAttr marshals v to a compact JSON string for embedding in an HTML
// attribute. templ HTML-escapes attribute values, so the resulting quotes are
// rendered as &#34; and read back correctly by JSON.parse in the browser.
// On marshal error it returns "{}" rather than failing the render.
func JSONAttr(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		return "{}"
	}
	return string(b)
}

// Brand holds white-label display values resolved from the incoming host.
type Brand struct {
	Name             string // Short product name, e.g. "ATLinks" or "Atlas"
	Tagline          string // Subtitle shown on login page
	FaviconFile      string // Path to favicon SVG
	OpenRegistration bool   // If true, no invite code required to register
}

// BrandFromHost resolves the Brand for a given Host header value.
func BrandFromHost(host string) Brand {
	// Strip port if present (e.g. "localhost:8080" → "localhost")
	if i := strings.LastIndex(host, ":"); i >= 0 {
		host = host[:i]
	}
	if host == "atlascloud.app" {
		return Brand{
			Name:             "Atlas",
			Tagline:          "Auto Transport Logistics Administration System",
			FaviconFile:      "/static/favicon-ac.svg",
			OpenRegistration: true,
		}
	}
	return Brand{
		Name:        "ATLinks",
		Tagline:     "Vehicle Transport Management",
		FaviconFile: "/static/favicon.svg",
	}
}

// PageContext holds data available to every page layout.
type PageContext struct {
	User        *auth.ContextUser
	CompanyName string
	Flash       string
	CSRFToken   string
	Features    models.FeatureSet
	Suspended   bool
	Brand       Brand
}

// PaginationData holds computed values for pagination controls.
type PaginationData struct {
	Page       int
	PageSize   int
	TotalCount int
	TotalPages int
	BaseURL    string // e.g. "/global/customers"
	TargetID   string // e.g. "#customer-table"
}

func NewPaginationData(page, pageSize, totalCount int, baseURL, targetID string) PaginationData {
	tp := 0
	if pageSize > 0 {
		tp = int(math.Ceil(float64(totalCount) / float64(pageSize)))
	}
	return PaginationData{
		Page:       page,
		PageSize:   pageSize,
		TotalCount: totalCount,
		TotalPages: tp,
		BaseURL:    baseURL,
		TargetID:   targetID,
	}
}

// ShowingStart returns 1-based index of first item on page.
func (p PaginationData) ShowingStart() int {
	return (p.Page-1)*p.PageSize + 1
}

// ShowingEnd returns 1-based index of last item on page.
func (p PaginationData) ShowingEnd() int {
	end := p.Page * p.PageSize
	if end > p.TotalCount {
		return p.TotalCount
	}
	return end
}

// Pages returns a slice [1..TotalPages] for iteration.
func (p PaginationData) Pages() []int {
	s := make([]int, p.TotalPages)
	for i := range s {
		s[i] = i + 1
	}
	return s
}

// --- Template helper functions (exported for use in .templ files) ---

// Deref returns the string value of a *string, or "" if nil.
func Deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// DerefInt returns the int value of a *int, or 0 if nil.
func DerefInt(i *int) int {
	if i == nil {
		return 0
	}
	return *i
}

// Money formats a stored numeric money string (e.g. "780.0000") as a display
// amount with thousands separators and two decimals, e.g. "780.00" or
// "1,234.50". Nil or empty input yields ""; non-numeric input is returned
// unchanged so unexpected values stay visible.
func Money(s *string) string {
	if s == nil {
		return ""
	}
	raw := strings.TrimSpace(*s)
	if raw == "" {
		return ""
	}
	clean := strings.ReplaceAll(strings.TrimPrefix(raw, "$"), ",", "")
	f, err := strconv.ParseFloat(clean, 64)
	if err != nil {
		return raw
	}
	neg := f < 0
	if neg {
		f = -f
	}
	cents := int64(math.Round(f * 100))
	dollars := cents / 100
	frac := cents % 100
	intStr := strconv.FormatInt(dollars, 10)
	var b strings.Builder
	for i, c := range intStr {
		if i > 0 && (len(intStr)-i)%3 == 0 {
			b.WriteByte(',')
		}
		b.WriteRune(c)
	}
	sign := ""
	if neg {
		sign = "-"
	}
	return sign + b.String() + fmt.Sprintf(".%02d", frac)
}

// FormatPhone formats a 10-digit phone string as (XXX) XXX-XXXX.
func FormatPhone(s *string) string {
	if s == nil {
		return ""
	}
	d := *s
	if len(d) == 10 {
		return "(" + d[:3] + ") " + d[3:6] + "-" + d[6:]
	}
	return d
}

// AssetVersion appends the build version query string for cache busting.
func AssetVersion(path string) string {
	return path + "?v=" + buildVersion
}

// buildVersion is set from handler.BuildVersion during init.
var buildVersion string

// SetBuildVersion is called from main.go or handler init to propagate the version.
func SetBuildVersion(v string) {
	buildVersion = v
}

// TotalPages calculates the number of pages.
func TotalPages(total, pageSize int) int {
	if pageSize == 0 {
		return 0
	}
	return int(math.Ceil(float64(total) / float64(pageSize)))
}

// Seq returns a slice of ints from start to end (inclusive).
func Seq(start, end int) []int {
	var s []int
	for i := start; i <= end; i++ {
		s = append(s, i)
	}
	return s
}

// FormatInt converts an int to string for use in templ attributes.
func FormatInt(i int) string {
	return fmt.Sprintf("%d", i)
}

// FormatDate formats a *time.Time as YYYY-MM-DD, or "" if nil.
func FormatDate(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.Format("2006-01-02")
}

// FormatBytes formats a byte count as a human-readable string.
func FormatBytes(b int64) string {
	switch {
	case b >= 1<<30:
		return fmt.Sprintf("%.2f GB", float64(b)/float64(1<<30))
	case b >= 1<<20:
		return fmt.Sprintf("%.1f MB", float64(b)/float64(1<<20))
	case b >= 1<<10:
		return fmt.Sprintf("%.0f KB", float64(b)/float64(1<<10))
	default:
		return fmt.Sprintf("%d B", b)
	}
}
