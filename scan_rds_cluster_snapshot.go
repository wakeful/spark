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
)

var (
	_ rds.DescribeDBClusterSnapshotsAPIClient = (rdsClusterSnapshotClient)(nil)
	_ runner                                  = (*rdsClusterSnapshotScan)(nil)
)

type rdsClusterSnapshotClient interface {
	rds.DescribeDBClusterSnapshotsAPIClient
}

type rdsClusterSnapshotScan struct {
	baseRunner
	client rdsClusterSnapshotClient
}

func newRDSClusterSnapshotRunner(cfg aws.Config) *rdsClusterSnapshotScan {
	client := rds.NewFromConfig(cfg)

	return &rdsClusterSnapshotScan{
		baseRunner: baseRunner{
			region:     cfg.Region,
			runnerType: rdsSnapshot,
		},
		client: client,
	}
}

func (r *rdsClusterSnapshotScan) scan(ctx context.Context, target string) ([]Result, error) {
	if target == "" {
		return nil, fmt.Errorf("%w: target account ID is required", errEmptyTarget)
	}

	slog.Debug(
		"starting RDS cluster snapshot scan",
		slog.String("region", r.region),
		slog.String("target", target),
	)

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
			if !strings.Contains(*snapshot.DBClusterSnapshotIdentifier, target) {
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

	slog.Debug("finished RDS cluster snapshot scan",
		slog.Int("count", len(output)),
		slog.String("region", r.region),
		slog.String("target", target),
	)

	return output, nil
}
