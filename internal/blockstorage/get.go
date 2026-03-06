package blockstorage

import (
	"context"
	"fmt"

	"github.com/gophercloud/gophercloud/v2"
	"github.com/gophercloud/gophercloud/v2/openstack/blockstorage/v3/volumes"

	"snapshot-cli/internal/auth"
	"snapshot-cli/internal/config"
	"snapshot-cli/internal/util"
)

// RunGetBlockStorage retrieves the Cinder volume identified by volID and writes
// its details to stdout in the requested output format.
func RunGetBlockStorage(ctx context.Context, volID, output string) error {
	if err := util.ValidateUUID(volID); err != nil {
		return err
	}
	authConfig, err := config.ReadAuthConfig()
	if err != nil {
		return err
	}
	blockClient, err := auth.NewBlockStorageClient(ctx, authConfig)
	if err != nil {
		return err
	}
	return getBlockStorage(ctx, volID, output, blockClient)
}

// getBlockStorage is the testable core: it accepts a pre-built client.
func getBlockStorage(ctx context.Context, volID, output string, blockClient *gophercloud.ServiceClient) error {
	vol, err := volumes.Get(ctx, blockClient, volID).Extract()
	if err != nil {
		return err
	}

	if vol == nil {
		fmt.Println("No blockstorage volume found")
		return nil
	}

	switch output {
	case util.OutputTable:
		return util.WriteAsTable(vol, volumeHeader)
	case util.OutputJSON:
		return util.WriteJSON(vol)
	default:
		return fmt.Errorf("unsupported output format: %q", output)
	}
}
