package xpRepo

import (
	"time"

	"github.com/google/uuid"
)

type XPEvent struct {
	UserID     uuid.UUID
	XPAmount   int
	SourceType string
	SourceID   uuid.UUID
}

type UserStats struct {
	UserID        uuid.UUID
	TotalXP       int64
	Completed     int
	LongestStreak int
	LastActiveAt  *time.Time
}
