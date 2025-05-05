package middlewares

import (
	"crypto/subtle"
	"fmt"
	"net"
	"net/http"

	"github.com/arabkood/backend/internal/server/interfaces/server"
	"github.com/gin-gonic/gin"
)

func InternalOnly(srv *server.Server) gin.HandlerFunc {
	return func(c *gin.Context) {
		clientIP := c.ClientIP()

		// Validate client IP
		ip := net.ParseIP(clientIP)
		// if ip == nil {
		// 	fmt.Println("no ip")
		// 	c.AbortWithStatus(http.StatusForbidden)
		// 	return
		// }
		//
		// // Retrieve allowed IPs/CIDRs from server configuration
		// allowedIPs := srv.Config.Server.InternalAllowList
		//
		// // Check against allow list
		// allowed := false
		// for _, entry := range allowedIPs {
		// 	// Try CIDR first
		// 	_, cidr, err := net.ParseCIDR(entry)
		// 	if err == nil {
		// 		if cidr.Contains(ip) {
		// 			allowed = true
		// 			break
		// 		}
		// 		continue
		// 	}
		//
		// 	// Try single IP
		// 	allowedIP := net.ParseIP(entry)
		// 	if allowedIP != nil && allowedIP.Equal(ip) {
		// 		allowed = true
		// 		break
		// 	}
		// }
		//
		// if !allowed {
		// 	fmt.Println("not allowed cidr")
		// 	c.AbortWithStatus(http.StatusForbidden)
		// 	return
		// }

		// Second check - Secret verification
		providedSecret := c.GetHeader("X-Internal-Secret")
		if providedSecret == "" {
			fmt.Println("no secret", ip)
			c.AbortWithStatus(http.StatusForbidden)
			return
		}

		// Use constant time comparison to prevent timing attacks
		expectedSecret := []byte(srv.Config.Server.InternalAuthSecret)
		providedBytes := []byte(providedSecret)
		if subtle.ConstantTimeCompare(expectedSecret, providedBytes) != 1 {
			fmt.Println("wrong secret", ip)
			c.AbortWithStatus(http.StatusForbidden)
			return
		}

		c.Next()
	}
}
