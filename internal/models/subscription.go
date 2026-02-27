package models

import "time"

// Tier constants for subscription tiers.
type Tier = string

const (
	TierBasic      Tier = "basic"
	TierPro        Tier = "pro"
	TierEnterprise Tier = "enterprise"
)

// Status constants for subscription billing status.
const (
	StatusActive    = "active"
	StatusSuspended = "suspended"
)

// Feature constants for gated features.
type Feature string

const (
	FeatureDispatch   Feature = "dispatch"
	FeatureAccounting Feature = "accounting"
	FeatureReports    Feature = "reports"
	FeatureLoadboard  Feature = "loadboard"
	FeatureEDI        Feature = "edi"
	FeatureQBO        Feature = "quickbooks"
)

// TierFeatures maps each tier to its included features (cumulative).
var TierFeatures = map[Tier][]Feature{
	TierBasic: {
		FeatureDispatch,
		FeatureAccounting,
		FeatureReports,
		FeatureQBO,
	},
	TierPro: {
		FeatureDispatch,
		FeatureAccounting,
		FeatureReports,
		FeatureLoadboard,
		FeatureQBO,
	},
	TierEnterprise: {
		FeatureDispatch,
		FeatureAccounting,
		FeatureReports,
		FeatureLoadboard,
		FeatureEDI,
		FeatureQBO,
	},
}

// Subscription holds a company's plan information.
type Subscription struct {
	ID              int
	CompanyID       int
	Tier            Tier
	Status          string
	AddonEDI        bool
	EDIMonthlyLimit *int
	ExternalID      *string
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// FeatureSet is a set of enabled features for a company.
type FeatureSet map[Feature]bool

// Has returns true if the feature is enabled.
func (fs FeatureSet) Has(f Feature) bool {
	return fs[f]
}

// BuildFeatureSet returns the FeatureSet for a subscription.
// If sub is nil, returns Basic features only.
func BuildFeatureSet(sub *Subscription) FeatureSet {
	fs := make(FeatureSet)
	tier := TierBasic
	if sub != nil {
		tier = sub.Tier
	}
	for _, f := range TierFeatures[tier] {
		fs[f] = true
	}
	// EDI add-on overrides (can grant EDI even on Pro)
	if sub != nil && sub.AddonEDI {
		fs[FeatureEDI] = true
	}
	return fs
}

// ValidTier returns true if the tier string is a known tier.
func ValidTier(t string) bool {
	return t == TierBasic || t == TierPro || t == TierEnterprise
}
