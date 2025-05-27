// Copyright 2025 variHQ OÜ
// SPDX-License-Identifier: BSD-3-Clause

package main

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
)

var (
	_ ec2.DescribeSnapshotsAPIClient = (ebsSnapshotClient)(nil)
	_ runner                         = (*ebsSnapshotScan)(nil)
)

type ebsSnapshotClient interface {
	ec2.DescribeSnapshotsAPIClient
}

type ebsSnapshotScan struct {
	client ebsSnapshotClient
	region string
}

func (s *ebsSnapshotScan) runType() runnerType {
	return ebsSnapshot
}

func newEBSSnapshotRunner(cfg aws.Config) *ebsSnapshotScan {
	client := ec2.NewFromConfig(cfg)

	return &ebsSnapshotScan{
		client: client,
		region: cfg.Region,
	}
}

func (s *ebsSnapshotScan) scan(ctx context.Context, target string) ([]Result, error) {
	if target == "" {
		return nil, fmt.Errorf("%w: target account ID is required", errEmptyTarget)
	}

	slog.Debug(
		"starting EBS snapshot scan",
		slog.String("region", s.region),
		slog.String("target", target),
	)

	var output []Result

	paginator := ec2.NewDescribeSnapshotsPaginator(s.client, &ec2.DescribeSnapshotsInput{
		DryRun:              nil,
		Filters:             nil,
		MaxResults:          nil,
		NextToken:           nil,
		OwnerIds:            []string{target},
		RestorableByUserIds: nil,
		SnapshotIds:         nil,
	})
	for paginator.HasMorePages() {
		if ctx.Err() != nil {
			return nil, fmt.Errorf("%w: %w", errCtxCancelled, ctx.Err())
		}

		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch snapshots, %w", err)
		}

		for _, snapshot := range page.Snapshots {
			output = append(output, Result{
				CreationDate: snapshot.CompletionTime.Format(time.RFC3339),
				Identifier:   *snapshot.SnapshotId,
				Region:       s.region,
				RType:        s.runType(),
			})
		}
	}

	slog.Debug("finished EBS snapshot scan",
		slog.Int("count", len(output)),
		slog.String("region", s.region),
		slog.String("target", target),
	)

	return output, nil
}
