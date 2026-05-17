package schema

import (
	"github.com/gophercloud/gophercloud/v2"
)

// ValidationResult holds the outcome of endpoint validation.
type ValidationResult struct {
	Valid   bool
	Message string
	Details string
}

// ValidateCinderV3 checks if a Cinder v3 client was successfully initialized.
// A successful initialization indicates the Cinder v3 endpoint is reachable.
func ValidateCinderV3(client *gophercloud.ServiceClient) *ValidationResult {
	if client == nil {
		return &ValidationResult{
			Valid:   false,
			Message: "Cinder client is nil",
			Details: "Failed to initialize Cinder v3 client",
		}
	}

	endpoint := client.ServiceURL()
	if endpoint == "" {
		return &ValidationResult{
			Valid:   false,
			Message: "Cinder endpoint is empty",
			Details: "Service URL could not be determined",
		}
	}

	return &ValidationResult{
		Valid:   true,
		Message: "Cinder v3 validation successful",
		Details: "Endpoint: " + endpoint,
	}
}

// ValidateManilaV2 checks if a Manila v2 client was successfully initialized.
// A successful initialization indicates the Manila v2 endpoint is reachable.
func ValidateManilaV2(client *gophercloud.ServiceClient) *ValidationResult {
	if client == nil {
		return &ValidationResult{
			Valid:   false,
			Message: "Manila client is nil",
			Details: "Failed to initialize Manila v2 client",
		}
	}

	endpoint := client.ServiceURL()
	if endpoint == "" {
		return &ValidationResult{
			Valid:   false,
			Message: "Manila endpoint is empty",
			Details: "Service URL could not be determined",
		}
	}

	return &ValidationResult{
		Valid:   true,
		Message: "Manila v2 validation successful",
		Details: "Endpoint: " + endpoint,
	}
}
