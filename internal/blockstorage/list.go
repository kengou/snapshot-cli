package blockstorage

import (
	"context"
	"fmt"

	"github.com/gophercloud/gophercloud/v2"
	"github.com/gophercloud/gophercloud/v2/openstack/blockstorage/v3/volumes"

	"snapshot-cli/internal/util"
)

var volumeHeader = []string{
	"id",
	"status",
	"size",
	"availability_zone",
	"attachments",
	"name",
	"description",
	"volume_type",
	"snapshot_id",
	"source_volid",
	"backup_id",
	"metadata",
	"user_id",
	"bootable",
	"encrypted",
	"replication_status",
	"consistencygroup_id",
	"multiattach",
	"volume_image_metadata",
	"os-vol-host-attr:host",
	"os-vol-tenant-attr:tenant_id",
}

// ListBlockStorage lists all Cinder volumes in the current project and writes
// them to stdout in the requested output format. Caller supplies the client.
func ListBlockStorage(ctx context.Context, output string, blockClient *gophercloud.ServiceClient) error {
	vol, err := volumes.List(blockClient, volumes.ListOpts{}).AllPages(ctx)
	if err != nil {
		return err
	}

	volumeList, err := volumes.ExtractVolumes(vol)
	if err != nil {
		return err
	}
	if len(volumeList) == 0 {
		fmt.Println("No volumes found")
		return nil
	}

	return util.Render(output, volumeList, volumeHeader)
}
