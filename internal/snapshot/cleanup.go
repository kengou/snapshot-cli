package snapshot

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/gophercloud/gophercloud/v2"
	blockSnapshot "github.com/gophercloud/gophercloud/v2/openstack/blockstorage/v3/snapshots"
	nfsSnapshot "github.com/gophercloud/gophercloud/v2/openstack/sharedfilesystems/v2/snapshots"

	"snapshot-cli/internal/auth"
	"snapshot-cli/internal/config"
	"snapshot-cli/internal/util"
)

// maxConcurrentDeletions limits parallel snapshot delete calls to avoid overwhelming the API.
const maxConcurrentDeletions = 5

// CleanupSnapshot deletes snapshots that are older than snapOpts.OlderThan and have
// status "available". Set snapOpts.Volume for block storage or snapOpts.Share for
// shared filesystems. Optionally scope to a specific resource via snapOpts.VolumeID
// or snapOpts.ShareID. Reuses snapOpts.client when already set (e.g. called from
// CreateSnapshotCmd) to avoid a redundant Keystone round-trip.
func CleanupSnapshot(ctx context.Context, snapOpts *SnapShotOpts, output string) error {
	olderThan := time.Now().Add(-ParseDurationOrFallback(snapOpts.OlderThan))

	switch {
	case snapOpts.Volume:
		// H1: reuse the client set by CreateSnapshotCmd when available.
		var client *gophercloud.ServiceClient
		if snapOpts.client != nil {
			client = snapOpts.client
		} else {
			authConfig, err := config.ReadAuthConfig()
			if err != nil {
				return err
			}
			client, err = auth.NewBlockStorageClient(ctx, authConfig)
			if err != nil {
				return err
			}
		}

		// M3: filter by status server-side to reduce data transferred.
		listOpts := blockSnapshot.ListOpts{Status: "available"}
		if snapOpts.VolumeID != "" {
			listOpts.VolumeID = snapOpts.VolumeID
		}
		pages, err := blockSnapshot.List(client, listOpts).AllPages(ctx)
		if err != nil {
			return err
		}
		allSnapshots, err := blockSnapshot.ExtractSnapshots(pages)
		if err != nil {
			return err
		}

		deletedSnapshots := deleteBlockSnapshots(ctx, client, allSnapshots, olderThan)

		switch output {
		case util.OutputTable:
			return util.WriteAsTable(deletedSnapshots, "")
		case util.OutputJSON:
			return util.WriteJSON(deletedSnapshots)
		default:
			return fmt.Errorf("unsupported output format: %q", output)
		}

	case snapOpts.Share:
		var client *gophercloud.ServiceClient
		if snapOpts.client != nil {
			client = snapOpts.client
		} else {
			authConfig, err := config.ReadAuthConfig()
			if err != nil {
				return err
			}
			client, err = auth.NewSharedFileSystemClient(ctx, authConfig)
			if err != nil {
				return err
			}
		}

		listOpts := nfsSnapshot.ListOpts{Status: "available"}
		if snapOpts.ShareID != "" {
			listOpts.ShareID = snapOpts.ShareID
		}

		pages, err := nfsSnapshot.ListDetail(client, listOpts).AllPages(ctx)
		if err != nil {
			return err
		}
		allSnapshots, err := nfsSnapshot.ExtractSnapshots(pages)
		if err != nil {
			return err
		}

		deletedSnapshots := deleteNFSSnapshots(ctx, client, allSnapshots, olderThan)

		switch output {
		case util.OutputTable:
			return util.WriteAsTable(deletedSnapshots, "")
		case util.OutputJSON:
			return util.WriteJSON(deletedSnapshots)
		default:
			return fmt.Errorf("unsupported output format: %q", output)
		}
	}

	return nil
}

// deleteBlockSnapshots deletes block storage snapshots older than olderThan in parallel
// and returns the IDs of successfully deleted snapshots.
func deleteBlockSnapshots(ctx context.Context, client *gophercloud.ServiceClient, snapshots []blockSnapshot.Snapshot, olderThan time.Time) []string {
	var (
		mu               sync.Mutex
		wg               sync.WaitGroup
		sem              = make(chan struct{}, maxConcurrentDeletions)
		deletedSnapshots []string
	)

	for _, snap := range snapshots {
		if !snap.CreatedAt.Before(olderThan) {
			continue
		}
		wg.Add(1)
		sem <- struct{}{}
		go func(id string) {
			defer wg.Done()
			defer func() { <-sem }()
			if delErr := blockSnapshot.Delete(ctx, client, id).ExtractErr(); delErr != nil {
				// M5: write errors to stderr so they don't corrupt JSON stdout.
				_, _ = fmt.Fprintf(os.Stderr, "failed to delete snapshot %s: %v\n", id, delErr)
			} else {
				mu.Lock()
				deletedSnapshots = append(deletedSnapshots, id)
				mu.Unlock()
			}
		}(snap.ID)
	}
	wg.Wait()
	return deletedSnapshots
}

// deleteNFSSnapshots deletes NFS snapshots older than olderThan in parallel
// and returns the IDs of successfully deleted snapshots.
func deleteNFSSnapshots(ctx context.Context, client *gophercloud.ServiceClient, snapshots []nfsSnapshot.Snapshot, olderThan time.Time) []string {
	var (
		mu               sync.Mutex
		wg               sync.WaitGroup
		sem              = make(chan struct{}, maxConcurrentDeletions)
		deletedSnapshots []string
	)

	for _, snap := range snapshots {
		if !snap.CreatedAt.Before(olderThan) {
			continue
		}
		wg.Add(1)
		sem <- struct{}{}
		go func(id string) {
			defer wg.Done()
			defer func() { <-sem }()
			if delErr := nfsSnapshot.Delete(ctx, client, id).ExtractErr(); delErr != nil {
				_, _ = fmt.Fprintf(os.Stderr, "failed to delete snapshot %s: %v\n", id, delErr)
			} else {
				mu.Lock()
				deletedSnapshots = append(deletedSnapshots, id)
				mu.Unlock()
			}
		}(snap.ID)
	}
	wg.Wait()
	return deletedSnapshots
}

// ParseDurationOrFallback parses a Go duration string (e.g. "168h", "720h").
// If value is empty or not a valid duration, it returns the default of 168h (7 days).
func ParseDurationOrFallback(value string) time.Duration {
	d, err := time.ParseDuration(value)
	if err != nil {
		return 168 * time.Hour // fallback to 7 days
	}
	return d
}
