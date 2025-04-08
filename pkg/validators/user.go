package validators

import (
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/go-playground/validator/v10"
)

const (
	MinUsernameLength = 3
	MaxUsernameLength = 64
	MinEmailLength    = 3
	MaxEmailLength    = 254
	MinPasswordLength = 8
	MaxPasswordLength = 256
)

// Reserved usernames that shouldn't be allowed
var reservedUsernames = map[string]bool{
	"admin":     true,
	"root":      true,
	"support":   true,
	"arabkood":  true,
	"system":    true,
	"moderator": true,
	"help":      true,
	"info":      true,
	"user":      true,
}

// Common disposable email domains to block
var blockedEmailDomains = map[string]bool{
	"tempmail.com": true,
}

// Username validation
var validateUsername validator.Func = func(fl validator.FieldLevel) bool {
	username := fl.Field().String()

	// Check UTF-8 validity
	if !utf8.ValidString(username) {
		return false
	}

	// Check length
	runeCount := utf8.RuneCountInString(username)
	if runeCount < MinUsernameLength || runeCount > MaxUsernameLength {
		return false
	}

	// Check if starts with invalid char
	firstRune, _ := utf8.DecodeRuneInString(username)
	if firstRune == '_' || firstRune == '-' {
		return false
	}

	// Check if allowed characters
	for _, char := range username {
		isSpecial := char == '_' || char == '-'
		if !unicode.IsLetter(char) && !unicode.IsNumber(char) && !isSpecial {
			return false
		}
	}

	return true
}

// Strict username validation with additional checks
var validateUsernameStrict validator.Func = func(fl validator.FieldLevel) bool {
	username := fl.Field().String()

	// First apply all standard username validations
	if !validateUsername(fl) {
		return false
	}

	lowercaseUsername := strings.ToLower(username)

	// Check for reserved usernames
	if reservedUsernames[lowercaseUsername] {
		return false
	}

	return true
}

// Email validation with comprehensive checks
var validateEmail validator.Func = func(fl validator.FieldLevel) bool {
	email := fl.Field().String()

	// Check UTF-8 validity
	if !utf8.ValidString(email) {
		return false
	}

	// Check overall length
	if len(email) > MaxEmailLength || len(email) < MinEmailLength {
		return false
	}

	// Block control characters and spaces
	for _, char := range email {
		if unicode.IsControl(char) || unicode.IsSpace(char) {
			return false
		}
	}

	// Split email into local and domain parts
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return false
	}

	localPart, domainPart := parts[0], parts[1]

	// Check local part
	if len(localPart) == 0 {
		return false
	}

	// Check domain part
	if len(domainPart) == 0 {
		return false
	}

	// Check for multiple dots and proper domain format
	domainParts := strings.Split(domainPart, ".")
	if len(domainParts) < 2 {
		return false
	}
	if len(domainParts[0]) == 0 || len(domainParts[1]) == 0 {
		return false
	}

	// Check for blocked email domains
	if blockedEmailDomains[strings.ToLower(domainPart)] {
		return false
	}

	return true
}

// Password validation
var validatePassword validator.Func = func(fl validator.FieldLevel) bool {
	password := fl.Field().String()

	// Check UTF-8 validity
	if !utf8.ValidString(password) {
		return false
	}

	// Check length
	if len(password) < MinPasswordLength || len(password) > MaxPasswordLength {
		return false
	}

	// Check for leading/trailing whitespace
	if strings.TrimSpace(password) != password {
		return false
	}

	// Block Control characters
	for _, char := range password {
		if unicode.IsControl(char) {
			return false
		}
	}

	return true

	// Complexity Checks
	// hasLower := false
	// hasUpper := false
	// hasNumber := false
	// hasSpecial := false
	//
	// for _, char := range password {
	// 	switch {
	// 	case unicode.IsControl(char):
	// 		return false
	// 	case unicode.IsLower(char):
	// 		hasLower = true
	// 	case unicode.IsUpper(char):
	// 		hasUpper = true
	// 	case unicode.IsNumber(char):
	// 		hasNumber = true
	// 	case unicode.IsPunct(char) || unicode.IsSymbol(char):
	// 		hasSpecial = true
	// 	}
	// }
	//
	// // Require at least 3 of: lowercase, uppercase, number, special char
	// complexity := 0
	// if hasLower {
	// 	complexity++
	// }
	// if hasUpper {
	// 	complexity++
	// }
	// if hasNumber {
	// 	complexity++
	// }
	// if hasSpecial {
	// 	complexity++
	// }
	//
	// return complexity >= 3
}
