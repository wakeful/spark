// Copyright 2025 variHQ OÜ
// SPDX-License-Identifier: BSD-3-Clause

package spark

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
	_ Runner                     = (*SSMDocumentScan)(nil)
	_ ssm.ListDocumentsAPIClient = (ssmDocumentClient)(nil)
)

type ssmDocumentClient interface {
	ssm.ListDocumentsAPIClient
}

// SSMDocumentFilter defines a function for filtering SSM documents.
type SSMDocumentFilter func(document *types.DocumentIdentifier, target string) bool

func isSSMDocumentOwner(document *types.DocumentIdentifier, target string) bool {
	return *document.Owner != target
}

// SSMDocumentScan scans SSM documents in a region using an SSM client and filter.
type SSMDocumentScan struct {
	baseRunner

	client ssmDocumentClient
	filter SSMDocumentFilter
}

// NewSSMDocumentScan creates a new SSMDocumentScan with the given config and filter.
func NewSSMDocumentScan(cfg aws.Config, filterFunc SSMDocumentFilter) *SSMDocumentScan {
	client := ssm.NewFromConfig(cfg)

	return &SSMDocumentScan{
		baseRunner: baseRunner{
			region:     cfg.Region,
			runnerType: DocumentSSM,
		},
		client: client,
		filter: filterFunc,
	}
}

// Scan retrieves SSM documents matching the filter.
func (s SSMDocumentScan) Scan(ctx context.Context, target string) ([]Result, error) {
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
			return nil, fmt.Errorf("%w: %w", ErrCtxCancelled, ctx.Err())
		}

		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch SSM document(s), %w", err)
		}

		for _, document := range page.DocumentIdentifiers {
			if s.filter != nil && s.filter(&document, target) {
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
				RType:        s.RunType(),
			})
		}
	}

	return output, nil
}
