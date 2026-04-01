package snapshot

import (
	"context"
	"fmt"

	blockSnapshot "github.com/gophercloud/gophercloud/v2/openstack/blockstorage/v3/snapshots"
	nfsSnapshot "github.com/gophercloud/gophercloud/v2/openstack/sharedfilesystems/v2/snapshots"

	"snapshot-cli/internal/auth"
	"snapshot-cli/internal/config"
	"snapshot-cli/internal/util"
)

// GetSnapshotCmd retrieves a single snapshot by ID.
// Set snapOpts.Volume for block storage or snapOpts.Share for shared filesystems.
// The snapshot ID must be provided in snapOpts.SnapshotID.
// If snapOpts.client is already set (e.g. in tests), auth is skipped.
func GetSnapshotCmd(ctx context.Context, snapOpts *SnapShotOpts, output string) error {
	if err := util.ValidateUUID(snapOpts.SnapshotID); err != nil {
		return err
	}

	if snapOpts.client == nil {
		authConfig, err := config.ReadAuthConfig()
		if err != nil {
			return err
		}
		switch {
		case snapOpts.Volume:
			snapOpts.client, err = auth.NewBlockStorageClient(ctx, authConfig)
		case snapOpts.Share:
			snapOpts.client, err = auth.NewSharedFileSystemClient(ctx, authConfig)
		}
		if err != nil {
			return err
		}
	}

	switch {
	case snapOpts.Volume:
		result, err := blockSnapshot.Get(ctx, snapOpts.client, snapOpts.SnapshotID).Extract()
		if err != nil {
			return err
		}
		switch output {
		case util.OutputTable:
			return util.WriteAsTable(result, snapshotBlockHeader)
		case util.OutputJSON:
			return util.WriteJSON(result)
		default:
			return fmt.Errorf("unsupported output format: %q", output)
		}

	case snapOpts.Share:
		result, err := nfsSnapshot.Get(ctx, snapOpts.client, snapOpts.SnapshotID).Extract()
		if err != nil {
			return err
		}
		switch output {
		case util.OutputTable:
			return util.WriteAsTable(result, snapshotNfsHeader)
		case util.OutputJSON:
			return util.WriteJSON(result)
		default:
			return fmt.Errorf("unsupported output format: %q", output)
		}
	}

	return nil
}
