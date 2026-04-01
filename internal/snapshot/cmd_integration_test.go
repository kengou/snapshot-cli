package snapshot

import (
	"context"
	"os"
	"strings"
	"testing"
)

// These tests verify the Cmd functions handle auth config errors correctly
// when snapOpts.client is nil (i.e., they need to read auth config).
// In the test environment, OS_AUTH_URL and other env vars are not set,
// so ReadAuthConfig() will return an error.

func TestCreateSnapshotCmd_NoClient_ReadConfigFails(t *testing.T) {
	opts := &SnapShotOpts{VolumeID: "12345678-1234-1234-1234-123456789012", Name: "test"}
	// client is nil, so CreateSnapshotCmd will try to read auth config
	// which will fail because OS_* env vars are not set
	err := CreateSnapshotCmd(context.Background(), opts, "json")
	if err == nil {
		t.Error("expected error when OS_AUTH_URL not set, got nil")
	}
	if !strings.Contains(err.Error(), "missing") && !strings.Contains(err.Error(), "OS_AUTH_URL") {
		t.Errorf("expected config error mentioning missing auth vars, got: %v", err)
	}
}

func TestGetSnapshotCmd_NoClient_ReadConfigFails(t *testing.T) {
	opts := &SnapShotOpts{SnapshotID: "12345678-1234-1234-1234-123456789012", Volume: true}
	// client is nil, so GetSnapshotCmd will try to read auth config
	err := GetSnapshotCmd(context.Background(), opts, "json")
	if err == nil {
		t.Error("expected error when OS_AUTH_URL not set, got nil")
	}
}

func TestListSnapshotsCmd_NoClient_ReadConfigFails(t *testing.T) {
	opts := &SnapShotOpts{Volume: true}
	// client is nil, so ListSnapshotsCmd will try to read auth config
	err := ListSnapshotsCmd(context.Background(), opts, "json")
	if err == nil {
		t.Error("expected error when OS_AUTH_URL not set, got nil")
	}
}

func TestDeleteSnapshotCmd_NoClient_ReadConfigFails(t *testing.T) {
	opts := &SnapShotOpts{SnapshotID: "12345678-1234-1234-1234-123456789012", Volume: true}
	// client is nil, so DeleteSnapshotCmd will try to read auth config
	err := DeleteSnapshotCmd(context.Background(), opts, "json")
	if err == nil {
		t.Error("expected error when OS_AUTH_URL not set, got nil")
	}
}

func TestCreateSnapshotCmd_Share_NoClient_ReadConfigFails(t *testing.T) {
	opts := &SnapShotOpts{ShareID: "12345678-1234-1234-1234-123456789012", Name: "test"}
	err := CreateSnapshotCmd(context.Background(), opts, "json")
	if err == nil {
		t.Error("expected error when OS_AUTH_URL not set, got nil")
	}
}

// Tests with mock auth config to cover client initialization paths
func TestCreateSnapshotCmd_WithAuthEnv_ClientCreationFails(t *testing.T) {
	// Set auth env vars to trigger client creation (which will fail due to invalid URL)
	oldAuthURL := os.Getenv("OS_AUTH_URL")
	oldUsername := os.Getenv("OS_USERNAME")
	oldPassword := os.Getenv("OS_PASSWORD")
	oldUserDomain := os.Getenv("OS_USER_DOMAIN_NAME")
	oldProjectName := os.Getenv("OS_PROJECT_NAME")
	oldProjectDomain := os.Getenv("OS_PROJECT_DOMAIN_NAME")
	oldRegion := os.Getenv("OS_REGION_NAME")

	defer func() {
		os.Setenv("OS_AUTH_URL", oldAuthURL)
		os.Setenv("OS_USERNAME", oldUsername)
		os.Setenv("OS_PASSWORD", oldPassword)
		os.Setenv("OS_USER_DOMAIN_NAME", oldUserDomain)
		os.Setenv("OS_PROJECT_NAME", oldProjectName)
		os.Setenv("OS_PROJECT_DOMAIN_NAME", oldProjectDomain)
		os.Setenv("OS_REGION_NAME", oldRegion)
	}()

	// Set all required env vars with invalid URL to trigger auth failure
	os.Setenv("OS_AUTH_URL", "http://invalid-openstack:5000/v3")
	os.Setenv("OS_USERNAME", "testuser")
	os.Setenv("OS_PASSWORD", "testpass")
	os.Setenv("OS_USER_DOMAIN_NAME", "Default")
	os.Setenv("OS_PROJECT_NAME", "testproject")
	os.Setenv("OS_PROJECT_DOMAIN_NAME", "Default")
	os.Setenv("OS_REGION_NAME", "RegionOne")

	opts := &SnapShotOpts{VolumeID: "12345678-1234-1234-1234-123456789012", Name: "test"}
	err := CreateSnapshotCmd(context.Background(), opts, "json")
	if err == nil {
		t.Error("expected error when connecting to invalid OpenStack URL, got nil")
	}
}
