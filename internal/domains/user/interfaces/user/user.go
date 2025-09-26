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

	// Subscription / Polar fields
	PremiumActive        bool        `json:"premium_active" db:"premium_active"`
	PolarLastSyncedAt    *time.Time  `json:"polar_last_synced_at,omitempty" db:"polar_last_synced_at"`
	PolarCustomerID      *uuid.UUID  `json:"polar_customer_id,omitempty" db:"polar_customer_id"`
	PolarSubscriptionIDs []uuid.UUID `json:"polar_subscription_ids,omitempty" db:"polar_subscription_ids"`

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

	CurrentStreak  int        `json:"current_streak" db:"current_streak"`
	LastActiveDate *time.Time `json:"last_active_date,omitempty" db:"last_active_date"`
}

type DailyStats struct {
	UserID         uuid.UUID `json:"user_id" db:"user_id"`
	Date           time.Time `json:"date" db:"date"`
	XPEarned       int64     `json:"xp_earned" db:"xp_earned"`
	ItemsCompleted int       `json:"items_completed" db:"items_completed"`
}
