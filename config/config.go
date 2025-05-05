package config

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/spf13/viper"
)

const (
	AppEnvDev  = "development"
	AppEnvProd = "production"
)

type Config struct {
	Email    EmailConfig    `validate:"required"`
	App      AppConfig      `validate:"required"`
	Auth     AuthConfig     `validate:"required"`
	Server   ServerConfig   `validate:"required"`
	Database DatabaseConfig `validate:"required"`
	Aws      AWSConfig      `validate:"required"`
	Other    OtherConfig    `validate:"required"`
}

type OtherConfig struct {
	SubmissionsDirectory string `validate:"required,dirpath"`
}

type AppConfig struct {
	Name        string `validate:"required"`
	Environment string `validate:"required,oneof=development production local"`
	BaseURL     string `validate:"required,url"`
	Debug       bool
}

type ServerConfig struct {
	Host               string        `validate:"required"`
	CorsOrigins        []string      `validate:"required,dive,url"`
	Port               int           `validate:"required,min=1,max=65535"`
	ReadTimeout        time.Duration `validate:"required"`
	WriteTimeout       time.Duration `validate:"required"`
	GracefulTimeout    time.Duration `validate:"required"`
	InternalAllowList  []string      `validate:"required"`
	InternalAuthSecret string        `validate:"required"`
}

type DatabaseConfig struct {
	Host     string `validate:"required"`
	UserName string `validate:"required"`
	Password string `validate:"required"`
	Name     string `validate:"required"`
	SSLMode  string `validate:"required,oneof=disable require verify-full prefer verify-ca"`
	Port     int    `validate:"required,min=1,max=65535"`

	// pgx config
	MaxConnLifetime       time.Duration `validate:"required"`
	MaxConnLifetimeJitter time.Duration `validate:"required"`
	MaxConnIdleTime       time.Duration `validate:"required"`
	MaxConns              int32         `validate:"required,min=1"`
	MinConns              int32         `validate:"required,min=0"`
	HealthCheckPeriod     time.Duration `validate:"required"`
}

type EmailConfig struct {
	Provider  string `validate:"required,oneof=sendgrid ses smtp"`
	FromEmail string `validate:"required,email"`
	FromName  string `validate:"required"`
	SMTPHost  string `validate:"required",mapstructure:"smtpHost"`
	SMTPPort  string `validate:"required",mapstructure:"smtpPort"`
	SMTPUser  string `validate:"required",mapstructure:"smtpUser"`
	SMTPPass  string `validate:"required",mapstructure:"smtpPass"`
}

type AWSConfig struct {
	Region              string `validate:"required"`
	SubmissionQueueURL  string `validate:"required,url"`
	CredentialsPath     string `validate:"required"`
	Profile             string `validate:"required"`
	Endpoint            string `validate:"required"`
	ExercisesBucketName string `validate:"required"`
	LogGroupName        string `validate:"required"`
}

type AuthConfig struct {
	SessionCookieName              string   `validate:"required"`
	CookiesDomains                 []string `validate:"required,dive,hostname"`
	SessionTokenExpiryHours        int      `validate:"required,min=1"`
	EmailVerificationExpiryMinutes int      `validate:"required,min=1"`
	EmailVerificationCodeLength    int      `validate:"required,min=4"`
	EmailVerificationCodeChars     string   `validate:"required,min=4"`
	PasswordResetExpiryMinutes     int      `validate:"required,min=1"`
	PasswordResetCodeLength        int      `validate:"required,min=4"`
	PasswordResetCodeChars         string   `validate:"required,min=4"`
}

func (d *DatabaseConfig) GetPostgresURL() string {
	query := url.Values{}
	if d.SSLMode != "" {
		query.Add("sslmode", d.SSLMode)
	}

	connStr := fmt.Sprintf("postgres://%s:%s@%s:%d/postgres",
		url.QueryEscape(d.UserName),
		url.QueryEscape(d.Password),
		url.QueryEscape(d.Host),
		d.Port)

	if len(query) > 0 {
		connStr += "?" + query.Encode()
	}

	return connStr
}

func (d *DatabaseConfig) GetDatabaseURL() string {
	// Create query params for postgres-specific options
	query := url.Values{}
	if d.SSLMode != "" {
		query.Add("sslmode", d.SSLMode)
	}

	// Build the connection string with only postgres params
	connStr := fmt.Sprintf("postgres://%s:%s@%s:%d/%s",
		url.QueryEscape(d.UserName),
		url.QueryEscape(d.Password),
		url.QueryEscape(d.Host),
		d.Port,
		url.QueryEscape(d.Name))

	// Add query parameters if any exist
	if len(query) > 0 {
		connStr += "?" + query.Encode()
	}

	return connStr
}

func NewConfigDefault() (*Config, error) {
	return NewConfig("config")
}

// NewConfig creates and loads the configuration
func NewConfig(configName string) (*Config, error) {
	v := viper.New()

	// Read config file
	v.SetConfigName(configName)        // config file name without extension
	v.SetConfigType("yaml")            // yaml, json, toml
	v.AddConfigPath(".")               // config file path
	v.AddConfigPath("config")          // config file path
	v.AddConfigPath("/etc/arabkood")   // fallback path
	v.AddConfigPath("/opt/go-app")     // fallback path
	v.AddConfigPath("$HOME/.arabkood") // fallback path

	// Read environment variables
	v.SetEnvPrefix("ARABKOOD")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	// Read config file
	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := v.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	validate := validator.New()
	if err := validate.Struct(&config); err != nil {
		return nil, fmt.Errorf("missing required config \n%w", err)
	}

	return &config, nil
}
