package security

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"warehouse/internal/roles"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// JWTMiddleware validates JWT and extracts claims.
func JWTMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header missing"})
			c.Abort()
			return
		}

		token, err := getTokenFromContext(c)

		if err != nil || !token.Valid {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			c.Abort()
			return
		}

		claims := token.Claims.(jwt.MapClaims)
		c.Set("userID", claims["userID"])
		c.Set("role", claims["role"])
		c.Set("username", claims["username"])
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

		userRoleType := roles.Role(userRole)
		requiredRoleType := roles.Role(requiredRole)

		if !userRoleType.IsValid() || !requiredRoleType.IsValid() {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Forbidden: insufficient permissions"})
			return
		}

		if !userRoleType.HasPermission(requiredRoleType) {
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

	userRoleType := roles.Role(userRole)
	requiredRoleType := roles.Role(requiredRole)

	if !userRoleType.IsValid() || !requiredRoleType.IsValid() {
		return false
	}

	return userRoleType.HasPermission(requiredRoleType)
}

// IsOwnerOrAllowed checks if the user is either the owner of the resource or has the required role.
func IsOwnerOrAllowed(c *gin.Context, resourceUserID int, requiredRole string) bool {
	authID, ok := c.Get("userID")
	if !ok {
		return false
	}

	authIDStr, ok := authID.(string)
	if !ok {
		return false
	}

	authIDInt, err := strconv.Atoi(authIDStr)
	if err != nil || authIDInt == 0 {
		return false
	}

	if authIDInt == resourceUserID {
		return true
	}

	return IsAllowed(c, requiredRole)
}

func RequireRole(requiredRole roles.Role) gin.HandlerFunc {
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

		roleType := roles.Role(userRole)
		if !roleType.HasPermission(requiredRole) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Forbidden: insufficient permissions"})
			return
		}

		c.Next()
	}
}

func getTokenFromContext(c *gin.Context) (*jwt.Token, error) {
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		return nil, fmt.Errorf("no token provided")
	}

	tokenString := strings.TrimPrefix(authHeader, "Bearer ")

	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return jwtSecret, nil
	})

	return token, err
}
