package snapshot

import (
	"context"

	blockSnapshot "github.com/gophercloud/gophercloud/v2/openstack/blockstorage/v3/snapshots"
	nfsSnapshot "github.com/gophercloud/gophercloud/v2/openstack/sharedfilesystems/v2/snapshots"

	"snapshot-cli/internal/auth"
	"snapshot-cli/internal/config"
	"snapshot-cli/internal/util"
)

func GetSnapshotCmd(ctx context.Context, snapOpts *SnapShotOpts, output string) error {
	authConfig, err := config.ReadAuthConfig()
	if err != nil {
		return err
	}

	if snapOpts.VolumeID != "" {
		snapOpts.client, err = auth.NewBlockStorageClient(ctx, authConfig)
		if err != nil {
			return err
		}

		snapShot := blockSnapshot.Get(ctx, snapOpts.client, snapOpts.VolumeID)
		result, err := snapShot.Extract()
		if err != nil {
			return err
		}

		switch output {
		case util.OutputTable:
			return util.WriteAsTable(result, snapshotBlockHeader)
		case util.OutputJSON:
			return util.WriteJSON(result)
		}
	} else if snapOpts.ShareID != "" {
		snapOpts.client, err = auth.NewSharedFileSystemClient(ctx, authConfig)
		if err != nil {
			return err
		}

		snapShot := nfsSnapshot.Get(ctx, snapOpts.client, snapOpts.ShareID)
		result, err := snapShot.Extract()

		if err != nil {
			return err
		}

		switch output {
		case util.OutputTable:
			return util.WriteAsTable(result, snapshotNfsHeader)
		case util.OutputJSON:
			return util.WriteJSON(result)
		}
	}

	return nil
}
