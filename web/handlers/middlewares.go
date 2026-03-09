package handlers

import (
	"net/http"
	"podcast/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v4"
	"github.com/spf13/cast"
	"github.com/youcd/toolkit/log"
)

// CorsMiddleware 处理跨域
func CorsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		method := c.Request.Method
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Headers", "Content-Type,AccessToken,X-CSRF-Token, Authorization, jwt, cache-control")
		c.Header("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
		c.Header("Access-Control-Expose-Headers", "Content-Length, Access-Control-Allow-Origin, Access-Control-Allow-Headers, Content-Type")
		c.Header("Access-Control-Allow-Credentials", "true")

		// 放行所有OPTIONS方法
		if method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
		}
		// 处理请求
		c.Next()
	}
}

func RequestIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 生成带 request_id 的 context
		ctx := log.SetRequestId(c.Request.Context())

		// 将新 context 设置回请求
		c.Request = c.Request.WithContext(ctx)
		value := ctx.Value("request_id").(string)
		c.Header("X-Request-Id", value)
		// 处理请求
		c.Next()
	}
}

// JwtMiddleware 需要注入 UserService，用于校验 token 并查询用户
func JwtMiddleware(userService *service.UserService) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "缺少Authorization header"})
			c.Abort()
			return
		}

		var tokenString string
		if len(authHeader) >= 7 && authHeader[:7] == "Bearer " {
			tokenString = authHeader[7:]
		} else {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header格式错误"})
			c.Abort()
			return
		}

		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			return []byte(userService.Token()), nil
		})

		if err != nil {
			if err == jwt.ErrSignatureInvalid {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "无效的token签名"})
				c.Abort()
				return
			}
			c.JSON(http.StatusUnauthorized, gin.H{"error": "无效的token"})
			c.Abort()
			return
		}

		if !token.Valid {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "无效的token"})
			c.Abort()
			return
		}

		if claims, ok := token.Claims.(jwt.MapClaims); ok {
			userID := cast.ToInt(claims["id"])

			user, err := userService.FindByID(c.Request.Context(), userID)
			if err != nil {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "用户不存在"})
				c.Abort()
				return
			}

			c.Set("user", user)
			c.Set("userID", userID)
		} else {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "无效的token claims"})
			c.Abort()
			return
		}

		c.Next()
	}
}
