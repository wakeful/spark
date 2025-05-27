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
	_ rds.DescribeDBSnapshotsAPIClient = (rdsSnapshotClient)(nil)
	_ runner                           = (*rdsSnapshotScan)(nil)
)

type rdsSnapshotClient interface {
	rds.DescribeDBSnapshotsAPIClient
}

type rdsSnapshotScan struct {
	client rdsSnapshotClient
	region string
}

func (r *rdsSnapshotScan) runType() runnerType {
	return rdsSnapshot
}

func newRDSSnapshotRunner(cfg aws.Config) *rdsSnapshotScan {
	client := rds.NewFromConfig(cfg)

	return &rdsSnapshotScan{
		client: client,
		region: cfg.Region,
	}
}

func (r *rdsSnapshotScan) scan(ctx context.Context, target string) ([]Result, error) {
	if target == "" {
		return nil, fmt.Errorf("%w: target account ID is required", errEmptyTarget)
	}

	slog.Debug(
		"starting RDS snapshot scan",
		slog.String("region", r.region),
		slog.String("target", target),
	)

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
			if !strings.Contains(*snapshot.DBSnapshotIdentifier, target) {
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

	slog.Debug("finished RDS snapshot scan",
		slog.Int("count", len(output)),
		slog.String("region", r.region),
		slog.String("target", target),
	)

	return output, nil
}
