package token

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// SessionToken represents a session token entity
type SessionToken struct {
	// Core fields
	Token  string    `json:"token"`
	UserID uuid.UUID `json:"user_id"`

	// Timestamps
	CreatedAt  time.Time  `json:"created_at"`
	ExpiresAt  time.Time  `json:"expires_at"`
	LastUsedAt *time.Time `json:"last_used_at,omitempty"`
}

// NewSessionToken creates a new session token with the provided parameters
func NewSessionToken(userID uuid.UUID, duration time.Duration) (*SessionToken, error) {
	now := time.Now()
	userIdStr := userID.String()
	if userIdStr == "" {
		return nil, errors.New("invalid user id")
	}
	token, err := GenerateSecureToken(userIdStr)
	if err != nil {
		return nil, err
	}

	return &SessionToken{
		Token:      token,
		UserID:     userID,
		CreatedAt:  now,
		ExpiresAt:  now.Add(duration),
		LastUsedAt: &now,
	}, nil
}
