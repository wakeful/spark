// Copyright 2025 variHQ OÜ
// SPDX-License-Identifier: BSD-3-Clause

package main

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
	_ runner                                  = (*rdsClusterSnapshotScan)(nil)
)

type rdsClusterSnapshotClient interface {
	rds.DescribeDBClusterSnapshotsAPIClient
}

type rdsClusterSnapshotFilter func(snapshot *types.DBClusterSnapshot, target string) bool

func isRDSClusterSnapshotOwner(snapshot *types.DBClusterSnapshot, target string) bool {
	return !strings.Contains(":"+*snapshot.DBClusterSnapshotIdentifier+":", target)
}

type rdsClusterSnapshotScan struct {
	baseRunner
	client rdsClusterSnapshotClient
	filter rdsClusterSnapshotFilter
}

func newRDSClusterSnapshotRunner(
	cfg aws.Config,
	filterFunc rdsClusterSnapshotFilter,
) *rdsClusterSnapshotScan {
	client := rds.NewFromConfig(cfg)

	return &rdsClusterSnapshotScan{
		baseRunner: baseRunner{
			region:     cfg.Region,
			runnerType: rdsSnapshot,
		},
		client: client,
		filter: filterFunc,
	}
}

func (r *rdsClusterSnapshotScan) scan(ctx context.Context, target string) ([]Result, error) {
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
			return nil, fmt.Errorf("%w: %w", errCtxCancelled, ctx.Err())
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
				RType:        r.runType(),
			})
		}
	}

	return output, nil
}
