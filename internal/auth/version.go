package auth

import (
	"context"
	"fmt"

	"github.com/gophercloud/gophercloud/v2"
	"github.com/gophercloud/gophercloud/v2/openstack"
)

// ServiceVersions holds detected OpenStack service API versions.
type ServiceVersions struct {
	CinderVersion string // Detected version: "v3"
	ManilaVersion string // Detected version: "v2"
}

// DetectVersions validates that required OpenStack services are available.
// It attempts to initialize Cinder v3 and Manila v2 clients to confirm versions are available.
// Returns error if required services are not found or incompatible.
func DetectVersions(ctx context.Context, provider *gophercloud.ProviderClient) (*ServiceVersions, error) {
	sv := &ServiceVersions{}

	// Try to initialize Cinder v3 client - this validates the service is available
	_, err := openstack.NewBlockStorageV3(provider, gophercloud.EndpointOpts{
		Availability: gophercloud.AvailabilityPublic,
	})
	if err != nil {
		return nil, fmt.Errorf("block storage (Cinder) service not available or incompatible version detected; ensure Cinder v3 is enabled: %w", err)
	}
	sv.CinderVersion = "v3"

	// Try to initialize Manila v2 client - this validates the service is available
	_, err = openstack.NewSharedFileSystemV2(provider, gophercloud.EndpointOpts{
		Availability: gophercloud.AvailabilityPublic,
	})
	if err != nil {
		return nil, fmt.Errorf("shared filesystem (Manila) service not available or incompatible version detected; ensure Manila v2 is enabled: %w", err)
	}
	sv.ManilaVersion = "v2"

	return sv, nil
}
