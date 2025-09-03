package auth

import (
	"context"

	"github.com/gophercloud/gophercloud/v2"
	"github.com/gophercloud/gophercloud/v2/openstack"
	"github.com/gophercloud/gophercloud/v2/openstack/identity/v3/tokens"

	"snapshot-cli/internal/config"
)

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
	provider.UseTokenLock()

	err = openstack.AuthenticateV3(ctx, provider, opts, gophercloud.EndpointOpts{})
	provider.UserAgent.Prepend("snapshot-creator")

	if provider.TokenID == "" {
		return nil, err
	}

	return provider, err
}
