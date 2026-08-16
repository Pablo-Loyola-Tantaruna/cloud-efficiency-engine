package gcp

import (
	"context"
	"fmt"

	"cloud.google.com/go/bigquery"
	compute "cloud.google.com/go/compute/apiv1"
	container "cloud.google.com/go/container/apiv1"
)

type Clients struct {
	ClusterManager        *container.ClusterManagerClient
	MachineTypes          *compute.MachineTypesClient
	InstanceGroupManagers *compute.InstanceGroupManagersClient
	BigQuery              *bigquery.Client
}

func NewClients(ctx context.Context, projectID string) (*Clients, error) {
	if projectID == "" {
		return nil, fmt.Errorf("GCP project ID must not be empty")
	}

	clusterManager, err := container.NewClusterManagerClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("create GCP GKE client: %w", err)
	}

	machineTypes, err := compute.NewMachineTypesRESTClient(ctx)
	if err != nil {
		_ = clusterManager.Close()
		return nil, fmt.Errorf("create GCP Compute machine types client: %w", err)
	}

	instanceGroupManagers, err := compute.NewInstanceGroupManagersRESTClient(ctx)
	if err != nil {
		_ = machineTypes.Close()
		_ = clusterManager.Close()
		return nil, fmt.Errorf("create GCP managed instance group client: %w", err)
	}

	bigQuery, err := bigquery.NewClient(ctx, projectID)
	if err != nil {
		_ = instanceGroupManagers.Close()
		_ = machineTypes.Close()
		_ = clusterManager.Close()
		return nil, fmt.Errorf("create GCP BigQuery client: %w", err)
	}

	return &Clients{
		ClusterManager:        clusterManager,
		MachineTypes:          machineTypes,
		InstanceGroupManagers: instanceGroupManagers,
		BigQuery:              bigQuery,
	}, nil
}

func (c *Clients) Close() error {
	if c == nil {
		return nil
	}

	if c.BigQuery != nil {
		if err := c.BigQuery.Close(); err != nil {
			return err
		}
	}
	if c.InstanceGroupManagers != nil {
		if err := c.InstanceGroupManagers.Close(); err != nil {
			return err
		}
	}
	if c.MachineTypes != nil {
		if err := c.MachineTypes.Close(); err != nil {
			return err
		}
	}
	if c.ClusterManager != nil {
		return c.ClusterManager.Close()
	}
	return nil
}
