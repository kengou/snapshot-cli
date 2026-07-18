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

	"github.com/kengou/snapshot-cli/internal/util"
)

// maxConcurrentDeletions limits parallel snapshot delete calls to avoid overwhelming the API.
const maxConcurrentDeletions = 5

// CleanupSnapshot deletes snapshots that are older than snapOpts.OlderThan and have
// status "available", then renders the deleted IDs in the requested output format.
// Set snapOpts.Volume for block storage or snapOpts.Share for shared filesystems.
// Optionally scope to a specific resource via snapOpts.VolumeID or snapOpts.ShareID.
// When snapOpts.DryRun is true, the IDs that would be deleted are rendered without
// issuing DELETE requests. Caller supplies the client.
func CleanupSnapshot(ctx context.Context, snapOpts *SnapShotOpts, output string, client *gophercloud.ServiceClient) error {
	deleted, err := cleanupSnapshots(ctx, snapOpts, client)
	if err != nil {
		return err
	}
	return util.Render(output, deleted, deletedHeader)
}

// cleanupSnapshots performs the cleanup and returns the deleted (or, in dry-run
// mode, the candidate) snapshot IDs without rendering any output. The returned
// slice is non-nil (empty rather than nil) to guarantee valid JSON `[]` output.
func cleanupSnapshots(ctx context.Context, snapOpts *SnapShotOpts, client *gophercloud.ServiceClient) (deleted []string, err error) {
	olderThanDuration := snapOpts.OlderThan
	if olderThanDuration <= 0 {
		olderThanDuration = DefaultOlderThan
	}
	olderThan := time.Now().Add(-olderThanDuration)

	if snapOpts.VolumeID != "" {
		if vErr := util.ValidateUUID(snapOpts.VolumeID); vErr != nil {
			return nil, vErr
		}
	}
	if snapOpts.ShareID != "" {
		if vErr := util.ValidateUUID(snapOpts.ShareID); vErr != nil {
			return nil, vErr
		}
	}

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
			return nil, lErr
		}
		allSnapshots, xErr := blockSnapshot.ExtractSnapshots(pages)
		if xErr != nil {
			return nil, xErr
		}

		candidates := expiredIDs(allSnapshots, olderThan, func(s blockSnapshot.Snapshot) (string, time.Time) {
			return s.ID, s.CreatedAt
		})
		if snapOpts.DryRun {
			return candidates, nil
		}
		return runParallelDeletes(ctx, candidates, func(id string) error {
			return blockSnapshot.Delete(ctx, client, id).ExtractErr()
		}), nil

	case snapOpts.Share:
		listOpts := nfsSnapshot.ListOpts{Status: "available"}
		if snapOpts.ShareID != "" {
			listOpts.ShareID = snapOpts.ShareID
		}

		pages, lErr := nfsSnapshot.ListDetail(client, listOpts).AllPages(ctx)
		if lErr != nil {
			return nil, lErr
		}
		allSnapshots, xErr := nfsSnapshot.ExtractSnapshots(pages)
		if xErr != nil {
			return nil, xErr
		}

		candidates := expiredIDs(allSnapshots, olderThan, func(s nfsSnapshot.Snapshot) (string, time.Time) {
			return s.ID, s.CreatedAt
		})
		if snapOpts.DryRun {
			return candidates, nil
		}
		return runParallelDeletes(ctx, candidates, func(id string) error {
			return nfsSnapshot.Delete(ctx, client, id).ExtractErr()
		}), nil
	}

	return []string{}, nil
}

// expiredIDs returns the IDs of all snapshots created before olderThan.
// meta extracts the ID and creation timestamp from a snapshot of either kind.
func expiredIDs[S any](snapshots []S, olderThan time.Time, meta func(S) (id string, createdAt time.Time)) []string {
	ids := make([]string, 0, len(snapshots))
	for _, snap := range snapshots {
		id, createdAt := meta(snap)
		if createdAt.Before(olderThan) {
			ids = append(ids, id)
		}
	}
	return ids
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
