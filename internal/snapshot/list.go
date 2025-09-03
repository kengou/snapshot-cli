package snapshot

import (
	"context"

	blockSnapshot "github.com/gophercloud/gophercloud/v2/openstack/blockstorage/v3/snapshots"
	nfsSnapshot "github.com/gophercloud/gophercloud/v2/openstack/sharedfilesystems/v2/snapshots"

	"snapshot-cli/internal/auth"
	"snapshot-cli/internal/config"
	"snapshot-cli/internal/util"
)

func ListSnapshotsCmd(ctx context.Context, snapOpts *SnapShotOpts, output string) error {
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

		pages, err := blockSnapshot.List(snapOpts.client, blockSnapshot.ListOpts{}).AllPages(ctx)
		if err != nil {
			return err
		}
		allSnapshots, err := blockSnapshot.ExtractSnapshots(pages)
		if err != nil {
			return err
		}

		switch output {
		case util.OutputTable:
			return util.WriteAsTable(allSnapshots, snapshotBlockHeader)
		case util.OutputJSON:
			return util.WriteJSON(allSnapshots)
		}
	case snapOpts.Share:
		snapOpts.client, err = auth.NewSharedFileSystemClient(ctx, authConfig)
		if err != nil {
			return err
		}

		pages, err := nfsSnapshot.ListDetail(snapOpts.client, nfsSnapshot.ListOpts{}).AllPages(ctx)
		if err != nil {
			return err
		}
		allSnapshots, err := nfsSnapshot.ExtractSnapshots(pages)
		if err != nil {
			return err
		}

		switch output {
		case util.OutputTable:
			return util.WriteAsTable(allSnapshots, snapshotNfsHeader)
		case util.OutputJSON:
			return util.WriteJSON(allSnapshots)
		}
	}
	return nil
}
