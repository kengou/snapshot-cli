package snapshot

import (
	"context"
	"time"

	"github.com/gophercloud/gophercloud/v2"
	blockSnapshot "github.com/gophercloud/gophercloud/v2/openstack/blockstorage/v3/snapshots"
	nfsSnapshot "github.com/gophercloud/gophercloud/v2/openstack/sharedfilesystems/v2/snapshots"

	"snapshot-cli/internal/auth"
	"snapshot-cli/internal/config"
	"snapshot-cli/internal/util"
)

func CleanupSnapshot(ctx context.Context, snapOpts *SnapShotOpts, output string) error {
	client := new(gophercloud.ServiceClient) //nolint:ineffassign
	authConfig, err := config.ReadAuthConfig()
	if err != nil {
		return err
	}

	olderThan := time.Now().Add(-ParseDurationOrFallback(snapOpts.OlderThan))

	switch {
	case snapOpts.Volume:

		client, err = auth.NewBlockStorageClient(ctx, authConfig)
		if err != nil {
			return err
		}

		pages, err := blockSnapshot.List(client, blockSnapshot.ListOpts{}).AllPages(ctx)
		if err != nil {
			return err
		}
		allSnapshots, err := blockSnapshot.ExtractSnapshots(pages)
		if err != nil {
			return err
		}

		var deletedSnapshots []string

		for _, snapshot := range allSnapshots {
			if snapshot.CreatedAt.Before(olderThan) && snapshot.Status == "available" {
				err = nfsSnapshot.Delete(ctx, client, snapshot.ID).ExtractErr()
				if err != nil {
					_ = util.WriteJSON("Failed to delete snapshot: " + snapshot.ID) //nolint:errcheck
				} else {
					deletedSnapshots = append(deletedSnapshots, snapshot.ID)
				}
			}
		}
		switch output {
		case util.OutputTable:
			return util.WriteAsTable(deletedSnapshots, "")
		case util.OutputJSON:
			return util.WriteJSON(deletedSnapshots)
		}

	case snapOpts.Share:

		client, err = auth.NewSharedFileSystemClient(ctx, authConfig)
		if err != nil {
			return err
		}

		pages, err := nfsSnapshot.ListDetail(client, nfsSnapshot.ListOpts{}).AllPages(ctx)
		if err != nil {
			return err
		}
		allSnapshots, err := nfsSnapshot.ExtractSnapshots(pages)
		if err != nil {
			return err
		}

		var deletedSnapshots []string

		for _, snapshot := range allSnapshots {
			if snapshot.CreatedAt.Before(olderThan) && snapshot.Status == "available" {
				err = nfsSnapshot.Delete(ctx, client, snapshot.ID).ExtractErr()
				if err != nil {
					_ = util.WriteJSON("Failed to delete snapshot: " + snapshot.ID) //nolint:errcheck
				} else {
					deletedSnapshots = append(deletedSnapshots, snapshot.ID)
				}
			}
		}
		switch output {
		case util.OutputTable:
			return util.WriteAsTable(deletedSnapshots, "")
		case util.OutputJSON:
			return util.WriteJSON(deletedSnapshots)
		}
	}

	return nil
}

func ParseDurationOrFallback(value string) time.Duration {
	d, err := time.ParseDuration(value)
	if err != nil {
		return 168 * time.Hour // fallback to 7 days
	}
	return d
}
