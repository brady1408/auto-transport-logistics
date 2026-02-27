package models

import "time"

type QBOConnection struct {
	ID           int
	CompanyID    int
	RealmID      string
	AccessToken  string
	RefreshToken string
	TokenExpiry  time.Time
	ConnectedBy  string
	ConnectedAt  time.Time
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type QBOSyncLog struct {
	ID           int64
	CompanyID    int
	EntityType   string // "customer", "invoice", "payment"
	EntityID     int
	QBOID        *string
	Action       string // "create", "update", "void"
	Status       string // "success", "failed"
	ErrorMessage *string
	AttemptedAt  time.Time
	CompletedAt  *time.Time
}
