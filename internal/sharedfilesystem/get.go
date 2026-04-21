package sharedfilesystem

import (
	"context"
	"fmt"

	"github.com/gophercloud/gophercloud/v2"
	"github.com/gophercloud/gophercloud/v2/openstack/sharedfilesystems/v2/shares"

	"snapshot-cli/internal/util"
)

// GetSharedFileSystem retrieves the Manila share identified by shareID and writes
// its details to stdout in the requested output format. Caller supplies the client.
func GetSharedFileSystem(ctx context.Context, shareID, output string, sharedClient *gophercloud.ServiceClient) error {
	if err := util.ValidateUUID(shareID); err != nil {
		return err
	}
	nfs, err := shares.Get(ctx, sharedClient, shareID).Extract()
	if err != nil {
		return err
	}

	if nfs == nil || nfs.ID == "" {
		fmt.Println("NFS share not found")
		return nil
	}

	return util.Render(output, nfs, nfsHeader)
}
