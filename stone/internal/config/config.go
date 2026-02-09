package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	Port string `envconfig:"PORT" default:"8080"`
	Env  string `envconfig:"ENV" default:"development"`

	DatabaseURL string `envconfig:"DATABASE_URL" required:"true"`
	RedisURL    string `envconfig:"REDIS_URL" required:"true"`

	JWTSecret                 string        `envconfig:"JWT_SECRET" required:"true"`
	JWTAccessTTL              time.Duration `envconfig:"JWT_ACCESS_TTL" default:"15m"`
	JWTRefreshTTL             time.Duration `envconfig:"JWT_REFRESH_TTL" default:"168h"`
	AuthCookieSecure          bool          `envconfig:"AUTH_COOKIE_SECURE" default:"false"`
	AuthRefreshCookieName     string        `envconfig:"AUTH_REFRESH_COOKIE_NAME" default:"pulse_refresh_token"`
	AuthRefreshCookiePath     string        `envconfig:"AUTH_REFRESH_COOKIE_PATH" default:""`
	AuthRefreshCookieDomain   string        `envconfig:"AUTH_REFRESH_COOKIE_DOMAIN" default:""`
	AuthRefreshCookieSameSite string        `envconfig:"AUTH_REFRESH_COOKIE_SAMESITE" default:"lax"`
	AuthCSRFCookieName        string        `envconfig:"AUTH_CSRF_COOKIE_NAME" default:"pulse_csrf_token"`
	AuthCSRFCookiePath        string        `envconfig:"AUTH_CSRF_COOKIE_PATH" default:"/"`
	AuthCSRFCookieDomain      string        `envconfig:"AUTH_CSRF_COOKIE_DOMAIN" default:""`
	AuthCSRFCookieSameSite    string        `envconfig:"AUTH_CSRF_COOKIE_SAMESITE" default:"lax"`
	AuthCSRFHeaderName        string        `envconfig:"AUTH_CSRF_HEADER_NAME" default:"X-CSRF-Token"`

	APIBasePath      string `envconfig:"API_BASE_PATH" default:"/api/v1"`
	StoragePath      string `envconfig:"STORAGE_PATH" default:"./uploads"`
	StorageBaseURL   string `envconfig:"STORAGE_BASE_URL" default:""`
	UploadPublicPath string `envconfig:"UPLOAD_PUBLIC_PATH" default:"/uploads"`
	StorageMaxSizeMB int    `envconfig:"STORAGE_MAX_SIZE_MB" default:"10"`

	CORSOrigins string `envconfig:"CORS_ORIGINS" default:""`
	WSOrigins   string `envconfig:"WS_ORIGINS" default:""`

	RateLimitRPS   int    `envconfig:"RATE_LIMIT_RPS" default:"100"`
	RateLimitBurst int    `envconfig:"RATE_LIMIT_BURST" default:"200"`
	TrustedProxies string `envconfig:"TRUSTED_PROXIES" default:"127.0.0.1,::1"`

	ReadTimeout     time.Duration `envconfig:"READ_TIMEOUT" default:"15s"`
	WriteTimeout    time.Duration `envconfig:"WRITE_TIMEOUT" default:"15s"`
	IdleTimeout     time.Duration `envconfig:"IDLE_TIMEOUT" default:"60s"`
	ShutdownTimeout time.Duration `envconfig:"SHUTDOWN_TIMEOUT" default:"10s"`
}

func Load() (*Config, error) {
	var cfg Config
	if err := envconfig.Process("", &cfg); err != nil {
		return nil, err
	}

	cfg.Env = strings.ToLower(strings.TrimSpace(cfg.Env))
	if cfg.Env == "" {
		cfg.Env = "development"
	}

	cfg.APIBasePath = normalizeURLPath(cfg.APIBasePath, "/api/v1")
	cfg.UploadPublicPath = normalizeURLPath(cfg.UploadPublicPath, "/uploads")
	if strings.TrimSpace(cfg.StorageBaseURL) == "" {
		cfg.StorageBaseURL = cfg.UploadPublicPath
	} else {
		cfg.StorageBaseURL = strings.TrimRight(strings.TrimSpace(cfg.StorageBaseURL), "/")
	}

	if strings.TrimSpace(cfg.AuthRefreshCookiePath) == "" {
		cfg.AuthRefreshCookiePath = cfg.APIBasePath + "/auth"
	}
	cfg.AuthRefreshCookiePath = normalizeURLPath(cfg.AuthRefreshCookiePath, cfg.APIBasePath+"/auth")
	cfg.AuthCSRFCookiePath = normalizeURLPath(cfg.AuthCSRFCookiePath, "/")

	if cfg.Env == "development" {
		if strings.TrimSpace(cfg.CORSOrigins) == "" {
			cfg.CORSOrigins = "http://localhost:5173,http://localhost:5174"
		}
		if strings.TrimSpace(cfg.WSOrigins) == "" {
			cfg.WSOrigins = cfg.CORSOrigins
		}
	}

	if err := cfg.validate(); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func normalizeURLPath(raw, fallback string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		trimmed = fallback
	}
	if !strings.HasPrefix(trimmed, "/") {
		trimmed = "/" + trimmed
	}
	if len(trimmed) > 1 {
		trimmed = strings.TrimRight(trimmed, "/")
	}
	return trimmed
}

func (c *Config) validate() error {
	if strings.TrimSpace(c.JWTSecret) == "" {
		return fmt.Errorf("JWT_SECRET is required")
	}
	if len(strings.TrimSpace(c.JWTSecret)) < 16 {
		return fmt.Errorf("JWT_SECRET must be at least 16 characters")
	}
	if c.JWTAccessTTL <= 0 || c.JWTRefreshTTL <= 0 {
		return fmt.Errorf("JWT_ACCESS_TTL and JWT_REFRESH_TTL must be > 0")
	}
	if strings.TrimSpace(c.CORSOrigins) == "" {
		return fmt.Errorf("CORS_ORIGINS is required")
	}
	if strings.TrimSpace(c.WSOrigins) == "" {
		return fmt.Errorf("WS_ORIGINS is required")
	}
	if strings.TrimSpace(c.TrustedProxies) == "" && c.Env != "development" {
		return fmt.Errorf("TRUSTED_PROXIES is required outside development")
	}
	if strings.TrimSpace(c.DatabaseURL) == "" || strings.TrimSpace(c.RedisURL) == "" {
		return fmt.Errorf("DATABASE_URL and REDIS_URL are required")
	}
	if c.StorageMaxSizeMB <= 0 {
		return fmt.Errorf("STORAGE_MAX_SIZE_MB must be > 0")
	}
	return nil
}
