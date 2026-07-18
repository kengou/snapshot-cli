package snapshot

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/gophercloud/gophercloud/v2"
	blockSnapshot "github.com/gophercloud/gophercloud/v2/openstack/blockstorage/v3/snapshots"
	nfsSnapshot "github.com/gophercloud/gophercloud/v2/openstack/sharedfilesystems/v2/snapshots"

	"github.com/kengou/snapshot-cli/internal/util"
)

// createWithCleanupResult is the single JSON document emitted by
// "snapshot create --cleanup": the created snapshot plus the IDs deleted by
// the follow-up cleanup.
type createWithCleanupResult struct {
	Snapshot         any      `json:"snapshot"`
	DeletedSnapshots []string `json:"deleted_snapshots"`
}

// CreateSnapshotCmd creates a snapshot of a block storage volume (snapOpts.VolumeID)
// or a shared filesystem (snapOpts.ShareID). The snapshot name is always suffixed
// with the current UTC timestamp; the base name defaults to the resource ID when
// snapOpts.Name is empty. If snapOpts.Cleanup is true, snapshots older than
// snapOpts.OlderThan are deleted afterwards on the same client; a cleanup failure
// is reported as a warning, not an error, since the snapshot was already created.
func CreateSnapshotCmd(ctx context.Context, snapOpts *SnapShotOpts, output string, client *gophercloud.ServiceClient) (err error) {
	ctx, span := startCreateSpan(ctx, snapOpts.VolumeID, snapOpts.ShareID, snapOpts.Name)
	defer func() {
		if err != nil {
			span.RecordError(err)
		}
		span.End()
	}()

	switch {
	case snapOpts.VolumeID != "":
		if err = util.ValidateUUID(snapOpts.VolumeID); err != nil {
			return err
		}
		snapOpts.Volume = true
	case snapOpts.ShareID != "":
		if err = util.ValidateUUID(snapOpts.ShareID); err != nil {
			return err
		}
		snapOpts.Share = true
	default:
		return nil
	}

	if snapOpts.Name == "" {
		if snapOpts.Volume {
			snapOpts.Name = snapOpts.VolumeID
		} else {
			snapOpts.Name = snapOpts.ShareID
		}
	}
	snapOpts.Name += "-" + time.Now().UTC().Format("200601021504")

	var result any
	var header []string
	if snapOpts.Volume {
		createOpts := blockSnapshot.CreateOpts{
			VolumeID:    snapOpts.VolumeID,
			Name:        snapOpts.Name,
			Description: snapOpts.Description,
			Force:       snapOpts.Force,
		}
		snap, cErr := blockSnapshot.Create(ctx, client, createOpts).Extract()
		if cErr != nil {
			return cErr
		}
		result, header = snap, snapshotBlockHeader
	} else {
		createOpts := nfsSnapshot.CreateOpts{
			ShareID:     snapOpts.ShareID,
			Name:        snapOpts.Name,
			Description: snapOpts.Description,
		}
		snap, cErr := nfsSnapshot.Create(ctx, client, createOpts).Extract()
		if cErr != nil {
			return cErr
		}
		result, header = snap, snapshotNfsHeader
	}

	if !snapOpts.Cleanup {
		return util.Render(output, result, header)
	}

	deleted, cleanupErr := cleanupSnapshots(ctx, snapOpts, client)
	if cleanupErr != nil {
		// The snapshot was created; failing the command here would make a
		// retry create a duplicate. Surface the cleanup problem as a warning.
		_, _ = fmt.Fprintf(os.Stderr, "warning: snapshot created but cleanup failed: %s\n", sanitize(cleanupErr.Error()))
		deleted = []string{}
	}

	if output == util.FormatTable {
		if err = util.Render(output, result, header); err != nil {
			return err
		}
		return util.Render(output, deleted, deletedHeader)
	}
	return util.Render(output, &createWithCleanupResult{Snapshot: result, DeletedSnapshots: deleted}, nil)
}
