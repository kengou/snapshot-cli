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

// DeleteSnapshotCmd deletes the snapshot identified by snapOpts.SnapshotID.
// Set snapOpts.Volume for block storage or snapOpts.Share for shared filesystems.
// If snapOpts.client is already set (e.g. in tests), auth is skipped.
func DeleteSnapshotCmd(ctx context.Context, snapOpts *SnapShotOpts, output string) error {
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
		result := blockSnapshot.Delete(ctx, snapOpts.client, snapOpts.SnapshotID)
		if result.Err != nil {
			return result.Err
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
		result := nfsSnapshot.Delete(ctx, snapOpts.client, snapOpts.SnapshotID)
		if result.Err != nil {
			return result.Err
		}
		switch output {
		case util.OutputTable:
			return util.WriteAsTable(result, snapshotBlockHeader)
		case util.OutputJSON:
			return util.WriteJSON(result)
		default:
			return fmt.Errorf("unsupported output format: %q", output)
		}
	}
	return nil
}
