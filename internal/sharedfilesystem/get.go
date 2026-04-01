package sharedfilesystem

import (
	"context"
	"fmt"

	"github.com/gophercloud/gophercloud/v2"
	"github.com/gophercloud/gophercloud/v2/openstack/sharedfilesystems/v2/shares"

	"snapshot-cli/internal/auth"
	"snapshot-cli/internal/config"
	"snapshot-cli/internal/util"
)

// RunGetSharedFileSystem retrieves the Manila share identified by shareID and writes
// its details to stdout in the requested output format.
func RunGetSharedFileSystem(ctx context.Context, shareID, output string) error {
	if err := util.ValidateUUID(shareID); err != nil {
		return err
	}
	authConfig, err := config.ReadAuthConfig()
	if err != nil {
		return err
	}
	sharedClient, err := auth.NewSharedFileSystemClient(ctx, authConfig)
	if err != nil {
		return err
	}
	return getSharedFileSystem(ctx, shareID, output, sharedClient)
}

// getSharedFileSystem is the testable core: it accepts a pre-built client.
func getSharedFileSystem(ctx context.Context, shareID, output string, sharedClient *gophercloud.ServiceClient) error {
	nfs, err := shares.Get(ctx, sharedClient, shareID).Extract()
	if err != nil {
		return err
	}

	if nfs == nil {
		fmt.Println("NFS share not found")
		return nil
	}

	switch output {
	case util.OutputTable:
		return util.WriteAsTable(nfs, nfsHeader)
	case util.OutputJSON:
		return util.WriteJSON(nfs)
	default:
		return fmt.Errorf("unsupported output format: %q", output)
	}
}
