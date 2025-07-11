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
	_ rds.DescribeDBSnapshotsAPIClient = (rdsSnapshotClient)(nil)
	_ runner                           = (*rdsSnapshotScan)(nil)
)

type rdsSnapshotClient interface {
	rds.DescribeDBSnapshotsAPIClient
}

type rdsSnapshotFilter func(snapshot *types.DBSnapshot, target string) bool

func isRDSSnapshotOwner(snapshot *types.DBSnapshot, target string) bool {
	return !strings.Contains(":"+*snapshot.DBSnapshotIdentifier+":", target)
}

type rdsSnapshotScan struct {
	baseRunner
	client rdsSnapshotClient
	filter rdsSnapshotFilter
}

func newRDSSnapshotRunner(cfg aws.Config, filterFunc rdsSnapshotFilter) *rdsSnapshotScan {
	client := rds.NewFromConfig(cfg)

	return &rdsSnapshotScan{
		baseRunner: baseRunner{
			region:     cfg.Region,
			runnerType: rdsSnapshot,
		},
		client: client,
		filter: filterFunc,
	}
}

func (r *rdsSnapshotScan) scan(ctx context.Context, target string) ([]Result, error) {
	var output []Result

	paginator := rds.NewDescribeDBSnapshotsPaginator(r.client, &rds.DescribeDBSnapshotsInput{
		DBInstanceIdentifier: nil,
		DBSnapshotIdentifier: nil,
		DbiResourceId:        nil,
		Filters:              nil,
		IncludePublic:        aws.Bool(true),
		IncludeShared:        aws.Bool(true),
		Marker:               nil,
		MaxRecords:           nil,
		SnapshotType:         nil,
	})
	for paginator.HasMorePages() {
		if ctx.Err() != nil {
			return nil, fmt.Errorf("%w: %w", errCtxCancelled, ctx.Err())
		}

		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch RDS snapshots, %w", err)
		}

		for _, snapshot := range page.DBSnapshots {
			if r.filter != nil && r.filter(&snapshot, target) {
				slog.Debug("skipping RDS snapshots",
					slog.String("name", *snapshot.DBSnapshotIdentifier),
					slog.String("region", r.region),
				)

				continue
			}

			output = append(output, Result{
				CreationDate: snapshot.SnapshotCreateTime.Format(time.RFC3339),
				Identifier:   *snapshot.DBSnapshotIdentifier,
				Region:       r.region,
				RType:        r.runType(),
			})
		}
	}

	return output, nil
}
