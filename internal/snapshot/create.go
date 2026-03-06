package snapshot

import (
	"context"
	"fmt"
	"time"

	blockSnapshot "github.com/gophercloud/gophercloud/v2/openstack/blockstorage/v3/snapshots"
	nfsSnapshot "github.com/gophercloud/gophercloud/v2/openstack/sharedfilesystems/v2/snapshots"

	"snapshot-cli/internal/auth"
	"snapshot-cli/internal/config"
	"snapshot-cli/internal/util"
)

// CreateSnapshotCmd creates a snapshot of a block storage volume (snapOpts.VolumeID)
// or a shared filesystem (snapOpts.ShareID). The snapshot name is auto-generated
// from the resource ID and the current UTC timestamp when snapOpts.Name is empty.
// If snapOpts.Cleanup is true, old snapshots older than snapOpts.OlderThan are
// deleted after the new snapshot is created.
// If snapOpts.client is already set (e.g. in tests), auth is skipped.
func CreateSnapshotCmd(ctx context.Context, snapOpts *SnapShotOpts, output string) error {
	// Generate default name if not provided
	if snapOpts.Name == "" {
		if snapOpts.VolumeID != "" {
			snapOpts.Name = snapOpts.VolumeID
		} else if snapOpts.ShareID != "" {
			snapOpts.Name = snapOpts.ShareID
		}
	}
	snapOpts.Name = snapOpts.Name + "-" + time.Now().UTC().Format("200601021504")

	if snapOpts.VolumeID != "" {
		if snapOpts.client == nil {
			authConfig, err := config.ReadAuthConfig()
			if err != nil {
				return err
			}
			snapOpts.client, err = auth.NewBlockStorageClient(ctx, authConfig)
			if err != nil {
				return err
			}
		}
		createOpts := blockSnapshot.CreateOpts{
			VolumeID:    snapOpts.VolumeID,
			Name:        snapOpts.Name,
			Description: snapOpts.Description,
			Force:       snapOpts.Force,
		}
		snapshotResult := blockSnapshot.Create(ctx, snapOpts.client, createOpts)
		result, err := snapshotResult.Extract()
		if err != nil {
			return err
		}
		if snapOpts.Cleanup {
			snapOpts.Volume = true
			if err = CleanupSnapshot(ctx, snapOpts, output); err != nil {
				return err
			}
		}
		switch output {
		case util.OutputTable:
			return util.WriteAsTable(result, snapshotBlockHeader)
		case util.OutputJSON:
			return util.WriteJSON(result)
		default:
			return fmt.Errorf("unsupported output format: %q", output)
		}
	} else if snapOpts.ShareID != "" {
		if snapOpts.client == nil {
			authConfig, err := config.ReadAuthConfig()
			if err != nil {
				return err
			}
			snapOpts.client, err = auth.NewSharedFileSystemClient(ctx, authConfig)
			if err != nil {
				return err
			}
		}
		createOpts := nfsSnapshot.CreateOpts{
			ShareID:     snapOpts.ShareID,
			Name:        snapOpts.Name,
			Description: snapOpts.Description,
		}
		snapshotResult := nfsSnapshot.Create(ctx, snapOpts.client, createOpts)
		result, err := snapshotResult.Extract()
		if err != nil {
			return err
		}
		if snapOpts.Cleanup {
			snapOpts.Share = true
			if err = CleanupSnapshot(ctx, snapOpts, output); err != nil {
				return err
			}
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
