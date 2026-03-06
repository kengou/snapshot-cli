package util

import (
	"testing"
)

func TestValidateUUID_ValidFormats(t *testing.T) {
	tests := []string{
		"aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
		"AAAAAAAA-AAAA-AAAA-AAAA-AAAAAAAAAAAA",
		"12345678-1234-1234-1234-123456789012",
		"f47ac10b-58cc-4372-a567-0e02b2c3d479",
		"F47AC10B-58CC-4372-A567-0E02B2C3D479",
	}
	for _, uuid := range tests {
		if err := ValidateUUID(uuid); err != nil {
			t.Errorf("ValidateUUID(%q) should pass, got error: %v", uuid, err)
		}
	}
}

func TestValidateUUID_InvalidFormats(t *testing.T) {
	tests := []string{
		"",
		"not-a-uuid",
		"12345678-1234-1234-1234",
		"12345678123412341234123456789012",
		"12345678-1234-1234-1234-12345678901",       // 11 chars in last segment
		"12345678-1234-1234-1234-1234567890123",     // 13 chars in last segment
		"gggggggg-gggg-gggg-gggg-gggggggggggg",      // invalid hex
		"12345678_1234_1234_1234_123456789012",      // underscores instead of dashes
		"12345678-1234-1234-1234-123456789012extra", // extra chars
	}
	for _, uuid := range tests {
		if err := ValidateUUID(uuid); err == nil {
			t.Errorf("ValidateUUID(%q) should fail, but got nil error", uuid)
		}
	}
}

func TestValidateUUID_EdgeCases(t *testing.T) {
	tests := []struct {
		uuid string
		want bool
	}{
		{"00000000-0000-0000-0000-000000000000", true},   // all zeros
		{"ffffffff-ffff-ffff-ffff-ffffffffffff", true},   // all f's
		{"12345678-1234-1234-1234-123456789012", true},   // mixed
		{" 12345678-1234-1234-1234-123456789012", false}, // leading space
		{"12345678-1234-1234-1234-123456789012 ", false}, // trailing space
	}
	for _, tt := range tests {
		err := ValidateUUID(tt.uuid)
		if tt.want && err != nil {
			t.Errorf("ValidateUUID(%q) should pass, got error: %v", tt.uuid, err)
		}
		if !tt.want && err == nil {
			t.Errorf("ValidateUUID(%q) should fail, but got nil error", tt.uuid)
		}
	}
}
