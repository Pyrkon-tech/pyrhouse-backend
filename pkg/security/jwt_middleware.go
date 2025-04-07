package security

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

const (
	RoleUser      = 1
	RoleModerator = 2
	RoleAdmin     = 3
)

var roleHierarchy = map[string]int{
	"user":      RoleUser,
	"moderator": RoleModerator,
	"admin":     RoleAdmin,
}

// JWTMiddleware validates JWT and extracts claims.
func JWTMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header missing"})
			c.Abort()
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")

		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method")
			}
			return jwtSecret, nil
		})

		if err != nil || !token.Valid {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			c.Abort()
			return
		}

		claims := token.Claims.(jwt.MapClaims)
		c.Set("userID", claims["userID"])
		c.Set("role", claims["role"])
		c.Next()
	}
}

// Authorize ensures the user has the required role.
func Authorize(requiredRole string) gin.HandlerFunc {
	return func(c *gin.Context) {
		role, exists := c.Get("role")
		if !exists {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Forbidden: insufficient permissions"})
			return
		}
		userRole, ok := role.(string)
		if !ok {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Invalid role format"})
			return
		}

		requiredRoleLevel, requiredExists := roleHierarchy[requiredRole]
		userRoleLevel, userExists := roleHierarchy[userRole]

		if !requiredExists || !userExists || userRoleLevel < requiredRoleLevel {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Forbidden: insufficient permissions"})
			return
		}

		c.Next()
	}
}

func IsAllowed(c *gin.Context, requiredRole string) bool {
	role, exists := c.Get("role")
	if !exists {
		return false
	}

	userRole, ok := role.(string)
	if !ok {
		return false
	}

	requiredRoleLevel, requiredExists := roleHierarchy[requiredRole]
	userRoleLevel, userExists := roleHierarchy[userRole]

	if !requiredExists || !userExists || userRoleLevel < requiredRoleLevel {
		return false
	}

	return true
}
