package sharedfilesystem

import (
	"context"
	"fmt"

	"github.com/gophercloud/gophercloud/v2/openstack/sharedfilesystems/v2/shares"

	"snapshot-cli/internal/auth"
	"snapshot-cli/internal/config"
	"snapshot-cli/internal/util"
)

var nfsHeader = []string{
	"availability_zone",
	"description",
	"display_description",
	"display_name",
	"has_replicas",
	"host",
	"id",
	"is_public",
	"links",
	"metadata",
	"name",
	"project_id",
	"replication_type",
	"share_network_id",
	"share_proto",
	"share_server_id",
	"share_type",
	"share_type_name",
	"share_group_id",
	"size",
	"snapshot_id",
	"status",
	"task_state",
	"volume_type",
	"consistency_group_id",
	"snapshot_support",
	"source_cgsnapshot_member_id",
	"create_share_from_snapshot_support",
	"created_at",
	"updated_at",
}

func RunListSharedFileSystemSnapshots(ctx context.Context, output string) error {
	authConfig, err := config.ReadAuthConfig()
	if err != nil {
		return err
	}
	sharedClient, err := auth.NewSharedFileSystemClient(ctx, authConfig)
	if err != nil {
		return err
	}

	nfsPage, err := shares.ListDetail(sharedClient, shares.ListOpts{}).AllPages(ctx)
	if err != nil {
		return err
	}

	nfsList, err := shares.ExtractShares(nfsPage)
	if err != nil {
		return err
	}
	if len(nfsList) == 0 {
		fmt.Println("No nfs found")
		return nil
	}

	switch output {
	case util.OutputTable:
		return util.WriteAsTable(nfsList, nfsHeader)
	case util.OutputJSON:
		return util.WriteJSON(nfsList)
	}

	return nil
}
