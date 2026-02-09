package service

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/pulse/stone/internal/model"
	"github.com/pulse/stone/internal/security"
)

// AuthService handles user registration, login, and JWT token management.
type AuthService struct {
	db         *gorm.DB
	jwtSecret  string
	accessTTL  time.Duration
	refreshTTL time.Duration
}

var (
	ErrInvalidCredentials  = errors.New("invalid credentials")
	ErrInvalidRefreshToken = errors.New("invalid refresh token")
)

// NewAuthService creates a new AuthService.
func NewAuthService(db *gorm.DB, jwtSecret string, accessTTL, refreshTTL time.Duration) *AuthService {
	return &AuthService{
		db:         db,
		jwtSecret:  jwtSecret,
		accessTTL:  accessTTL,
		refreshTTL: refreshTTL,
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
		return nil, "", "", fmt.Errorf("failed to create user: %w", err)
	}

	accessToken, refreshToken, err := s.generateTokens(user.ID)
	if err != nil {
		return nil, "", "", fmt.Errorf("failed to generate tokens: %w", err)
	}

	return user, accessToken, refreshToken, nil
}

// Login authenticates a user by email and password, returning the user with tokens.
func (s *AuthService) Login(email, password string) (*model.User, string, string, error) {
	var user model.User
	if err := s.db.Where("email = ?", email).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, "", "", ErrInvalidCredentials
		}
		return nil, "", "", fmt.Errorf("failed to find user: %w", err)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		return nil, "", "", ErrInvalidCredentials
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

		if session.RevokedAt != nil || session.ExpiresAt.Before(time.Now()) {
			return ErrInvalidRefreshToken
		}

		newAccessToken, replacementToken, replacementID, err := s.generateTokensTx(tx, parsed.UserID)
		if err != nil {
			return err
		}

		now := time.Now()
		if err := tx.Model(&session).
			Where("id = ? AND revoked_at IS NULL", session.ID).
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
