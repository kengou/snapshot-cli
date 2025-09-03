package sharedfilesystem

import (
	"context"
	"fmt"

	"github.com/gophercloud/gophercloud/v2/openstack/sharedfilesystems/v2/shares"

	"snapshot-cli/internal/auth"
	"snapshot-cli/internal/config"
	"snapshot-cli/internal/util"
)

func RunGetSharedFileSystem(ctx context.Context, shareID, output string) error {
	authConfig, err := config.ReadAuthConfig()
	if err != nil {
		return err
	}
	sharedClient, err := auth.NewSharedFileSystemClient(ctx, authConfig)
	if err != nil {
		return err
	}

	nfs, err := shares.Get(ctx, sharedClient, shareID).Extract()
	if err != nil {
		return err
	}

	if nfs == nil {
		fmt.Println("NFS share not found")
		return nil
	}

	switch output {
	case util.OutputTable:
		return util.WriteAsTable(nfs, nfsHeader)
	case util.OutputJSON:
		return util.WriteJSON(nfs)
	}

	return nil
}
