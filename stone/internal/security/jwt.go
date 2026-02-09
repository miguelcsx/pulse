package security

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

var (
	ErrInvalidToken     = errors.New("invalid token")
	ErrInvalidTokenType = errors.New("invalid token type")
	ErrInvalidTokenSub  = errors.New("invalid token subject")
)

// Claims defines JWT claims used by Pulse auth.
type Claims struct {
	Type string `json:"type"`
	jwt.RegisteredClaims
}

// ParsedToken is a strongly-typed projection of validated JWT claims.
type ParsedToken struct {
	UserID    uuid.UUID
	TokenType string
	TokenID   *uuid.UUID
	ExpiresAt time.Time
}

// ParseAndValidateJWT verifies signature + standard claims and enforces token type.
func ParseAndValidateJWT(tokenStr, secret, expectedType string) (*ParsedToken, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(
		tokenStr,
		claims,
		func(t *jwt.Token) (any, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, jwt.ErrSignatureInvalid
			}
			return []byte(secret), nil
		},
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithLeeway(2*time.Second),
	)
	if err != nil || !token.Valid {
		return nil, ErrInvalidToken
	}

	if expectedType != "" && claims.Type != expectedType {
		return nil, ErrInvalidTokenType
	}

	userID, err := uuid.Parse(claims.Subject)
	if err != nil {
		return nil, ErrInvalidTokenSub
	}

	var tokenID *uuid.UUID
	if claims.ID != "" {
		id, err := uuid.Parse(claims.ID)
		if err != nil {
			return nil, fmt.Errorf("invalid token id: %w", err)
		}
		tokenID = &id
	}

	expiresAt := time.Time{}
	if claims.ExpiresAt != nil {
		expiresAt = claims.ExpiresAt.Time
	}

	return &ParsedToken{
		UserID:    userID,
		TokenType: claims.Type,
		TokenID:   tokenID,
		ExpiresAt: expiresAt,
	}, nil
}
