package auth

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/gophercloud/gophercloud/v2"
)

// newProviderWithLocator builds a ProviderClient whose EndpointLocator
// returns the caller-supplied endpoint/error for the given service type.
// Any unexpected service type returns ErrEndpointNotFound.
func newProviderWithLocator(locator gophercloud.EndpointLocator) *gophercloud.ProviderClient {
	return &gophercloud.ProviderClient{
		IdentityEndpoint: "http://example.invalid/identity/",
		EndpointLocator:  locator,
	}
}

func TestDetectVersions_BothServicesAvailable(t *testing.T) {
	provider := newProviderWithLocator(func(eo gophercloud.EndpointOpts) (string, error) {
		switch eo.Type {
		case "block-storage", "volumev3":
			return "http://cinder.invalid/v3/", nil
		case "shared-file-system", "sharev2":
			return "http://manila.invalid/v2/", nil
		}
		return "", gophercloud.ErrEndpointNotFound{}
	})

	sv, err := DetectVersions(context.Background(), provider)
	if err != nil {
		t.Fatalf("DetectVersions returned error: %v", err)
	}
	if sv.CinderVersion != "v3" {
		t.Errorf("CinderVersion = %q, want v3", sv.CinderVersion)
	}
	if sv.ManilaVersion != "v2" {
		t.Errorf("ManilaVersion = %q, want v2", sv.ManilaVersion)
	}
}

func TestDetectVersions_CinderMissing(t *testing.T) {
	provider := newProviderWithLocator(func(eo gophercloud.EndpointOpts) (string, error) {
		switch eo.Type {
		case "shared-file-system", "sharev2":
			return "http://manila.invalid/v2/", nil
		}
		return "", gophercloud.ErrEndpointNotFound{}
	})

	sv, err := DetectVersions(context.Background(), provider)
	if err == nil {
		t.Fatalf("expected error when Cinder endpoint missing, got nil (sv=%+v)", sv)
	}
	if !strings.Contains(err.Error(), "Cinder") {
		t.Errorf("error should mention Cinder, got: %v", err)
	}
}

func TestDetectVersions_ManilaMissing(t *testing.T) {
	provider := newProviderWithLocator(func(eo gophercloud.EndpointOpts) (string, error) {
		switch eo.Type {
		case "block-storage", "volumev3":
			return "http://cinder.invalid/v3/", nil
		}
		return "", gophercloud.ErrEndpointNotFound{}
	})

	sv, err := DetectVersions(context.Background(), provider)
	if err == nil {
		t.Fatalf("expected error when Manila endpoint missing, got nil (sv=%+v)", sv)
	}
	if !strings.Contains(err.Error(), "Manila") {
		t.Errorf("error should mention Manila, got: %v", err)
	}
}

func TestDetectVersions_BothMissing(t *testing.T) {
	provider := newProviderWithLocator(func(eo gophercloud.EndpointOpts) (string, error) {
		return "", gophercloud.ErrEndpointNotFound{}
	})

	sv, err := DetectVersions(context.Background(), provider)
	if err == nil {
		t.Fatalf("expected error when no endpoints available, got nil (sv=%+v)", sv)
	}
	// Cinder is checked first, so that's the one we expect to see in the error.
	if !strings.Contains(err.Error(), "Cinder") {
		t.Errorf("error should mention Cinder (checked first), got: %v", err)
	}
}

func TestDetectVersions_LocatorReturnsGenericError(t *testing.T) {
	sentinel := errors.New("keystone unreachable")
	provider := newProviderWithLocator(func(eo gophercloud.EndpointOpts) (string, error) {
		return "", sentinel
	})

	_, err := DetectVersions(context.Background(), provider)
	if err == nil {
		t.Fatal("expected error when locator fails, got nil")
	}
	if !errors.Is(err, sentinel) {
		t.Errorf("returned error should wrap %v, got: %v", sentinel, err)
	}
}

func TestDetectVersions_CinderMissingDoesNotShortCircuitOnManila(t *testing.T) {
	// Confirm Manila check is skipped entirely when Cinder fails
	// (i.e. we fail fast, not partially).
	manilaCalled := false
	provider := newProviderWithLocator(func(eo gophercloud.EndpointOpts) (string, error) {
		switch eo.Type {
		case "block-storage", "volumev3":
			return "", errors.New("cinder catalog entry missing")
		case "shared-file-system", "sharev2":
			manilaCalled = true
			return "http://manila.invalid/v2/", nil
		}
		return "", gophercloud.ErrEndpointNotFound{}
	})

	_, err := DetectVersions(context.Background(), provider)
	if err == nil {
		t.Fatal("expected error when Cinder fails, got nil")
	}
	if manilaCalled {
		t.Error("Manila locator should not be invoked after Cinder failure; fail fast")
	}
}
