//go:build integration

package integration

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/gophercloud/gophercloud/v2"

	"snapshot-cli/internal/auth"
	"snapshot-cli/internal/config"
)

// blockClient and nfsClient are initialised once in TestMain and reused across
// all helper functions. Tests call CLI functions (which do their own auth from
// env vars) for the "act" step, and use these clients only for setup/teardown.
var (
	ctx         = context.Background()
	blockClient *gophercloud.ServiceClient
	nfsClient   *gophercloud.ServiceClient
)

func TestMain(m *testing.M) {
	if os.Getenv("OS_AUTH_URL") == "" {
		fmt.Println("integration tests skipped: OS_AUTH_URL not set")
		os.Exit(0)
	}

	cfg, err := config.ReadAuthConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "read auth config: %v\n", err)
		os.Exit(1)
	}

	blockClient, err = auth.NewBlockStorageClient(ctx, cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "block storage client: %v\n", err)
		os.Exit(1)
	}

	nfsClient, err = auth.NewSharedFileSystemClient(ctx, cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "shared filesystem client: %v\n", err)
		os.Exit(1)
	}

	os.Exit(m.Run())
}
