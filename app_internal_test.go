// Copyright 2025 variHQ OÜ
// SPDX-License-Identifier: BSD-3-Clause

package spark

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"
)

func TestApp_Run(t *testing.T) {
	t.Parallel()

	withTimeout, _ := context.WithTimeout(t.Context(), -time.Minute) //nolint:govet

	mockRunners := []Runner{
		&EBSSnapshotScan{
			baseRunner: baseRunner{
				region:     "eu-west-1",
				runnerType: SnapshotEBS,
			},
			client: &mockEBSSnapshotClient{
				mockSnapshot:    nil,
				mockSnapshotErr: nil,
			},
		},
	}

	tests := []struct {
		name    string
		ctx     context.Context //nolint:containedctx
		runners []Runner
		target  string
		want    []Result
		wantErr bool
	}{
		{
			name:    "successful run",
			ctx:     t.Context(),
			runners: mockRunners,
			target:  "42",
			want:    nil,
			wantErr: false,
		},
		{
			name:    "fail with timeout",
			ctx:     withTimeout,
			runners: mockRunners,
			target:  "self",
			want:    nil,
			wantErr: true,
		},
		{
			name:    "error during the scan",
			ctx:     t.Context(),
			runners: mockRunners,
			target:  "",
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			a := &App{
				Runners:     tt.runners,
				workerLimit: 1,
			}

			got, err := a.Run(tt.ctx, tt.target)
			if (err != nil) != tt.wantErr {
				t.Errorf("Run() error = %v, wantErr %v", err, tt.wantErr)

				return
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Run() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestApp_GetAccountID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		stsClient stsClient
		wantErr   bool
	}{
		{
			name: "fail with error",
			stsClient: &mockSTSClient{
				mockAccountID:            "",
				mockGetCallerIdentityErr: errors.New("some error"),
			},
			wantErr: true,
		},
		{
			name: "fail with empty account id",
			stsClient: &mockSTSClient{
				mockAccountID:            "",
				mockGetCallerIdentityErr: nil,
			},
			wantErr: true,
		},
		{
			name: "obtain account ID without any errors",
			stsClient: &mockSTSClient{
				mockAccountID:            "42",
				mockGetCallerIdentityErr: nil,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			a := &App{
				stsClient: tt.stsClient,
			}

			err := a.GetAccountID(t.Context())
			if (err != nil) != tt.wantErr {
				t.Errorf("getAccountID() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
