package blockstorage

import (
	"context"
	"fmt"

	"github.com/gophercloud/gophercloud/v2/openstack/blockstorage/v3/snapshots"
	"github.com/gophercloud/gophercloud/v2/openstack/blockstorage/v3/volumes"

	"snapshot-cli/internal/auth"
	"snapshot-cli/internal/config"
)

type SnapShotOpts struct {
	VolumeID    string
	Force       bool
	Name        string
	Description string
}

func CreateSnapshotBlockStorage(ctx context.Context, snapOpts *SnapShotOpts) error {
	authConfig, err := config.ReadAuthConfig()
	if err != nil {
		return err
	}
	blockClient, err := auth.NewBlockStorageClient(ctx, authConfig)
	if err != nil {
		return err
	}

	vol, err := volumes.Get(ctx, blockClient, snapOpts.VolumeID).Extract()
	if err != nil {
		return err
	}

	if vol == nil {
		fmt.Println("No volume found")
		return nil
	}

	result, err := snapshots.Create(ctx, blockClient, snapshots.CreateOpts{
		VolumeID:    snapOpts.VolumeID,
		Force:       snapOpts.Force,
		Name:        snapOpts.Name,
		Description: snapOpts.Description,
	}).Extract()

	if err != nil {
		return fmt.Errorf("failed to create snapshot for volume %s: %w", vol.ID, err)
	}

	fmt.Printf("Snapshot created successfully: ID=%s, Status=%s\n", result.ID, result.Status)

	return nil
}
