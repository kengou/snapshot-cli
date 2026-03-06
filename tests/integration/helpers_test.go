//go:build integration

package integration

import (
	"bytes"
	"io"
	"os"
	"testing"
	"time"

	blockSnap "github.com/gophercloud/gophercloud/v2/openstack/blockstorage/v3/snapshots"
	"github.com/gophercloud/gophercloud/v2/openstack/blockstorage/v3/volumes"
	nfsSnap "github.com/gophercloud/gophercloud/v2/openstack/sharedfilesystems/v2/snapshots"
	"github.com/gophercloud/gophercloud/v2/openstack/sharedfilesystems/v2/shares"
)

const (
	pollInterval  = 2 * time.Second
	pollTimeout   = 2 * time.Minute
	shareTimeout  = 4 * time.Minute // Manila LVM takes longer than Cinder
)

// ── Stdout capture ────────────────────────────────────────────────────────────

// captureOutput redirects os.Stdout while fn runs and returns whatever was
// written. CLI functions write directly to os.Stdout, so this is required to
// inspect their output in integration tests.
func captureOutput(t *testing.T, fn func() error) (string, error) {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	old := os.Stdout
	os.Stdout = w

	fnErr := fn()

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	if _, copyErr := io.Copy(&buf, r); copyErr != nil {
		t.Fatalf("io.Copy: %v", copyErr)
	}
	return buf.String(), fnErr
}

// ── Cinder volume helpers ─────────────────────────────────────────────────────

// createVolume creates a 1 GiB Cinder volume and waits for it to become
// available. A t.Cleanup is registered to delete the volume after the test;
// because Go runs cleanups in LIFO order, any snapshot cleanups registered
// later will fire first, keeping the deletion order correct.
func createVolume(t *testing.T) string {
	t.Helper()
	v, err := volumes.Create(ctx, blockClient, volumes.CreateOpts{
		Size: 1,
		Name: "snapcli-it-" + t.Name(),
	}).Extract()
	if err != nil {
		t.Fatalf("create volume: %v", err)
	}
	t.Cleanup(func() { deleteVolume(t, v.ID) })
	waitForVolume(t, v.ID, "available")
	return v.ID
}

func deleteVolume(t *testing.T, id string) {
	t.Helper()
	if err := volumes.Delete(ctx, blockClient, id, volumes.DeleteOpts{}).ExtractErr(); err != nil {
		t.Logf("delete volume %s: %v", id, err)
	}
}

func waitForVolume(t *testing.T, id, status string) {
	t.Helper()
	deadline := time.Now().Add(pollTimeout)
	for time.Now().Before(deadline) {
		v, err := volumes.Get(ctx, blockClient, id).Extract()
		if err != nil {
			t.Fatalf("poll volume %s: %v", id, err)
		}
		if v.Status == status {
			return
		}
		if v.Status == "error" || v.Status == "error_deleting" {
			t.Fatalf("volume %s entered error state (status=%s)", id, v.Status)
		}
		time.Sleep(pollInterval)
	}
	t.Fatalf("volume %s did not reach %q within %s", id, status, pollTimeout)
}

// ── Cinder snapshot helpers ───────────────────────────────────────────────────

// createBlockSnapshot creates a snapshot of volID and waits for "available".
// Registers a cleanup that fires before the parent volume cleanup (LIFO).
func createBlockSnapshot(t *testing.T, volID string) string {
	t.Helper()
	s, err := blockSnap.Create(ctx, blockClient, blockSnap.CreateOpts{
		VolumeID: volID,
		Name:     "snapcli-it-snap-" + t.Name(),
		Force:    true,
	}).Extract()
	if err != nil {
		t.Fatalf("create block snapshot: %v", err)
	}
	t.Cleanup(func() { deleteBlockSnapshot(t, s.ID) })
	waitForBlockSnapshot(t, s.ID, "available")
	return s.ID
}

func deleteBlockSnapshot(t *testing.T, id string) {
	t.Helper()
	if err := blockSnap.Delete(ctx, blockClient, id).ExtractErr(); err != nil {
		t.Logf("delete block snapshot %s: %v", id, err)
		return
	}
	waitForBlockSnapshotGone(t, id)
}

func waitForBlockSnapshot(t *testing.T, id, status string) {
	t.Helper()
	deadline := time.Now().Add(pollTimeout)
	for time.Now().Before(deadline) {
		s, err := blockSnap.Get(ctx, blockClient, id).Extract()
		if err != nil {
			t.Fatalf("poll block snapshot %s: %v", id, err)
		}
		if s.Status == status {
			return
		}
		if s.Status == "error" {
			t.Fatalf("block snapshot %s entered error state", id)
		}
		time.Sleep(pollInterval)
	}
	t.Fatalf("block snapshot %s did not reach %q within %s", id, status, pollTimeout)
}

func waitForBlockSnapshotGone(t *testing.T, id string) {
	t.Helper()
	deadline := time.Now().Add(pollTimeout)
	for time.Now().Before(deadline) {
		if _, err := blockSnap.Get(ctx, blockClient, id).Extract(); err != nil {
			return // 404 or any error means it is gone
		}
		time.Sleep(pollInterval)
	}
	t.Fatalf("block snapshot %s was not removed within %s", id, pollTimeout)
}

// ── Manila share helpers ──────────────────────────────────────────────────────

func createShare(t *testing.T) string {
	t.Helper()
	s, err := shares.Create(ctx, nfsClient, shares.CreateOpts{
		ShareProto: "NFS",
		Size:       1,
		Name:       "snapcli-it-" + t.Name(),
	}).Extract()
	if err != nil {
		t.Fatalf("create share: %v", err)
	}
	t.Cleanup(func() { deleteShare(t, s.ID) })
	waitForShare(t, s.ID, "available")
	return s.ID
}

func deleteShare(t *testing.T, id string) {
	t.Helper()
	if err := shares.Delete(ctx, nfsClient, id).ExtractErr(); err != nil {
		t.Logf("delete share %s: %v", id, err)
	}
}

func waitForShare(t *testing.T, id, status string) {
	t.Helper()
	deadline := time.Now().Add(shareTimeout)
	for time.Now().Before(deadline) {
		s, err := shares.Get(ctx, nfsClient, id).Extract()
		if err != nil {
			t.Fatalf("poll share %s: %v", id, err)
		}
		if s.Status == status {
			return
		}
		if s.Status == "error" {
			t.Fatalf("share %s entered error state", id)
		}
		time.Sleep(pollInterval)
	}
	t.Fatalf("share %s did not reach %q within %s", id, status, shareTimeout)
}

// ── Manila snapshot helpers ───────────────────────────────────────────────────

func createNFSSnapshot(t *testing.T, shareID string) string {
	t.Helper()
	s, err := nfsSnap.Create(ctx, nfsClient, nfsSnap.CreateOpts{
		ShareID: shareID,
		Name:    "snapcli-it-snap-" + t.Name(),
	}).Extract()
	if err != nil {
		t.Fatalf("create NFS snapshot: %v", err)
	}
	t.Cleanup(func() { deleteNFSSnapshot(t, s.ID) })
	waitForNFSSnapshot(t, s.ID, "available")
	return s.ID
}

func deleteNFSSnapshot(t *testing.T, id string) {
	t.Helper()
	if err := nfsSnap.Delete(ctx, nfsClient, id).ExtractErr(); err != nil {
		t.Logf("delete NFS snapshot %s: %v", id, err)
		return
	}
	waitForNFSSnapshotGone(t, id)
}

func waitForNFSSnapshot(t *testing.T, id, status string) {
	t.Helper()
	deadline := time.Now().Add(shareTimeout)
	for time.Now().Before(deadline) {
		s, err := nfsSnap.Get(ctx, nfsClient, id).Extract()
		if err != nil {
			t.Fatalf("poll NFS snapshot %s: %v", id, err)
		}
		if s.Status == status {
			return
		}
		if s.Status == "error" {
			t.Fatalf("NFS snapshot %s entered error state", id)
		}
		time.Sleep(pollInterval)
	}
	t.Fatalf("NFS snapshot %s did not reach %q within %s", id, status, shareTimeout)
}

func waitForNFSSnapshotGone(t *testing.T, id string) {
	t.Helper()
	deadline := time.Now().Add(shareTimeout)
	for time.Now().Before(deadline) {
		if _, err := nfsSnap.Get(ctx, nfsClient, id).Extract(); err != nil {
			return
		}
		time.Sleep(pollInterval)
	}
	t.Fatalf("NFS snapshot %s was not removed within %s", id, shareTimeout)
}
