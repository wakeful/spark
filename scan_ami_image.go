// Copyright 2025 variHQ OÜ
// SPDX-License-Identifier: BSD-3-Clause

package main

import (
	"context"
	"fmt"

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

	return output, nil
}
