package util

import (
	"regexp"
	"strings"
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

// TestValidateUUID_AdversarialInputs covers injection-style attacks and
// encoding tricks that a naive validator might miss.
func TestValidateUUID_AdversarialInputs(t *testing.T) {
	rejected := []string{
		"12345678-1234-1234-1234-12345678901\x00",    // trailing null byte
		"\x0012345678-1234-1234-1234-123456789012",   // leading null byte
		"12345678-1234-1234-1234-12345\x00789012",    // embedded null
		"12345678-1234-1234-1234-123456789012\n",     // trailing newline
		"\t12345678-1234-1234-1234-123456789012",     // leading tab
		"12345678-1234-1234-1234-123456789012\u00a0", // non-breaking space
		"12345678\u2013 1234-1234-1234-123456789012", // en-dash, not ASCII dash
		"12345678_1234-1234-1234-123456789012",       // underscore in slot 0
		"12345678-1234_1234-1234-123456789012",       // underscore in slot 1
		"1234567-81234-1234-1234-123456789012",       // dashes shifted
		"12345678-1234--234-1234-123456789012",       // double dash
		"12345678-1234-1234-1234-123456789012 extra", // suffix payload
		"12345678-1234-1234-1234-123456789012; DROP", // SQL-injection-shaped
		"12345678-1234-1234-1234-12345678901\u202e2", // right-to-left override
		"12345678-1234-1234-1234-123456789012\r\n",   // CRLF
		"12345678-1234-1234-1234-1234567890\xff\xff", // invalid UTF-8 bytes
	}
	for _, input := range rejected {
		if err := ValidateUUID(input); err == nil {
			t.Errorf("ValidateUUID(%q) should fail, but got nil error", input)
		}
	}
}

// FuzzValidateUUID confirms the validator never panics on any input
// and that its accept/reject decision is consistent with the canonical
// UUID regex (no false positives that could slip past into API calls).
func FuzzValidateUUID(f *testing.F) {
	seeds := []string{
		"",
		"12345678-1234-1234-1234-123456789012",
		"AAAAAAAA-AAAA-AAAA-AAAA-AAAAAAAAAAAA",
		"00000000-0000-0000-0000-000000000000",
		"not-a-uuid",
		" 12345678-1234-1234-1234-123456789012",
		"12345678-1234-1234-1234-123456789012\x00",
		"12345678-1234-1234-1234-12345678901\u202e2",
		strings.Repeat("a", 36),
		strings.Repeat("-", 36),
	}
	for _, s := range seeds {
		f.Add(s)
	}

	// Independent reference: this regex is explicit about anchoring and character
	// classes, mirroring the production validator. If they disagree, we have a bug.
	ref := regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

	f.Fuzz(func(t *testing.T, s string) {
		err := ValidateUUID(s)
		wantAccept := ref.MatchString(s)
		gotAccept := err == nil
		if wantAccept != gotAccept {
			t.Errorf("ValidateUUID(%q) = accept:%v, reference regex = accept:%v", s, gotAccept, wantAccept)
		}
	})
}
