// Copyright 2025 variHQ OÜ
// SPDX-License-Identifier: BSD-3-Clause

package spark

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
)

var (
	_ ec2.DescribeSnapshotsAPIClient = (ebsSnapshotClient)(nil)
	_ Runner                         = (*EBSSnapshotScan)(nil)
)

type ebsSnapshotClient interface {
	ec2.DescribeSnapshotsAPIClient
}

// EBSSnapshotScan scans EBS snapshots in a region using an EC2 client.
type EBSSnapshotScan struct {
	baseRunner

	client ebsSnapshotClient
}

// NewEBSSnapshotRunner creates a new EBSSnapshotScan with the given config.
func NewEBSSnapshotRunner(cfg aws.Config) *EBSSnapshotScan {
	client := ec2.NewFromConfig(cfg)

	return &EBSSnapshotScan{
		baseRunner: baseRunner{
			region:     cfg.Region,
			runnerType: SnapshotEBS,
		},
		client: client,
	}
}

// Scan retrieves EBS snapshots for the target AWS account.
func (s *EBSSnapshotScan) Scan(ctx context.Context, target string) ([]Result, error) {
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
			return nil, fmt.Errorf("%w: %w", ErrCtxCancelled, ctx.Err())
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
				RType:        s.RunType(),
			})
		}
	}

	return output, nil
}
