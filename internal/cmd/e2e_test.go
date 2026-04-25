package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

// ── E2E resource IDs ─────────────────────────────────────────────────────────

const (
	e2eVolumeID    = "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"
	e2eShareID     = "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"
	e2eBlockSnapID = "cccccccc-cccc-cccc-cccc-cccccccccccc"
	e2eNFSSnapID   = "dddddddd-dddd-dddd-dddd-dddddddddddd"
	e2eNotFoundID  = "eeeeeeee-eeee-eeee-eeee-eeeeeeeeeeee"
)

// ── Fake JSON payloads ───────────────────────────────────────────────────────

func e2eVolumeJSON(id string) string {
	return `{"id":"` + id + `","status":"available","size":10,"name":"e2e-vol",` +
		`"availability_zone":"nova","bootable":"false","encrypted":false,` +
		`"replication_status":"enabled","multiattach":false}`
}

// e2eBlockSnapJSON uses the JSONRFC3339MilliNoZ format ("no trailing Z") that
// gophercloud uses for Cinder snapshot CreatedAt fields. The 2022 date ensures
// the snapshot is always "old enough" for cleanup tests using --older-than 1h.
func e2eBlockSnapJSON(id, volID string) string {
	return `{"id":"` + id + `","name":"e2e-block-snap","status":"available",` +
		`"volume_id":"` + volID + `","size":10,` +
		`"created_at":"2022-01-01T00:00:00.000000"}`
}

// e2eShareJSON uses the same JSONRFC3339MilliNoZ format required by Manila.
func e2eShareJSON(id string) string {
	return `{"id":"` + id + `","name":"e2e-share","status":"available","size":5,` +
		`"share_proto":"NFS","availability_zone":"nova","is_public":false,` +
		`"created_at":"2022-01-01T00:00:00.000000","updated_at":"2022-01-01T00:00:00.000000"}`
}

func e2eNFSSnapJSON(id, shareID string) string {
	return `{"id":"` + id + `","name":"e2e-nfs-snap","status":"available",` +
		`"share_id":"` + shareID + `","size":5,"share_proto":"NFS","share_size":5,` +
		`"created_at":"2022-01-01T00:00:00.000000"}`
}

// ── Fake OpenStack server ────────────────────────────────────────────────────

// setupE2EServer starts an httptest.Server that handles:
//   - POST /v3/auth/tokens  → Keystone auth with service catalog
//   - GET/LIST/CREATE/DELETE on /block/* and /nfs/* for all resource types
//
// The catalog URLs are populated dynamically with the server's own address so
// gophercloud authenticates and calls APIs against the same fake server.
func setupE2EServer(t *testing.T) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()

	// srv is declared here and assigned after all handlers are registered.
	// Handlers close over the variable; by the time any request arrives the
	// server is running and srv.URL is populated.
	var srv *httptest.Server

	// ── Keystone ──────────────────────────────────────────────────────────────
	mux.HandleFunc("/v3/auth/tokens", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		catalog := fmt.Sprintf(`{
			"token":{
				"catalog":[
					{"type":"block-storage","id":"bs1","endpoints":[
						{"interface":"public","url":"%s/block/","region_id":"RegionOne","id":"bse1"}
					]},
					{"type":"shared-file-system","id":"sfs1","endpoints":[
						{"interface":"public","url":"%s/nfs/","region_id":"RegionOne","id":"sfse1"}
					]}
				],
				"expires_at":"2099-01-01T00:00:00.000000Z",
				"issued_at":"2024-01-01T00:00:00.000000Z",
				"methods":["password"],
				"user":{"id":"u1","name":"testuser","domain":{"id":"default","name":"Default"}},
				"project":{"id":"proj1","name":"testproject","domain":{"id":"default","name":"Default"}}
			}
		}`, srv.URL, srv.URL)
		w.Header().Set("X-Subject-Token", "fake-e2e-token")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		if _, err := w.Write([]byte(catalog)); err != nil {
			panic("keystone handler write: " + err.Error())
		}
	})

	// ── Version discovery ────────────────────────────────────────────────────
	// gophercloud GETs the service base URL before making API calls to
	// negotiate API version and microversion support. Handlers for more
	// specific paths (/block/volumes/, etc.) always take priority in ServeMux.
	mux.HandleFunc("/block/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/block/" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		body := fmt.Sprintf(`{"version":{"id":"v3.0","status":"CURRENT","version":"3.67","min_version":"3.0","links":[{"href":"%s/block/","rel":"self"}]}}`, srv.URL)
		if _, err := w.Write([]byte(body)); err != nil {
			panic("block version discovery write: " + err.Error())
		}
	})
	mux.HandleFunc("/nfs/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/nfs/" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		body := fmt.Sprintf(`{"version":{"id":"v2.0","status":"CURRENT","version":"2.70","min_version":"2.0","links":[{"href":"%s/nfs/","rel":"self"}]}}`, srv.URL)
		if _, err := w.Write([]byte(body)); err != nil {
			panic("nfs version discovery write: " + err.Error())
		}
	})

	// ── Block Storage: volumes ────────────────────────────────────────────────
	// GET /block/volumes/detail → list
	// Note: exact pattern takes priority over the subtree "/block/volumes/" below.
	mux.HandleFunc("/block/volumes/detail", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		body := `{"volumes":[` + e2eVolumeJSON(e2eVolumeID) + `]}`
		if _, err := w.Write([]byte(body)); err != nil {
			panic("volumes/detail handler write: " + err.Error())
		}
	})
	// GET /block/volumes/{id}
	mux.HandleFunc("/block/volumes/", func(w http.ResponseWriter, r *http.Request) {
		id := strings.TrimPrefix(r.URL.Path, "/block/volumes/")
		if id == e2eNotFoundID {
			w.WriteHeader(http.StatusNotFound)
			if _, err := w.Write([]byte(`{"itemNotFound":{"message":"Volume not found","code":404}}`)); err != nil {
				panic("volumes/{id} 404 write: " + err.Error())
			}
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte(`{"volume":` + e2eVolumeJSON(id) + `}`)); err != nil { //nolint:gosec
			panic("volumes/{id} handler write: " + err.Error())
		}
	})

	// ── Block Storage: snapshots ──────────────────────────────────────────────
	// GET /block/snapshots  → list  (blockSnapshot.List uses /snapshots, not /snapshots/detail)
	// POST /block/snapshots → create
	mux.HandleFunc("/block/snapshots", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.Method {
		case http.MethodGet:
			w.WriteHeader(http.StatusOK)
			body := `{"snapshots":[` + e2eBlockSnapJSON(e2eBlockSnapID, e2eVolumeID) + `]}`
			if _, err := w.Write([]byte(body)); err != nil {
				panic("block snapshots list write: " + err.Error())
			}
		case http.MethodPost:
			w.WriteHeader(http.StatusAccepted)
			body := `{"snapshot":` + e2eBlockSnapJSON(e2eBlockSnapID, e2eVolumeID) + `}`
			if _, err := w.Write([]byte(body)); err != nil {
				panic("block snapshots create write: " + err.Error())
			}
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})
	// GET /block/snapshots/{id}  → get
	// DELETE /block/snapshots/{id} → delete
	mux.HandleFunc("/block/snapshots/", func(w http.ResponseWriter, r *http.Request) {
		id := strings.TrimPrefix(r.URL.Path, "/block/snapshots/")
		if id == e2eNotFoundID {
			w.WriteHeader(http.StatusNotFound)
			if _, err := w.Write([]byte(`{"itemNotFound":{"message":"Snapshot not found","code":404}}`)); err != nil {
				panic("block snapshots/{id} 404 write: " + err.Error())
			}
			return
		}
		switch r.Method {
		case http.MethodGet:
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			body := `{"snapshot":` + e2eBlockSnapJSON(id, e2eVolumeID) + `}`
			if _, err := w.Write([]byte(body)); err != nil { //nolint:gosec
				panic("block snapshots/{id} get write: " + err.Error())
			}
		case http.MethodDelete:
			w.WriteHeader(http.StatusAccepted)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})

	// ── Shared Filesystems: shares ────────────────────────────────────────────
	// GET /nfs/shares/detail → list
	mux.HandleFunc("/nfs/shares/detail", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		body := `{"shares":[` + e2eShareJSON(e2eShareID) + `]}`
		if _, err := w.Write([]byte(body)); err != nil {
			panic("shares/detail handler write: " + err.Error())
		}
	})
	// GET /nfs/shares/{id}
	mux.HandleFunc("/nfs/shares/", func(w http.ResponseWriter, r *http.Request) {
		id := strings.TrimPrefix(r.URL.Path, "/nfs/shares/")
		if id == e2eNotFoundID {
			w.WriteHeader(http.StatusNotFound)
			if _, err := w.Write([]byte(`{"itemNotFound":{"message":"Share not found","code":404}}`)); err != nil {
				panic("shares/{id} 404 write: " + err.Error())
			}
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte(`{"share":` + e2eShareJSON(id) + `}`)); err != nil { //nolint:gosec
			panic("shares/{id} handler write: " + err.Error())
		}
	})

	// ── Shared Filesystems: snapshots ─────────────────────────────────────────
	// GET /nfs/snapshots/detail → list (nfsSnapshot.ListDetail uses /snapshots/detail)
	mux.HandleFunc("/nfs/snapshots/detail", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		body := `{"snapshots":[` + e2eNFSSnapJSON(e2eNFSSnapID, e2eShareID) + `]}`
		if _, err := w.Write([]byte(body)); err != nil {
			panic("nfs snapshots/detail write: " + err.Error())
		}
	})
	// POST /nfs/snapshots → create
	mux.HandleFunc("/nfs/snapshots", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		body := `{"snapshot":` + e2eNFSSnapJSON(e2eNFSSnapID, e2eShareID) + `}`
		if _, err := w.Write([]byte(body)); err != nil {
			panic("nfs snapshots create write: " + err.Error())
		}
	})
	// GET /nfs/snapshots/{id}  → get
	// DELETE /nfs/snapshots/{id} → delete
	mux.HandleFunc("/nfs/snapshots/", func(w http.ResponseWriter, r *http.Request) {
		id := strings.TrimPrefix(r.URL.Path, "/nfs/snapshots/")
		if id == e2eNotFoundID {
			w.WriteHeader(http.StatusNotFound)
			if _, err := w.Write([]byte(`{"itemNotFound":{"message":"NFS snapshot not found","code":404}}`)); err != nil {
				panic("nfs snapshots/{id} 404 write: " + err.Error())
			}
			return
		}
		switch r.Method {
		case http.MethodGet:
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			body := `{"snapshot":` + e2eNFSSnapJSON(id, e2eShareID) + `}`
			if _, err := w.Write([]byte(body)); err != nil { //nolint:gosec
				panic("nfs snapshots/{id} get write: " + err.Error())
			}
		case http.MethodDelete:
			w.WriteHeader(http.StatusAccepted)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})

	// Start the server after all handlers are registered so closures that
	// capture srv (e.g. the Keystone handler) can read srv.URL at call time.
	srv = httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

// ── CLI runner ───────────────────────────────────────────────────────────────

// runE2ECmd sets OpenStack env vars to point at srv, builds the root cobra
// command, executes it with args, and returns captured stdout + any error.
func runE2ECmd(t *testing.T, srv *httptest.Server, args ...string) (string, error) {
	t.Helper()

	t.Setenv("OS_AUTH_URL", srv.URL+"/v3")
	t.Setenv("OS_USERNAME", "testuser")
	t.Setenv("OS_PASSWORD", "testpass")
	t.Setenv("OS_USER_DOMAIN_NAME", "Default")
	t.Setenv("OS_PROJECT_NAME", "testproject")
	t.Setenv("OS_PROJECT_DOMAIN_NAME", "Default")
	t.Setenv("OS_REGION_NAME", "RegionOne")

	v := &VersionInfo{Version: "test", GitCommitHash: "abc", BuildDate: "2024-01-01"}
	root := newRootCmd(v)
	root.SetArgs(args)

	pr, pw, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	oldStdout := os.Stdout
	os.Stdout = pw

	execErr := root.ExecuteContext(context.Background())

	pw.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	if _, err = io.Copy(&buf, pr); err != nil {
		t.Fatalf("io.Copy: %v", err)
	}
	return buf.String(), execErr
}

// ── volumes get ──────────────────────────────────────────────────────────────

func TestE2E_VolumesGet_JSON(t *testing.T) {
	srv := setupE2EServer(t)
	out, err := runE2ECmd(t, srv, "volumes", "get", "--volume-id", e2eVolumeID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, e2eVolumeID) {
		t.Errorf("expected volume ID in output, got: %s", out)
	}
	var m map[string]any
	if jsonErr := json.Unmarshal([]byte(strings.TrimSpace(out)), &m); jsonErr != nil {
		t.Errorf("output is not valid JSON: %v\noutput: %s", jsonErr, out)
	}
}

func TestE2E_VolumesGet_Table(t *testing.T) {
	srv := setupE2EServer(t)
	out, err := runE2ECmd(t, srv, "volumes", "get", "--volume-id", e2eVolumeID, "--output", "table")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, e2eVolumeID) {
		t.Errorf("expected volume ID in table output, got: %s", out)
	}
}

func TestE2E_VolumesGet_NotFound(t *testing.T) {
	srv := setupE2EServer(t)
	_, err := runE2ECmd(t, srv, "volumes", "get", "--volume-id", e2eNotFoundID)
	if err == nil {
		t.Error("expected error for not-found volume, got nil")
	}
}

// ── volumes list ─────────────────────────────────────────────────────────────

func TestE2E_VolumesList_JSON(t *testing.T) {
	srv := setupE2EServer(t)
	out, err := runE2ECmd(t, srv, "volumes", "list")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, e2eVolumeID) {
		t.Errorf("expected volume ID in output, got: %s", out)
	}
	var list []map[string]any
	if jsonErr := json.Unmarshal([]byte(strings.TrimSpace(out)), &list); jsonErr != nil {
		t.Errorf("output is not valid JSON array: %v\noutput: %s", jsonErr, out)
	}
}

func TestE2E_VolumesList_Table(t *testing.T) {
	srv := setupE2EServer(t)
	out, err := runE2ECmd(t, srv, "volumes", "list", "--output", "table")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, e2eVolumeID) {
		t.Errorf("expected volume ID in table output, got: %s", out)
	}
}

// ── volumes snapshot (create block snapshot via volumes sub-command) ──────────

func TestE2E_VolumesSnapshot_Create_JSON(t *testing.T) {
	srv := setupE2EServer(t)
	out, err := runE2ECmd(t, srv, "volumes", "snapshot", "--volume-id", e2eVolumeID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, e2eBlockSnapID) {
		t.Errorf("expected block snapshot ID in output, got: %s", out)
	}
}

func TestE2E_VolumesSnapshot_Create_Table(t *testing.T) {
	srv := setupE2EServer(t)
	out, err := runE2ECmd(t, srv, "volumes", "snapshot",
		"--volume-id", e2eVolumeID,
		"--output", "table")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, e2eBlockSnapID) {
		t.Errorf("expected block snapshot ID in table output, got: %s", out)
	}
}

// ── nfs get ──────────────────────────────────────────────────────────────────

func TestE2E_NFSGet_JSON(t *testing.T) {
	srv := setupE2EServer(t)
	out, err := runE2ECmd(t, srv, "nfs", "get", "--share-id", e2eShareID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, e2eShareID) {
		t.Errorf("expected share ID in output, got: %s", out)
	}
	var m map[string]any
	if jsonErr := json.Unmarshal([]byte(strings.TrimSpace(out)), &m); jsonErr != nil {
		t.Errorf("output is not valid JSON: %v\noutput: %s", jsonErr, out)
	}
}

func TestE2E_NFSGet_Table(t *testing.T) {
	srv := setupE2EServer(t)
	out, err := runE2ECmd(t, srv, "nfs", "get", "--share-id", e2eShareID, "--output", "table")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, e2eShareID) {
		t.Errorf("expected share ID in table output, got: %s", out)
	}
}

func TestE2E_NFSGet_NotFound(t *testing.T) {
	srv := setupE2EServer(t)
	_, err := runE2ECmd(t, srv, "nfs", "get", "--share-id", e2eNotFoundID)
	if err == nil {
		t.Error("expected error for not-found share, got nil")
	}
}

// ── nfs list ─────────────────────────────────────────────────────────────────

func TestE2E_NFSList_JSON(t *testing.T) {
	srv := setupE2EServer(t)
	out, err := runE2ECmd(t, srv, "nfs", "list")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, e2eShareID) {
		t.Errorf("expected share ID in output, got: %s", out)
	}
	var list []map[string]any
	if jsonErr := json.Unmarshal([]byte(strings.TrimSpace(out)), &list); jsonErr != nil {
		t.Errorf("output is not valid JSON array: %v\noutput: %s", jsonErr, out)
	}
}

func TestE2E_NFSList_Table(t *testing.T) {
	srv := setupE2EServer(t)
	out, err := runE2ECmd(t, srv, "nfs", "list", "--output", "table")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, e2eShareID) {
		t.Errorf("expected share ID in table output, got: %s", out)
	}
}

// ── snapshot get ──────────────────────────────────────────────────────────────

func TestE2E_SnapshotGet_Volume_JSON(t *testing.T) {
	srv := setupE2EServer(t)
	out, err := runE2ECmd(t, srv, "snapshot", "get", "--snapshot-id", e2eBlockSnapID, "--volume")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, e2eBlockSnapID) {
		t.Errorf("expected snapshot ID in output, got: %s", out)
	}
	var m map[string]any
	if jsonErr := json.Unmarshal([]byte(strings.TrimSpace(out)), &m); jsonErr != nil {
		t.Errorf("output is not valid JSON: %v\noutput: %s", jsonErr, out)
	}
}

func TestE2E_SnapshotGet_Volume_Table(t *testing.T) {
	srv := setupE2EServer(t)
	out, err := runE2ECmd(t, srv, "snapshot", "get",
		"--snapshot-id", e2eBlockSnapID,
		"--volume",
		"--output", "table")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, e2eBlockSnapID) {
		t.Errorf("expected snapshot ID in table output, got: %s", out)
	}
}

func TestE2E_SnapshotGet_Share_JSON(t *testing.T) {
	srv := setupE2EServer(t)
	out, err := runE2ECmd(t, srv, "snapshot", "get", "--snapshot-id", e2eNFSSnapID, "--share")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, e2eNFSSnapID) {
		t.Errorf("expected NFS snapshot ID in output, got: %s", out)
	}
}

func TestE2E_SnapshotGet_Share_Table(t *testing.T) {
	srv := setupE2EServer(t)
	out, err := runE2ECmd(t, srv, "snapshot", "get",
		"--snapshot-id", e2eNFSSnapID,
		"--share",
		"--output", "table")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, e2eNFSSnapID) {
		t.Errorf("expected NFS snapshot ID in table output, got: %s", out)
	}
}

func TestE2E_SnapshotGet_NotFound(t *testing.T) {
	srv := setupE2EServer(t)
	_, err := runE2ECmd(t, srv, "snapshot", "get", "--snapshot-id", e2eNotFoundID, "--volume")
	if err == nil {
		t.Error("expected error for not-found snapshot, got nil")
	}
}

// ── snapshot list ─────────────────────────────────────────────────────────────

func TestE2E_SnapshotList_Volume_JSON(t *testing.T) {
	srv := setupE2EServer(t)
	out, err := runE2ECmd(t, srv, "snapshot", "list", "--volume")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, e2eBlockSnapID) {
		t.Errorf("expected block snapshot ID in output, got: %s", out)
	}
}

func TestE2E_SnapshotList_Volume_Table(t *testing.T) {
	srv := setupE2EServer(t)
	out, err := runE2ECmd(t, srv, "snapshot", "list", "--volume", "--output", "table")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, e2eBlockSnapID) {
		t.Errorf("expected block snapshot ID in table output, got: %s", out)
	}
}

func TestE2E_SnapshotList_Share_JSON(t *testing.T) {
	srv := setupE2EServer(t)
	out, err := runE2ECmd(t, srv, "snapshot", "list", "--share")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, e2eNFSSnapID) {
		t.Errorf("expected NFS snapshot ID in output, got: %s", out)
	}
}

func TestE2E_SnapshotList_Share_Table(t *testing.T) {
	srv := setupE2EServer(t)
	out, err := runE2ECmd(t, srv, "snapshot", "list", "--share", "--output", "table")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, e2eNFSSnapID) {
		t.Errorf("expected NFS snapshot ID in table output, got: %s", out)
	}
}

// ── snapshot create ───────────────────────────────────────────────────────────

func TestE2E_SnapshotCreate_Volume_JSON(t *testing.T) {
	srv := setupE2EServer(t)
	out, err := runE2ECmd(t, srv, "snapshot", "create", "--volume-id", e2eVolumeID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, e2eBlockSnapID) {
		t.Errorf("expected block snapshot ID in output, got: %s", out)
	}
}

func TestE2E_SnapshotCreate_Volume_Table(t *testing.T) {
	srv := setupE2EServer(t)
	out, err := runE2ECmd(t, srv, "snapshot", "create",
		"--volume-id", e2eVolumeID,
		"--output", "table")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, e2eBlockSnapID) {
		t.Errorf("expected block snapshot ID in table output, got: %s", out)
	}
}

func TestE2E_SnapshotCreate_Volume_WithNameAndDescription(t *testing.T) {
	srv := setupE2EServer(t)
	out, err := runE2ECmd(t, srv, "snapshot", "create",
		"--volume-id", e2eVolumeID,
		"--name", "my-snapshot",
		"--description", "e2e test snapshot")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, e2eBlockSnapID) {
		t.Errorf("expected block snapshot ID in output, got: %s", out)
	}
}

// TestE2E_SnapshotCreate_Volume_WithCleanup tests the full create+cleanup flow:
// create a snapshot, then delete snapshots older than 1h (all fake snapshots
// use a 2022 timestamp so they always qualify).
func TestE2E_SnapshotCreate_Volume_WithCleanup(t *testing.T) {
	srv := setupE2EServer(t)
	out, err := runE2ECmd(t, srv, "snapshot", "create",
		"--volume-id", e2eVolumeID,
		"--cleanup",
		"--older-than", "1h")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Output contains cleanup result (deleted IDs) followed by created snapshot.
	if !strings.Contains(out, e2eBlockSnapID) {
		t.Errorf("expected block snapshot ID in output, got: %s", out)
	}
}

func TestE2E_SnapshotCreate_Share_JSON(t *testing.T) {
	srv := setupE2EServer(t)
	out, err := runE2ECmd(t, srv, "snapshot", "create", "--share-id", e2eShareID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, e2eNFSSnapID) {
		t.Errorf("expected NFS snapshot ID in output, got: %s", out)
	}
}

func TestE2E_SnapshotCreate_Share_WithCleanup(t *testing.T) {
	srv := setupE2EServer(t)
	out, err := runE2ECmd(t, srv, "snapshot", "create",
		"--share-id", e2eShareID,
		"--cleanup",
		"--older-than", "1h")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, e2eNFSSnapID) {
		t.Errorf("expected NFS snapshot ID in output, got: %s", out)
	}
}

// ── snapshot delete ───────────────────────────────────────────────────────────

func TestE2E_SnapshotDelete_Volume(t *testing.T) {
	srv := setupE2EServer(t)
	_, err := runE2ECmd(t, srv, "snapshot", "delete",
		"--snapshot-id", e2eBlockSnapID,
		"--volume")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestE2E_SnapshotDelete_Volume_Table(t *testing.T) {
	srv := setupE2EServer(t)
	_, err := runE2ECmd(t, srv, "snapshot", "delete",
		"--snapshot-id", e2eBlockSnapID,
		"--volume",
		"--output", "table")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestE2E_SnapshotDelete_Share(t *testing.T) {
	srv := setupE2EServer(t)
	_, err := runE2ECmd(t, srv, "snapshot", "delete",
		"--snapshot-id", e2eNFSSnapID,
		"--share")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestE2E_SnapshotDelete_Volume_NotFound(t *testing.T) {
	srv := setupE2EServer(t)
	_, err := runE2ECmd(t, srv, "snapshot", "delete",
		"--snapshot-id", e2eNotFoundID,
		"--volume")
	if err == nil {
		t.Error("expected error for not-found snapshot, got nil")
	}
}

// ── cleanup ───────────────────────────────────────────────────────────────────

// TestE2E_Cleanup_Volume verifies that cleanup lists block snapshots, deletes
// those older than --older-than, and outputs the deleted IDs.
func TestE2E_Cleanup_Volume(t *testing.T) {
	srv := setupE2EServer(t)
	out, err := runE2ECmd(t, srv, "cleanup", "--volume", "--older-than", "1h")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, e2eBlockSnapID) {
		t.Errorf("expected deleted snapshot ID in cleanup output, got: %s", out)
	}
}

func TestE2E_Cleanup_Volume_Table(t *testing.T) {
	srv := setupE2EServer(t)
	_, err := runE2ECmd(t, srv, "cleanup", "--volume", "--older-than", "1h", "--output", "table")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestE2E_Cleanup_Volume_WithVolumeID scopes cleanup to a specific volume.
func TestE2E_Cleanup_Volume_WithVolumeID(t *testing.T) {
	srv := setupE2EServer(t)
	out, err := runE2ECmd(t, srv, "cleanup",
		"--volume",
		"--volume-id", e2eVolumeID,
		"--older-than", "1h")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, e2eBlockSnapID) {
		t.Errorf("expected deleted snapshot ID in cleanup output, got: %s", out)
	}
}

func TestE2E_Cleanup_Share(t *testing.T) {
	srv := setupE2EServer(t)
	out, err := runE2ECmd(t, srv, "cleanup", "--share", "--older-than", "1h")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, e2eNFSSnapID) {
		t.Errorf("expected deleted NFS snapshot ID in cleanup output, got: %s", out)
	}
}

func TestE2E_Cleanup_Share_Table(t *testing.T) {
	srv := setupE2EServer(t)
	_, err := runE2ECmd(t, srv, "cleanup", "--share", "--older-than", "1h", "--output", "table")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestE2E_Cleanup_Share_WithShareID scopes cleanup to a specific share.
func TestE2E_Cleanup_Share_WithShareID(t *testing.T) {
	srv := setupE2EServer(t)
	out, err := runE2ECmd(t, srv, "cleanup",
		"--share",
		"--share-id", e2eShareID,
		"--older-than", "1h")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, e2eNFSSnapID) {
		t.Errorf("expected deleted NFS snapshot ID in cleanup output, got: %s", out)
	}
}

// ── error / validation cases ──────────────────────────────────────────────────

// TestE2E_MissingEnvVars confirms the CLI returns a descriptive error when the
// required OpenStack env vars are not set.
func TestE2E_MissingEnvVars(t *testing.T) {
	t.Setenv("OS_AUTH_URL", "")
	t.Setenv("OS_USERNAME", "")
	t.Setenv("OS_PASSWORD", "")
	t.Setenv("OS_USER_DOMAIN_NAME", "")
	t.Setenv("OS_PROJECT_NAME", "")
	t.Setenv("OS_PROJECT_DOMAIN_NAME", "")

	v := &VersionInfo{Version: "test", GitCommitHash: "abc", BuildDate: "2024-01-01"}
	root := newRootCmd(v)
	root.SetArgs([]string{"volumes", "list"})
	err := root.ExecuteContext(context.Background())
	if err == nil {
		t.Error("expected error when OS_ env vars are missing, got nil")
	}
	if !strings.Contains(err.Error(), "missing") {
		t.Errorf("expected 'missing' in error, got: %v", err)
	}
}
