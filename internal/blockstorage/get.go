package blockstorage

import (
	"context"
	"fmt"

	"github.com/gophercloud/gophercloud/v2"
	"github.com/gophercloud/gophercloud/v2/openstack/blockstorage/v3/volumes"

	"snapshot-cli/internal/util"
)

// GetBlockStorage retrieves the Cinder volume identified by volID and writes
// its details to stdout in the requested output format. The caller supplies
// the gophercloud client so auth is decoupled from the business logic.
func GetBlockStorage(ctx context.Context, volID, output string, blockClient *gophercloud.ServiceClient) error {
	if err := util.ValidateUUID(volID); err != nil {
		return err
	}
	vol, err := volumes.Get(ctx, blockClient, volID).Extract()
	if err != nil {
		return err
	}

	if vol == nil || vol.ID == "" {
		fmt.Println("No blockstorage volume found")
		return nil
	}

	return util.Render(output, vol, volumeHeader)
}
