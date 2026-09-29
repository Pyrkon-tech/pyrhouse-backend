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

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token claims"})
			c.Abort()
			return
		}
		// Routes without Authorize still type-assert these, so reject tokens that lack them.
		userID, _ := claims["userID"].(string)
		role, _ := claims["role"].(string)
		if userID == "" || role == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token claims"})
			c.Abort()
			return
		}
		c.Set("userID", userID)
		c.Set("role", role)
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
		return jwtSecret, nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}), jwt.WithExpirationRequired())
	if err != nil {
		return nil, err
	}

	if err := checkWarehouseAudience(token); err != nil {
		return nil, err
	}

	return token, nil
}

// checkWarehouseAudience accepts tokens issued for the warehouse and legacy tokens
// without an aud claim (issued before audiences existed, valid until they expire).
// Any other audience, notably the shop's, is rejected.
func checkWarehouseAudience(token *jwt.Token) error {
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return fmt.Errorf("unexpected claims type")
	}
	if _, present := claims["aud"]; !present {
		return nil
	}

	// GetAudience silently maps unsupported types (e.g. a number) to an empty list,
	// so a present claim must yield at least one audience.
	aud, err := claims.GetAudience()
	if err != nil {
		return fmt.Errorf("invalid aud claim: %w", err)
	}
	if len(aud) == 0 {
		return fmt.Errorf("invalid aud claim")
	}
	for _, a := range aud {
		if a != AudienceWarehouse {
			return fmt.Errorf("token audience %q is not allowed", a)
		}
	}
	return nil
}
