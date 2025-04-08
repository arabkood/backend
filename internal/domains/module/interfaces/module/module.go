package module

import (
	"time"

	"github.com/google/uuid"
)

type Module struct {
	// Core Information
	ID          uuid.UUID `json:"id" db:"id"`
	Slug        string    `json:"slug" db:"slug"`
	TrackID     uuid.UUID `json:"track_id" db:"track_id"`
	Title       string    `json:"title" db:"title"`
	Description string    `json:"description,omitempty" db:"description"`

	// Order and Structure
	OrderNumber int       `json:"order_number" db:"order_number"`
	SectionID   uuid.UUID `json:"section_id" db:"section_id"`

	// Timestamps
	CreatedAt time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt time.Time  `json:"updated_at" db:"updated_at"`
	DeletedAt *time.Time `json:"deleted_at,omitempty" db:"deleted_at"` // Nullable for soft deletion

	// Rewards and Metrics
	XPReward         int `json:"xp_reward" db:"xp_reward"`
	EstimatedMinutes int `json:"estimated_minutes,omitempty" db:"estimated_minutes"`

	// Attributes
	Difficulty   string      `json:"difficulty" db:"difficulty"`
	Dependencies []uuid.UUID `json:"dependencies,omitempty" db:"dependencies"` // Array of UUIDs
	PremiumOnly  bool        `json:"premium_only" db:"premium_only"`

	// Type and Source
	Type    string   `json:"type" db:"type"`
	Source  string   `json:"source" db:"source"`
	PreArgs []string `json:"pre_args" db:"pre_args"`
}
