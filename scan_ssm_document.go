// Copyright 2025 variHQ OÜ
// SPDX-License-Identifier: BSD-3-Clause

package main

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/aws/aws-sdk-go-v2/service/ssm/types"
)

var (
	_ runner                     = (*ssmDocumentScan)(nil)
	_ ssm.ListDocumentsAPIClient = (ssmDocumentClient)(nil)
)

type ssmDocumentClient interface {
	ssm.ListDocumentsAPIClient
}

type ssmDocumentScan struct {
	baseRunner
	client ssmDocumentClient
}

func newSSMDocumentScan(cfg aws.Config) *ssmDocumentScan {
	client := ssm.NewFromConfig(cfg)

	return &ssmDocumentScan{
		baseRunner: baseRunner{
			region:     cfg.Region,
			runnerType: ssmDocument,
		},
		client: client,
	}
}

func (s ssmDocumentScan) scan(ctx context.Context, target string) ([]Result, error) {
	var output []Result

	paginator := ssm.NewListDocumentsPaginator(s.client, &ssm.ListDocumentsInput{
		DocumentFilterList: nil,
		Filters: []types.DocumentKeyValuesFilter{
			{
				Key:    aws.String("Owner"),
				Values: []string{"Public"},
			},
		},
		MaxResults: nil,
		NextToken:  nil,
	})
	for paginator.HasMorePages() {
		if ctx.Err() != nil {
			return nil, fmt.Errorf("%w: %w", errCtxCancelled, ctx.Err())
		}

		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch SSM document(s), %w", err)
		}

		for _, document := range page.DocumentIdentifiers {
			if *document.Owner != target {
				slog.Debug("skipping SSM document",
					slog.String("name", *document.Name),
					slog.String("region", s.region),
					slog.String("owner", *document.Owner),
				)

				continue
			}

			output = append(output, Result{
				CreationDate: document.CreatedDate.Format(time.RFC3339),
				Identifier:   *document.Name,
				Region:       s.region,
				RType:        s.runType(),
			})
		}
	}

	return output, nil
}
