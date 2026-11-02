package api

import (
	"strings"

	"t11-unstoppables/internal/auth"
	"t11-unstoppables/internal/users"

	"github.com/gin-gonic/gin"
)

func CORSMiddleware(origins []string) gin.HandlerFunc {
	allowed := make(map[string]struct{}, len(origins))
	for _, origin := range origins {
		allowed[strings.TrimSpace(origin)] = struct{}{}
	}
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if origin == "" {
			c.Next()
			return
		}
		if _, ok := allowed[origin]; !ok {
			c.AbortWithStatusJSON(403, gin.H{"success": false, "message": "origin is not allowed"})
			return
		}
		c.Header("Access-Control-Allow-Origin", origin)
		c.Header("Vary", "Origin")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PATCH, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Authorization, Content-Type, Idempotency-Key")
		c.Header("Access-Control-Max-Age", "600")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	}
}

func RequireAuth(service *auth.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		authorization := c.GetHeader("Authorization")
		if !strings.HasPrefix(authorization, "Bearer ") {
			c.AbortWithStatusJSON(401, gin.H{"success": false, "message": "authentication required"})
			return
		}
		claims, err := service.ParseToken(strings.TrimPrefix(authorization, "Bearer "))
		if err != nil {
			c.AbortWithStatusJSON(401, gin.H{"success": false, "message": "invalid or expired session"})
			return
		}
		c.Set("userID", claims.Subject)
		c.Set("role", claims.Role)
		c.Next()
	}
}

func RequireRole(roles ...string) gin.HandlerFunc {
	allowed := make(map[string]struct{}, len(roles))
	for _, role := range roles {
		allowed[role] = struct{}{}
	}
	return func(c *gin.Context) {
		role := users.Role(c.GetString("role"))
		if users.ValidateRole(role) != nil {
			c.AbortWithStatusJSON(403, gin.H{"success": false, "message": "unsupported role"})
			return
		}
		if _, ok := allowed[string(role)]; !ok {
			c.AbortWithStatusJSON(403, gin.H{"success": false, "message": "insufficient role"})
			return
		}
		c.Next()
	}
}
