package models

import "time"

type MigrationRun struct {
	ID             int64
	CompanyID      int64
	CompanyName    string // joined
	Status         string // pending | running | complete | failed
	BackupFilename string
	Log            string
	Stats          []MigrationTableStat
	StartedAt      *time.Time
	FinishedAt     *time.Time
	CreatedAt      time.Time
}

type MigrationTableStat struct {
	Table    string `json:"table"`
	Source   int    `json:"source"`
	Inserted int    `json:"inserted"`
	Skipped  int    `json:"skipped"`
}
