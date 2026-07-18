package snapshot

import (
	"context"

	"github.com/gophercloud/gophercloud/v2"
	blockSnapshot "github.com/gophercloud/gophercloud/v2/openstack/blockstorage/v3/snapshots"
	nfsSnapshot "github.com/gophercloud/gophercloud/v2/openstack/sharedfilesystems/v2/snapshots"

	"github.com/kengou/snapshot-cli/internal/util"
)

// deletedSnapshot is the output document confirming a successful delete.
type deletedSnapshot struct {
	Deleted string `json:"deleted"`
}

// DeleteSnapshotCmd deletes the snapshot identified by snapOpts.SnapshotID.
// Set snapOpts.Volume for block storage or snapOpts.Share for shared filesystems.
// Caller supplies the gophercloud ServiceClient appropriate for the resource kind.
func DeleteSnapshotCmd(ctx context.Context, snapOpts *SnapShotOpts, output string, client *gophercloud.ServiceClient) (err error) {
	if err = util.ValidateUUID(snapOpts.SnapshotID); err != nil {
		return err
	}

	ctx, span := startDeleteSpan(ctx, snapOpts.SnapshotID)
	defer func() {
		if err != nil {
			span.RecordError(err)
		}
		span.End()
	}()

	switch {
	case snapOpts.Volume:
		if err = blockSnapshot.Delete(ctx, client, snapOpts.SnapshotID).ExtractErr(); err != nil {
			return err
		}
	case snapOpts.Share:
		if err = nfsSnapshot.Delete(ctx, client, snapOpts.SnapshotID).ExtractErr(); err != nil {
			return err
		}
	default:
		return nil
	}
	return util.Render(output, deletedSnapshot{Deleted: snapOpts.SnapshotID}, deletedHeader)
}
