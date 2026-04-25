package util

import (
	"fmt"
	"regexp"
)

var uuidRE = regexp.MustCompile(`(?i)^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

// ValidateUUID returns an error if id is not a valid UUID.
func ValidateUUID(id string) error {
	if !uuidRE.MatchString(id) {
		return fmt.Errorf("invalid ID %q: expected UUID format xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx", id)
	}
	return nil
}
