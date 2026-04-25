package snapshot

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
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

	client := newFakeClient(server)
	opts := &SnapShotOpts{SnapshotID: snapID, Volume: true}
	out := captureStdout(t, func() {
		if err := GetSnapshotCmd(context.Background(), opts, "json", client); err != nil {
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

	client := newFakeClient(server)
	opts := &SnapShotOpts{SnapshotID: snapID, Volume: true}
	out := captureStdout(t, func() {
		if err := GetSnapshotCmd(context.Background(), opts, "table", client); err != nil {
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

	client := newFakeClient(server)
	opts := &SnapShotOpts{SnapshotID: snapID, Volume: true}
	err := GetSnapshotCmd(context.Background(), opts, "yaml", client)
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

	client := newFakeClient(server)
	opts := &SnapShotOpts{SnapshotID: snapID, Volume: true}
	if err := GetSnapshotCmd(context.Background(), opts, "json", client); err == nil {
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

	client := newFakeClient(server)
	opts := &SnapShotOpts{SnapshotID: snapID, Share: true}
	out := captureStdout(t, func() {
		if err := GetSnapshotCmd(context.Background(), opts, "json", client); err != nil {
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
	if err := GetSnapshotCmd(context.Background(), opts, "json", nil); err == nil || !strings.Contains(err.Error(), "invalid ID") {
		t.Errorf("expected UUID validation error, got: %v", err)
	}
}

func TestDeleteSnapshotCmd_InvalidUUID(t *testing.T) {
	opts := &SnapShotOpts{SnapshotID: "bad-id", Volume: true}
	if err := DeleteSnapshotCmd(context.Background(), opts, "json", nil); err == nil || !strings.Contains(err.Error(), "invalid ID") {
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

	client := newFakeClient(server)
	opts := &SnapShotOpts{Volume: true}
	out := captureStdout(t, func() {
		if err := ListSnapshotsCmd(context.Background(), opts, "json", client); err != nil {
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

	client := newFakeClient(server)
	opts := &SnapShotOpts{Volume: true}
	out := captureStdout(t, func() {
		if err := ListSnapshotsCmd(context.Background(), opts, "table", client); err != nil {
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

	client := newFakeClient(server)
	opts := &SnapShotOpts{Volume: true}
	if err := ListSnapshotsCmd(context.Background(), opts, "csv", client); err == nil || !strings.Contains(err.Error(), "unsupported output format") {
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

	client := newFakeClient(server)
	opts := &SnapShotOpts{Share: true}
	out := captureStdout(t, func() {
		if err := ListSnapshotsCmd(context.Background(), opts, "json", client); err != nil {
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

	client := newFakeClient(server)
	opts := &SnapShotOpts{SnapshotID: snapID, Volume: true}
	if err := DeleteSnapshotCmd(context.Background(), opts, "json", client); err != nil {
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

	client := newFakeClient(server)
	opts := &SnapShotOpts{SnapshotID: snapID, Volume: true}
	if err := DeleteSnapshotCmd(context.Background(), opts, "json", client); err == nil {
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

	client := newFakeClient(server)
	opts := &SnapShotOpts{SnapshotID: snapID, Share: true}
	if err := DeleteSnapshotCmd(context.Background(), opts, "json", client); err != nil {
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

	client := newFakeClient(server)
	opts := &SnapShotOpts{VolumeID: validUUID2, Name: "test"}
	out := captureStdout(t, func() {
		if err := CreateSnapshotCmd(context.Background(), opts, "json", client); err != nil {
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

	client := newFakeClient(server)
	opts := &SnapShotOpts{VolumeID: validUUID2, Name: "test"}
	if err := CreateSnapshotCmd(context.Background(), opts, "yaml", client); err == nil || !strings.Contains(err.Error(), "unsupported output format") {
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

	client := newFakeClient(server)
	opts := &SnapShotOpts{ShareID: validUUID2, Name: "nfs-test"}
	out := captureStdout(t, func() {
		if err := CreateSnapshotCmd(context.Background(), opts, "json", client); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})
	if !strings.Contains(out, validUUID) {
		t.Errorf("expected snapshot ID in JSON output, got: %s", out)
	}
}

func TestCreateSnapshotCmd_Volume_Table(t *testing.T) {
	server := th.SetupHTTP()
	defer server.Teardown()

	ts := timeNoZ(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC))
	resp := `{"snapshot":{"id":"` + validUUID + `","volume_id":"` + validUUID2 + `","name":"test-snap","status":"creating","size":10,"created_at":"` + ts + `"}}`
	server.Mux.HandleFunc("/snapshots", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		writeBody(w, resp)
	})

	client := newFakeClient(server)
	opts := &SnapShotOpts{VolumeID: validUUID2, Name: "test"}
	out := captureStdout(t, func() {
		if err := CreateSnapshotCmd(context.Background(), opts, "table", client); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})
	if !strings.Contains(out, validUUID) {
		t.Errorf("expected snapshot ID in table output, got: %s", out)
	}
}

func TestCreateSnapshotCmd_Share_Table(t *testing.T) {
	server := th.SetupHTTP()
	defer server.Teardown()

	ts := timeNoZ(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC))
	resp := `{"snapshot":{"id":"` + validUUID + `","share_id":"` + validUUID2 + `","name":"nfs-snap","status":"creating","size":5,"created_at":"` + ts + `"}}`
	server.Mux.HandleFunc("/snapshots", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		writeBody(w, resp)
	})

	client := newFakeClient(server)
	opts := &SnapShotOpts{ShareID: validUUID2, Name: "nfs-test"}
	out := captureStdout(t, func() {
		if err := CreateSnapshotCmd(context.Background(), opts, "table", client); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})
	if !strings.Contains(out, validUUID) {
		t.Errorf("expected snapshot ID in table output, got: %s", out)
	}
}

func TestGetSnapshotCmd_Share_Table(t *testing.T) {
	server := th.SetupHTTP()
	defer server.Teardown()

	snapID := validUUID
	server.Mux.HandleFunc("/snapshots/"+snapID, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		writeBody(w, nfsSnapJSON(snapID, validUUID2))
	})

	client := newFakeClient(server)
	opts := &SnapShotOpts{SnapshotID: snapID, Share: true}
	out := captureStdout(t, func() {
		if err := GetSnapshotCmd(context.Background(), opts, "table", client); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})
	if !strings.Contains(out, snapID) {
		t.Errorf("expected snapshot ID in table output, got: %s", out)
	}
}

func TestDeleteSnapshotCmd_Volume_Table(t *testing.T) {
	server := th.SetupHTTP()
	defer server.Teardown()

	snapID := validUUID
	server.Mux.HandleFunc("/snapshots/"+snapID, func(w http.ResponseWriter, r *http.Request) {
		th.TestMethod(t, r, "DELETE")
		w.WriteHeader(http.StatusNoContent)
	})

	client := newFakeClient(server)
	opts := &SnapShotOpts{SnapshotID: snapID, Volume: true}
	if err := DeleteSnapshotCmd(context.Background(), opts, "table", client); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestDeleteSnapshotCmd_Share_Table(t *testing.T) {
	server := th.SetupHTTP()
	defer server.Teardown()

	snapID := validUUID
	server.Mux.HandleFunc("/snapshots/"+snapID, func(w http.ResponseWriter, r *http.Request) {
		th.TestMethod(t, r, "DELETE")
		w.WriteHeader(http.StatusNoContent)
	})

	client := newFakeClient(server)
	opts := &SnapShotOpts{SnapshotID: snapID, Share: true}
	if err := DeleteSnapshotCmd(context.Background(), opts, "table", client); err != nil {
		t.Errorf("unexpected error deleting NFS snapshot in table format: %v", err)
	}
}

func TestCreateSnapshotCmd_APIError(t *testing.T) {
	server := th.SetupHTTP()
	defer server.Teardown()

	server.Mux.HandleFunc("/snapshots", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		writeBody(w, `{"badRequest":{"message":"Invalid volume_id","code":400}}`)
	})

	client := newFakeClient(server)
	opts := &SnapShotOpts{VolumeID: validUUID2}
	if err := CreateSnapshotCmd(context.Background(), opts, "json", client); err == nil {
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

	client := newFakeClient(server)
	opts := &SnapShotOpts{Volume: true, OlderThan: "168h"}
	captureStdout(t, func() {
		if err := CleanupSnapshot(context.Background(), opts, "json", client); err != nil {
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

	client := newFakeClient(server)
	opts := &SnapShotOpts{Volume: true, OlderThan: "168h"}
	captureStdout(t, func() {
		if err := CleanupSnapshot(context.Background(), opts, "json", client); err != nil {
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

	client := newFakeClient(server)
	opts := &SnapShotOpts{Volume: true, OlderThan: "168h"}
	if err := CleanupSnapshot(context.Background(), opts, "yaml", client); err == nil || !strings.Contains(err.Error(), "unsupported output format") {
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

	client := newFakeClient(server)
	opts := &SnapShotOpts{Share: true, OlderThan: "168h"}
	captureStdout(t, func() {
		if err := CleanupSnapshot(context.Background(), opts, "json", client); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})
	if !deletedNFS {
		t.Error("expected old NFS snapshot to be deleted")
	}
}

func TestCleanupSnapshot_Volume_Table(t *testing.T) {
	server := th.SetupHTTP()
	defer server.Teardown()

	oldID := validUUID
	list := blockSnapListWithAges([]struct{ id, age string }{{oldID, "200h"}})

	server.Mux.HandleFunc("/snapshots", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		writeBody(w, list)
	})

	server.Mux.HandleFunc("/snapshots/"+oldID, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete {
			w.WriteHeader(http.StatusNoContent)
		}
	})

	client := newFakeClient(server)
	opts := &SnapShotOpts{Volume: true, OlderThan: "168h"}
	captureStdout(t, func() {
		if err := CleanupSnapshot(context.Background(), opts, "table", client); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})
}

func TestCleanupSnapshot_Share_Table(t *testing.T) {
	server := th.SetupHTTP()
	defer server.Teardown()

	oldID := validUUID
	server.Mux.HandleFunc("/snapshots/detail", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		writeBody(w, nfsSnapListWithAge(oldID, "200h"))
	})

	server.Mux.HandleFunc("/snapshots/"+oldID, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete {
			w.WriteHeader(http.StatusNoContent)
		}
	})

	client := newFakeClient(server)
	opts := &SnapShotOpts{Share: true, OlderThan: "168h"}
	captureStdout(t, func() {
		if err := CleanupSnapshot(context.Background(), opts, "table", client); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})
}

// ============================================================
// CreateSnapshotCmd — missing VolumeID and ShareID
// ============================================================

func TestCreateSnapshotCmd_NoVolumeOrShare_ReturnsNil(t *testing.T) {
	client := newFakeClient(th.SetupHTTP())
	opts := &SnapShotOpts{Name: "test"}
	if err := CreateSnapshotCmd(context.Background(), opts, "json", client); err != nil {
		t.Errorf("expected no error when neither volume nor share specified, got: %v", err)
	}
}

// ============================================================
// Additional error path tests
// ============================================================

func TestCreateSnapshotCmd_Volume_InvalidOutputFormat(t *testing.T) {
	server := th.SetupHTTP()
	defer server.Teardown()

	ts := timeNoZ(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC))
	resp := `{"snapshot":{"id":"` + validUUID + `","volume_id":"` + validUUID2 + `","status":"creating","size":10,"created_at":"` + ts + `"}}`
	server.Mux.HandleFunc("/snapshots", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		writeBody(w, resp)
	})

	client := newFakeClient(server)
	opts := &SnapShotOpts{VolumeID: validUUID2}
	if err := CreateSnapshotCmd(context.Background(), opts, "xml", client); err == nil || !strings.Contains(err.Error(), "unsupported output format") {
		t.Errorf("expected unsupported format error, got: %v", err)
	}
}

func TestCreateSnapshotCmd_Share_InvalidOutputFormat(t *testing.T) {
	server := th.SetupHTTP()
	defer server.Teardown()

	ts := timeNoZ(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC))
	resp := `{"snapshot":{"id":"` + validUUID + `","share_id":"` + validUUID2 + `","status":"creating","created_at":"` + ts + `"}}`
	server.Mux.HandleFunc("/snapshots", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		writeBody(w, resp)
	})

	client := newFakeClient(server)
	opts := &SnapShotOpts{ShareID: validUUID2}
	if err := CreateSnapshotCmd(context.Background(), opts, "xml", client); err == nil || !strings.Contains(err.Error(), "unsupported output format") {
		t.Errorf("expected unsupported format error, got: %v", err)
	}
}

func TestListSnapshotsCmd_Volume_Empty(t *testing.T) {
	server := th.SetupHTTP()
	defer server.Teardown()

	server.Mux.HandleFunc("/snapshots", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		writeBody(w, `{"snapshots":[]}`)
	})

	client := newFakeClient(server)
	opts := &SnapShotOpts{Volume: true}
	out := captureStdout(t, func() {
		if err := ListSnapshotsCmd(context.Background(), opts, "json", client); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})
	trimmed := strings.TrimSpace(out)
	if trimmed != "[]" && trimmed != "null" {
		t.Errorf("expected empty list, got: %s", out)
	}
}

func TestListSnapshotsCmd_Share_Empty(t *testing.T) {
	server := th.SetupHTTP()
	defer server.Teardown()

	server.Mux.HandleFunc("/snapshots/detail", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		writeBody(w, `{"snapshots":[]}`)
	})

	client := newFakeClient(server)
	opts := &SnapShotOpts{Share: true}
	out := captureStdout(t, func() {
		if err := ListSnapshotsCmd(context.Background(), opts, "json", client); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})
	trimmed := strings.TrimSpace(out)
	if trimmed != "[]" && trimmed != "null" {
		t.Errorf("expected empty list, got: %s", out)
	}
}

func TestDeleteSnapshotCmd_Share_404(t *testing.T) {
	server := th.SetupHTTP()
	defer server.Teardown()

	snapID := validUUID
	server.Mux.HandleFunc("/snapshots/"+snapID, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		writeBody(w, `{"itemNotFound":{"message":"Snapshot not found","code":404}}`)
	})

	client := newFakeClient(server)
	opts := &SnapShotOpts{SnapshotID: snapID, Share: true}
	if err := DeleteSnapshotCmd(context.Background(), opts, "json", client); err == nil {
		t.Error("expected error for 404 response, got nil")
	}
}

// ============================================================
// Cleanup — partial failure & cancellation
// ============================================================

// blockSnapListOldEnough builds a snapshots list with all ids aged 500h
// (well past the 168h default cleanup threshold).
func blockSnapListOldEnough(ids []string) string {
	ts := timeNoZ(time.Now().Add(-500 * time.Hour))
	parts := make([]string, len(ids))
	for i, id := range ids {
		parts[i] = `{"id":"` + id + `","volume_id":"vol1","name":"snap","status":"available","size":10,"metadata":{},"created_at":"` + ts + `","updated_at":"` + ts + `"}`
	}
	return `{"snapshots":[` + strings.Join(parts, ",") + `]}`
}

// TestCleanupSnapshot_Volume_PartialFail simulates a scenario where one DELETE returns
// 500 and the others succeed; the returned JSON must contain only the successful IDs.
func TestCleanupSnapshot_Volume_PartialFail(t *testing.T) {
	server := th.SetupHTTP()
	defer server.Teardown()

	okA := "aaaaaaaa-1111-1111-1111-111111111111"
	okB := "bbbbbbbb-2222-2222-2222-222222222222"
	failC := "cccccccc-3333-3333-3333-333333333333"

	server.Mux.HandleFunc("/snapshots", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		writeBody(w, blockSnapListOldEnough([]string{okA, okB, failC}))
	})

	for _, id := range []string{okA, okB} {
		server.Mux.HandleFunc("/snapshots/"+id, func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodDelete {
				w.WriteHeader(http.StatusNoContent)
			}
		})
	}
	server.Mux.HandleFunc("/snapshots/"+failC, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete {
			w.WriteHeader(http.StatusInternalServerError)
			writeBody(w, `{"error":{"message":"backend unavailable","code":500}}`)
		}
	})

	client := newFakeClient(server)
	opts := &SnapShotOpts{Volume: true, OlderThan: "168h"}
	out := captureStdout(t, func() {
		if err := CleanupSnapshot(context.Background(), opts, "json", client); err != nil {
			t.Fatalf("CleanupSnapshot returned error: %v", err)
		}
	})
	var deleted []string
	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &deleted); err != nil {
		t.Fatalf("output is not valid JSON array: %v\nout=%s", err, out)
	}
	if len(deleted) != 2 {
		t.Fatalf("expected exactly 2 successful deletions, got %d: %v", len(deleted), deleted)
	}
	for _, id := range deleted {
		if id == failC {
			t.Errorf("failed snapshot %s should NOT appear in deleted list", failC)
		}
	}
}

// TestCleanupSnapshot_Volume_DryRun_SkipsDeletes verifies that --dry-run reports
// the candidate IDs without issuing any DELETE calls.
func TestCleanupSnapshot_Volume_DryRun_SkipsDeletes(t *testing.T) {
	server := th.SetupHTTP()
	defer server.Teardown()

	oldID := validUUID
	server.Mux.HandleFunc("/snapshots", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		writeBody(w, blockSnapListOldEnough([]string{oldID}))
	})
	server.Mux.HandleFunc("/snapshots/"+oldID, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete {
			t.Error("dry-run must NOT issue DELETE requests")
		}
	})

	client := newFakeClient(server)
	opts := &SnapShotOpts{Volume: true, OlderThan: "168h", DryRun: true}
	out := captureStdout(t, func() {
		if err := CleanupSnapshot(context.Background(), opts, "json", client); err != nil {
			t.Fatalf("CleanupSnapshot returned error: %v", err)
		}
	})
	if !strings.Contains(out, oldID) {
		t.Errorf("dry-run output should list candidate %s, got: %s", oldID, out)
	}
}

// TestCleanupSnapshot_Volume_CtxCancelledBeforeStart verifies that a cancelled
// context produces no deletions and does not panic. We cancel before CleanupSnapshot
// runs, then confirm the list call itself fails cleanly.
func TestCleanupSnapshot_Volume_CtxCancelledBeforeStart(t *testing.T) {
	server := th.SetupHTTP()
	defer server.Teardown()

	server.Mux.HandleFunc("/snapshots", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		writeBody(w, blockSnapListOldEnough([]string{validUUID}))
	})
	server.Mux.HandleFunc("/snapshots/"+validUUID, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete {
			t.Error("cancelled context must NOT issue DELETE requests")
		}
	})

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel before calling

	client := newFakeClient(server)
	opts := &SnapShotOpts{Volume: true, OlderThan: "168h"}
	// With a cancelled context, the AllPages call should return an error.
	// We don't assert the exact error string — only that we don't panic and
	// return promptly rather than hanging.
	done := make(chan struct{})
	go func() {
		defer close(done)
		//nolint:errcheck // intentionally discarded — test asserts on timing, not error
		CleanupSnapshot(ctx, opts, "json", client)
	}()
	select {
	case <-done:
		// ok — returned quickly
	case <-time.After(2 * time.Second):
		t.Fatal("CleanupSnapshot did not return within 2s on cancelled context")
	}
}

// TestCleanupSnapshot_Volume_CtxCancelledMidway queues many deletes with a small
// semaphore so some are waiting on it, then cancels the context. Cleanup must
// return promptly — the in-progress workers exit cleanly and the queued ones
// are skipped.
func TestCleanupSnapshot_Volume_CtxCancelledMidway(t *testing.T) {
	server := th.SetupHTTP()
	defer server.Teardown()

	// Use 20 IDs; semaphore is 5, so most wait.
	ids := make([]string, 20)
	for i := range ids {
		ids[i] = fmt.Sprintf("aaaaaaaa-0000-0000-0000-%012d", i)
	}

	server.Mux.HandleFunc("/snapshots", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		writeBody(w, blockSnapListOldEnough(ids))
	})

	// Each DELETE takes 200ms — plenty of time to cancel.
	deleted := make(map[string]bool)
	var mu sync.Mutex
	for _, id := range ids {
		server.Mux.HandleFunc("/snapshots/"+id, func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodDelete {
				return
			}
			time.Sleep(200 * time.Millisecond)
			mu.Lock()
			deleted[id] = true
			mu.Unlock()
			w.WriteHeader(http.StatusNoContent)
		})
	}

	ctx, cancel := context.WithCancel(context.Background())
	client := newFakeClient(server)
	opts := &SnapShotOpts{Volume: true, OlderThan: "168h"}

	// Cancel 100ms after Cleanup starts — some DELETEs will be in flight but
	// most will be waiting for the semaphore.
	go func() {
		time.Sleep(100 * time.Millisecond)
		cancel()
	}()

	done := make(chan struct{})
	// Stderr gets written to on cancellation; silence it so the test output stays clean.
	stderrOld := os.Stderr
	stderrR, stderrW, pipeErr := os.Pipe()
	if pipeErr != nil {
		t.Fatalf("os.Pipe: %v", pipeErr)
	}
	os.Stderr = stderrW
	go func() {
		defer close(done)
		captureStdout(t, func() {
			//nolint:errcheck // intentionally discarded — test asserts on timing, not error
			CleanupSnapshot(ctx, opts, "json", client)
		})
	}()

	select {
	case <-done:
		// ok
	case <-time.After(5 * time.Second):
		t.Fatal("CleanupSnapshot did not return within 5s after cancellation")
	}
	if err := stderrW.Close(); err != nil {
		t.Logf("stderrW.Close: %v", err)
	}
	os.Stderr = stderrOld
	// Drain stderr so the pipe doesn't block GC.
	if _, err := io.Copy(io.Discard, stderrR); err != nil {
		t.Logf("stderr drain: %v", err)
	}

	mu.Lock()
	n := len(deleted)
	mu.Unlock()
	if n >= len(ids) {
		t.Errorf("cancellation should have stopped some deletions, got %d/%d", n, len(ids))
	}
}

// ============================================================
// ListSnapshotsCmd — pagination
// ============================================================

// TestListSnapshotsCmd_Volume_Paginated verifies that AllPages follows the
// snapshots_links[next] URL and returns the union of all pages.
func TestListSnapshotsCmd_Volume_Paginated(t *testing.T) {
	server := th.SetupHTTP()
	defer server.Teardown()

	p1ID := "11111111-1111-1111-1111-111111111111"
	p2ID := "22222222-2222-2222-2222-222222222222"

	ts := timeNoZ(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC))
	nextURL := server.Server.URL + "/snapshots?marker=" + p1ID
	p1 := `{"snapshots":[{"id":"` + p1ID + `","volume_id":"vol1","name":"snap1","status":"available","size":10,"metadata":{},"created_at":"` + ts + `","updated_at":"` + ts + `"}],"snapshots_links":[{"rel":"next","href":"` + nextURL + `"}]}`
	p2 := `{"snapshots":[{"id":"` + p2ID + `","volume_id":"vol1","name":"snap2","status":"available","size":10,"metadata":{},"created_at":"` + ts + `","updated_at":"` + ts + `"}]}`

	server.Mux.HandleFunc("/snapshots", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if r.URL.Query().Get("marker") == p1ID {
			writeBody(w, p2)
		} else {
			writeBody(w, p1)
		}
	})

	client := newFakeClient(server)
	opts := &SnapShotOpts{Volume: true}
	out := captureStdout(t, func() {
		if err := ListSnapshotsCmd(context.Background(), opts, "json", client); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
	if !strings.Contains(out, p1ID) {
		t.Errorf("expected page-1 ID %s in output: %s", p1ID, out)
	}
	if !strings.Contains(out, p2ID) {
		t.Errorf("expected page-2 ID %s in output (pagination not followed): %s", p2ID, out)
	}
}

// Note: NFS (Manila) pagination is NOT covered by an analogous test here because
// gophercloud's NFS SnapshotPage.NextPageURL relies on the "limit" query parameter
// being set on the original request. Our production code passes ListOpts{} with no
// Limit, so Manila always returns a single page regardless of server-side links —
// pagination is effectively off for the NFS path. A test exercising it would not
// reflect production behavior.
