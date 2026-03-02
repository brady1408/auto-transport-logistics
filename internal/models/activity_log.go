package models

import "time"

// ActivityLog represents a single HTTP request tracked in the activity_log table.
type ActivityLog struct {
	ID         int64
	UserID     *int
	Username   *string
	CompanyID  *int
	Method     string
	Path       string
	StatusCode int
	DurationMS int
	IPAddress  *string
	UserAgent  *string
	CreatedAt  time.Time
}

// ActivityFilter holds query parameters for listing activity logs.
type ActivityFilter struct {
	UserID    int
	CompanyID int
	Method    string
	Path      string
	DateFrom  string
	DateTo    string
	Page      int
	PageSize  int
}

// ActivityListResult is returned from ActivityStore.List.
type ActivityListResult struct {
	Items      []ActivityLog
	TotalCount int
	Page       int
	PageSize   int
}

// PathCount represents a URL path and its request count.
type PathCount struct {
	Path  string
	Count int
}

// UserActivity represents a user and their recent request count.
type UserActivity struct {
	UserID   int
	Username string
	Count    int
}

// HourlyCount represents request count for a given hour (0–23).
type HourlyCount struct {
	Hour  int
	Count int
}

// ActivityStats holds aggregated statistics for the dashboard.
type ActivityStats struct {
	TotalRequests  int
	UniqueUsers    int
	TopPaths       []PathCount
	ActiveUsers    []UserActivity
	RecentLogins   []ActivityLog
	HourlyRequests []HourlyCount
}
