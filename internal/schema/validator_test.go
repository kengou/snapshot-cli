package schema

import (
	"testing"

	"github.com/gophercloud/gophercloud/v2"
)

func TestValidateCinderV3_Success(t *testing.T) {
	client := &gophercloud.ServiceClient{
		ProviderClient: &gophercloud.ProviderClient{},
		Endpoint:       "https://cinder.example.com/v3",
	}

	result := ValidateCinderV3(client)
	if !result.Valid {
		t.Errorf("ValidateCinderV3() Valid = %v, want true", result.Valid)
	}
	if result.Message != "Cinder v3 validation successful" {
		t.Errorf("ValidateCinderV3() Message = %q, want %q", result.Message, "Cinder v3 validation successful")
	}
}

func TestValidateCinderV3_NilClient(t *testing.T) {
	result := ValidateCinderV3(nil)
	if result.Valid {
		t.Errorf("ValidateCinderV3(nil) Valid = %v, want false", result.Valid)
	}
	if result.Message != "Cinder client is nil" {
		t.Errorf("ValidateCinderV3(nil) Message = %q, want nil client message", result.Message)
	}
}

func TestValidateCinderV3_EmptyEndpoint(t *testing.T) {
	client := &gophercloud.ServiceClient{
		ProviderClient: &gophercloud.ProviderClient{},
		Endpoint:       "",
	}

	result := ValidateCinderV3(client)
	if result.Valid {
		t.Errorf("ValidateCinderV3() Valid = %v, want false for empty endpoint", result.Valid)
	}
}

func TestValidateManilaV2_Success(t *testing.T) {
	client := &gophercloud.ServiceClient{
		ProviderClient: &gophercloud.ProviderClient{},
		Endpoint:       "https://manila.example.com/v2",
	}

	result := ValidateManilaV2(client)
	if !result.Valid {
		t.Errorf("ValidateManilaV2() Valid = %v, want true", result.Valid)
	}
	if result.Message != "Manila v2 validation successful" {
		t.Errorf("ValidateManilaV2() Message = %q, want %q", result.Message, "Manila v2 validation successful")
	}
}

func TestValidateManilaV2_NilClient(t *testing.T) {
	result := ValidateManilaV2(nil)
	if result.Valid {
		t.Errorf("ValidateManilaV2(nil) Valid = %v, want false", result.Valid)
	}
	if result.Message != "Manila client is nil" {
		t.Errorf("ValidateManilaV2(nil) Message = %q, want nil client message", result.Message)
	}
}

func TestValidateManilaV2_EmptyEndpoint(t *testing.T) {
	client := &gophercloud.ServiceClient{
		ProviderClient: &gophercloud.ProviderClient{},
		Endpoint:       "",
	}

	result := ValidateManilaV2(client)
	if result.Valid {
		t.Errorf("ValidateManilaV2() Valid = %v, want false for empty endpoint", result.Valid)
	}
}
