package itemInterface

import (
	"time"

	"github.com/google/uuid"
)

type Submission struct {
	// Core Identifiers
	ID     uuid.UUID `json:"id" db:"id"`
	UserID uuid.UUID `json:"user_id" db:"user_id"`
	ItemID uuid.UUID `json:"item_id" db:"item_id"`

	// Submission State
	Status   string `json:"status" db:"status"` // 'wait', 'pass', 'fail', 'error'
	XPReward int    `json:"xp_reward" db:"xp_reward"`
	Attempts int    `json:"attempts" db:"attempts"`

	// Timestamps
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`

	// JSON Fields
	Metadata []byte `json:"metadata,omitempty" db:"metadata"`
	Data     []byte `json:"data,omitempty" db:"data"`
	Results  []byte `json:"results,omitempty" db:"results"`
}
