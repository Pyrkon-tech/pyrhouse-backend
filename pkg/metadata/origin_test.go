package metadata

import (
	"testing"
)

// TestIsValid tests the IsValid method of the Origin type.
func TestIsValid(t *testing.T) {
	tests := []struct {
		name     string
		origin   Origin
		expected bool
	}{
		{"druga-era origin", OriginDrugaEra, true},
		{"probis origin", OriginProbis, true},
		{"targi origin", OriginTargi, true},
		{"personal origin", OriginPersonal, false},
		{"unknown origin", Origin("unknown"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.origin.IsValid(); got != tt.expected {
				t.Errorf("IsValid() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestNewOrigin(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"valid druga-era", "druga-era", false},
		{"valid uppercase PROBIS", "PROBIS", false},
		{"invalid unknown", "unknown", true},
		{"valid personal with spaces", "  personal ", false},
		{"valid targowe", "targowe", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewOrigin(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewOrigin() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err == nil && !got.IsValid() && !got.isPredefined() {
				t.Errorf("NewOrigin() = %v is neither valid nor predefined", got)
			}
		})
	}
}

func TestContainsKeyword(t *testing.T) {
	tests := []struct {
		name     string
		origin   Origin
		keyword  string
		expected bool
	}{
		{"druga-era contains druga", OriginDrugaEra, "druga", true},
		{"probis contains pro", OriginProbis, "pro", true},
		{"personal contains sonal", OriginPersonal, "sonal", true},
		{"unknown contains know", Origin("unknown"), "know", true},
		{"case sensitive match", Origin("targowe"), "PROBIS", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.origin.ContainsKeyword(tt.keyword); got != tt.expected {
				t.Errorf("ContainsKeyword() = %v, want %v", got, tt.expected)
			}
		})
	}
}
