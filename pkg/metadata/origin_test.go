package metadata

import (
	"testing"
)

// TestIsValid tests the IsValid method of the Origin type.
func TestIsValid(t *testing.T) {
	tests := []struct {
		origin       Origin
		expectedBool bool
	}{
		{OriginDrugaEra, true},
		{OriginProbis, true},
		{OriginTargi, true},
		{OriginPersonal, false}, // Not predefined but handled with ContainsKeyword.
		{Origin("unknown"), false},
	}

	for _, tt := range tests {
		t.Run(string(tt.origin), func(t *testing.T) {
			if isValid := tt.origin.IsValid(); isValid != tt.expectedBool {
				t.Errorf("Expected %v for %s, got %v", tt.expectedBool, tt.origin, isValid)
			}
		})
	}
}

func TestNewOrigin(t *testing.T) {
	tests := []struct {
		input         string
		expectedError bool
	}{
		{"druga-era", false},
		{"PROBIS", false},      // Should be converted to lowercase.
		{"unknown", true},      // Should fail as it's not predefined.
		{"  personal ", false}, // Should trim spaces and normalize.
		{"targowe", false},     // Valid value.
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			_, err := NewOrigin(tt.input)
			if tt.expectedError && err == nil {
				t.Errorf("Expected error for input %s, but got none", tt.input)
			}
			if !tt.expectedError && err != nil {
				t.Errorf("Did not expect error for input %s, but got %v", tt.input, err)
			}
		})
	}
}

func TestContainsKeyword(t *testing.T) {
	tests := []struct {
		origin   Origin
		keyword  string
		expected bool
	}{
		{OriginDrugaEra, "druga", true},
		{OriginProbis, "pro", true},
		{OriginPersonal, "sonal", true},
		{Origin("unknown"), "know", true},
		{Origin("targowe"), "PROBIS", false}, // Should be case-sensitive.
	}

	for _, tt := range tests {
		t.Run(string(tt.origin), func(t *testing.T) {
			if result := tt.origin.ContainsKeyword(tt.keyword); result != tt.expected {
				t.Errorf("Expected %v for origin %s with keyword %s, got %v", tt.expected, tt.origin, tt.keyword, result)
			}
		})
	}
}
