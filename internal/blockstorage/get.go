package blockstorage

import (
	"context"
	"fmt"

	"github.com/gophercloud/gophercloud/v2/openstack/blockstorage/v3/volumes"

	"snapshot-cli/internal/auth"
	"snapshot-cli/internal/config"
	"snapshot-cli/internal/util"
)

func RunGetBlockStorage(ctx context.Context, volID, output string) error {
	authConfig, err := config.ReadAuthConfig()
	if err != nil {
		return err
	}
	blockClient, err := auth.NewBlockStorageClient(ctx, authConfig)
	if err != nil {
		return err
	}

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
	}

	return nil
}
