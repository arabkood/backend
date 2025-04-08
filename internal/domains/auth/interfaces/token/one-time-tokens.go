package token

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type OneTimeTokenType string

const (
	TokenTypeEmailConfirmation OneTimeTokenType = "email_confirmation"
	TokenTypeEmailChange       OneTimeTokenType = "email_change"
	TokenTypePasswordChange    OneTimeTokenType = "password_change"
	TokenTypePasswordRecovery  OneTimeTokenType = "password_recovery"
)

type OneTimeToken struct {
	UserID    uuid.UUID        `json:"user_id" db:"user_id"`
	Type      OneTimeTokenType `json:"type" db:"type"`
	Token     string           `json:"token" db:"token"`
	UpdatedAt time.Time        `json:"updated_at" db:"updated_at"`
	ExpiresAt time.Time        `json:"expires_at" db:"expires_at"`
	Metadata  json.RawMessage  `json:"metadata,omitempty" db:"metadata"`
}
