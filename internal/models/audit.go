package models

import "time"

type AuditEntry struct {
	ID        int               `json:"id"`
	CompanyID *int              `json:"company_id,omitempty"`
	TableName string            `json:"table_name"`
	RecordID  int               `json:"record_id"`
	Action    string            `json:"action"`
	OldValues map[string]any    `json:"old_values,omitempty"`
	NewValues map[string]any    `json:"new_values,omitempty"`
	UserID    *int              `json:"user_id,omitempty"`
	Username  string            `json:"username"`
	IPAddress string            `json:"ip_address"`
	CreatedAt time.Time         `json:"created_at"`
}
