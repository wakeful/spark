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
	"github.com/aws/aws-sdk-go-v2/service/rds"
	"github.com/aws/aws-sdk-go-v2/service/rds/types"
)

type mockRDSClusterSnapshotClient struct {
	mockDBClusterSnapshot    []types.DBClusterSnapshot
	mockDBClusterSnapshotErr error
}

func (m *mockRDSClusterSnapshotClient) DescribeDBClusterSnapshots(
	_ context.Context,
	_ *rds.DescribeDBClusterSnapshotsInput,
	_ ...func(*rds.Options),
) (*rds.DescribeDBClusterSnapshotsOutput, error) {
	return &rds.DescribeDBClusterSnapshotsOutput{
		DBClusterSnapshots: m.mockDBClusterSnapshot,
	}, m.mockDBClusterSnapshotErr
}

var _ rdsClusterSnapshotClient = (*mockRDSClusterSnapshotClient)(nil)

func Test_rdsClusterSnapshotScan_scan(t *testing.T) {
	t.Parallel()
	withTimeout, _ := context.WithTimeout(t.Context(), -time.Minute) //nolint:govet
	now := time.Now()

	tests := []struct {
		name    string
		ctx     context.Context //nolint:containedctx
		client  rdsClusterSnapshotClient
		region  string
		target  string
		want    []Result
		wantErr bool
	}{
		{
			name: "should fail when ctx is cancelled",
			ctx:  withTimeout,
			client: &mockRDSClusterSnapshotClient{
				mockDBClusterSnapshot:    nil,
				mockDBClusterSnapshotErr: nil,
			},
			region:  "eu-west-1",
			target:  "self",
			want:    nil,
			wantErr: true,
		},
		{
			name: "should fail when instance api returns error",
			ctx:  t.Context(),
			client: &mockRDSClusterSnapshotClient{
				mockDBClusterSnapshot:    nil,
				mockDBClusterSnapshotErr: errors.New("some error"),
			},
			region:  "eu-west-1",
			target:  "self",
			want:    nil,
			wantErr: true,
		},
		{
			name: "should succeed with zero snapshots when no snapshots found",
			ctx:  t.Context(),
			client: &mockRDSClusterSnapshotClient{
				mockDBClusterSnapshot:    nil,
				mockDBClusterSnapshotErr: nil,
			},
			region:  "eu-west-1",
			target:  "self",
			want:    nil,
			wantErr: false,
		},
		{
			name: "should succeed with one instance snapshot",
			ctx:  t.Context(),
			client: &mockRDSClusterSnapshotClient{
				mockDBClusterSnapshot: []types.DBClusterSnapshot{
					{
						SnapshotCreateTime:          &now,
						DBClusterSnapshotIdentifier: aws.String("test-self-id"),
					},
					{
						SnapshotCreateTime:          &now,
						DBClusterSnapshotIdentifier: aws.String("test-skip-id"),
					},
				},
				mockDBClusterSnapshotErr: nil,
			},
			region: "eu-west-1",
			target: "self",
			want: []Result{
				{
					CreationDate: now.Format(time.RFC3339),
					Identifier:   "test-self-id",
					Region:       "eu-west-1",
					RType:        rdsSnapshot,
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			r := &rdsClusterSnapshotScan{
				baseRunner: baseRunner{
					region:     tt.region,
					runnerType: rdsSnapshot,
				},
				client: tt.client,
				filter: isRDSClusterSnapshotOwner,
			}

			got, err := r.scan(tt.ctx, tt.target)
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
