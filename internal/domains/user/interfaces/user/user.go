package user

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	// Core Identity
	ID       uuid.UUID `json:"id" db:"id"`
	Email    string    `json:"email" db:"email"`
	Username string    `json:"username" db:"username"`
	Role     string    `json:"role" db:"role"`

	// Authentication
	EncryptedPassword string     `json:"encrypted_password,omitempty" db:"encrypted_password"`
	EmailVerified     bool       `json:"email_verified" db:"email_verified"`
	EmailVerifiedAt   *time.Time `json:"email_verified_at,omitempty" db:"email_verified_at"`

	// Timestamps
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

type Stats struct {
	UserID         uuid.UUID  `json:"user_id" db:"user_id"`
	TotalXP        int64      `json:"total_xp" db:"total_xp"`
	CompletedItems int        `json:"completed_items" db:"completed_items"`
	LongestStreak  int        `json:"longest_streak" db:"longest_streak"`
	LastActiveAt   *time.Time `json:"last_active_at,omitempty" db:"last_active_at"`
}

type DailyStats struct {
	UserID         uuid.UUID `json:"user_id" db:"user_id"`
	Date           time.Time `json:"date" db:"date"`
	XPEarned       int64     `json:"xp_earned" db:"xp_earned"`
	ItemsCompleted int       `json:"items_completed" db:"items_completed"`
}
