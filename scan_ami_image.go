// Copyright 2025 variHQ OÜ
// SPDX-License-Identifier: BSD-3-Clause

package main

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
)

var (
	_ ec2.DescribeImagesAPIClient = (amiClient)(nil)
	_ runner                      = (*amiImageScan)(nil)
)

type amiClient interface {
	ec2.DescribeImagesAPIClient
}

type amiImageScan struct {
	baseRunner
	client amiClient
}

func newAMIImageScan(cfg aws.Config) *amiImageScan {
	client := ec2.NewFromConfig(cfg)

	return &amiImageScan{
		baseRunner: baseRunner{
			region:     cfg.Region,
			runnerType: amiImage,
		},
		client: client,
	}
}

func (s *amiImageScan) scan(ctx context.Context, target string) ([]Result, error) {
	if target == "" {
		return nil, fmt.Errorf("%w: target account ID is required", errEmptyTarget)
	}

	slog.Debug(
		"starting AMI scan",
		slog.String("region", s.region),
		slog.String("target", target),
	)

	var output []Result

	paginator := ec2.NewDescribeImagesPaginator(s.client, &ec2.DescribeImagesInput{
		DryRun:            nil,
		ExecutableUsers:   nil,
		Filters:           nil,
		ImageIds:          nil,
		IncludeDeprecated: nil,
		IncludeDisabled:   nil,
		MaxResults:        nil,
		NextToken:         nil,
		Owners:            []string{target},
	})
	for paginator.HasMorePages() {
		if ctx.Err() != nil {
			return nil, fmt.Errorf("%w: %w", errCtxCancelled, ctx.Err())
		}

		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch AMI(s), %w", err)
		}

		for _, image := range page.Images {
			output = append(output, Result{
				CreationDate: *image.CreationDate,
				Identifier:   *image.ImageId,
				Region:       s.region,
				RType:        s.runType(),
			})
		}
	}

	slog.Debug("finished AMI scan",
		slog.Int("count", len(output)),
		slog.String("region", s.region),
		slog.String("target", target),
	)

	return output, nil
}
