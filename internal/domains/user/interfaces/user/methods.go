package user

import (
	"github.com/gin-gonic/gin"
)

// Domain methods - Business rules and validations
func (u *User) CanSubmitSolution() bool {
	return u.EmailVerified
}

// User method to convert to public user data
func (u *User) ToPublicUser() gin.H {
	return gin.H{
		"id":       u.ID,
		"username": u.Username,
		// "name":           u.Name,
		// "avatar_url":     u.AvatarURL,
		// "premium":        u.Premium,
		"created_at": u.CreatedAt,
		// "last_seen_at":   u.LastSeenAt,
		"email_verified": u.EmailVerified,
	}
}

// User method to check if account is locked
// func (u *User) IsLocked() bool {
// 	if u.BlockedUntil != nil {
// 		if time.Now().Before(*u.BlockedUntil) {
// 			return true
// 		}
// 	}
// 	return false
// }
