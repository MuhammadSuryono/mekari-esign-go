package config

import (
	"strings"
	"time"

	"github.com/spf13/viper"
)

// AuthType constants
const (
	AuthTypeOAuth2 = "oauth2"
	AuthTypeHMAC   = "hmac"
)

type Config struct {
	App      AppConfig      `mapstructure:"app"`
	Mekari   MekariConfig   `mapstructure:"mekari"`
	Database DatabaseConfig `mapstructure:"database"`
	Redis    RedisConfig    `mapstructure:"redis"`
	OAuth    OAuthConfig    `mapstructure:"oauth"`
	Document DocumentConfig `mapstructure:"document"`
	Logging  LoggingConfig  `mapstructure:"logging"`
	NAV      NAVConfig      `mapstructure:"nav"`
}

type AppConfig struct {
	Name    string `mapstructure:"name"`
	Port    int    `mapstructure:"port"`
	Env     string `mapstructure:"env"`
	BaseURL string `mapstructure:"base_url"`
}

type MekariConfig struct {
	AuthType   string            `mapstructure:"auth_type"` // "oauth2" or "hmac"
	BaseURL    string            `mapstructure:"base_url"`
	SsoBaseURL string            `mapstructure:"sso_base_url"`
	AuthURL    string            `mapstructure:"auth_url"`
	Timeout    time.Duration     `mapstructure:"timeout"`
	OAuth2     OAuth2Credentials `mapstructure:"oauth2"` // OAuth2 credentials
	HMAC       HMACCredentials   `mapstructure:"hmac"`   // HMAC credentials
}

// OAuth2Credentials stores OAuth2 client credentials
type OAuth2Credentials struct {
	ClientID     string `mapstructure:"client_id"`
	ClientSecret string `mapstructure:"client_secret"`
}

// HMACCredentials stores HMAC client credentials
type HMACCredentials struct {
	ClientID     string `mapstructure:"client_id"`
	ClientSecret string `mapstructure:"client_secret"`
}

// GetClientID returns the client ID based on auth type
func (m *MekariConfig) GetClientID() string {
	if m.AuthType == AuthTypeHMAC {
		return m.HMAC.ClientID
	}
	return m.OAuth2.ClientID
}

// GetClientSecret returns the client secret based on auth type
func (m *MekariConfig) GetClientSecret() string {
	if m.AuthType == AuthTypeHMAC {
		return m.HMAC.ClientSecret
	}
	return m.OAuth2.ClientSecret
}

// IsOAuth2 returns true if auth type is OAuth2
func (m *MekariConfig) IsOAuth2() bool {
	return m.AuthType == AuthTypeOAuth2 || m.AuthType == ""
}

// IsHMAC returns true if auth type is HMAC
func (m *MekariConfig) IsHMAC() bool {
	return m.AuthType == AuthTypeHMAC
}

type DatabaseConfig struct {
	Driver   string `mapstructure:"driver"`
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
	DBName   string `mapstructure:"dbname"`
	SSLMode  string `mapstructure:"sslmode"`
}

type RedisConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
}

type OAuthConfig struct {
	RefreshTokenAgeDays int `mapstructure:"refresh_token_age_days"`
}

type DocumentConfig struct {
	BasePath       string `mapstructure:"base_path"`       // Base path for documents
	ReadyFolder    string `mapstructure:"ready_folder"`    // Folder for documents ready to send
	ProgressFolder string `mapstructure:"progress_folder"` // Folder for documents in progress
	FinishFolder   string `mapstructure:"finish_folder"`   // Folder for completed documents
	FilePrefix     string `mapstructure:"file_prefix"`     // Optional prefix for files
	FileExtension  string `mapstructure:"file_extension"`  // File extension (default: .pdf)
}

type LoggingConfig struct {
	Level  string `mapstructure:"level"`
	Format string `mapstructure:"format"`
}

type NAVConfig struct {
	Enabled  bool   `mapstructure:"enabled"`
	BaseURL  string `mapstructure:"base_url"`
	Company  string `mapstructure:"company"`
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
	Timeout  int    `mapstructure:"timeout"`
}

func NewConfig() (*Config, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("./config")

	// Enable environment variable override
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	if err := viper.ReadInConfig(); err != nil {
		return nil, err
	}

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, err
	}

	// Convert timeout to duration
	cfg.Mekari.Timeout = cfg.Mekari.Timeout * time.Second

	// Default auth type to oauth2 if not specified
	if cfg.Mekari.AuthType == "" {
		cfg.Mekari.AuthType = AuthTypeOAuth2
	}

	return &cfg, nil
}

func (c *Config) IsDevelopment() bool {
	return c.App.Env == "development"
}

func (c *Config) IsProduction() bool {
	return c.App.Env == "production"
}
