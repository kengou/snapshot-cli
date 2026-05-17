package snapshot

import (
	"testing"
	"time"
)

func TestParseDurationOrFallback_ValidDuration(t *testing.T) {
	tests := []struct {
		input string
		want  time.Duration
	}{
		{"1h", time.Hour},
		{"24h", 24 * time.Hour},
		{"30m", 30 * time.Minute},
		{"168h", 168 * time.Hour},
		{"72h", 72 * time.Hour},
		{"0s", 0},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := parseDurationOrFallback(tt.input)
			if got != tt.want {
				t.Errorf("parseDurationOrFallback(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestParseDurationOrFallback_InvalidInput_Returns7Days(t *testing.T) {
	fallback := 168 * time.Hour

	invalid := []string{
		"",
		"invalid",
		"1d",    // Go doesn't support days
		"1week", // not a valid Go duration
		"abc",
		"999", // number without unit
		"-1h", // negative - actually valid in Go, but keep to show behavior
	}

	// Filter: only test truly invalid ones (negative is valid in Go)
	trulyInvalid := []string{"", "invalid", "1d", "1week", "abc", "999"}

	for _, input := range trulyInvalid {
		t.Run("invalid:"+input, func(t *testing.T) {
			got := parseDurationOrFallback(input)
			if got != fallback {
				t.Errorf("parseDurationOrFallback(%q) = %v, want fallback %v", input, got, fallback)
			}
		})
	}

	_ = invalid
}

func TestParseDurationOrFallback_NegativeDuration_ParsedAsIs(t *testing.T) {
	// Go's time.ParseDuration accepts negative durations
	got := parseDurationOrFallback("-1h")
	if got != -time.Hour {
		t.Errorf("parseDurationOrFallback(%q) = %v, want %v", "-1h", got, -time.Hour)
	}
}

func TestParseDurationOrFallback_FallbackIs7Days(t *testing.T) {
	const expected = 168 * time.Hour
	got := parseDurationOrFallback("not-a-duration")
	if got != expected {
		t.Errorf("fallback should be 168h (7 days), got %v", got)
	}
}

// FuzzParseDurationOrFallback verifies that parseDurationOrFallback never panics
// and always returns either the parsed duration or the 7-day fallback.
func FuzzParseDurationOrFallback(f *testing.F) {
	// Seed corpus: valid durations, invalid strings, edge cases
	for _, seed := range []string{"168h", "1h30m", "0s", "", "invalid", "1d", "999", "-1h", "99999h"} {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, s string) {
		result := parseDurationOrFallback(s)
		// Must always return the fallback (168h) for unparseable input,
		// or the parsed value for valid input — never panic.
		parsed, err := time.ParseDuration(s)
		if err != nil {
			if result != 168*time.Hour {
				t.Errorf("parseDurationOrFallback(%q): invalid input should return 168h fallback, got %v", s, result)
			}
		} else {
			if result != parsed {
				t.Errorf("parseDurationOrFallback(%q): valid input should return %v, got %v", s, parsed, result)
			}
		}
	})
}
