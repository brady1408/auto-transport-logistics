package models

import "time"

type NotificationItem struct {
	Type        string    // "loadboard_message", "feedback_comment"
	EntityID    int       // claim_id or feedback_id
	Title       string    // "Message on LD-001234" or "Comment on Bug Report"
	Description string    // truncated message body
	Author      string    // sender username
	URL         string    // direct link
	CreatedAt   time.Time
	IsNew       bool      // created after notifications_last_checked_at
}
