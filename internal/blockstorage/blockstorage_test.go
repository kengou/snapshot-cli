package blockstorage

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"

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

// newFakeBlockClient creates a gophercloud ServiceClient pointing at the fake server.
func newFakeBlockClient(server th.FakeServer) *gophercloud.ServiceClient {
	c := thclient.ServiceClient(server)
	c.ResourceBase = server.Endpoint()
	return c
}

func fakeVolume(id string) string {
	return `{
		"volume": {
			"id": "` + id + `",
			"status": "available",
			"size": 10,
			"availability_zone": "nova",
			"name": "test-vol",
			"description": "desc",
			"volume_type": "ssd",
			"bootable": "false",
			"encrypted": false,
			"replication_status": "enabled",
			"multiattach": false
		}
	}`
}

func fakeVolumeList() string {
	return `{
		"volumes": [
			{"id": "vol-aaa", "status": "available", "size": 10, "availability_zone": "nova",
			 "name": "v1", "bootable": "false", "encrypted": false, "replication_status": "enabled", "multiattach": false},
			{"id": "vol-bbb", "status": "in-use", "size": 20, "availability_zone": "nova",
			 "name": "v2", "bootable": "true", "encrypted": false, "replication_status": "enabled", "multiattach": false}
		]
	}`
}

// --- getBlockStorage tests ---

func TestGetBlockStorage_ValidUUID_JSON(t *testing.T) {
	server := th.SetupHTTP()
	defer server.Teardown()

	volID := "11111111-1111-1111-1111-111111111111"
	server.Mux.HandleFunc("/volumes/"+volID, func(w http.ResponseWriter, r *http.Request) {
		th.TestMethod(t, r, "GET")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		writeBody(w, fakeVolume(volID))
	})

	client := newFakeBlockClient(server)
	out := captureStdout(t, func() {
		if err := GetBlockStorage(context.Background(), volID, "json", client); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	if !strings.Contains(out, volID) {
		t.Errorf("expected volume ID in JSON output, got: %s", out)
	}
	var m map[string]any
	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &m); err != nil {
		t.Errorf("output is not valid JSON: %v\noutput: %s", err, out)
	}
}

func TestGetBlockStorage_ValidUUID_Table(t *testing.T) {
	server := th.SetupHTTP()
	defer server.Teardown()

	volID := "22222222-2222-2222-2222-222222222222"
	server.Mux.HandleFunc("/volumes/"+volID, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		writeBody(w, fakeVolume(volID))
	})

	client := newFakeBlockClient(server)
	out := captureStdout(t, func() {
		if err := GetBlockStorage(context.Background(), volID, "table", client); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})
	if !strings.Contains(out, volID) {
		t.Errorf("expected volume ID in table output, got: %s", out)
	}
}

func TestGetBlockStorage_UnsupportedFormat_ReturnsError(t *testing.T) {
	server := th.SetupHTTP()
	defer server.Teardown()

	volID := "33333333-3333-3333-3333-333333333333"
	server.Mux.HandleFunc("/volumes/"+volID, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		writeBody(w, fakeVolume(volID))
	})

	client := newFakeBlockClient(server)
	err := GetBlockStorage(context.Background(), volID, "yaml", client)
	if err == nil || !strings.Contains(err.Error(), "unsupported output format") {
		t.Errorf("expected unsupported format error, got: %v", err)
	}
}

func TestGetBlockStorage_404_ReturnsError(t *testing.T) {
	server := th.SetupHTTP()
	defer server.Teardown()

	volID := "44444444-4444-4444-4444-444444444444"
	server.Mux.HandleFunc("/volumes/"+volID, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		writeBody(w, `{"itemNotFound": {"message": "Volume not found", "code": 404}}`)
	})

	client := newFakeBlockClient(server)
	err := GetBlockStorage(context.Background(), volID, "json", client)
	if err == nil {
		t.Error("expected error for 404 response, got nil")
	}
}

// UUID validation runs before any client call, so nil client is fine here.

func TestGetBlockStorage_InvalidUUID(t *testing.T) {
	err := GetBlockStorage(context.Background(), "not-a-uuid", "json", nil)
	if err == nil || !strings.Contains(err.Error(), "invalid ID") {
		t.Errorf("expected UUID validation error, got: %v", err)
	}
}

func TestGetBlockStorage_EmptyUUID(t *testing.T) {
	err := GetBlockStorage(context.Background(), "", "json", nil)
	if err == nil || !strings.Contains(err.Error(), "invalid ID") {
		t.Errorf("expected UUID validation error for empty ID, got: %v", err)
	}
}

// --- listBlockStorage tests ---

func TestListBlockStorage_JSON(t *testing.T) {
	server := th.SetupHTTP()
	defer server.Teardown()

	server.Mux.HandleFunc("/volumes/detail", func(w http.ResponseWriter, r *http.Request) {
		th.TestMethod(t, r, "GET")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		writeBody(w, fakeVolumeList())
	})

	client := newFakeBlockClient(server)
	out := captureStdout(t, func() {
		if err := ListBlockStorage(context.Background(), "json", client); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	if !strings.Contains(out, "vol-aaa") {
		t.Errorf("expected first volume ID in output, got: %s", out)
	}
	if !strings.Contains(out, "vol-bbb") {
		t.Errorf("expected second volume ID in output, got: %s", out)
	}

	var list []map[string]any
	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &list); err != nil {
		t.Errorf("output is not valid JSON array: %v\noutput: %s", err, out)
	}
	if len(list) != 2 {
		t.Errorf("expected 2 volumes, got %d", len(list))
	}
}

func TestListBlockStorage_Table(t *testing.T) {
	server := th.SetupHTTP()
	defer server.Teardown()

	server.Mux.HandleFunc("/volumes/detail", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		writeBody(w, fakeVolumeList())
	})

	client := newFakeBlockClient(server)
	out := captureStdout(t, func() {
		if err := ListBlockStorage(context.Background(), "table", client); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})
	if !strings.Contains(out, "vol-aaa") {
		t.Errorf("expected volume in table output, got: %s", out)
	}
}

func TestListBlockStorage_Empty(t *testing.T) {
	server := th.SetupHTTP()
	defer server.Teardown()

	server.Mux.HandleFunc("/volumes/detail", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		writeBody(w, `{"volumes": []}`)
	})

	client := newFakeBlockClient(server)
	out := captureStdout(t, func() {
		if err := ListBlockStorage(context.Background(), "json", client); err != nil {
			t.Errorf("unexpected error for empty list: %v", err)
		}
	})
	if !strings.Contains(out, "No volumes found") {
		t.Errorf("expected 'No volumes found' message, got: %s", out)
	}
}

func TestListBlockStorage_UnsupportedFormat(t *testing.T) {
	server := th.SetupHTTP()
	defer server.Teardown()

	server.Mux.HandleFunc("/volumes/detail", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		writeBody(w, fakeVolumeList())
	})

	client := newFakeBlockClient(server)
	err := ListBlockStorage(context.Background(), "csv", client)
	if err == nil || !strings.Contains(err.Error(), "unsupported output format") {
		t.Errorf("expected unsupported format error, got: %v", err)
	}
}

func TestListBlockStorage_APIError(t *testing.T) {
	server := th.SetupHTTP()
	defer server.Teardown()

	server.Mux.HandleFunc("/volumes/detail", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		writeBody(w, `{"error": {"message": "internal server error"}}`)
	})

	client := newFakeBlockClient(server)
	err := ListBlockStorage(context.Background(), "json", client)
	if err == nil {
		t.Error("expected error for 500 response, got nil")
	}
}

func TestGetBlockStorage_NilVolume_PrintsMessage(t *testing.T) {
	server := th.SetupHTTP()
	defer server.Teardown()

	volID := "55555555-5555-5555-5555-555555555555"
	server.Mux.HandleFunc("/volumes/"+volID, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		// Return a response that extracts to nil
		writeBody(w, `{"volume": null}`)
	})

	client := newFakeBlockClient(server)
	out := captureStdout(t, func() {
		if err := GetBlockStorage(context.Background(), volID, "json", client); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})
	if !strings.Contains(out, "No blockstorage volume found") {
		t.Errorf("expected 'No blockstorage volume found' message, got: %s", out)
	}
}
