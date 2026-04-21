package snapshot

import (
	"context"

	"github.com/gophercloud/gophercloud/v2"
	blockSnapshot "github.com/gophercloud/gophercloud/v2/openstack/blockstorage/v3/snapshots"
	nfsSnapshot "github.com/gophercloud/gophercloud/v2/openstack/sharedfilesystems/v2/snapshots"

	"snapshot-cli/internal/util"
)

// GetSnapshotCmd retrieves a single snapshot by ID.
// Set snapOpts.Volume for block storage or snapOpts.Share for shared filesystems.
// The snapshot ID must be provided in snapOpts.SnapshotID. Caller supplies the client.
func GetSnapshotCmd(ctx context.Context, snapOpts *SnapShotOpts, output string, client *gophercloud.ServiceClient) (err error) {
	if err = util.ValidateUUID(snapOpts.SnapshotID); err != nil {
		return err
	}

	ctx, span := startGetSpan(ctx, snapOpts.SnapshotID)
	defer func() {
		if err != nil {
			span.RecordError(err)
		}
		span.End()
	}()

	switch {
	case snapOpts.Volume:
		result, gErr := blockSnapshot.Get(ctx, client, snapOpts.SnapshotID).Extract()
		if gErr != nil {
			return gErr
		}
		return util.Render(output, result, snapshotBlockHeader)

	case snapOpts.Share:
		result, gErr := nfsSnapshot.Get(ctx, client, snapOpts.SnapshotID).Extract()
		if gErr != nil {
			return gErr
		}
		return util.Render(output, result, snapshotNfsHeader)
	}

	return nil
}
