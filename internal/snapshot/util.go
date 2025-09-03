package snapshot

import "github.com/gophercloud/gophercloud/v2"

type SnapShotOpts struct {
	client      *gophercloud.ServiceClient
	SnapshotID  string
	Share       bool
	ShareID     string
	Volume      bool
	VolumeID    string
	Force       bool
	Name        string
	Description string
}

var snapshotNfsHeader = []string{
	"ID", "Name", "Share ID", "Share Proto", "Share Size", "Size", "Status", "Project ID", "Links",
}

var snapshotBlockHeader = []string{
	"ID", "Name", "Description", "Volume ID", "Status", "Size", "Metadata", "Progress", "Project ID", "Group Snapshot ID", "User ID", "Consumes Quota",
}
