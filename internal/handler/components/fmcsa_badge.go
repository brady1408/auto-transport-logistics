package components

import (
	"strings"

	"github.com/brady1408/auto-transport-logistics/internal/models"
)

// fmcsaNumberMatchesSaved reports whether the number verified against FMCSA
// matches the company's saved DOT or MC number, comparing digits only.
func fmcsaNumberMatchesSaved(c models.Company) bool {
	verified := digitsOnly(Deref(c.FMCSAVerifiedNumber))
	if verified == "" {
		return false
	}
	if d := digitsOnly(Deref(c.DOTNumber)); d != "" && d == verified {
		return true
	}
	if m := digitsOnly(Deref(c.MCNumber)); m != "" && m == verified {
		return true
	}
	return false
}

func fmcsaBadgeTitle(c models.Company) string {
	parts := make([]string, 0, 2)
	if n := Deref(c.FMCSAVerifiedNumber); n != "" {
		parts = append(parts, "Checked "+n)
	}
	if s := Deref(c.FMCSAStatusSummary); s != "" {
		parts = append(parts, s)
	}
	return strings.Join(parts, " · ")
}

func digitsOnly(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	return b.String()
}
