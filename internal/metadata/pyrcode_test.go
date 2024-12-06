package metadata

import (
	"testing"
	"warehouse/pkg/models"

	"github.com/stretchr/testify/assert"
)

func TestNewPyrCode(t *testing.T) {
	tests := []struct {
		name     string
		asset    models.Asset
		expected string
	}{
		{
			name: "Basic Case - No Accessories",
			asset: models.Asset{
				ID: 123,
				Category: models.ItemCategory{
					PyrID: "LT",
				},
				Accessories: []models.AssetAccessories{},
			},
			expected: "PYR-LT123",
		},
		{
			name: "With Accessories",
			asset: models.Asset{
				ID: 456,
				Category: models.ItemCategory{
					PyrID: "PC",
				},
				Accessories: []models.AssetAccessories{
					{Name: "Mouse", Label: "Mouse"},
					{Name: "Keyboard", Label: "Keyboard"},
				},
			},
			expected: "PYR-PC45611", // Accessories represented by "11"
		},
		{
			name: "With One Accessory",
			asset: models.Asset{
				ID: 789,
				Category: models.ItemCategory{
					PyrID: "MT",
				},
				Accessories: []models.AssetAccessories{
					{Name: "Microphone", Label: "Mic"},
				},
			},
			expected: "PYR-MT7891", // Accessories represented by "1"
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pyrCode := NewPyrCode(&tt.asset)
			actual := pyrCode.GeneratePyrCode()
			assert.Equal(t, tt.expected, actual)
		})
	}
}
