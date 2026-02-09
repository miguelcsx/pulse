package middleware

import (
	"net/http"
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"
)

func CORS(allowedOrigins string) gin.HandlerFunc {
	matchOrigin := NewOriginMatcher(allowedOrigins)

	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if origin != "" && matchOrigin(origin) {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Authorization, X-Request-ID, X-CSRF-Token")
			c.Header("Access-Control-Allow-Credentials", "true")
			c.Header("Access-Control-Max-Age", "86400")
			c.Header("Vary", "Origin, Access-Control-Request-Method, Access-Control-Request-Headers")
		}

		if c.Request.Method == http.MethodOptions {
			if origin != "" && !matchOrigin(origin) {
				c.AbortWithStatus(http.StatusForbidden)
				return
			}
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

// NewOriginMatcher returns an origin validator that supports exact origins and
// suffix wildcards in the form "https://*.example.com". A single "*" allows all origins.
func NewOriginMatcher(allowedOrigins string) func(string) bool {
	var exact []string
	var wildcardPatterns []string
	allowAll := false

	for _, raw := range strings.Split(allowedOrigins, ",") {
		origin := strings.TrimSpace(raw)
		if origin == "" {
			continue
		}
		if origin == "*" {
			allowAll = true
			continue
		}
		if strings.Contains(origin, "*.") {
			wildcardPatterns = append(wildcardPatterns, origin)
			continue
		}
		exact = append(exact, origin)
	}

	return func(origin string) bool {
		if origin == "" {
			return false
		}
		if allowAll {
			return true
		}

		parsed, err := url.Parse(origin)
		if err != nil || parsed.Scheme == "" || parsed.Host == "" {
			return false
		}

		for _, eo := range exact {
			if origin == eo {
				return true
			}
		}

		for _, pattern := range wildcardPatterns {
			patternURL, err := url.Parse(pattern)
			if err != nil || patternURL.Scheme == "" || patternURL.Host == "" {
				continue
			}
			if parsed.Scheme != patternURL.Scheme {
				continue
			}

			patternHost := patternURL.Hostname()
			if !strings.HasPrefix(patternHost, "*.") {
				continue
			}
			if strings.TrimPrefix(patternHost, "*.") == parsed.Hostname() {
				continue // only subdomains, not root domain
			}
			if strings.HasSuffix(parsed.Hostname(), strings.TrimPrefix(patternHost, "*")) {
				// If wildcard origin pattern includes an explicit port, enforce it.
				if patternURL.Port() == "" || patternURL.Port() == parsed.Port() {
					return true
				}
			}
		}
		return false
	}
}
