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
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

type mockEBSSnapshotClient struct {
	mockSnapshot    []types.Snapshot
	mockSnapshotErr error
}

func (m *mockEBSSnapshotClient) DescribeSnapshots(
	_ context.Context,
	_ *ec2.DescribeSnapshotsInput,
	_ ...func(*ec2.Options),
) (*ec2.DescribeSnapshotsOutput, error) {
	return &ec2.DescribeSnapshotsOutput{Snapshots: m.mockSnapshot}, m.mockSnapshotErr
}

var _ ebsSnapshotClient = (*mockEBSSnapshotClient)(nil)

func Test_ebsSnapshotScan_scan(t *testing.T) {
	t.Parallel()
	withTimeout, _ := context.WithTimeout(t.Context(), -time.Minute) //nolint:govet
	now := time.Now()

	tests := []struct {
		name    string
		ctx     context.Context //nolint:containedctx
		client  ebsSnapshotClient
		region  string
		target  string
		want    []Result
		wantErr bool
	}{
		{
			name: "should fail when ctx is cancelled",
			ctx:  withTimeout,
			client: &mockEBSSnapshotClient{
				mockSnapshot:    nil,
				mockSnapshotErr: nil,
			},
			region:  "eu-west-1",
			target:  "self",
			want:    nil,
			wantErr: true,
		},
		{
			name: "should fail when api returns error",
			ctx:  t.Context(),
			client: &mockEBSSnapshotClient{
				mockSnapshot:    nil,
				mockSnapshotErr: errors.New("some error"),
			},
			region:  "eu-west-1",
			target:  "self",
			want:    nil,
			wantErr: true,
		},
		{
			name: "should succeed with zero snapshots when no snapshots found",
			ctx:  t.Context(),
			client: &mockEBSSnapshotClient{
				mockSnapshot:    nil,
				mockSnapshotErr: nil,
			},
			region:  "eu-west-1",
			target:  "self",
			want:    nil,
			wantErr: false,
		},
		{
			name: "should succeed with one snapshot",
			ctx:  t.Context(),
			client: &mockEBSSnapshotClient{
				mockSnapshot: []types.Snapshot{
					{
						CompletionTime: &now,
						SnapshotId:     aws.String("test-snapshot-id"),
					},
				},
				mockSnapshotErr: nil,
			},
			region: "eu-west-1",
			target: "self",
			want: []Result{
				{
					CreationDate: now.Format(time.RFC3339),
					Identifier:   "test-snapshot-id",
					Region:       "eu-west-1",
					RType:        ebsSnapshot,
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			s := &ebsSnapshotScan{
				baseRunner: baseRunner{
					region:     tt.region,
					runnerType: ebsSnapshot,
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
