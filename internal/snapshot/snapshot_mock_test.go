package snapshot

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gophercloud/gophercloud/v2"
	th "github.com/gophercloud/gophercloud/v2/testhelper"
	thclient "github.com/gophercloud/gophercloud/v2/testhelper/client"
)

// captureStdout captures anything written to os.Stdout during fn.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	old := os.Stdout
	os.Stdout = w
	fn()
	w.Close()
	os.Stdout = old
	var buf bytes.Buffer
	if _, err = io.Copy(&buf, r); err != nil {
		t.Fatalf("io.Copy: %v", err)
	}
	return buf.String()
}

// writeBody writes body to w; panics on error (acceptable in test HTTP handlers).
func writeBody(w http.ResponseWriter, body string) {
	if _, err := w.Write([]byte(body)); err != nil {
		panic("test handler write: " + err.Error())
	}
}

// newFakeClient builds a gophercloud ServiceClient pointing at the fake test server.
func newFakeClient(server th.FakeServer) *gophercloud.ServiceClient {
	c := thclient.ServiceClient(server)
	c.ResourceBase = server.Endpoint()
	return c
}

// timeNoZ formats a time in JSONRFC3339MilliNoZ style used by gophercloud.
func timeNoZ(t time.Time) string {
	return t.UTC().Format("2006-01-02T15:04:05.000000")
}

const validUUID = "12345678-1234-1234-1234-123456789012"
const validUUID2 = "abcdefab-abcd-abcd-abcd-abcdefabcdef"

// --- helpers ---

func blockSnapJSON(id, volID string) string {
	ts := timeNoZ(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC))
	return `{"snapshot":{"id":"` + id + `","volume_id":"` + volID + `","name":"snap-` + id + `","description":"test","status":"available","size":10,"metadata":{},"created_at":"` + ts + `","updated_at":"` + ts + `"}}`
}

func blockSnapListJSON(ids []string) string {
	ts := timeNoZ(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC))
	parts := make([]string, len(ids))
	for i, id := range ids {
		parts[i] = `{"id":"` + id + `","volume_id":"` + validUUID + `","name":"snap","status":"available","size":10,"metadata":{},"created_at":"` + ts + `","updated_at":"` + ts + `"}`
	}
	return `{"snapshots":[` + strings.Join(parts, ",") + `]}`
}

func nfsSnapJSON(id, shareID string) string {
	ts := timeNoZ(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC))
	return `{"snapshot":{"id":"` + id + `","share_id":"` + shareID + `","name":"nfs-snap","description":"test","status":"available","size":5,"share_proto":"NFS","share_size":5,"created_at":"` + ts + `"}}`
}

func nfsSnapListJSON(ids []string) string {
	ts := timeNoZ(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC))
	parts := make([]string, len(ids))
	for i, id := range ids {
		parts[i] = `{"id":"` + id + `","share_id":"` + validUUID + `","name":"nfs-snap","status":"available","size":5,"share_proto":"NFS","share_size":5,"created_at":"` + ts + `"}`
	}
	return `{"snapshots":[` + strings.Join(parts, ",") + `]}`
}

func blockSnapListWithAges(snaps []struct{ id, age string }) string {
	parts := make([]string, len(snaps))
	for i, s := range snaps {
		ts := timeNoZ(time.Now().Add(-parseAge(s.age)))
		parts[i] = `{"id":"` + s.id + `","volume_id":"vol1","name":"snap","status":"available","size":10,"metadata":{},"created_at":"` + ts + `","updated_at":"` + ts + `"}`
	}
	return `{"snapshots":[` + strings.Join(parts, ",") + `]}`
}

func nfsSnapListWithAge(id, age string) string {
	ts := timeNoZ(time.Now().Add(-parseAge(age)))
	return `{"snapshots":[{"id":"` + id + `","share_id":"sh1","name":"old-nfs","status":"available","size":5,"share_proto":"NFS","share_size":5,"created_at":"` + ts + `"}]}`
}

func parseAge(s string) time.Duration {
	d, err := time.ParseDuration(s)
	if err != nil {
		panic("invalid duration in test: " + err.Error())
	}
	return d
}

// ============================================================
// GetSnapshotCmd — block storage
// ============================================================

func TestGetSnapshotCmd_Volume_JSON(t *testing.T) {
	server := th.SetupHTTP()
	defer server.Teardown()

	snapID := validUUID
	server.Mux.HandleFunc("/snapshots/"+snapID, func(w http.ResponseWriter, r *http.Request) {
		th.TestMethod(t, r, "GET")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		writeBody(w, blockSnapJSON(snapID, validUUID2))
	})

	opts := &SnapShotOpts{SnapshotID: snapID, Volume: true, client: newFakeClient(server)}
	out := captureStdout(t, func() {
		if err := GetSnapshotCmd(context.Background(), opts, "json"); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})
	if !strings.Contains(out, snapID) {
		t.Errorf("expected snapshot ID in JSON output, got: %s", out)
	}
	var m map[string]any
	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &m); err != nil {
		t.Errorf("output is not valid JSON: %v\noutput: %s", err, out)
	}
}

func TestGetSnapshotCmd_Volume_Table(t *testing.T) {
	server := th.SetupHTTP()
	defer server.Teardown()

	snapID := validUUID
	server.Mux.HandleFunc("/snapshots/"+snapID, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		writeBody(w, blockSnapJSON(snapID, validUUID2))
	})

	opts := &SnapShotOpts{SnapshotID: snapID, Volume: true, client: newFakeClient(server)}
	out := captureStdout(t, func() {
		if err := GetSnapshotCmd(context.Background(), opts, "table"); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})
	if !strings.Contains(out, snapID) {
		t.Errorf("expected snapshot ID in table output, got: %s", out)
	}
}

func TestGetSnapshotCmd_Volume_UnsupportedFormat(t *testing.T) {
	server := th.SetupHTTP()
	defer server.Teardown()

	snapID := validUUID
	server.Mux.HandleFunc("/snapshots/"+snapID, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		writeBody(w, blockSnapJSON(snapID, validUUID2))
	})

	opts := &SnapShotOpts{SnapshotID: snapID, Volume: true, client: newFakeClient(server)}
	err := GetSnapshotCmd(context.Background(), opts, "yaml")
	if err == nil || !strings.Contains(err.Error(), "unsupported output format") {
		t.Errorf("expected unsupported format error, got: %v", err)
	}
}

func TestGetSnapshotCmd_Volume_404(t *testing.T) {
	server := th.SetupHTTP()
	defer server.Teardown()

	snapID := validUUID
	server.Mux.HandleFunc("/snapshots/"+snapID, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		writeBody(w, `{"itemNotFound":{"message":"Snapshot not found","code":404}}`)
	})

	opts := &SnapShotOpts{SnapshotID: snapID, Volume: true, client: newFakeClient(server)}
	if err := GetSnapshotCmd(context.Background(), opts, "json"); err == nil {
		t.Error("expected error for 404 response, got nil")
	}
}

func TestGetSnapshotCmd_Share_JSON(t *testing.T) {
	server := th.SetupHTTP()
	defer server.Teardown()

	snapID := validUUID
	server.Mux.HandleFunc("/snapshots/"+snapID, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		writeBody(w, nfsSnapJSON(snapID, validUUID2))
	})

	opts := &SnapShotOpts{SnapshotID: snapID, Share: true, client: newFakeClient(server)}
	out := captureStdout(t, func() {
		if err := GetSnapshotCmd(context.Background(), opts, "json"); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})
	if !strings.Contains(out, snapID) {
		t.Errorf("expected snapshot ID in NFS output, got: %s", out)
	}
}

// ============================================================
// UUID validation
// ============================================================

func TestGetSnapshotCmd_InvalidUUID(t *testing.T) {
	opts := &SnapShotOpts{SnapshotID: "not-a-uuid", Volume: true}
	if err := GetSnapshotCmd(context.Background(), opts, "json"); err == nil || !strings.Contains(err.Error(), "invalid ID") {
		t.Errorf("expected UUID validation error, got: %v", err)
	}
}

func TestDeleteSnapshotCmd_InvalidUUID(t *testing.T) {
	opts := &SnapShotOpts{SnapshotID: "bad-id", Volume: true}
	if err := DeleteSnapshotCmd(context.Background(), opts, "json"); err == nil || !strings.Contains(err.Error(), "invalid ID") {
		t.Errorf("expected UUID validation error, got: %v", err)
	}
}

// ============================================================
// ListSnapshotsCmd — block storage
// ============================================================

func TestListSnapshotsCmd_Volume_JSON(t *testing.T) {
	server := th.SetupHTTP()
	defer server.Teardown()

	// block storage List uses /snapshots (not /snapshots/detail).
	server.Mux.HandleFunc("/snapshots", func(w http.ResponseWriter, r *http.Request) {
		th.TestMethod(t, r, "GET")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		writeBody(w, blockSnapListJSON([]string{validUUID, validUUID2}))
	})

	opts := &SnapShotOpts{Volume: true, client: newFakeClient(server)}
	out := captureStdout(t, func() {
		if err := ListSnapshotsCmd(context.Background(), opts, "json"); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})
	if !strings.Contains(out, validUUID) {
		t.Errorf("expected first snapshot ID in output, got: %s", out)
	}
	if !strings.Contains(out, validUUID2) {
		t.Errorf("expected second snapshot ID in output, got: %s", out)
	}
	var list []map[string]any
	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &list); err != nil {
		t.Errorf("output is not valid JSON array: %v", err)
	}
	if len(list) != 2 {
		t.Errorf("expected 2 snapshots, got %d", len(list))
	}
}

func TestListSnapshotsCmd_Volume_Table(t *testing.T) {
	server := th.SetupHTTP()
	defer server.Teardown()

	server.Mux.HandleFunc("/snapshots", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		writeBody(w, blockSnapListJSON([]string{validUUID}))
	})

	opts := &SnapShotOpts{Volume: true, client: newFakeClient(server)}
	out := captureStdout(t, func() {
		if err := ListSnapshotsCmd(context.Background(), opts, "table"); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})
	if !strings.Contains(out, validUUID) {
		t.Errorf("expected snapshot in table output, got: %s", out)
	}
}

func TestListSnapshotsCmd_Volume_UnsupportedFormat(t *testing.T) {
	server := th.SetupHTTP()
	defer server.Teardown()

	server.Mux.HandleFunc("/snapshots", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		writeBody(w, blockSnapListJSON([]string{validUUID}))
	})

	opts := &SnapShotOpts{Volume: true, client: newFakeClient(server)}
	if err := ListSnapshotsCmd(context.Background(), opts, "csv"); err == nil || !strings.Contains(err.Error(), "unsupported output format") {
		t.Errorf("expected unsupported format error, got: %v", err)
	}
}

// ============================================================
// ListSnapshotsCmd — NFS
// ============================================================

func TestListSnapshotsCmd_Share_JSON(t *testing.T) {
	server := th.SetupHTTP()
	defer server.Teardown()

	server.Mux.HandleFunc("/snapshots/detail", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		writeBody(w, nfsSnapListJSON([]string{validUUID, validUUID2}))
	})

	opts := &SnapShotOpts{Share: true, client: newFakeClient(server)}
	out := captureStdout(t, func() {
		if err := ListSnapshotsCmd(context.Background(), opts, "json"); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})
	if !strings.Contains(out, validUUID) {
		t.Errorf("expected first NFS snapshot in output, got: %s", out)
	}
}

// ============================================================
// DeleteSnapshotCmd
// ============================================================

func TestDeleteSnapshotCmd_Volume_Success(t *testing.T) {
	server := th.SetupHTTP()
	defer server.Teardown()

	snapID := validUUID
	server.Mux.HandleFunc("/snapshots/"+snapID, func(w http.ResponseWriter, r *http.Request) {
		th.TestMethod(t, r, "DELETE")
		w.WriteHeader(http.StatusNoContent)
	})

	opts := &SnapShotOpts{SnapshotID: snapID, Volume: true, client: newFakeClient(server)}
	if err := DeleteSnapshotCmd(context.Background(), opts, "json"); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestDeleteSnapshotCmd_Volume_404(t *testing.T) {
	server := th.SetupHTTP()
	defer server.Teardown()

	snapID := validUUID
	server.Mux.HandleFunc("/snapshots/"+snapID, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		writeBody(w, `{"itemNotFound":{"message":"Snapshot not found","code":404}}`)
	})

	opts := &SnapShotOpts{SnapshotID: snapID, Volume: true, client: newFakeClient(server)}
	if err := DeleteSnapshotCmd(context.Background(), opts, "json"); err == nil {
		t.Error("expected error for 404 response, got nil")
	}
}

func TestDeleteSnapshotCmd_Share_Success(t *testing.T) {
	server := th.SetupHTTP()
	defer server.Teardown()

	snapID := validUUID
	server.Mux.HandleFunc("/snapshots/"+snapID, func(w http.ResponseWriter, r *http.Request) {
		th.TestMethod(t, r, "DELETE")
		w.WriteHeader(http.StatusNoContent)
	})

	opts := &SnapShotOpts{SnapshotID: snapID, Share: true, client: newFakeClient(server)}
	if err := DeleteSnapshotCmd(context.Background(), opts, "json"); err != nil {
		t.Errorf("unexpected error deleting NFS snapshot: %v", err)
	}
}

// ============================================================
// CreateSnapshotCmd — block storage
// ============================================================

func TestCreateSnapshotCmd_Volume_JSON(t *testing.T) {
	server := th.SetupHTTP()
	defer server.Teardown()

	ts := timeNoZ(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC))
	resp := `{"snapshot":{"id":"` + validUUID + `","volume_id":"` + validUUID2 + `","name":"test-snap","status":"creating","size":10,"metadata":{},"created_at":"` + ts + `","updated_at":"` + ts + `"}}`
	server.Mux.HandleFunc("/snapshots", func(w http.ResponseWriter, r *http.Request) {
		th.TestMethod(t, r, "POST")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		writeBody(w, resp)
	})

	opts := &SnapShotOpts{VolumeID: validUUID2, Name: "test", client: newFakeClient(server)}
	out := captureStdout(t, func() {
		if err := CreateSnapshotCmd(context.Background(), opts, "json"); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})
	if !strings.Contains(out, validUUID) {
		t.Errorf("expected snapshot ID in JSON output, got: %s", out)
	}
	var m map[string]any
	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &m); err != nil {
		t.Errorf("output is not valid JSON: %v", err)
	}
}

func TestCreateSnapshotCmd_Volume_UnsupportedFormat(t *testing.T) {
	server := th.SetupHTTP()
	defer server.Teardown()

	ts := timeNoZ(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC))
	resp := `{"snapshot":{"id":"` + validUUID + `","volume_id":"` + validUUID2 + `","name":"test-snap","status":"creating","size":10,"metadata":{},"created_at":"` + ts + `","updated_at":"` + ts + `"}}`
	server.Mux.HandleFunc("/snapshots", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		writeBody(w, resp)
	})

	opts := &SnapShotOpts{VolumeID: validUUID2, Name: "test", client: newFakeClient(server)}
	if err := CreateSnapshotCmd(context.Background(), opts, "yaml"); err == nil || !strings.Contains(err.Error(), "unsupported output format") {
		t.Errorf("expected unsupported format error, got: %v", err)
	}
}

func TestCreateSnapshotCmd_Share_JSON(t *testing.T) {
	server := th.SetupHTTP()
	defer server.Teardown()

	ts := timeNoZ(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC))
	resp := `{"snapshot":{"id":"` + validUUID + `","share_id":"` + validUUID2 + `","name":"nfs-snap","status":"creating","size":5,"share_size":5,"share_proto":"NFS","created_at":"` + ts + `"}}`
	server.Mux.HandleFunc("/snapshots", func(w http.ResponseWriter, r *http.Request) {
		th.TestMethod(t, r, "POST")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		writeBody(w, resp)
	})

	opts := &SnapShotOpts{ShareID: validUUID2, Name: "nfs-test", client: newFakeClient(server)}
	out := captureStdout(t, func() {
		if err := CreateSnapshotCmd(context.Background(), opts, "json"); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})
	if !strings.Contains(out, validUUID) {
		t.Errorf("expected snapshot ID in JSON output, got: %s", out)
	}
}

func TestCreateSnapshotCmd_APIError(t *testing.T) {
	server := th.SetupHTTP()
	defer server.Teardown()

	server.Mux.HandleFunc("/snapshots", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		writeBody(w, `{"badRequest":{"message":"Invalid volume_id","code":400}}`)
	})

	opts := &SnapShotOpts{VolumeID: validUUID2, client: newFakeClient(server)}
	if err := CreateSnapshotCmd(context.Background(), opts, "json"); err == nil {
		t.Error("expected error for 400 response, got nil")
	}
}

// ============================================================
// CleanupSnapshot — block storage
// ============================================================

func TestCleanupSnapshot_Volume_DeletesOld(t *testing.T) {
	server := th.SetupHTTP()
	defer server.Teardown()

	oldID := validUUID
	recentID := validUUID2
	list := blockSnapListWithAges([]struct{ id, age string }{{oldID, "200h"}, {recentID, "1h"}})

	server.Mux.HandleFunc("/snapshots", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		writeBody(w, list)
	})

	deletedIDs := []string{}
	server.Mux.HandleFunc("/snapshots/"+oldID, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete {
			deletedIDs = append(deletedIDs, oldID)
			w.WriteHeader(http.StatusNoContent)
		}
	})
	server.Mux.HandleFunc("/snapshots/"+recentID, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete {
			t.Error("recent snapshot should NOT be deleted")
		}
	})

	opts := &SnapShotOpts{Volume: true, OlderThan: "168h", client: newFakeClient(server)}
	captureStdout(t, func() {
		if err := CleanupSnapshot(context.Background(), opts, "json"); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})
	if len(deletedIDs) != 1 || deletedIDs[0] != oldID {
		t.Errorf("expected old snapshot %s deleted, got: %v", oldID, deletedIDs)
	}
}

func TestCleanupSnapshot_Volume_NoneOldEnough(t *testing.T) {
	server := th.SetupHTTP()
	defer server.Teardown()

	list := blockSnapListWithAges([]struct{ id, age string }{{validUUID, "1h"}})
	server.Mux.HandleFunc("/snapshots", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		writeBody(w, list)
	})

	opts := &SnapShotOpts{Volume: true, OlderThan: "168h", client: newFakeClient(server)}
	captureStdout(t, func() {
		if err := CleanupSnapshot(context.Background(), opts, "json"); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})
}

func TestCleanupSnapshot_Volume_UnsupportedFormat(t *testing.T) {
	server := th.SetupHTTP()
	defer server.Teardown()

	list := blockSnapListWithAges([]struct{ id, age string }{{validUUID, "200h"}})
	server.Mux.HandleFunc("/snapshots", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		writeBody(w, list)
	})
	server.Mux.HandleFunc("/snapshots/"+validUUID, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	opts := &SnapShotOpts{Volume: true, OlderThan: "168h", client: newFakeClient(server)}
	if err := CleanupSnapshot(context.Background(), opts, "yaml"); err == nil || !strings.Contains(err.Error(), "unsupported output format") {
		t.Errorf("expected unsupported format error, got: %v", err)
	}
}

// ============================================================
// CleanupSnapshot — NFS
// ============================================================

func TestCleanupSnapshot_Share_DeletesOld(t *testing.T) {
	server := th.SetupHTTP()
	defer server.Teardown()

	oldID := validUUID
	server.Mux.HandleFunc("/snapshots/detail", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		writeBody(w, nfsSnapListWithAge(oldID, "200h"))
	})

	deletedNFS := false
	server.Mux.HandleFunc("/snapshots/"+oldID, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete {
			deletedNFS = true
			w.WriteHeader(http.StatusNoContent)
		}
	})

	opts := &SnapShotOpts{Share: true, OlderThan: "168h", client: newFakeClient(server)}
	captureStdout(t, func() {
		if err := CleanupSnapshot(context.Background(), opts, "json"); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})
	if !deletedNFS {
		t.Error("expected old NFS snapshot to be deleted")
	}
}
