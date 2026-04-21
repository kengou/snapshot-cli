package snapshot

import (
	"context"

	"github.com/gophercloud/gophercloud/v2"
	blockSnapshot "github.com/gophercloud/gophercloud/v2/openstack/blockstorage/v3/snapshots"
	nfsSnapshot "github.com/gophercloud/gophercloud/v2/openstack/sharedfilesystems/v2/snapshots"

	"snapshot-cli/internal/util"
)

// ListSnapshotsCmd lists all snapshots for either block storage (snapOpts.Volume)
// or shared filesystems (snapOpts.Share). Caller supplies the client.
func ListSnapshotsCmd(ctx context.Context, snapOpts *SnapShotOpts, output string, client *gophercloud.ServiceClient) (err error) {
	ctx, span := startListSpan(ctx, snapOpts.VolumeID, snapOpts.ShareID)
	defer func() {
		if err != nil {
			span.RecordError(err)
		}
		span.End()
	}()

	switch {
	case snapOpts.Volume:
		pages, lErr := blockSnapshot.List(client, blockSnapshot.ListOpts{}).AllPages(ctx)
		if lErr != nil {
			return lErr
		}
		allSnapshots, xErr := blockSnapshot.ExtractSnapshots(pages)
		if xErr != nil {
			return xErr
		}
		return util.Render(output, allSnapshots, snapshotBlockHeader)

	case snapOpts.Share:
		pages, lErr := nfsSnapshot.ListDetail(client, nfsSnapshot.ListOpts{}).AllPages(ctx)
		if lErr != nil {
			return lErr
		}
		allSnapshots, xErr := nfsSnapshot.ExtractSnapshots(pages)
		if xErr != nil {
			return xErr
		}
		return util.Render(output, allSnapshots, snapshotNfsHeader)
	}
	return nil
}
