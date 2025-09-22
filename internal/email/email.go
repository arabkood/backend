package email

import (
	"context"
	"embed"
	"fmt"
	"html/template"
	"log"
	"net/smtp"
	"os"
	"path"
	"strings"

	appConfig "github.com/arabkood/backend/config"
)

// Embed templates into the binary
//
//go:embed templates/*.tmpl
var templateFS embed.FS

// EmailService defines methods for sending various types of emails
type EmailService interface {
	SendVerificationEmail(ctx context.Context, email *VerificationEmail) error
	SendPasswordResetEmail(ctx context.Context, email *PasswordResetEmail) error
	SendWelcomeEmail(ctx context.Context, email *WelcomeEmail) error
}

// Email data structures
type VerificationEmail struct {
	To       string
	Username string
	Token    string
}

type PasswordResetEmail struct {
	To       string
	Username string
	Token    string
}

type WelcomeEmail struct {
	To       string
	Username string
}

// ProductionEmailService implements the EmailService interface
type ProductionEmailService struct {
	smtpHost  string
	smtpPort  string
	smtpUser  string
	smtpPass  string
	from      string
	baseURL   string
	templates map[string]*template.Template
	logger    *log.Logger
}

// NewProductionEmailService initializes the email service with templates and SMTP configuration
func NewProductionEmailService(config *appConfig.Config) (*ProductionEmailService, error) {
	logger := log.New(os.Stderr, "[EMAIL] ", log.LstdFlags)

	// Load templates from the embedded file system
	templates, err := loadTemplates()
	if err != nil {
		return nil, fmt.Errorf("failed to load email templates: %w", err)
	}

	return &ProductionEmailService{
		smtpHost:  config.Email.SMTPHost,
		smtpPort:  config.Email.SMTPPort,
		smtpUser:  config.Email.SMTPUser,
		smtpPass:  config.Email.SMTPPass,
		from:      config.Email.FromEmail,
		baseURL:   config.App.BaseURL,
		templates: templates,
		logger:    logger,
	}, nil
}

// loadTemplates loads all HTML templates from the embedded file system
func loadTemplates() (map[string]*template.Template, error) {
	templates := make(map[string]*template.Template)

	// Walk through embedded templates and parse them
	entries, err := templateFS.ReadDir("templates")
	if err != nil {
		return nil, fmt.Errorf("failed to read embedded templates: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".tmpl") {
			name := strings.TrimSuffix(entry.Name(), ".tmpl") // Template name without extension
			filePath := path.Join("templates", entry.Name())

			tpl, err := template.ParseFS(templateFS, filePath)
			if err != nil {
				return nil, fmt.Errorf("failed to parse template %s: %w", filePath, err)
			}
			templates[name] = tpl
		}
	}

	return templates, nil
}

// renderEmail renders an email template with the provided data
func (s *ProductionEmailService) renderEmail(templateName string, data map[string]interface{}) (string, error) {
	tpl, exists := s.templates[templateName]
	if !exists {
		return "", fmt.Errorf("template not found: %s", templateName)
	}

	// Add base URL to data
	data["BaseURL"] = s.baseURL

	var output strings.Builder
	if err := tpl.Execute(&output, data); err != nil {
		return "", fmt.Errorf("error rendering template: %w", err)
	}

	return output.String(), nil
}

// sendEmail sends an email via SMTP with TLS
func (s *ProductionEmailService) sendEmail(to, content, subject string) error {
	auth := smtp.PlainAuth("", s.smtpUser, s.smtpPass, s.smtpHost)

	headers := fmt.Sprintf("From: %s\r\n"+
		"To: %s\r\n"+
		"Subject: %s\r\n"+
		"MIME-Version: 1.0\r\n"+
		"Content-Type: text/html; charset=\"UTF-8\"\r\n\r\n",
		s.from, to, subject)

	message := []byte(headers + content)

	addr := fmt.Sprintf("%s:%s", s.smtpHost, s.smtpPort)

	// Use smtp.SendMail which handles STARTTLS automatically
	return smtp.SendMail(addr, auth, s.from, []string{to}, message)
}

// Email sending functions
func (s *ProductionEmailService) SendVerificationEmail(ctx context.Context, email *VerificationEmail) error {
	data := map[string]interface{}{
		"Username": email.Username,
		"Token":    email.Token,
		"LogoURL":  "https://www.arabkood.com/icon-32x32.png",
	}
	content, err := s.renderEmail("verification", data)
	if err != nil {
		return err
	}

	return s.sendEmail(email.To, content, "تأكيد عنوان البريد الإلكتروني")
}

func (s *ProductionEmailService) SendPasswordResetEmail(ctx context.Context, email *PasswordResetEmail) error {
	data := map[string]interface{}{
		"Username": email.Username,
		"Token":    email.Token,
		"LogoURL":  "https://www.akood.com/icon-32x32.png",
	}
	content, err := s.renderEmail("password_reset", data)
	if err != nil {
		return err
	}

	return s.sendEmail(email.To, content, "password reset")
}

func (s *ProductionEmailService) SendWelcomeEmail(ctx context.Context, email *WelcomeEmail) error {
	data := map[string]interface{}{
		"Username": email.Username,
		"LogoURL":  "https://www.akood.com/icon-32x32.png",
	}
	content, err := s.renderEmail("welcome", data)
	if err != nil {
		return err
	}

	return s.sendEmail(email.To, content, "welcome")
}
