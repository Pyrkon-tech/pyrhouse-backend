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
			name: "Basic Case",
			asset: models.Asset{
				ID: 123,
				Category: models.ItemCategory{
					PyrID: "LT",
				},
			},
			expected: "PYR-LT123",
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
