package snapshot

import (
	"context"

	blockSnapshot "github.com/gophercloud/gophercloud/v2/openstack/blockstorage/v3/snapshots"
	nfsSnapshot "github.com/gophercloud/gophercloud/v2/openstack/sharedfilesystems/v2/snapshots"

	"snapshot-cli/internal/auth"
	"snapshot-cli/internal/config"
	"snapshot-cli/internal/util"
)

func DeleteSnapshotCmd(ctx context.Context, snapOpts *SnapShotOpts, output string) error {
	authConfig, err := config.ReadAuthConfig()
	if err != nil {
		return err
	}

	switch {
	case snapOpts.Volume:
		snapOpts.client, err = auth.NewBlockStorageClient(ctx, authConfig)
		if err != nil {
			return err
		}
		result := blockSnapshot.Delete(ctx, snapOpts.client, snapOpts.SnapshotID)
		if result.Err != nil {
			return result.Err
		}
		switch output {
		case util.OutputTable:
			return util.WriteAsTable(result, snapshotBlockHeader)
		case util.OutputJSON:
			return util.WriteJSON(result)
		}
	case snapOpts.Share:
		snapOpts.client, err = auth.NewSharedFileSystemClient(ctx, authConfig)
		if err != nil {
			return err
		}
		result := nfsSnapshot.Delete(ctx, snapOpts.client, snapOpts.SnapshotID)
		if result.Err != nil {
			return result.Err
		}
		switch output {
		case util.OutputTable:
			return util.WriteAsTable(result, snapshotBlockHeader)
		case util.OutputJSON:
			return util.WriteJSON(result)
		}
	}
	return nil
}
