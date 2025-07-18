// Copyright 2025 variHQ OÜ
// SPDX-License-Identifier: BSD-3-Clause

package spark

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

type mockRDSSnapshotClient struct {
	mockDBSnapshot             []types.DBSnapshot
	mockDescribeDBSnapshotsErr error
}

func (m *mockRDSSnapshotClient) DescribeDBSnapshots(
	_ context.Context,
	_ *rds.DescribeDBSnapshotsInput,
	_ ...func(*rds.Options),
) (*rds.DescribeDBSnapshotsOutput, error) {
	return &rds.DescribeDBSnapshotsOutput{
		DBSnapshots: m.mockDBSnapshot,
	}, m.mockDescribeDBSnapshotsErr
}

var _ rdsSnapshotClient = (*mockRDSSnapshotClient)(nil)

func Test_rdsSnapshotScan_scan(t *testing.T) {
	t.Parallel()
	withTimeout, _ := context.WithTimeout(t.Context(), -time.Minute) //nolint:govet
	now := time.Now()

	tests := []struct {
		name    string
		ctx     context.Context //nolint:containedctx
		client  rdsSnapshotClient
		region  string
		target  string
		want    []Result
		wantErr bool
	}{
		{
			name: "should fail when ctx is cancelled",
			ctx:  withTimeout,
			client: &mockRDSSnapshotClient{
				mockDBSnapshot:             nil,
				mockDescribeDBSnapshotsErr: nil,
			},
			region:  "eu-west-1",
			target:  "self",
			want:    nil,
			wantErr: true,
		},
		{
			name: "should fail when instance api returns error",
			ctx:  t.Context(),
			client: &mockRDSSnapshotClient{
				mockDBSnapshot:             nil,
				mockDescribeDBSnapshotsErr: errors.New("some error"),
			},
			region:  "eu-west-1",
			target:  "self",
			want:    nil,
			wantErr: true,
		},
		{
			name: "should succeed with zero snapshots when no snapshots found",
			ctx:  t.Context(),
			client: &mockRDSSnapshotClient{
				mockDBSnapshot:             nil,
				mockDescribeDBSnapshotsErr: nil,
			},
			region:  "eu-west-1",
			target:  "self",
			want:    nil,
			wantErr: false,
		},
		{
			name: "should succeed with one instance snapshot",
			ctx:  t.Context(),
			client: &mockRDSSnapshotClient{
				mockDBSnapshot: []types.DBSnapshot{
					{
						SnapshotCreateTime:   &now,
						DBSnapshotIdentifier: aws.String("test-self-id"),
					},
					{
						SnapshotCreateTime:   &now,
						DBSnapshotIdentifier: aws.String("test-skip-id"),
					},
				},
				mockDescribeDBSnapshotsErr: nil,
			},
			region: "eu-west-1",
			target: "self",
			want: []Result{
				{
					CreationDate: now.Format(time.RFC3339),
					Identifier:   "test-self-id",
					Region:       "eu-west-1",
					RType:        SnapshotRDS,
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			r := &RDSSnapshotScan{
				baseRunner: baseRunner{
					region:     tt.region,
					runnerType: SnapshotRDS,
				},
				client: tt.client,
				filter: isRDSSnapshotOwner,
			}

			got, err := r.Scan(tt.ctx, tt.target)
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
