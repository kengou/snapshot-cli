package sharedfilesystem

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

func newFakeNFSClient(server th.FakeServer) *gophercloud.ServiceClient {
	c := thclient.ServiceClient(server)
	c.ResourceBase = server.Endpoint()
	return c
}

func fakeShare(id string) string {
	// Manila uses JSONRFC3339MilliNoZ format (no trailing Z).
	return `{
		"share": {
			"id": "` + id + `",
			"name": "test-share",
			"description": "a test NFS share",
			"status": "available",
			"size": 5,
			"share_proto": "NFS",
			"availability_zone": "nova",
			"is_public": false,
			"share_type": "default",
			"created_at": "2024-01-01T00:00:00.000000",
			"updated_at": "2024-01-01T00:00:00.000000"
		}
	}`
}

func fakeShareList() string {
	return `{
		"shares": [
			{
				"id": "share-aaa",
				"name": "nfs-one",
				"status": "available",
				"size": 5,
				"share_proto": "NFS",
				"availability_zone": "nova",
				"is_public": false
			},
			{
				"id": "share-bbb",
				"name": "nfs-two",
				"status": "in-use",
				"size": 10,
				"share_proto": "CIFS",
				"availability_zone": "nova",
				"is_public": true
			}
		]
	}`
}

// --- getSharedFileSystem tests ---

func TestGetSharedFileSystem_JSON(t *testing.T) {
	server := th.SetupHTTP()
	defer server.Teardown()

	shareID := "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"
	server.Mux.HandleFunc("/shares/"+shareID, func(w http.ResponseWriter, r *http.Request) {
		th.TestMethod(t, r, "GET")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		writeBody(w, fakeShare(shareID))
	})

	client := newFakeNFSClient(server)
	out := captureStdout(t, func() {
		if err := GetSharedFileSystem(context.Background(), shareID, "json", client); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	if !strings.Contains(out, shareID) {
		t.Errorf("expected share ID in JSON output, got: %s", out)
	}
	var m map[string]any
	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &m); err != nil {
		t.Errorf("output is not valid JSON: %v\noutput: %s", err, out)
	}
}

func TestGetSharedFileSystem_Table(t *testing.T) {
	server := th.SetupHTTP()
	defer server.Teardown()

	shareID := "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"
	server.Mux.HandleFunc("/shares/"+shareID, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		writeBody(w, fakeShare(shareID))
	})

	client := newFakeNFSClient(server)
	out := captureStdout(t, func() {
		if err := GetSharedFileSystem(context.Background(), shareID, "table", client); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})
	if !strings.Contains(out, shareID) {
		t.Errorf("expected share ID in table output, got: %s", out)
	}
}

func TestGetSharedFileSystem_UnsupportedFormat(t *testing.T) {
	server := th.SetupHTTP()
	defer server.Teardown()

	shareID := "cccccccc-cccc-cccc-cccc-cccccccccccc"
	server.Mux.HandleFunc("/shares/"+shareID, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		writeBody(w, fakeShare(shareID))
	})

	client := newFakeNFSClient(server)
	err := GetSharedFileSystem(context.Background(), shareID, "yaml", client)
	if err == nil || !strings.Contains(err.Error(), "unsupported output format") {
		t.Errorf("expected unsupported format error, got: %v", err)
	}
}

func TestGetSharedFileSystem_404_ReturnsError(t *testing.T) {
	server := th.SetupHTTP()
	defer server.Teardown()

	shareID := "dddddddd-dddd-dddd-dddd-dddddddddddd"
	server.Mux.HandleFunc("/shares/"+shareID, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		writeBody(w, `{"itemNotFound": {"message": "Share not found", "code": 404}}`)
	})

	client := newFakeNFSClient(server)
	err := GetSharedFileSystem(context.Background(), shareID, "json", client)
	if err == nil {
		t.Error("expected error for 404 response, got nil")
	}
}

// UUID validation runs before any client call, so nil client is fine here.

func TestGetSharedFileSystem_InvalidUUID(t *testing.T) {
	err := GetSharedFileSystem(context.Background(), "not-a-uuid", "json", nil)
	if err == nil || !strings.Contains(err.Error(), "invalid ID") {
		t.Errorf("expected UUID validation error, got: %v", err)
	}
}

func TestGetSharedFileSystem_EmptyUUID(t *testing.T) {
	err := GetSharedFileSystem(context.Background(), "", "json", nil)
	if err == nil || !strings.Contains(err.Error(), "invalid ID") {
		t.Errorf("expected UUID validation error for empty ID, got: %v", err)
	}
}

// --- listSharedFileSystems tests ---

func TestListSharedFileSystems_JSON(t *testing.T) {
	server := th.SetupHTTP()
	defer server.Teardown()

	server.Mux.HandleFunc("/shares/detail", func(w http.ResponseWriter, r *http.Request) {
		th.TestMethod(t, r, "GET")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		writeBody(w, fakeShareList())
	})

	client := newFakeNFSClient(server)
	out := captureStdout(t, func() {
		if err := ListSharedFileSystems(context.Background(), "json", client); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	if !strings.Contains(out, "share-aaa") {
		t.Errorf("expected first share ID in output, got: %s", out)
	}
	if !strings.Contains(out, "share-bbb") {
		t.Errorf("expected second share ID in output, got: %s", out)
	}

	var list []map[string]any
	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &list); err != nil {
		t.Errorf("output is not valid JSON array: %v\noutput: %s", err, out)
	}
	if len(list) != 2 {
		t.Errorf("expected 2 shares, got %d", len(list))
	}
}

func TestListSharedFileSystems_Table(t *testing.T) {
	server := th.SetupHTTP()
	defer server.Teardown()

	server.Mux.HandleFunc("/shares/detail", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		writeBody(w, fakeShareList())
	})

	client := newFakeNFSClient(server)
	out := captureStdout(t, func() {
		if err := ListSharedFileSystems(context.Background(), "table", client); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})
	if !strings.Contains(out, "share-aaa") {
		t.Errorf("expected share in table output, got: %s", out)
	}
}

func TestListSharedFileSystems_Empty(t *testing.T) {
	server := th.SetupHTTP()
	defer server.Teardown()

	server.Mux.HandleFunc("/shares/detail", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		writeBody(w, `{"shares": []}`)
	})

	client := newFakeNFSClient(server)
	out := captureStdout(t, func() {
		if err := ListSharedFileSystems(context.Background(), "json", client); err != nil {
			t.Errorf("unexpected error for empty list: %v", err)
		}
	})
	if !strings.Contains(out, "No nfs found") {
		t.Errorf("expected 'No nfs found' message, got: %s", out)
	}
}

func TestListSharedFileSystems_UnsupportedFormat(t *testing.T) {
	server := th.SetupHTTP()
	defer server.Teardown()

	server.Mux.HandleFunc("/shares/detail", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		writeBody(w, fakeShareList())
	})

	client := newFakeNFSClient(server)
	err := ListSharedFileSystems(context.Background(), "xml", client)
	if err == nil || !strings.Contains(err.Error(), "unsupported output format") {
		t.Errorf("expected unsupported format error, got: %v", err)
	}
}

func TestListSharedFileSystems_APIError(t *testing.T) {
	server := th.SetupHTTP()
	defer server.Teardown()

	server.Mux.HandleFunc("/shares/detail", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		writeBody(w, `{"error": {"message": "internal server error"}}`)
	})

	client := newFakeNFSClient(server)
	err := ListSharedFileSystems(context.Background(), "json", client)
	if err == nil {
		t.Error("expected error for 500 response, got nil")
	}
}

func TestGetSharedFileSystem_NilShare_PrintsMessage(t *testing.T) {
	server := th.SetupHTTP()
	defer server.Teardown()

	shareID := "eeeeeeee-eeee-eeee-eeee-eeeeeeeeeeee"
	server.Mux.HandleFunc("/shares/"+shareID, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		writeBody(w, `{"share": null}`)
	})

	client := newFakeNFSClient(server)
	out := captureStdout(t, func() {
		if err := GetSharedFileSystem(context.Background(), shareID, "json", client); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})
	if !strings.Contains(out, "NFS share not found") {
		t.Errorf("expected 'NFS share not found' message, got: %s", out)
	}
}
