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

// PlanType mirrors the auth.plan_type enum.
type PlanType string

const (
	PlanFree    PlanType = "free"
	PlanPro     PlanType = "pro"
	PlanPastDue PlanType = "past_due"
)

// PlanInterval mirrors the auth.plan_interval enum.
type PlanInterval string

const (
	IntervalMonthly PlanInterval = "monthly"
	IntervalYearly  PlanInterval = "yearly"
)

// Subscription maps the auth.user_subscriptions table (Stripe-backed).
type Subscription struct {
	UserID               uuid.UUID     `json:"user_id" db:"user_id"`
	StripeCustomerID     string        `json:"stripe_customer_id" db:"stripe_customer_id"`
	StripeSubscriptionID *string       `json:"stripe_subscription_id,omitempty" db:"stripe_subscription_id"`
	Plan                 PlanType      `json:"plan" db:"plan"`
	PlanInterval         *PlanInterval `json:"plan_interval,omitempty" db:"plan_interval"`
	ProUntil             *time.Time    `json:"pro_until,omitempty" db:"pro_until"`
	CancelAtPeriodEnd    bool          `json:"cancel_at_period_end" db:"cancel_at_period_end"`
	CreatedAt            time.Time     `json:"created_at" db:"created_at"`
	UpdatedAt            time.Time     `json:"updated_at" db:"updated_at"`
}

// IsPro reports whether the subscription currently grants active pro access.
func (s *Subscription) IsPro() bool {
	if s.Plan != PlanPro {
		return false
	}
	if s.ProUntil != nil && time.Now().After(*s.ProUntil) {
		return false
	}
	return true
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
