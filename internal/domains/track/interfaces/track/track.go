package track

import (
	"time"

	"github.com/arabkood/backend/internal/domains/module/interfaces/module"
	"github.com/google/uuid"
)

type Track struct {
	// Core Information
	ID          uuid.UUID `json:"id" db:"id"`
	Title       string    `json:"title" db:"title"`
	Slug        string    `json:"slug" db:"slug"`
	Description string    `json:"description,omitempty" db:"description"`
	Logo        string    `json:"logo,omitempty" db:"logo"`

	// Attributes
	ProgrammingLanguages []string `json:"programming_languages,omitempty" db:"programming_languages"`
	Difficulty           string   `json:"difficulty" db:"difficulty"`
	PremiumOnly          bool     `json:"premium_only" db:"premium_only"`
	Skills               []string `json:"skills,omitempty" db:"skills"`
	Tags                 []string `json:"tags,omitempty" db:"tags"`

	// Metrics
	TotalXP        int `json:"total_xp" db:"total_xp"`
	TotalModules   int `json:"total_modules" db:"total_modules"`
	EstimatedHours int `json:"estimated_hours,omitempty" db:"estimated_hours"`
	Students       int `json:"students" db:"students"`

	// Timestamps
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
	DeletedAt time.Time `json:"deleted_at,omitempty" db:"deleted_at"`

	// Additional Data
	Outcomes     []string `json:"outcomes,omitempty" db:"outcomes"`
	Requirements []string `json:"requirements,omitempty" db:"requirements"`
}

type Section struct {
	// Core Information
	ID          uuid.UUID `json:"id" db:"id"`
	TrackID     uuid.UUID `json:"track_id" db:"track_id"`
	Title       string    `json:"title" db:"title"`
	Description string    `json:"description" db:"description"`
	OrderNumber int       `json:"order_number" db:"order_number"`

	// Timestamps
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
	DeletedAt time.Time `json:"deleted_at" db:"deleted_at"`
}

type SectionWithModules struct {
	Section
	Modules []module.Module `json:"modules"`
}
