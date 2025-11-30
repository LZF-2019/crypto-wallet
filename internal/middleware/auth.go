package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"

	"crypto-wallet-api/internal/service"
	"crypto-wallet-api/internal/utils"
)

// AuthMiddleware JWT认证中间件
func AuthMiddleware(authService *service.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 1. 从Header获取Token
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			utils.Unauthorized(c, "missing authorization header")
			c.Abort()
			return
		}

		// 2. 解析Bearer Token
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			utils.Unauthorized(c, "invalid authorization header format")
			c.Abort()
			return
		}

		tokenString := parts[1]

		// 3. 验证Token
		userID, err := authService.ValidateToken(tokenString)
		if err != nil {
			utils.Unauthorized(c, "invalid or expired token")
			c.Abort()
			return
		}

		// 4. 将用户ID存入上下文
		c.Set("user_id", userID)

		// 5. 继续处理请求
		c.Next()
	}
}
