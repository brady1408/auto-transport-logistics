package models

import "time"

type ApiKey struct {
	ID          int
	KeyHash     string
	UserID      int
	Label       string
	Active      bool
	CreatedAt   time.Time
	LastUsedAt  *time.Time
	// Joined from users table
	Username    string
	CompanyID   int
}
