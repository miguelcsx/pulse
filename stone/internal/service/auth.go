package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/redis/go-redis/v9"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/pulse/stone/internal/model"
	"github.com/pulse/stone/internal/security"
)

// AuthService handles user registration, login, and JWT token management.
type AuthService struct {
	db              *gorm.DB
	rdb             *redis.Client
	jwtSecret       string
	accessTTL       time.Duration
	refreshTTL      time.Duration
	maxAttempts     int
	lockoutDuration time.Duration
}

var (
	ErrInvalidCredentials  = errors.New("invalid credentials")
	ErrInvalidRefreshToken = errors.New("invalid refresh token")
	ErrAccountLocked       = errors.New("account temporarily locked due to too many failed login attempts")
	ErrDuplicateUser       = errors.New("handle or email already taken")
	ErrInvalidDemoHandle   = errors.New("demo handle must be 3-30 letters, numbers, underscores, dots, or hyphens")
)

const refreshReplayGrace = 15 * time.Second

// NewAuthService creates a new AuthService.
func NewAuthService(db *gorm.DB, rdb *redis.Client, jwtSecret string, accessTTL, refreshTTL time.Duration, maxAttempts int, lockoutDuration time.Duration) *AuthService {
	if maxAttempts <= 0 {
		maxAttempts = 10
	}
	if lockoutDuration <= 0 {
		lockoutDuration = 15 * time.Minute
	}
	return &AuthService{
		db:              db,
		rdb:             rdb,
		jwtSecret:       jwtSecret,
		accessTTL:       accessTTL,
		refreshTTL:      refreshTTL,
		maxAttempts:     maxAttempts,
		lockoutDuration: lockoutDuration,
	}
}

// Register creates a new user account and returns the user with access and refresh tokens.
func (s *AuthService) Register(handle, email, password, displayName string) (*model.User, string, string, error) {
	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, "", "", fmt.Errorf("failed to hash password: %w", err)
	}

	user := &model.User{
		Handle:      handle,
		Email:       email,
		Password:    string(hashed),
		DisplayName: displayName,
	}

	if err := s.db.Create(user).Error; err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return nil, "", "", ErrDuplicateUser
		}
		return nil, "", "", fmt.Errorf("failed to create user: %w", err)
	}

	accessToken, refreshToken, err := s.generateTokens(user.ID)
	if err != nil {
		return nil, "", "", fmt.Errorf("failed to generate tokens: %w", err)
	}

	return user, accessToken, refreshToken, nil
}

// Login authenticates a user by email and password, returning the user with tokens.
// Implements progressive lockout: after maxAttempts consecutive failures within
// the lockout window, the account is temporarily locked.
func (s *AuthService) Login(email, password string) (*model.User, string, string, error) {
	// Check if the account is locked before doing anything else.
	if locked, remaining := s.isAccountLocked(email); locked {
		return nil, "", "", fmt.Errorf("%w: try again in %s", ErrAccountLocked, remaining.Truncate(time.Second))
	}

	var user model.User
	if err := s.db.Where("email = ?", email).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// Record the failed attempt even for non-existent accounts to prevent
			// email enumeration timing attacks.
			s.recordFailedAttempt(email)
			return nil, "", "", ErrInvalidCredentials
		}
		return nil, "", "", fmt.Errorf("failed to find user: %w", err)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		s.recordFailedAttempt(email)
		return nil, "", "", ErrInvalidCredentials
	}

	// Successful login — clear the failed attempt counter.
	s.clearFailedAttempts(email)

	accessToken, refreshToken, err := s.generateTokens(user.ID)
	if err != nil {
		return nil, "", "", fmt.Errorf("failed to generate tokens: %w", err)
	}

	return &user, accessToken, refreshToken, nil
}

// DemoLogin finds or creates a throwaway demo user addressed only by handle.
func (s *AuthService) DemoLogin(handle string) (*model.User, string, string, error) {
	normalized, err := normalizeDemoHandle(handle)
	if err != nil {
		return nil, "", "", err
	}

	email := normalized + "@demo.pulse.local"
	displayName := "@" + normalized

	var user model.User
	err = s.db.Where("handle = ?", normalized).First(&user).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, "", "", fmt.Errorf("failed to find demo user: %w", err)
	}

	if errors.Is(err, gorm.ErrRecordNotFound) {
		password, err := randomDemoPassword()
		if err != nil {
			return nil, "", "", err
		}
		hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			return nil, "", "", fmt.Errorf("failed to hash demo password: %w", err)
		}

		user = model.User{
			Handle:      normalized,
			Email:       email,
			Password:    string(hashed),
			DisplayName: displayName,
			Bio:         "Demo day account",
		}
		if err := s.db.Create(&user).Error; err != nil {
			var pgErr *pgconn.PgError
			if errors.As(err, &pgErr) && pgErr.Code == "23505" {
				if err := s.db.Where("handle = ?", normalized).First(&user).Error; err != nil {
					return nil, "", "", fmt.Errorf("failed to load existing demo user: %w", err)
				}
			} else {
				return nil, "", "", fmt.Errorf("failed to create demo user: %w", err)
			}
		}
	}

	accessToken, refreshToken, err := s.generateTokens(user.ID)
	if err != nil {
		return nil, "", "", fmt.Errorf("failed to generate tokens: %w", err)
	}

	return &user, accessToken, refreshToken, nil
}

// RefreshToken validates a refresh token and issues a new access/refresh token pair.
func (s *AuthService) RefreshToken(refreshToken string) (string, string, error) {
	parsed, err := security.ParseAndValidateJWT(refreshToken, s.jwtSecret, "refresh")
	if err != nil {
		return "", "", ErrInvalidRefreshToken
	}
	if parsed.TokenID == nil {
		return "", "", ErrInvalidRefreshToken
	}

	var accessToken string
	var newRefreshToken string
	err = s.db.Transaction(func(tx *gorm.DB) error {
		var session model.RefreshToken
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			First(&session, "id = ? AND user_id = ?", *parsed.TokenID, parsed.UserID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrInvalidRefreshToken
			}
			return fmt.Errorf("failed to load refresh session: %w", err)
		}

		now := time.Now()
		activeSession, err := resolveRefreshReplay(tx, session, parsed.UserID, now)
		if err != nil {
			return err
		}

		if activeSession.ExpiresAt.Before(now) {
			return ErrInvalidRefreshToken
		}

		newAccessToken, replacementToken, replacementID, err := s.generateTokensTx(tx, parsed.UserID)
		if err != nil {
			return err
		}

		if err := tx.Model(&activeSession).
			Where("id = ? AND revoked_at IS NULL", activeSession.ID).
			Updates(map[string]any{
				"revoked_at":  now,
				"replaced_by": replacementID,
				"updated_at":  now,
			}).Error; err != nil {
			return fmt.Errorf("failed to rotate refresh token: %w", err)
		}

		accessToken = newAccessToken
		newRefreshToken = replacementToken
		return nil
	})
	if err != nil {
		if errors.Is(err, ErrInvalidRefreshToken) {
			return "", "", ErrInvalidRefreshToken
		}
		return "", "", err
	}
	return accessToken, newRefreshToken, nil
}

func resolveRefreshReplay(tx *gorm.DB, session model.RefreshToken, userID uuid.UUID, now time.Time) (model.RefreshToken, error) {
	for range 3 {
		if session.RevokedAt == nil {
			return session, nil
		}
		if session.ReplacedBy == nil || now.Sub(*session.RevokedAt) > refreshReplayGrace {
			return model.RefreshToken{}, ErrInvalidRefreshToken
		}

		var replacement model.RefreshToken
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			First(&replacement, "id = ? AND user_id = ?", *session.ReplacedBy, userID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return model.RefreshToken{}, ErrInvalidRefreshToken
			}
			return model.RefreshToken{}, fmt.Errorf("failed to load replacement refresh session: %w", err)
		}
		session = replacement
	}

	return model.RefreshToken{}, ErrInvalidRefreshToken
}

// RevokeRefreshToken invalidates a refresh token session if it exists and is active.
func (s *AuthService) RevokeRefreshToken(refreshToken string) error {
	parsed, err := security.ParseAndValidateJWT(refreshToken, s.jwtSecret, "refresh")
	if err != nil || parsed.TokenID == nil {
		return nil
	}

	now := time.Now()
	if err := s.db.Model(&model.RefreshToken{}).
		Where("id = ? AND user_id = ? AND revoked_at IS NULL", *parsed.TokenID, parsed.UserID).
		Updates(map[string]any{
			"revoked_at": now,
			"updated_at": now,
		}).Error; err != nil {
		return fmt.Errorf("failed to revoke refresh token: %w", err)
	}
	return nil
}

// --- Account lockout ---

// lockoutKey returns the Redis key for tracking failed login attempts.
func (s *AuthService) lockoutKey(email string) string {
	return "login_attempts:" + email
}

// recordFailedAttempt increments the failed login counter for the given email.
// The counter expires after the lockout duration so it auto-resets.
func (s *AuthService) recordFailedAttempt(email string) {
	if s.rdb == nil {
		return
	}
	ctx := context.Background()
	key := s.lockoutKey(email)

	count, err := s.rdb.Incr(ctx, key).Result()
	if err != nil {
		return // fail-open: don't block logins if Redis is unavailable
	}

	// Set expiry on first attempt or refresh it on every attempt to create
	// a sliding window.
	if count == 1 {
		s.rdb.Expire(ctx, key, s.lockoutDuration)
	}
}

// isAccountLocked checks whether the email has exceeded the maximum failed
// login attempts within the lockout window.
func (s *AuthService) isAccountLocked(email string) (bool, time.Duration) {
	if s.rdb == nil {
		return false, 0
	}
	ctx := context.Background()
	key := s.lockoutKey(email)

	countStr, err := s.rdb.Get(ctx, key).Result()
	if err != nil {
		return false, 0 // key doesn't exist or Redis error — not locked
	}

	count, err := strconv.Atoi(countStr)
	if err != nil {
		return false, 0
	}

	if count < s.maxAttempts {
		return false, 0
	}

	// Account is locked — report how long until the lockout expires.
	ttl, err := s.rdb.TTL(ctx, key).Result()
	if err != nil || ttl <= 0 {
		return true, s.lockoutDuration
	}
	return true, ttl
}

// clearFailedAttempts removes the failed login counter after a successful login.
func (s *AuthService) clearFailedAttempts(email string) {
	if s.rdb == nil {
		return
	}
	s.rdb.Del(context.Background(), s.lockoutKey(email))
}

func normalizeDemoHandle(raw string) (string, error) {
	handle := strings.TrimPrefix(strings.TrimSpace(raw), "@")
	handle = strings.ToLower(handle)
	if len(handle) < 3 || len(handle) > 30 {
		return "", ErrInvalidDemoHandle
	}
	for _, r := range handle {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_' || r == '-' || r == '.' {
			continue
		}
		return "", ErrInvalidDemoHandle
	}
	return handle, nil
}

func randomDemoPassword() (string, error) {
	b := make([]byte, 24)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("failed to generate demo password: %w", err)
	}
	return hex.EncodeToString(b), nil
}

// --- Token generation ---

// generateTokens creates a new access and refresh JWT token pair for the given user ID.
func (s *AuthService) generateTokens(userID uuid.UUID) (string, string, error) {
	access, refresh, _, err := s.generateTokensTx(s.db, userID)
	return access, refresh, err
}

func (s *AuthService) generateTokensTx(tx *gorm.DB, userID uuid.UUID) (string, string, uuid.UUID, error) {
	now := time.Now()

	accessClaims := security.Claims{
		Type: "access",
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID.String(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(s.accessTTL)),
		},
	}
	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
	accessStr, err := accessToken.SignedString([]byte(s.jwtSecret))
	if err != nil {
		return "", "", uuid.Nil, fmt.Errorf("failed to sign access token: %w", err)
	}

	refreshID := uuid.New()
	refreshExp := now.Add(s.refreshTTL)
	session := model.RefreshToken{
		ID:        refreshID,
		UserID:    userID,
		ExpiresAt: refreshExp,
	}
	if err := tx.Create(&session).Error; err != nil {
		return "", "", uuid.Nil, fmt.Errorf("failed to create refresh session: %w", err)
	}

	refreshClaims := security.Claims{
		Type: "refresh",
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        refreshID.String(),
			Subject:   userID.String(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(refreshExp),
		},
	}
	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims)
	refreshStr, err := refreshToken.SignedString([]byte(s.jwtSecret))
	if err != nil {
		return "", "", uuid.Nil, fmt.Errorf("failed to sign refresh token: %w", err)
	}

	return accessStr, refreshStr, refreshID, nil
}
