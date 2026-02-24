package models

import "time"

type Feedback struct {
	ID           int       `json:"id"`
	CompanyID    int       `json:"company_id"`
	UserID       int       `json:"user_id"`
	Username     string    `json:"username"`
	PageURL      string    `json:"page_url"`
	Category     string    `json:"category"`
	Message      string    `json:"message"`
	Status       string    `json:"status"`
	CompanyName  string    `json:"company_name,omitempty"`
	CommentCount int       `json:"comment_count"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type FeedbackComment struct {
	ID         int       `json:"id"`
	FeedbackID int       `json:"feedback_id"`
	UserID     int       `json:"user_id"`
	Username   string    `json:"username"`
	UserRole   string    `json:"user_role"`
	CompanyID  int       `json:"company_id"`
	Message    string    `json:"message"`
	Internal   bool      `json:"internal"`
	CreatedAt  time.Time `json:"created_at"`
}

type FeedbackFilter struct {
	Status   string
	Category string
	Page     int
	PageSize int
}

type FeedbackListResult struct {
	Items      []Feedback
	TotalCount int
	Page       int
	PageSize   int
}
