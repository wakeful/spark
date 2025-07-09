// Copyright 2025 variHQ OÜ
// SPDX-License-Identifier: BSD-3-Clause

package main

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/aws/aws-sdk-go-v2/service/ssm/types"
)

type mockSSMDocumentClient struct {
	mockDocumentIdentifiers []types.DocumentIdentifier
	mockListDocumentsErr    error
}

func (m mockSSMDocumentClient) ListDocuments(
	_ context.Context,
	_ *ssm.ListDocumentsInput,
	_ ...func(*ssm.Options),
) (*ssm.ListDocumentsOutput, error) {
	return &ssm.ListDocumentsOutput{
		DocumentIdentifiers: m.mockDocumentIdentifiers,
	}, m.mockListDocumentsErr
}

var _ ssmDocumentClient = (*mockSSMDocumentClient)(nil)

func Test_ssmDocumentScan_scan(t *testing.T) {
	t.Parallel()
	withTimeout, _ := context.WithTimeout(t.Context(), -time.Minute) //nolint:govet
	now := time.Now()

	tests := []struct {
		name    string
		ctx     context.Context //nolint:containedctx
		client  ssmDocumentClient
		region  string
		target  string
		want    []Result
		wantErr bool
	}{
		{
			name: "should fail when ctx is cancelled",
			ctx:  withTimeout,
			client: &mockSSMDocumentClient{
				mockDocumentIdentifiers: nil,
				mockListDocumentsErr:    nil,
			},
			region:  "eu-west-1",
			target:  "self",
			want:    nil,
			wantErr: true,
		},
		{
			name: "should fail when api returns error",
			ctx:  t.Context(),
			client: &mockSSMDocumentClient{
				mockDocumentIdentifiers: nil,
				mockListDocumentsErr:    errors.New("some error"),
			},
			region:  "eu-west-1",
			target:  "self",
			want:    nil,
			wantErr: true,
		},
		{
			name: "should succeed with zero snapshots when no snapshots found",
			ctx:  t.Context(),
			client: &mockSSMDocumentClient{
				mockDocumentIdentifiers: nil,
				mockListDocumentsErr:    nil,
			},
			region:  "eu-west-1",
			target:  "self",
			want:    nil,
			wantErr: false,
		},
		{
			name: "should succeed with one snapshot",
			ctx:  t.Context(),
			client: &mockSSMDocumentClient{
				mockDocumentIdentifiers: []types.DocumentIdentifier{
					{
						CreatedDate: &now,
						Name:        aws.String("test-document-name"),
						Owner:       aws.String("self"),
					},
					{
						CreatedDate: &now,
						Name:        aws.String("document-to-skip"),
						Owner:       aws.String("AWS"),
					},
				},
				mockListDocumentsErr: nil,
			},
			region: "eu-west-1",
			target: "self",
			want: []Result{
				{
					CreationDate: now.Format(time.RFC3339),
					Identifier:   "test-document-name",
					Region:       "eu-west-1",
					RType:        ssmDocument,
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			s := ssmDocumentScan{
				baseRunner: baseRunner{
					region:     tt.region,
					runnerType: ssmDocument,
				},
				client: tt.client,
			}

			got, err := s.scan(tt.ctx, tt.target)
			if (err != nil) != tt.wantErr {
				t.Errorf("scan() error = %v, wantErr %v", err, tt.wantErr)

				return
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("scan() got = %v, want %v", got, tt.want)
			}
		})
	}
}
