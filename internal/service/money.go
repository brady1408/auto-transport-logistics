package service

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

// parseCents converts a monetary string like "123.45" to integer cents (12345).
// Returns 0 for empty/invalid strings.
func parseCents(s string) int64 {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0
	}
	return int64(math.Round(f * 100))
}

// parseCentsPtr is like parseCents but for *string.
func parseCentsPtr(s *string) int64 {
	if s == nil {
		return 0
	}
	return parseCents(*s)
}

// centsToStr converts integer cents to a monetary string like "123.45".
func centsToStr(cents int64) string {
	if cents < 0 {
		return "-" + centsToStr(-cents)
	}
	return fmt.Sprintf("%d.%02d", cents/100, cents%100)
}
