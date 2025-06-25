package itemInterface

import (
	"time"

	"github.com/google/uuid"
)

type Item struct {
	// Core Information
	ID        uuid.UUID `json:"id" db:"id"`
	ModuleID  uuid.UUID `json:"module_id" db:"module_id"`
	Slug      string    `json:"slug" db:"slug"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
	Hash      string    `json:"hash,omitempty" db:"hash"`
	// Attributes
	Type        *string `json:"type,omitempty" db:"type"`
	Position    *int    `json:"position,omitempty" db:"position"`
	Title       string  `json:"title" db:"title"`
	Blurb       *string `json:"blurb,omitempty" db:"blurb"`
	Difficulty  *string `json:"difficulty,omitempty" db:"difficulty"`
	PremiumOnly bool    `json:"premium_only" db:"premium_only"`
	BaseXP      int     `json:"base_xp" db:"base_xp"`
	S3Path      *string `json:"s3_path,omitempty" db:"s3_path"`
}
