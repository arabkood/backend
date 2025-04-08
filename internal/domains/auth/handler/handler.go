package handler

import (
	"net/url"
	"strings"
	"time"

	appConfig "github.com/arabkood/backend/config"
	domainToken "github.com/arabkood/backend/internal/domains/auth/interfaces/token"
	emailService "github.com/arabkood/backend/internal/email"
	appLogger "github.com/arabkood/backend/pkg/logger"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AuthHandler struct {
	config   *appConfig.Config
	db       *pgxpool.Pool
	emailSvc *emailService.ProductionEmailService
	logger   *appLogger.Logger
}

func NewAuthHandler(config *appConfig.Config, db *pgxpool.Pool, emailSvc *emailService.ProductionEmailService, logger *appLogger.Logger) *AuthHandler {
	return &AuthHandler{
		config,
		db,
		emailSvc,
		logger,
	}
}

func normalizeDomain(rawURL string) (string, error) {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}
	domain := strings.TrimSuffix(strings.ToLower(parsedURL.Hostname()), ".")
	return domain, nil
}

func (h *AuthHandler) setCookie(c *gin.Context, name, value string, maxAge int, path string, secure, httpOnly bool) {
	if h.config.App.Environment == appConfig.AppEnvDev {
		c.SetCookie(
			name,
			value,
			maxAge,
			path,
			"",
			secure,
			httpOnly,
		)
		return
	}
	for _, domain := range h.config.Auth.CookiesDomains {
		c.SetCookie(
			name,
			value,
			maxAge,
			path,
			domain,
			secure,
			httpOnly,
		)
	}
}

func (h *AuthHandler) setAuthCookies(c *gin.Context, sessionToken *domainToken.SessionToken) {
	h.setCookie(
		c,
		h.config.Auth.SessionCookieName,
		sessionToken.Token,
		h.config.Auth.SessionTokenExpiryHours*int(time.Hour.Seconds()),
		"/",
		// TODO: in production "secure":   h.config.App.Environment != appConfig.AppEnvDev,
		false,
		true,
	)
	h.logger.Debug().Any("cookie auth", map[string]interface{}{
		"name":   h.config.Auth.SessionCookieName,
		"value":  sessionToken.Token,
		"maxAge": h.config.Auth.SessionTokenExpiryHours * int(time.Hour.Seconds()),
		"path":   "/",
		// TODO: in production "secure":   h.config.App.Environment != appConfig.AppEnvDev,
		"secure":   false,
		"httpOnly": true,
	}).Msg("Session token set")
}

func (h *AuthHandler) unsetAuthCookies(c *gin.Context) {
	h.setCookie(
		c,
		h.config.Auth.SessionCookieName,
		"",
		-1,
		"/",
		h.config.App.Environment != appConfig.AppEnvDev,
		true,
	)
}
