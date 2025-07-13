// Copyright 2025 variHQ OÜ
// SPDX-License-Identifier: BSD-3-Clause

package spark

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/rds"
	"github.com/aws/aws-sdk-go-v2/service/rds/types"
)

var (
	_ rds.DescribeDBClusterSnapshotsAPIClient = (rdsClusterSnapshotClient)(nil)
	_ Runner                                  = (*RDSClusterSnapshotScan)(nil)
)

type rdsClusterSnapshotClient interface {
	rds.DescribeDBClusterSnapshotsAPIClient
}

// RdsClusterSnapshotFilter defines a function for filtering RDS cluster snapshots.
type RdsClusterSnapshotFilter func(snapshot *types.DBClusterSnapshot, target string) bool

func isRDSClusterSnapshotOwner(snapshot *types.DBClusterSnapshot, target string) bool {
	return !strings.Contains(":"+*snapshot.DBClusterSnapshotIdentifier+":", target)
}

// RDSClusterSnapshotScan scans RDS cluster snapshots in a region using an RDS client and filter.
type RDSClusterSnapshotScan struct {
	baseRunner

	client rdsClusterSnapshotClient
	filter RdsClusterSnapshotFilter
}

// NewRDSClusterSnapshotRunner creates a new RDSClusterSnapshotScan with the given config and filter.
func NewRDSClusterSnapshotRunner(
	cfg aws.Config,
	filterFunc RdsClusterSnapshotFilter,
) *RDSClusterSnapshotScan {
	client := rds.NewFromConfig(cfg)

	return &RDSClusterSnapshotScan{
		baseRunner: baseRunner{
			region:     cfg.Region,
			runnerType: SnapshotRDS,
		},
		client: client,
		filter: filterFunc,
	}
}

// Scan retrieves RDS cluster snapshots for the target AWS account.
func (r *RDSClusterSnapshotScan) Scan(ctx context.Context, target string) ([]Result, error) {
	var output []Result

	paginator := rds.NewDescribeDBClusterSnapshotsPaginator(
		r.client,
		&rds.DescribeDBClusterSnapshotsInput{
			DBClusterIdentifier:         nil,
			DBClusterSnapshotIdentifier: nil,
			DbClusterResourceId:         nil,
			Filters:                     nil,
			IncludePublic:               aws.Bool(true),
			IncludeShared:               aws.Bool(true),
			Marker:                      nil,
			MaxRecords:                  nil,
			SnapshotType:                nil,
		},
	)
	for paginator.HasMorePages() {
		if ctx.Err() != nil {
			return nil, fmt.Errorf("%w: %w", ErrCtxCancelled, ctx.Err())
		}

		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch RDS cluster snapshots, %w", err)
		}

		for _, snapshot := range page.DBClusterSnapshots {
			if r.filter != nil && r.filter(&snapshot, target) {
				slog.Debug("skipping RDS cluster snapshots",
					slog.String("name", *snapshot.DBClusterSnapshotIdentifier),
					slog.String("region", r.region),
				)

				continue
			}

			output = append(output, Result{
				CreationDate: snapshot.SnapshotCreateTime.Format(time.RFC3339),
				Identifier:   *snapshot.DBClusterSnapshotIdentifier,
				Region:       r.region,
				RType:        r.RunType(),
			})
		}
	}

	return output, nil
}
