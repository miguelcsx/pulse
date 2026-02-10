package middleware

import (
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

func RateLimit(rdb *redis.Client, rps int, burst int, failOpen bool) gin.HandlerFunc {
	if rps <= 0 {
		rps = 1
	}
	if burst < rps {
		burst = rps
	}
	windowMS := int64(1000)
	if burst > rps {
		windowMS = int64(float64(burst) / float64(rps) * 1000.0)
		if windowMS < 1000 {
			windowMS = 1000
		}
	}

	return func(c *gin.Context) {
		nowMS := time.Now().UnixMilli()
		windowStart := nowMS - (nowMS % windowMS)
		key := fmt.Sprintf("rl:%s:%d", c.ClientIP(), windowStart)
		ctx := c.Request.Context()

		count, err := rdb.Incr(ctx, key).Result()
		if err != nil {
			slog.Warn("rate limiter redis error", "error", err, "client_ip", c.ClientIP())
			if failOpen {
				c.Next()
				return
			}
			c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{
				"error": "service temporarily unavailable",
			})
			return
		}

		if count == 1 {
			rdb.PExpire(ctx, key, time.Duration(windowMS*2)*time.Millisecond)
		}

		if count > int64(burst) {
			retryAfterSeconds := int((windowMS + 999) / 1000)
			if retryAfterSeconds < 1 {
				retryAfterSeconds = 1
			}
			c.Header("Retry-After", strconv.Itoa(retryAfterSeconds))
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error": "rate limit exceeded",
			})
			return
		}

		c.Next()
	}
}
