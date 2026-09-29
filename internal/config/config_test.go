package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWithOrigin(t *testing.T) {
	tests := []struct {
		name    string
		origins []string
		origin  string
		want    []string
	}{
		{"empty origin is ignored", []string{"http://localhost:3000"}, "", []string{"http://localhost:3000"}},
		{"new origin is appended", []string{"http://localhost:3000"}, "https://shop.pyrhouse.space", []string{"http://localhost:3000", "https://shop.pyrhouse.space"}},
		{"existing origin is not duplicated", []string{"https://shop.pyrhouse.space"}, "https://shop.pyrhouse.space", []string{"https://shop.pyrhouse.space"}},
		{"existing origin with trailing slash is not duplicated", []string{"https://shop.pyrhouse.space/"}, "https://shop.pyrhouse.space", []string{"https://shop.pyrhouse.space/"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, withOrigin(tt.origins, tt.origin))
		})
	}
}

func TestLoad_ShopURLAddedToCORS(t *testing.T) {
	t.Setenv("CORS_ALLOWED_ORIGINS", "https://pyrhouse.space")
	t.Setenv("SHOP_URL", "https://shop.pyrhouse.space/")

	cfg, err := Load()
	assert.NoError(t, err)
	assert.Equal(t, "https://shop.pyrhouse.space", cfg.Shop.URL)
	assert.Equal(t, []string{"https://pyrhouse.space", "https://shop.pyrhouse.space"}, cfg.CORS.AllowedOrigins)
	assert.Contains(t, cfg.CORS.AllowedHeaders, "Idempotency-Key")
}
