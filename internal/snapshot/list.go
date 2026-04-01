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

// ListSnapshotsCmd lists all snapshots for either block storage (snapOpts.Volume)
// or shared filesystems (snapOpts.Share).
// If snapOpts.client is already set (e.g. in tests), auth is skipped.
func ListSnapshotsCmd(ctx context.Context, snapOpts *SnapShotOpts, output string) error {
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
		default:
			return fmt.Errorf("unsupported output format: %q", output)
		}

	case snapOpts.Share:
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
		default:
			return fmt.Errorf("unsupported output format: %q", output)
		}
	}
	return nil
}
