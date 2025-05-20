package exercise

import (
	"time"

	"github.com/google/uuid"
)

type Submission struct {
	// Keys
	ID     uuid.UUID `json:"id" db:"id"`
	UserID uuid.UUID `json:"user_id" db:"user_id"`
	ItemID uuid.UUID `json:"item_id" db:"item_id"`

	// Code Evaluated
	XpReward  int64     `json:"xp_reward" db:"xp_reward"`
	Attempts  uint16    `json:"attempts" db:"attempts"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`

	UserFiles []byte         `json:"user_files" db:"user_files"`
	Args      map[string]any `json:"args" db:"args"`
	Results   map[string]any `json:"results" db:"results"`

	Version *string `json:"version" db:"version"`
}
