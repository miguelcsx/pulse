package config

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	Port string `envconfig:"PORT" default:"8080"`
	Env  string `envconfig:"ENV" default:"development"`

	DatabaseURL string `envconfig:"DATABASE_URL" required:"true"`

	RedisURL string `envconfig:"REDIS_URL" required:"true"`

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
	DemoAuthEnabled           bool          `envconfig:"DEMO_AUTH_ENABLED" default:"false"`

	APIBasePath      string `envconfig:"API_BASE_PATH" default:"/api/v1"`
	StoragePath      string `envconfig:"STORAGE_PATH" default:"./uploads"`
	StorageBaseURL   string `envconfig:"STORAGE_BASE_URL" default:""`
	UploadPublicPath string `envconfig:"UPLOAD_PUBLIC_PATH" default:"/uploads"`
	StorageMaxSizeMB int    `envconfig:"STORAGE_MAX_SIZE_MB" default:"10"`
	StorageDriver    string `envconfig:"STORAGE_DRIVER" default:"local"` // "local" or "s3"

	// S3/Supabase Storage
	S3Endpoint  string `envconfig:"S3_ENDPOINT" default:""`
	S3Region    string `envconfig:"S3_REGION" default:"us-east-1"`
	S3Bucket    string `envconfig:"S3_BUCKET" default:""`
	S3AccessKey string `envconfig:"S3_ACCESS_KEY" default:""`
	S3SecretKey string `envconfig:"S3_SECRET_KEY" default:""`
	S3PublicURL string `envconfig:"S3_PUBLIC_URL" default:""`

	CORSOrigins string `envconfig:"CORS_ORIGINS" default:""`
	WSOrigins   string `envconfig:"WS_ORIGINS" default:""`

	RateLimitRPS      int    `envconfig:"RATE_LIMIT_RPS" default:"100"`
	RateLimitBurst    int    `envconfig:"RATE_LIMIT_BURST" default:"200"`
	RateLimitFailOpen bool   `envconfig:"RATE_LIMIT_FAIL_OPEN" default:"false"`
	TrustedProxies    string `envconfig:"TRUSTED_PROXIES" default:"127.0.0.1,::1"`

	ReadTimeout     time.Duration `envconfig:"READ_TIMEOUT" default:"15s"`
	WriteTimeout    time.Duration `envconfig:"WRITE_TIMEOUT" default:"15s"`
	IdleTimeout     time.Duration `envconfig:"IDLE_TIMEOUT" default:"60s"`
	ShutdownTimeout time.Duration `envconfig:"SHUTDOWN_TIMEOUT" default:"10s"`

	// Database pool
	DBMaxOpenConns    int           `envconfig:"DB_MAX_OPEN_CONNS" default:"25"`
	DBMaxIdleConns    int           `envconfig:"DB_MAX_IDLE_CONNS" default:"5"`
	DBConnMaxLifetime time.Duration `envconfig:"DB_CONN_MAX_LIFETIME" default:"5m"`

	// Scheduler
	SchedulerInterval      time.Duration `envconfig:"SCHEDULER_INTERVAL" default:"5m"`
	MediaRecoveryThreshold time.Duration `envconfig:"MEDIA_RECOVERY_THRESHOLD" default:"15m"`

	// Auth brute-force protection
	LoginMaxAttempts     int           `envconfig:"LOGIN_MAX_ATTEMPTS" default:"10"`
	LoginLockoutDuration time.Duration `envconfig:"LOGIN_LOCKOUT_DURATION" default:"15m"`

	// Tag cache
	TagCacheTTL time.Duration `envconfig:"TAG_CACHE_TTL" default:"5m"`

	// Room
	RoomTTL              time.Duration `envconfig:"ROOM_TTL" default:"24h"`
	RoomExplorationRatio float64       `envconfig:"ROOM_EXPLORATION_RATIO" default:"0.2"`

	// WebSocket
	WSMaxMessageSize int64 `envconfig:"WS_MAX_MESSAGE_SIZE" default:"4096"`

	// Observability
	MetricsEnabled bool `envconfig:"METRICS_ENABLED" default:"false"`

	// Vector search
	EmbeddingDimensions int `envconfig:"EMBEDDING_DIMENSIONS" default:"1024"`
	VectorTopK          int `envconfig:"VECTOR_TOP_K" default:"12"`

	// Local AI (Ollama for semantic embeddings)
	OllamaBaseURL string        `envconfig:"OLLAMA_BASE_URL" default:"http://localhost:11434"`
	OllamaModel   string        `envconfig:"OLLAMA_MODEL" default:"qwen3-embedding"`
	OllamaTimeout time.Duration `envconfig:"OLLAMA_TIMEOUT" default:"2s"`

	// Affinity decay half-lives (in days)
	AffinityHalfLife7DDays  float64 `envconfig:"AFFINITY_HALF_LIFE_7D_DAYS" default:"3.5"`
	AffinityHalfLife30DDays float64 `envconfig:"AFFINITY_HALF_LIFE_30D_DAYS" default:"15"`

	// Feed algorithm weights and mix ratios
	FeedAffinityWeight float64       `envconfig:"FEED_AFFINITY_WEIGHT" default:"0.4"`
	FeedRecencyWeight  float64       `envconfig:"FEED_RECENCY_WEIGHT" default:"0.35"`
	FeedQualityWeight  float64       `envconfig:"FEED_QUALITY_WEIGHT" default:"0.25"`
	FeedDiscoveryRatio float64       `envconfig:"FEED_DISCOVERY_RATIO" default:"0.3"`
	FeedTrendingMaxAge time.Duration `envconfig:"FEED_TRENDING_MAX_AGE" default:"48h"`
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
	if cfg.Env == "development" {
		if _, explicitlySet := os.LookupEnv("DEMO_AUTH_ENABLED"); !explicitlySet {
			cfg.DemoAuthEnabled = true
		}
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
	if strings.TrimSpace(c.DatabaseURL) == "" {
		return fmt.Errorf("DATABASE_URL is required")
	}
	if strings.TrimSpace(c.RedisURL) == "" {
		return fmt.Errorf("REDIS_URL is required")
	}
	if c.StorageMaxSizeMB <= 0 {
		return fmt.Errorf("STORAGE_MAX_SIZE_MB must be > 0")
	}
	if c.StorageDriver == "s3" {
		if strings.TrimSpace(c.S3Endpoint) == "" {
			return fmt.Errorf("S3_ENDPOINT is required when STORAGE_DRIVER=s3")
		}
		if strings.TrimSpace(c.S3Bucket) == "" {
			return fmt.Errorf("S3_BUCKET is required when STORAGE_DRIVER=s3")
		}
		if strings.TrimSpace(c.S3AccessKey) == "" {
			return fmt.Errorf("S3_ACCESS_KEY is required when STORAGE_DRIVER=s3")
		}
		if strings.TrimSpace(c.S3SecretKey) == "" {
			return fmt.Errorf("S3_SECRET_KEY is required when STORAGE_DRIVER=s3")
		}
	}
	if c.EmbeddingDimensions <= 0 {
		return fmt.Errorf("EMBEDDING_DIMENSIONS must be > 0")
	}
	if c.VectorTopK <= 0 {
		return fmt.Errorf("VECTOR_TOP_K must be > 0")
	}
	if c.OllamaTimeout <= 0 {
		return fmt.Errorf("OLLAMA_TIMEOUT must be > 0")
	}
	if c.RoomExplorationRatio < 0 || c.RoomExplorationRatio >= 1 {
		return fmt.Errorf("ROOM_EXPLORATION_RATIO must be >= 0 and < 1")
	}
	return nil
}

// IsProduction reports whether the server is running in production mode.
func (c *Config) IsProduction() bool {
	return c.Env == "production"
}
