package security

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func signTestToken(t *testing.T, claims jwt.MapClaims) string {
	t.Helper()
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(jwtSecret)
	require.NoError(t, err)
	return token
}

func baseClaims() jwt.MapClaims {
	return jwt.MapClaims{
		"userID":   "42",
		"role":     "user",
		"username": "tester",
		"exp":      time.Now().Add(time.Hour).Unix(),
	}
}

func withAud(aud any) jwt.MapClaims {
	c := baseClaims()
	c["aud"] = aud
	return c
}

func TestGenerateJWT_SetsWarehouseAudience(t *testing.T) {
	initTestJWT(t)

	tokenString, err := GenerateJWT("42", "user", "tester")
	require.NoError(t, err)

	token, _, err := jwt.NewParser().ParseUnverified(tokenString, jwt.MapClaims{})
	require.NoError(t, err)
	aud, err := token.Claims.GetAudience()
	require.NoError(t, err)
	assert.Equal(t, jwt.ClaimStrings{AudienceWarehouse}, aud)
}

func TestWarehouseAudience(t *testing.T) {
	initTestJWT(t)
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name     string
		claims   jwt.MapClaims
		accepted bool
	}{
		{"warehouse audience", withAud(AudienceWarehouse), true},
		{"warehouse audience as list", withAud([]string{AudienceWarehouse}), true},
		{"token without aud (pre-2026-09-29)", baseClaims(), false},
		{"shop audience", withAud(AudienceShop), false},
		{"shop audience in list with warehouse", withAud([]string{AudienceWarehouse, AudienceShop}), false},
		{"unknown audience", withAud("something-else"), false},
		{"malformed aud claim", withAud(123), false},
		{"empty aud string", withAud(""), false},
		{"empty aud list", withAud([]string{}), false},
		{"null aud", withAud(nil), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			header := "Bearer " + signTestToken(t, tt.claims)

			t.Run("JWTMiddleware", func(t *testing.T) {
				r := gin.New()
				r.GET("/p", JWTMiddleware(), func(c *gin.Context) { c.Status(http.StatusOK) })
				req := httptest.NewRequest(http.MethodGet, "/p", nil)
				req.Header.Set("Authorization", header)
				w := httptest.NewRecorder()
				r.ServeHTTP(w, req)

				if tt.accepted {
					assert.Equal(t, http.StatusOK, w.Code)
				} else {
					assert.Equal(t, http.StatusUnauthorized, w.Code)
				}
			})

			// Optional-auth handlers (e.g. service desk createRequest) read the token
			// directly, bypassing JWTMiddleware, so the check must hold here too.
			t.Run("GetUserIDFromToken", func(t *testing.T) {
				c, _ := gin.CreateTestContext(httptest.NewRecorder())
				c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
				c.Request.Header.Set("Authorization", header)

				userID, err := GetUserIDFromToken(c)
				if tt.accepted {
					require.NoError(t, err)
					assert.Equal(t, "42", userID)
				} else {
					assert.Error(t, err)
					assert.Empty(t, userID)
				}
			})
		})
	}
}

func TestWarehouseToken_Validation(t *testing.T) {
	initTestJWT(t)
	gin.SetMode(gin.TestMode)

	signWith := func(method jwt.SigningMethod, claims jwt.MapClaims, key []byte) string {
		token, err := jwt.NewWithClaims(method, claims).SignedString(key)
		require.NoError(t, err)
		return token
	}
	without := func(key string) jwt.MapClaims {
		c := withAud(AudienceWarehouse)
		delete(c, key)
		return c
	}
	expired := withAud(AudienceWarehouse)
	expired["exp"] = time.Now().Add(-time.Minute).Unix()

	tests := []struct {
		name  string
		token string
		code  int
	}{
		{"valid token", signWith(jwt.SigningMethodHS256, withAud(AudienceWarehouse), jwtSecret), http.StatusOK},
		{"expired token", signWith(jwt.SigningMethodHS256, expired, jwtSecret), http.StatusUnauthorized},
		{"token without exp", signWith(jwt.SigningMethodHS256, without("exp"), jwtSecret), http.StatusUnauthorized},
		{"wrong secret", signWith(jwt.SigningMethodHS256, withAud(AudienceWarehouse), []byte("other-secret")), http.StatusUnauthorized},
		{"HS512 instead of HS256", signWith(jwt.SigningMethodHS512, withAud(AudienceWarehouse), jwtSecret), http.StatusUnauthorized},
		{"token without userID", signWith(jwt.SigningMethodHS256, without("userID"), jwtSecret), http.StatusUnauthorized},
		{"token without role", signWith(jwt.SigningMethodHS256, without("role"), jwtSecret), http.StatusUnauthorized},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := gin.New()
			r.GET("/p", JWTMiddleware(), func(c *gin.Context) { c.Status(http.StatusOK) })
			req := httptest.NewRequest(http.MethodGet, "/p", nil)
			req.Header.Set("Authorization", "Bearer "+tt.token)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			assert.Equal(t, tt.code, w.Code)
		})
	}
}

func shopClaims() jwt.MapClaims {
	return jwt.MapClaims{
		"sub": "7",
		"aud": []string{AudienceShop},
		"exp": time.Now().Add(time.Hour).Unix(),
	}
}

func TestShopToken_RoundTrip(t *testing.T) {
	initTestJWT(t)

	token, err := GenerateShopJWT(7)
	require.NoError(t, err)

	accountID, err := ParseShopToken(token)
	require.NoError(t, err)
	assert.Equal(t, 7, accountID)
}

func TestParseShopToken_Rejects(t *testing.T) {
	initTestJWT(t)

	warehouseToken, err := GenerateJWT("42", "admin", "tester")
	require.NoError(t, err)

	mutate := func(f func(jwt.MapClaims)) string {
		c := shopClaims()
		f(c)
		return signTestToken(t, c)
	}

	tests := []struct {
		name  string
		token string
	}{
		{"warehouse token", warehouseToken},
		{"legacy token without aud", signTestToken(t, baseClaims())},
		{"shop token without aud", mutate(func(c jwt.MapClaims) { delete(c, "aud") })},
		{"both audiences", mutate(func(c jwt.MapClaims) { c["aud"] = []string{AudienceShop, AudienceWarehouse} })},
		{"expired", mutate(func(c jwt.MapClaims) { c["exp"] = time.Now().Add(-time.Minute).Unix() })},
		{"without exp", mutate(func(c jwt.MapClaims) { delete(c, "exp") })},
		{"without sub", mutate(func(c jwt.MapClaims) { delete(c, "sub") })},
		{"non-numeric sub", mutate(func(c jwt.MapClaims) { c["sub"] = "admin" })},
		{"zero sub", mutate(func(c jwt.MapClaims) { c["sub"] = "0" })},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseShopToken(tt.token)
			assert.Error(t, err)
		})
	}
}

// A shop token must never authenticate against warehouse routes, even when a buggy issuer
// drops its aud claim (it then lacks userID/role and is still rejected).
func TestShopToken_RejectedByWarehouse(t *testing.T) {
	initTestJWT(t)
	gin.SetMode(gin.TestMode)

	shopToken, err := GenerateShopJWT(42)
	require.NoError(t, err)
	stripped := shopClaims()
	delete(stripped, "aud")

	for name, token := range map[string]string{
		"shop token":             shopToken,
		"shop token without aud": signTestToken(t, stripped),
	} {
		t.Run(name, func(t *testing.T) {
			r := gin.New()
			r.GET("/p", JWTMiddleware(), func(c *gin.Context) { c.Status(http.StatusOK) })
			req := httptest.NewRequest(http.MethodGet, "/p", nil)
			req.Header.Set("Authorization", "Bearer "+token)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			assert.Equal(t, http.StatusUnauthorized, w.Code)
		})
	}

	t.Run("GetUserIDFromToken", func(t *testing.T) {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
		c.Request.Header.Set("Authorization", "Bearer "+shopToken)
		_, err := GetUserIDFromToken(c)
		assert.Error(t, err)
	})
}
