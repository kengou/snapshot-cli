package snapshot

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gophercloud/gophercloud/v2"
	blockSnapshot "github.com/gophercloud/gophercloud/v2/openstack/blockstorage/v3/snapshots"
	nfsSnapshot "github.com/gophercloud/gophercloud/v2/openstack/sharedfilesystems/v2/snapshots"

	"snapshot-cli/internal/util"
)

// maxConcurrentDeletions limits parallel snapshot delete calls to avoid overwhelming the API.
const maxConcurrentDeletions = 5

// CleanupSnapshot deletes snapshots that are older than snapOpts.OlderThan and have
// status "available". Set snapOpts.Volume for block storage or snapOpts.Share for
// shared filesystems. Optionally scope to a specific resource via snapOpts.VolumeID
// or snapOpts.ShareID. When snapOpts.DryRun is true, the function returns the IDs
// that would be deleted without issuing DELETE requests. Caller supplies the client.
func CleanupSnapshot(ctx context.Context, snapOpts *SnapShotOpts, output string, client *gophercloud.ServiceClient) (err error) {
	olderThanDuration := parseDurationOrFallback(snapOpts.OlderThan)
	olderThan := time.Now().Add(-olderThanDuration)

	ctx, span := startCleanupSpan(ctx, snapOpts.VolumeID, snapOpts.ShareID, int64(olderThanDuration.Seconds()))
	defer func() {
		if err != nil {
			span.RecordError(err)
		}
		span.End()
	}()

	switch {
	case snapOpts.Volume:
		listOpts := blockSnapshot.ListOpts{Status: "available"}
		if snapOpts.VolumeID != "" {
			listOpts.VolumeID = snapOpts.VolumeID
		}
		pages, lErr := blockSnapshot.List(client, listOpts).AllPages(ctx)
		if lErr != nil {
			return lErr
		}
		allSnapshots, xErr := blockSnapshot.ExtractSnapshots(pages)
		if xErr != nil {
			return xErr
		}

		deletedSnapshots := deleteBlockSnapshots(ctx, client, allSnapshots, olderThan, snapOpts.DryRun)
		return util.Render(output, deletedSnapshots, "")

	case snapOpts.Share:
		listOpts := nfsSnapshot.ListOpts{Status: "available"}
		if snapOpts.ShareID != "" {
			listOpts.ShareID = snapOpts.ShareID
		}

		pages, lErr := nfsSnapshot.ListDetail(client, listOpts).AllPages(ctx)
		if lErr != nil {
			return lErr
		}
		allSnapshots, xErr := nfsSnapshot.ExtractSnapshots(pages)
		if xErr != nil {
			return xErr
		}

		deletedSnapshots := deleteNFSSnapshots(ctx, client, allSnapshots, olderThan, snapOpts.DryRun)
		return util.Render(output, deletedSnapshots, "")
	}

	return nil
}

// deleteBlockSnapshots deletes block storage snapshots older than olderThan in parallel
// and returns the IDs of successfully deleted snapshots. When dryRun is true, it returns
// the IDs that would be deleted without issuing DELETE requests. The returned slice is
// non-nil (empty slice rather than nil) to guarantee valid JSON `[]` output.
func deleteBlockSnapshots(ctx context.Context, client *gophercloud.ServiceClient, snapshots []blockSnapshot.Snapshot, olderThan time.Time, dryRun bool) []string {
	candidates := make([]string, 0, len(snapshots))
	for _, snap := range snapshots {
		if snap.CreatedAt.Before(olderThan) {
			candidates = append(candidates, snap.ID)
		}
	}
	if dryRun {
		return candidates
	}
	return runParallelDeletes(ctx, candidates, func(id string) error {
		return blockSnapshot.Delete(ctx, client, id).ExtractErr()
	})
}

// deleteNFSSnapshots deletes NFS snapshots older than olderThan in parallel
// and returns the IDs of successfully deleted snapshots.
func deleteNFSSnapshots(ctx context.Context, client *gophercloud.ServiceClient, snapshots []nfsSnapshot.Snapshot, olderThan time.Time, dryRun bool) []string {
	candidates := make([]string, 0, len(snapshots))
	for _, snap := range snapshots {
		if snap.CreatedAt.Before(olderThan) {
			candidates = append(candidates, snap.ID)
		}
	}
	if dryRun {
		return candidates
	}
	return runParallelDeletes(ctx, candidates, func(id string) error {
		return nfsSnapshot.Delete(ctx, client, id).ExtractErr()
	})
}

// runParallelDeletes invokes del for each id with a semaphore-bounded worker pool.
// Each worker has a panic recover so a panic in gophercloud cannot deadlock wg.Wait.
// Respects ctx cancellation both before acquiring the semaphore and before each call.
func runParallelDeletes(ctx context.Context, ids []string, del func(id string) error) []string {
	var (
		mu               sync.Mutex
		wg               sync.WaitGroup
		sem              = make(chan struct{}, maxConcurrentDeletions)
		deletedSnapshots = make([]string, 0, len(ids))
	)

	for _, id := range ids {
		select {
		case <-ctx.Done():
			wg.Wait()
			// Strip non-printable bytes to prevent log injection from ctx error surfaces.
			_, _ = fmt.Fprintf(os.Stderr, "cleanup cancelled: %s\n", sanitize(ctx.Err().Error()))
			return deletedSnapshots
		case sem <- struct{}{}:
		}

		wg.Add(1)
		go func(id string) {
			defer wg.Done()
			defer func() { <-sem }()
			defer func() {
				if r := recover(); r != nil {
					_, _ = fmt.Fprintf(os.Stderr, "panic deleting snapshot %s: %v\n", id, r)
				}
			}()
			if ctx.Err() != nil {
				return
			}
			if delErr := del(id); delErr != nil {
				_, _ = fmt.Fprintf(os.Stderr, "failed to delete snapshot %s: %s\n", id, sanitize(delErr.Error()))
				return
			}
			mu.Lock()
			deletedSnapshots = append(deletedSnapshots, id)
			mu.Unlock()
		}(id)
	}
	wg.Wait()
	return deletedSnapshots
}

// sanitize replaces CR/LF/TAB in a string with spaces to prevent log/span
// injection when server-returned error messages include control characters.
func sanitize(s string) string {
	return strings.NewReplacer("\r", " ", "\n", " ", "\t", " ").Replace(s)
}

// parseDurationOrFallback parses a Go duration string (e.g. "168h", "720h").
// If value is empty or not a valid duration, it returns the default of 168h (7 days).
func parseDurationOrFallback(value string) time.Duration {
	d, err := time.ParseDuration(value)
	if err != nil {
		return 168 * time.Hour // fallback to 7 days
	}
	return d
}
