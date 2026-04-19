package auth

import (
	"testing"
)

// Note: Full integration tests for DetectVersions require a real OpenStack environment
// or complex HTTP mocking. Unit test coverage is limited by gophercloud v2's
// internal endpoint resolution logic. Validation is primarily tested via E2E tests
// in cmd_integration_test.go with mocked OpenStack endpoints.

func TestDetectVersions_DocumentationOnly(t *testing.T) {
	// This test documents the DetectVersions function behavior.
	// Actual testing is done in cmd_integration_test.go where we mock
	// OpenStack API responses and validate version detection works end-to-end.
	//
	// DetectVersions(ctx, provider) returns:
	// - ServiceVersions{CinderVersion: "v3", ManilaVersion: "v2"} on success
	// - error if either Cinder v3 or Manila v2 cannot be initialized
	//
	// To test this function:
	// 1. Set up OpenStack environment variables (OS_AUTH_URL, etc.)
	// 2. Run against a real OpenStack instance
	// 3. Or use mocked HTTP endpoints in integration tests
	t.Logf("DetectVersions validates Cinder v3 and Manila v2 availability")
}
