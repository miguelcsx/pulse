package api

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type registerRequest struct {
	Handle      string `json:"handle" binding:"required"`
	Email       string `json:"email" binding:"required,email"`
	Password    string `json:"password" binding:"required,min=8"`
	DisplayName string `json:"display_name" binding:"required"`
}

type loginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type refreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

func (s *Server) Register(c *gin.Context) {
	var req registerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, accessToken, refreshToken, err := s.authService.Register(req.Handle, req.Email, req.Password, req.DisplayName)
	if err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		return
	}

	s.issueCSRFCookie(c)
	s.setRefreshCookie(c, refreshToken)
	c.JSON(http.StatusCreated, gin.H{
		"access_token": accessToken,
		"user":         toSelfUserResponse(user),
	})
}

func (s *Server) Login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, accessToken, refreshToken, err := s.authService.Login(req.Email, req.Password)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}

	s.issueCSRFCookie(c)
	s.setRefreshCookie(c, refreshToken)
	c.JSON(http.StatusOK, gin.H{
		"access_token": accessToken,
		"user":         toSelfUserResponse(user),
	})
}

func (s *Server) Refresh(c *gin.Context) {
	var req refreshRequest
	if err := c.ShouldBindJSON(&req); err != nil && !errors.Is(err, io.EOF) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid refresh payload"})
		return
	}

	refreshToken := strings.TrimSpace(req.RefreshToken)
	refreshFromCookie := false
	if refreshToken == "" {
		if cookie, err := c.Cookie(s.cfg.AuthRefreshCookieName); err == nil {
			refreshToken = strings.TrimSpace(cookie)
			refreshFromCookie = refreshToken != ""
		}
	}
	if refreshToken == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing refresh token"})
		return
	}
	if refreshFromCookie && !s.validateCSRFCookieHeader(c) {
		return
	}

	accessToken, rotatedRefreshToken, err := s.authService.RefreshToken(refreshToken)
	if err != nil {
		s.clearCSRFCookie(c)
		s.clearRefreshCookie(c)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid refresh token"})
		return
	}

	s.issueCSRFCookie(c)
	s.setRefreshCookie(c, rotatedRefreshToken)
	c.JSON(http.StatusOK, gin.H{
		"access_token": accessToken,
	})
}

func (s *Server) Logout(c *gin.Context) {
	refreshToken := ""
	if cookie, err := c.Cookie(s.cfg.AuthRefreshCookieName); err == nil {
		refreshToken = strings.TrimSpace(cookie)
	}
	if refreshToken != "" && !s.validateCSRFCookieHeader(c) {
		return
	}
	if refreshToken != "" {
		_ = s.authService.RevokeRefreshToken(refreshToken)
	}
	s.clearCSRFCookie(c)
	s.clearRefreshCookie(c)
	c.JSON(http.StatusOK, gin.H{"message": "logged out"})
}

func (s *Server) issueCSRFCookie(c *gin.Context) {
	token, err := generateCSRFToken()
	if err != nil {
		return
	}
	s.setCSRFCookie(c, token)
}

func (s *Server) setCSRFCookie(c *gin.Context, token string) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     s.cfg.AuthCSRFCookieName,
		Value:    token,
		Path:     s.cfg.AuthCSRFCookiePath,
		Domain:   strings.TrimSpace(s.cfg.AuthCSRFCookieDomain),
		MaxAge:   int(s.cfg.JWTRefreshTTL.Seconds()),
		Expires:  time.Now().Add(s.cfg.JWTRefreshTTL),
		HttpOnly: false,
		Secure:   s.cookieSecure(),
		SameSite: parseSameSite(s.cfg.AuthCSRFCookieSameSite),
	})
}

func (s *Server) setRefreshCookie(c *gin.Context, refreshToken string) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     s.cfg.AuthRefreshCookieName,
		Value:    refreshToken,
		Path:     s.cfg.AuthRefreshCookiePath,
		Domain:   strings.TrimSpace(s.cfg.AuthRefreshCookieDomain),
		MaxAge:   int(s.cfg.JWTRefreshTTL.Seconds()),
		Expires:  time.Now().Add(s.cfg.JWTRefreshTTL),
		HttpOnly: true,
		Secure:   s.cookieSecure(),
		SameSite: parseSameSite(s.cfg.AuthRefreshCookieSameSite),
	})
}

func (s *Server) clearRefreshCookie(c *gin.Context) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     s.cfg.AuthRefreshCookieName,
		Value:    "",
		Path:     s.cfg.AuthRefreshCookiePath,
		Domain:   strings.TrimSpace(s.cfg.AuthRefreshCookieDomain),
		MaxAge:   -1,
		Expires:  time.Unix(0, 0),
		HttpOnly: true,
		Secure:   s.cookieSecure(),
		SameSite: parseSameSite(s.cfg.AuthRefreshCookieSameSite),
	})
}

func (s *Server) clearCSRFCookie(c *gin.Context) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     s.cfg.AuthCSRFCookieName,
		Value:    "",
		Path:     s.cfg.AuthCSRFCookiePath,
		Domain:   strings.TrimSpace(s.cfg.AuthCSRFCookieDomain),
		MaxAge:   -1,
		Expires:  time.Unix(0, 0),
		HttpOnly: false,
		Secure:   s.cookieSecure(),
		SameSite: parseSameSite(s.cfg.AuthCSRFCookieSameSite),
	})
}

func (s *Server) validateCSRFCookieHeader(c *gin.Context) bool {
	headerName := strings.TrimSpace(s.cfg.AuthCSRFHeaderName)
	if headerName == "" {
		headerName = "X-CSRF-Token"
	}
	headerValue := strings.TrimSpace(c.GetHeader(headerName))
	cookieValue, err := c.Cookie(s.cfg.AuthCSRFCookieName)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "missing csrf cookie"})
		return false
	}
	cookieValue = strings.TrimSpace(cookieValue)
	if headerValue == "" || cookieValue == "" {
		c.JSON(http.StatusForbidden, gin.H{"error": "missing csrf token"})
		return false
	}
	if subtle.ConstantTimeCompare([]byte(headerValue), []byte(cookieValue)) != 1 {
		c.JSON(http.StatusForbidden, gin.H{"error": "invalid csrf token"})
		return false
	}
	return true
}

func (s *Server) cookieSecure() bool {
	return s.cfg.AuthCookieSecure || s.cfg.Env == "production"
}

func parseSameSite(raw string) http.SameSite {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "strict":
		return http.SameSiteStrictMode
	case "none":
		return http.SameSiteNoneMode
	default:
		return http.SameSiteLaxMode
	}
}

func generateCSRFToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("failed to generate csrf token: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
