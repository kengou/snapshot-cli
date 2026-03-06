//go:build integration

package integration

import (
	"encoding/json"
	"strings"
	"testing"

	"snapshot-cli/internal/blockstorage"
	"snapshot-cli/internal/sharedfilesystem"
	"snapshot-cli/internal/snapshot"
	"snapshot-cli/internal/util"
)

// ── Block Storage: volumes ────────────────────────────────────────────────────

func TestIntegration_VolumesList_JSON(t *testing.T) {
	createVolume(t) // ensures at least one volume exists

	out, err := captureOutput(t, func() error {
		return blockstorage.RunListBlockStorage(ctx, util.OutputJSON)
	})
	if err != nil {
		t.Fatalf("RunListBlockStorage: %v", err)
	}
	var list []map[string]any
	if jsonErr := json.Unmarshal([]byte(strings.TrimSpace(out)), &list); jsonErr != nil {
		t.Errorf("output is not valid JSON array: %v\noutput: %s", jsonErr, out)
	}
	if len(list) == 0 {
		t.Error("expected at least one volume in list, got empty")
	}
}

func TestIntegration_VolumesList_Table(t *testing.T) {
	createVolume(t)

	out, err := captureOutput(t, func() error {
		return blockstorage.RunListBlockStorage(ctx, util.OutputTable)
	})
	if err != nil {
		t.Fatalf("RunListBlockStorage table: %v", err)
	}
	if !strings.Contains(out, "id") {
		t.Errorf("expected table header in output, got: %s", out)
	}
}

func TestIntegration_VolumesGet_JSON(t *testing.T) {
	volID := createVolume(t)

	out, err := captureOutput(t, func() error {
		return blockstorage.RunGetBlockStorage(ctx, volID, util.OutputJSON)
	})
	if err != nil {
		t.Fatalf("RunGetBlockStorage: %v", err)
	}
	if !strings.Contains(out, volID) {
		t.Errorf("expected volume ID %s in output, got: %s", volID, out)
	}
	var m map[string]any
	if jsonErr := json.Unmarshal([]byte(strings.TrimSpace(out)), &m); jsonErr != nil {
		t.Errorf("output is not valid JSON: %v\noutput: %s", jsonErr, out)
	}
}

func TestIntegration_VolumesGet_Table(t *testing.T) {
	volID := createVolume(t)

	out, err := captureOutput(t, func() error {
		return blockstorage.RunGetBlockStorage(ctx, volID, util.OutputTable)
	})
	if err != nil {
		t.Fatalf("RunGetBlockStorage table: %v", err)
	}
	if !strings.Contains(out, volID) {
		t.Errorf("expected volume ID %s in table output, got: %s", volID, out)
	}
}

func TestIntegration_VolumesGet_NotFound(t *testing.T) {
	const notFoundID = "ffffffff-ffff-ffff-ffff-ffffffffffff"
	_, err := captureOutput(t, func() error {
		return blockstorage.RunGetBlockStorage(ctx, notFoundID, util.OutputJSON)
	})
	if err == nil {
		t.Error("expected error for not-found volume, got nil")
	}
}

// ── Shared Filesystem: shares ─────────────────────────────────────────────────

func TestIntegration_NFSList_JSON(t *testing.T) {
	createShare(t)

	out, err := captureOutput(t, func() error {
		return sharedfilesystem.RunListSharedFileSystems(ctx, util.OutputJSON)
	})
	if err != nil {
		t.Fatalf("RunListSharedFileSystem: %v", err)
	}
	var list []map[string]any
	if jsonErr := json.Unmarshal([]byte(strings.TrimSpace(out)), &list); jsonErr != nil {
		t.Errorf("output is not valid JSON array: %v\noutput: %s", jsonErr, out)
	}
	if len(list) == 0 {
		t.Error("expected at least one share in list, got empty")
	}
}

func TestIntegration_NFSList_Table(t *testing.T) {
	createShare(t)

	out, err := captureOutput(t, func() error {
		return sharedfilesystem.RunListSharedFileSystems(ctx, util.OutputTable)
	})
	if err != nil {
		t.Fatalf("RunListSharedFileSystem table: %v", err)
	}
	if !strings.Contains(out, "id") {
		t.Errorf("expected table header in output, got: %s", out)
	}
}

func TestIntegration_NFSGet_JSON(t *testing.T) {
	shareID := createShare(t)

	out, err := captureOutput(t, func() error {
		return sharedfilesystem.RunGetSharedFileSystem(ctx, shareID, util.OutputJSON)
	})
	if err != nil {
		t.Fatalf("RunGetSharedFileSystem: %v", err)
	}
	if !strings.Contains(out, shareID) {
		t.Errorf("expected share ID %s in output, got: %s", shareID, out)
	}
	var m map[string]any
	if jsonErr := json.Unmarshal([]byte(strings.TrimSpace(out)), &m); jsonErr != nil {
		t.Errorf("output is not valid JSON: %v\noutput: %s", jsonErr, out)
	}
}

func TestIntegration_NFSGet_Table(t *testing.T) {
	shareID := createShare(t)

	out, err := captureOutput(t, func() error {
		return sharedfilesystem.RunGetSharedFileSystem(ctx, shareID, util.OutputTable)
	})
	if err != nil {
		t.Fatalf("RunGetSharedFileSystem table: %v", err)
	}
	if !strings.Contains(out, shareID) {
		t.Errorf("expected share ID %s in table output, got: %s", shareID, out)
	}
}

func TestIntegration_NFSGet_NotFound(t *testing.T) {
	const notFoundID = "ffffffff-ffff-ffff-ffff-ffffffffffff"
	_, err := captureOutput(t, func() error {
		return sharedfilesystem.RunGetSharedFileSystem(ctx, notFoundID, util.OutputJSON)
	})
	if err == nil {
		t.Error("expected error for not-found share, got nil")
	}
}

// ── Block Storage: snapshot get / list ───────────────────────────────────────

func TestIntegration_SnapshotGet_Volume_JSON(t *testing.T) {
	volID := createVolume(t)
	snapID := createBlockSnapshot(t, volID)

	out, err := captureOutput(t, func() error {
		return snapshot.GetSnapshotCmd(ctx, &snapshot.SnapShotOpts{
			Volume:     true,
			SnapshotID: snapID,
		}, util.OutputJSON)
	})
	if err != nil {
		t.Fatalf("GetSnapshotCmd volume: %v", err)
	}
	if !strings.Contains(out, snapID) {
		t.Errorf("expected snapshot ID %s in output, got: %s", snapID, out)
	}
	var m map[string]any
	if jsonErr := json.Unmarshal([]byte(strings.TrimSpace(out)), &m); jsonErr != nil {
		t.Errorf("output is not valid JSON: %v\noutput: %s", jsonErr, out)
	}
}

func TestIntegration_SnapshotGet_Volume_Table(t *testing.T) {
	volID := createVolume(t)
	snapID := createBlockSnapshot(t, volID)

	out, err := captureOutput(t, func() error {
		return snapshot.GetSnapshotCmd(ctx, &snapshot.SnapShotOpts{
			Volume:     true,
			SnapshotID: snapID,
		}, util.OutputTable)
	})
	if err != nil {
		t.Fatalf("GetSnapshotCmd volume table: %v", err)
	}
	if !strings.Contains(out, snapID) {
		t.Errorf("expected snapshot ID %s in table output, got: %s", snapID, out)
	}
}

func TestIntegration_SnapshotList_Volume_JSON(t *testing.T) {
	volID := createVolume(t)
	snapID := createBlockSnapshot(t, volID)

	out, err := captureOutput(t, func() error {
		return snapshot.ListSnapshotsCmd(ctx, &snapshot.SnapShotOpts{
			Volume: true,
		}, util.OutputJSON)
	})
	if err != nil {
		t.Fatalf("ListSnapshotsCmd volume: %v", err)
	}
	if !strings.Contains(out, snapID) {
		t.Errorf("expected snapshot ID %s in list output, got: %s", snapID, out)
	}
}

func TestIntegration_SnapshotList_Volume_Table(t *testing.T) {
	volID := createVolume(t)
	createBlockSnapshot(t, volID)

	out, err := captureOutput(t, func() error {
		return snapshot.ListSnapshotsCmd(ctx, &snapshot.SnapShotOpts{
			Volume: true,
		}, util.OutputTable)
	})
	if err != nil {
		t.Fatalf("ListSnapshotsCmd volume table: %v", err)
	}
	if !strings.Contains(out, "ID") {
		t.Errorf("expected table header in output, got: %s", out)
	}
}

func TestIntegration_SnapshotGet_NotFound(t *testing.T) {
	const notFoundID = "ffffffff-ffff-ffff-ffff-ffffffffffff"
	_, err := captureOutput(t, func() error {
		return snapshot.GetSnapshotCmd(ctx, &snapshot.SnapShotOpts{
			Volume:     true,
			SnapshotID: notFoundID,
		}, util.OutputJSON)
	})
	if err == nil {
		t.Error("expected error for not-found snapshot, got nil")
	}
}

// ── NFS: snapshot get / list ──────────────────────────────────────────────────

func TestIntegration_SnapshotGet_Share_JSON(t *testing.T) {
	shareID := createShare(t)
	snapID := createNFSSnapshot(t, shareID)

	out, err := captureOutput(t, func() error {
		return snapshot.GetSnapshotCmd(ctx, &snapshot.SnapShotOpts{
			Share:      true,
			SnapshotID: snapID,
		}, util.OutputJSON)
	})
	if err != nil {
		t.Fatalf("GetSnapshotCmd share: %v", err)
	}
	if !strings.Contains(out, snapID) {
		t.Errorf("expected NFS snapshot ID %s in output, got: %s", snapID, out)
	}
	var m map[string]any
	if jsonErr := json.Unmarshal([]byte(strings.TrimSpace(out)), &m); jsonErr != nil {
		t.Errorf("output is not valid JSON: %v\noutput: %s", jsonErr, out)
	}
}

func TestIntegration_SnapshotList_Share_JSON(t *testing.T) {
	shareID := createShare(t)
	snapID := createNFSSnapshot(t, shareID)

	out, err := captureOutput(t, func() error {
		return snapshot.ListSnapshotsCmd(ctx, &snapshot.SnapShotOpts{
			Share: true,
		}, util.OutputJSON)
	})
	if err != nil {
		t.Fatalf("ListSnapshotsCmd share: %v", err)
	}
	if !strings.Contains(out, snapID) {
		t.Errorf("expected NFS snapshot ID %s in list output, got: %s", snapID, out)
	}
}

// ── Snapshot create ───────────────────────────────────────────────────────────

func TestIntegration_SnapshotCreate_Volume_JSON(t *testing.T) {
	volID := createVolume(t)

	var createdSnapID string
	out, err := captureOutput(t, func() error {
		return snapshot.CreateSnapshotCmd(ctx, &snapshot.SnapShotOpts{
			VolumeID: volID,
			Force:    true,
		}, util.OutputJSON)
	})
	if err != nil {
		t.Fatalf("CreateSnapshotCmd volume: %v", err)
	}
	var m map[string]any
	if jsonErr := json.Unmarshal([]byte(strings.TrimSpace(out)), &m); jsonErr != nil {
		t.Fatalf("output is not valid JSON: %v\noutput: %s", jsonErr, out)
	}
	if id, ok := m["id"].(string); ok {
		createdSnapID = id
		t.Cleanup(func() { deleteBlockSnapshot(t, createdSnapID) })
	} else {
		t.Error("response JSON has no 'id' field")
	}
}

func TestIntegration_SnapshotCreate_Volume_Table(t *testing.T) {
	volID := createVolume(t)

	out, err := captureOutput(t, func() error {
		return snapshot.CreateSnapshotCmd(ctx, &snapshot.SnapShotOpts{
			VolumeID: volID,
			Force:    true,
		}, util.OutputTable)
	})
	if err != nil {
		t.Fatalf("CreateSnapshotCmd volume table: %v", err)
	}
	if !strings.Contains(out, volID) {
		t.Errorf("expected volume ID %s in create output, got: %s", volID, out)
	}
}

func TestIntegration_SnapshotCreate_Share_JSON(t *testing.T) {
	shareID := createShare(t)

	out, err := captureOutput(t, func() error {
		return snapshot.CreateSnapshotCmd(ctx, &snapshot.SnapShotOpts{
			ShareID: shareID,
		}, util.OutputJSON)
	})
	if err != nil {
		t.Fatalf("CreateSnapshotCmd share: %v", err)
	}
	var m map[string]any
	if jsonErr := json.Unmarshal([]byte(strings.TrimSpace(out)), &m); jsonErr != nil {
		t.Fatalf("output is not valid JSON: %v\noutput: %s", jsonErr, out)
	}
	if id, ok := m["id"].(string); ok {
		t.Cleanup(func() { deleteNFSSnapshot(t, id) })
	} else {
		t.Error("response JSON has no 'id' field")
	}
}

// ── Snapshot delete ───────────────────────────────────────────────────────────

func TestIntegration_SnapshotDelete_Volume(t *testing.T) {
	volID := createVolume(t)
	snapID := createBlockSnapshot(t, volID)

	// Delete via CLI; the cleanup registered by createBlockSnapshot will also
	// attempt deletion — that's fine, gophercloud treats 404 as a no-op.
	_, err := captureOutput(t, func() error {
		return snapshot.DeleteSnapshotCmd(ctx, &snapshot.SnapShotOpts{
			Volume:     true,
			SnapshotID: snapID,
		}, util.OutputJSON)
	})
	if err != nil {
		t.Fatalf("DeleteSnapshotCmd volume: %v", err)
	}
}

func TestIntegration_SnapshotDelete_Share(t *testing.T) {
	shareID := createShare(t)
	snapID := createNFSSnapshot(t, shareID)

	_, err := captureOutput(t, func() error {
		return snapshot.DeleteSnapshotCmd(ctx, &snapshot.SnapShotOpts{
			Share:      true,
			SnapshotID: snapID,
		}, util.OutputJSON)
	})
	if err != nil {
		t.Fatalf("DeleteSnapshotCmd share: %v", err)
	}
}

func TestIntegration_SnapshotDelete_NotFound(t *testing.T) {
	const notFoundID = "ffffffff-ffff-ffff-ffff-ffffffffffff"
	_, err := captureOutput(t, func() error {
		return snapshot.DeleteSnapshotCmd(ctx, &snapshot.SnapShotOpts{
			Volume:     true,
			SnapshotID: notFoundID,
		}, util.OutputJSON)
	})
	if err == nil {
		t.Error("expected error deleting non-existent snapshot, got nil")
	}
}

// ── Cleanup ───────────────────────────────────────────────────────────────────

func TestIntegration_Cleanup_Volume(t *testing.T) {
	volID := createVolume(t)
	snapID := createBlockSnapshot(t, volID)

	// 0s means "older than right now", so any snapshot qualifies.
	out, err := captureOutput(t, func() error {
		return snapshot.CleanupSnapshot(ctx, &snapshot.SnapShotOpts{
			Volume:    true,
			OlderThan: "0s",
		}, util.OutputJSON)
	})
	if err != nil {
		t.Fatalf("CleanupSnapshot volume: %v", err)
	}
	if !strings.Contains(out, snapID) {
		t.Errorf("expected deleted snapshot ID %s in cleanup output, got: %s", snapID, out)
	}
}

func TestIntegration_Cleanup_Volume_Table(t *testing.T) {
	volID := createVolume(t)
	createBlockSnapshot(t, volID)

	_, err := captureOutput(t, func() error {
		return snapshot.CleanupSnapshot(ctx, &snapshot.SnapShotOpts{
			Volume:    true,
			OlderThan: "0s",
		}, util.OutputTable)
	})
	if err != nil {
		t.Fatalf("CleanupSnapshot volume table: %v", err)
	}
}

func TestIntegration_Cleanup_Share(t *testing.T) {
	shareID := createShare(t)
	snapID := createNFSSnapshot(t, shareID)

	out, err := captureOutput(t, func() error {
		return snapshot.CleanupSnapshot(ctx, &snapshot.SnapShotOpts{
			Share:     true,
			OlderThan: "0s",
		}, util.OutputJSON)
	})
	if err != nil {
		t.Fatalf("CleanupSnapshot share: %v", err)
	}
	if !strings.Contains(out, snapID) {
		t.Errorf("expected deleted NFS snapshot ID %s in cleanup output, got: %s", snapID, out)
	}
}

func TestIntegration_Cleanup_NothingToDelete(t *testing.T) {
	// Use a very short --older-than so freshly created snapshots are NOT deleted.
	volID := createVolume(t)
	createBlockSnapshot(t, volID)

	out, err := captureOutput(t, func() error {
		return snapshot.CleanupSnapshot(ctx, &snapshot.SnapShotOpts{
			Volume:    true,
			OlderThan: "999h",
		}, util.OutputJSON)
	})
	if err != nil {
		t.Fatalf("CleanupSnapshot NothingToDelete: %v", err)
	}
	// Expect an empty JSON array — no snapshots are old enough.
	trimmed := strings.TrimSpace(out)
	if trimmed != "[]" && trimmed != "null" {
		t.Errorf("expected empty list when no snapshots qualify, got: %s", out)
	}
}
