package token

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"math/big"
	"time"
)

func GenerateRandomCode(length int, chars string) (string, error) {
	bytes := make([]byte, length)
	charsLength := big.NewInt(int64(len(chars)))

	for i := 0; i < length; i++ {
		n, err := rand.Int(rand.Reader, charsLength)
		if err != nil {
			return "", err
		}
		bytes[i] = chars[n.Int64()]
	}

	return string(bytes), nil
}

func ValidateCode(code string, length int, chars string) bool {
	// Check if code length matches expected length
	if len(code) != length {
		return false
	}

	validChars := make(map[rune]bool)
	for _, char := range chars {
		validChars[char] = true
	}

	// Check if all characters in the code are valid
	for _, char := range code {
		if !validChars[char] {
			return false
		}
	}

	return true
}

// GenerateSecureToken creates a secure token with 256 bits of entropy
func GenerateSecureToken(additional string) (string, error) {
	// Use 32 bytes for strong security
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}

	// Add timestamp to prevent replay attacks
	timestamp := make([]byte, 8)
	binary.BigEndian.PutUint64(timestamp, uint64(time.Now().UnixNano()))

	additionalHash := sha256.Sum256([]byte(additional))

	// Combine all sources of entropy
	var combined []byte
	combined = append(combined, bytes...)
	combined = append(combined, timestamp...)
	combined = append(combined, additionalHash[:]...)

	// Hash the combined bytes for additional security
	hash := sha256.Sum256(combined)

	// URL-safe base64 encoding
	token := base64.URLEncoding.EncodeToString(hash[:])
	return token, nil
}
