package models

import "time"

type Attachment struct {
	ID          int       `json:"id"`
	CompanyID   int       `json:"company_id"`
	Category    string    `json:"category"`
	EntityID    int       `json:"entity_id"`
	Filename    string    `json:"filename"`
	StorageKey  string    `json:"storage_key"`
	ContentType string    `json:"content_type"`
	SizeBytes   int64     `json:"size_bytes"`
	UploadedBy  *int      `json:"uploaded_by,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}
