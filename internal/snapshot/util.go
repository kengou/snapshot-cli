package snapshot

import (
	"time"
)

// DefaultOlderThan is the fallback retention window used when no --older-than flag is provided.
const DefaultOlderThan = 168 * time.Hour

// SnapShotOpts carries options common to all snapshot sub-commands.
// Exactly one of Volume/VolumeID or Share/ShareID should be set per invocation.
// The caller constructs a gophercloud ServiceClient and passes it separately;
// this struct stays free of auth concerns.
type SnapShotOpts struct {
	SnapshotID  string
	Share       bool
	ShareID     string
	Volume      bool
	VolumeID    string
	Force       bool
	Name        string
	Description string
	OlderThan   string
	Cleanup     bool
	// DryRun — when true, CleanupSnapshot returns the IDs that would be
	// deleted without issuing DELETE requests. Useful for previewing.
	DryRun bool
}

var snapshotNfsHeader = []string{
	"ID", "Name", "Share ID", "Share Proto", "Share Size", "Size", "Status", "Project ID", "Links",
}

var snapshotBlockHeader = []string{
	"ID", "Name", "Description", "Volume ID", "Status", "Size", "Metadata", "Progress", "Project ID", "Group Snapshot ID", "User ID", "Consumes Quota",
}
