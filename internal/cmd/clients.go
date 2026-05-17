package cmd

import (
	"context"

	"github.com/gophercloud/gophercloud/v2"

	"snapshot-cli/internal/auth"
	"snapshot-cli/internal/config"
)

// buildBlockClient reads the auth config and returns a Cinder v3 client.
// Centralised here so the business packages stay free of config/auth imports.
func buildBlockClient(ctx context.Context) (*gophercloud.ServiceClient, error) {
	authConfig, err := config.ReadAuthConfig()
	if err != nil {
		return nil, err
	}
	return auth.NewBlockStorageClient(ctx, authConfig)
}

// buildSharedClient reads the auth config and returns a Manila v2 client.
func buildSharedClient(ctx context.Context) (*gophercloud.ServiceClient, error) {
	authConfig, err := config.ReadAuthConfig()
	if err != nil {
		return nil, err
	}
	return auth.NewSharedFileSystemClient(ctx, authConfig)
}
