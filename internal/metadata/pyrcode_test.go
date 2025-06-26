package metadata

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewPyrCode(t *testing.T) {
	tests := []struct {
		name     string
		pyrID    string
		assetID  int
		expected string
	}{
		{
			name:     "Basic Case",
			pyrID:    "LT",
			assetID:  123,
			expected: "PYR-LT123",
		},
		{
			name:     "Different Category",
			pyrID:    "KB",
			assetID:  456,
			expected: "PYR-KB456",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pyrCode := NewPyrCode(tt.pyrID, tt.assetID)
			actual := pyrCode.GeneratePyrCode()
			assert.Equal(t, tt.expected, actual)
		})
	}
}
