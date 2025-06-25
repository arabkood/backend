package track

import (
	"time"

	"github.com/google/uuid"
)

type Track struct {
	// Core Information
	ID      uuid.UUID `json:"id" db:"id"`
	TopicID uuid.UUID `json:"topic_id" db:"topic_id"`
	Title   string    `json:"title" db:"title"`
	Slug    string    `json:"slug" db:"slug"`
	Blurb   string    `json:"blurb,omitempty" db:"blurb"`
	Logo    string    `json:"logo,omitempty" db:"logo"`
	Hash    string    `json:"hash,omitempty" db:"hash"`

	// Attributes
	PremiumOnly bool `json:"premium_only" db:"premium_only"`

	// Timestamps
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}
