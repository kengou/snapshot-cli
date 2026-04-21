package snapshot

import (
	"context"
	"time"

	"github.com/gophercloud/gophercloud/v2"
	blockSnapshot "github.com/gophercloud/gophercloud/v2/openstack/blockstorage/v3/snapshots"
	nfsSnapshot "github.com/gophercloud/gophercloud/v2/openstack/sharedfilesystems/v2/snapshots"

	"snapshot-cli/internal/util"
)

// CreateSnapshotCmd creates a snapshot of a block storage volume (snapOpts.VolumeID)
// or a shared filesystem (snapOpts.ShareID). The snapshot name is auto-generated
// from the resource ID and the current UTC timestamp when snapOpts.Name is empty.
// If snapOpts.Cleanup is true, old snapshots older than snapOpts.OlderThan are
// deleted after the new snapshot is created, on the same client.
func CreateSnapshotCmd(ctx context.Context, snapOpts *SnapShotOpts, output string, client *gophercloud.ServiceClient) (err error) {
	ctx, span := startCreateSpan(ctx, snapOpts.VolumeID, snapOpts.ShareID, snapOpts.Name)
	defer func() {
		if err != nil {
			span.RecordError(err)
		}
		span.End()
	}()

	if snapOpts.Name == "" {
		if snapOpts.VolumeID != "" {
			snapOpts.Name = snapOpts.VolumeID
		} else if snapOpts.ShareID != "" {
			snapOpts.Name = snapOpts.ShareID
		}
	}
	snapOpts.Name = snapOpts.Name + "-" + time.Now().UTC().Format("200601021504")

	switch {
	case snapOpts.VolumeID != "":
		snapOpts.Volume = true
	case snapOpts.ShareID != "":
		snapOpts.Share = true
	}

	if snapOpts.VolumeID != "" {
		createOpts := blockSnapshot.CreateOpts{
			VolumeID:    snapOpts.VolumeID,
			Name:        snapOpts.Name,
			Description: snapOpts.Description,
			Force:       snapOpts.Force,
		}
		snapshotResult := blockSnapshot.Create(ctx, client, createOpts)
		result, err := snapshotResult.Extract()
		if err != nil {
			return err
		}
		if snapOpts.Cleanup {
			if err = CleanupSnapshot(ctx, snapOpts, output, client); err != nil {
				return err
			}
		}
		return util.Render(output, result, snapshotBlockHeader)
	} else if snapOpts.ShareID != "" {
		createOpts := nfsSnapshot.CreateOpts{
			ShareID:     snapOpts.ShareID,
			Name:        snapOpts.Name,
			Description: snapOpts.Description,
		}
		snapshotResult := nfsSnapshot.Create(ctx, client, createOpts)
		result, err := snapshotResult.Extract()
		if err != nil {
			return err
		}
		if snapOpts.Cleanup {
			if err = CleanupSnapshot(ctx, snapOpts, output, client); err != nil {
				return err
			}
		}
		return util.Render(output, result, snapshotNfsHeader)
	}

	return nil
}
