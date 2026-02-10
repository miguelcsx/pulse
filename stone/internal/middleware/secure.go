package middleware

import (
	"github.com/gin-gonic/gin"
)

// SecureHeaders returns middleware that sets security-related HTTP headers.
// In production, stricter policies (HSTS, tight CSP) are applied.
func SecureHeaders() gin.HandlerFunc {
	return SecureHeadersWithEnv("development")
}

// SecureHeadersWithEnv returns middleware that sets security headers appropriate
// for the given environment ("production", "development", etc.).
func SecureHeadersWithEnv(env string) gin.HandlerFunc {
	isProduction := env == "production"

	return func(c *gin.Context) {
		// Prevent MIME-type sniffing
		c.Header("X-Content-Type-Options", "nosniff")

		// Prevent clickjacking
		c.Header("X-Frame-Options", "DENY")

		// Legacy XSS filter (still useful for older browsers)
		c.Header("X-XSS-Protection", "1; mode=block")

		// Control referrer information leakage
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")

		// Restrict browser features the app doesn't need
		c.Header("Permissions-Policy", "camera=(), microphone=(), geolocation=(), payment=()")

		// Content Security Policy
		if isProduction {
			c.Header("Content-Security-Policy",
				"default-src 'self'; "+
					"img-src 'self' data: blob: https:; "+
					"media-src 'self' blob: https:; "+
					"style-src 'self' 'unsafe-inline'; "+
					"script-src 'self'; "+
					"connect-src 'self' wss: https:; "+
					"font-src 'self' data:; "+
					"frame-ancestors 'none'; "+
					"base-uri 'self'; "+
					"form-action 'self'",
			)
		} else {
			// Development: more permissive to allow Vite HMR, inline styles, etc.
			c.Header("Content-Security-Policy",
				"default-src 'self' http://localhost:* ws://localhost:*; "+
					"img-src 'self' data: blob: http://localhost:*; "+
					"media-src 'self' blob: http://localhost:*; "+
					"style-src 'self' 'unsafe-inline'; "+
					"script-src 'self' 'unsafe-inline' 'unsafe-eval' http://localhost:*; "+
					"connect-src 'self' ws://localhost:* http://localhost:*; "+
					"font-src 'self' data:; "+
					"frame-ancestors 'none'",
			)
		}

		// HSTS: only in production (requires TLS)
		if isProduction {
			// max-age=63072000 = 2 years; includeSubDomains; preload
			c.Header("Strict-Transport-Security", "max-age=63072000; includeSubDomains; preload")
		}

		// Prevent the page from being embedded in cross-origin contexts
		c.Header("Cross-Origin-Opener-Policy", "same-origin")

		c.Next()
	}
}
