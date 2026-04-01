package auth

import (
	"context"
	"time"

	"github.com/gophercloud/gophercloud/v2"
	"github.com/gophercloud/gophercloud/v2/openstack"
	"github.com/gophercloud/gophercloud/v2/openstack/identity/v3/tokens"

	"snapshot-cli/internal/config"
)

// NewSharedFileSystemClient creates an authenticated Manila (Shared File System v2) service client.
// It performs a full Keystone v3 authentication on every call.
func NewSharedFileSystemClient(ctx context.Context, auth *config.Auth) (*gophercloud.ServiceClient, error) {
	provider, err := newAuthenticatedProviderClient(ctx, auth)
	if err != nil {
		return nil, err
	}

	client, err := openstack.NewSharedFileSystemV2(provider, gophercloud.EndpointOpts{
		Region:       auth.RegionName,
		Availability: gophercloud.AvailabilityPublic,
	})
	if err != nil {
		return nil, err
	}

	return client, nil
}

// NewBlockStorageClient creates an authenticated Cinder (Block Storage v3) service client.
// It performs a full Keystone v3 authentication on every call.
func NewBlockStorageClient(ctx context.Context, auth *config.Auth) (*gophercloud.ServiceClient, error) {
	provider, err := newAuthenticatedProviderClient(ctx, auth)
	if err != nil {
		return nil, err
	}

	client, err := openstack.NewBlockStorageV3(provider, gophercloud.EndpointOpts{
		Region:       auth.RegionName,
		Availability: gophercloud.AvailabilityPublic,
	})
	if err != nil {
		return nil, err
	}

	return client, nil
}

func newAuthenticatedProviderClient(ctx context.Context, auth *config.Auth) (*gophercloud.ProviderClient, error) {
	opts := &tokens.AuthOptions{
		IdentityEndpoint: auth.AuthURL,
		Username:         auth.Username,
		Password:         auth.Password,
		DomainName:       auth.UserDomainName,
		AllowReauth:      true,
		Scope: tokens.Scope{
			ProjectName: auth.ProjectName,
			DomainName:  auth.ProjectDomainName,
		},
	}

	provider, err := openstack.NewClient(auth.AuthURL)
	if err != nil {
		return nil, err
	}
	provider.UserAgent.Prepend("snapshot-creator")
	provider.HTTPClient.Timeout = 30 * time.Second
	provider.UseTokenLock()

	if err = openstack.AuthenticateV3(ctx, provider, opts, gophercloud.EndpointOpts{}); err != nil {
		return nil, err
	}

	return provider, nil
}
