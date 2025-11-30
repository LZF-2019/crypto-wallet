package middleware

import (
	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"

	"crypto-wallet-api/internal/utils"
)

// RateLimitMiddleware 限流中间件（令牌桶算法）
func RateLimitMiddleware(requestsPerSecond float64, burst int) gin.HandlerFunc {
	// 创建限流器
	limiter := rate.NewLimiter(rate.Limit(requestsPerSecond), burst)

	return func(c *gin.Context) {
		// 尝试获取令牌
		if !limiter.Allow() {
			utils.ErrorJson(c, 429, utils.CodeInvalidParams, "rate limit exceeded")
			c.Abort()
			return
		}

		c.Next()
	}
}
