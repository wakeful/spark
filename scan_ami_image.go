// Copyright 2025 variHQ OÜ
// SPDX-License-Identifier: BSD-3-Clause

package spark

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
)

var (
	_ ec2.DescribeImagesAPIClient = (amiClient)(nil)
	_ Runner                      = (*AMIScan)(nil)
)

type amiClient interface {
	ec2.DescribeImagesAPIClient
}

// AMIScan scans Amazon Machine Images in a region using an EC2 client.
type AMIScan struct {
	baseRunner

	client amiClient
}

// NewAMIScan creates a new AMIScan with the given config.
func NewAMIScan(cfg aws.Config) *AMIScan {
	client := ec2.NewFromConfig(cfg)

	return &AMIScan{
		baseRunner: baseRunner{
			region:     cfg.Region,
			runnerType: ImageAMI,
		},
		client: client,
	}
}

// Scan retrieves Amazon Machine Images for the target AWS account.
func (s *AMIScan) Scan(ctx context.Context, target string) ([]Result, error) {
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
			return nil, fmt.Errorf("%w: %w", ErrCtxCancelled, ctx.Err())
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
				RType:        s.RunType(),
			})
		}
	}

	return output, nil
}
