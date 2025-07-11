// Copyright 2025 variHQ OÜ
// SPDX-License-Identifier: BSD-3-Clause

package main

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"
)

func Test_newApp(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		check       []runnerType
		regions     []string
		workerLimit int
		wantApp     bool
		wantErr     bool
		wantWorkers []runnerType
	}{
		{
			name:    "fails when no regions are provided",
			check:   nil,
			regions: nil,
			wantApp: false,
			wantErr: true,
		},
		{
			name:    "fails when no checkers are provided",
			check:   nil,
			regions: []string{"eu-west-1"},
			wantApp: false,
			wantErr: true,
		},
		{
			name:    "succeeds",
			check:   []runnerType{amiImage, ebsSnapshot, ssmDocument},
			regions: []string{"eu-west-1"},
			wantApp: true,
			wantErr: false,
		},
		{
			name:    "succeeds with multiple uniq regions",
			check:   []runnerType{amiImage, ebsSnapshot, ssmDocument},
			regions: []string{"eu-west-1", "eu-west-1"},
			wantApp: true,
			wantErr: false,
			wantWorkers: []runnerType{
				amiImage,
				ebsSnapshot,
				ssmDocument,
			},
		},
		{
			name: "successes with multiple regions",
			check: []runnerType{
				amiImage,
				ebsSnapshot,
				ssmDocument,
				rdsSnapshot,
			},
			regions: []string{"eu-west-1", "eu-west-2"},
			wantWorkers: []runnerType{
				amiImage,
				ebsSnapshot,
				ssmDocument,
				rdsSnapshot,
				rdsSnapshot,
				amiImage,
				ebsSnapshot,
				ssmDocument,
				rdsSnapshot,
				rdsSnapshot,
			},
			wantApp: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := newApp(t.Context(), tt.check, tt.regions, tt.workerLimit)
			if (err != nil) != tt.wantErr {
				t.Errorf("newApp() error = %v, wantErr %v", err, tt.wantErr)

				return
			}

			if (got != nil) != tt.wantApp {
				t.Errorf("got app = %v, want non-nil: %v", got, tt.wantApp)
			}

			if len(tt.wantWorkers) > 0 {
				var collected []runnerType
				for _, worker := range got.runners {
					collected = append(collected, worker.runType())
				}

				if !reflect.DeepEqual(collected, tt.wantWorkers) {
					t.Errorf("workers got = %v, want %v", collected, tt.wantWorkers)
				}
			}
		})
	}
}

func TestApp_Run(t *testing.T) {
	t.Parallel()

	withTimeout, _ := context.WithTimeout(t.Context(), -time.Minute) //nolint:govet

	mockRunners := []runner{
		&ebsSnapshotScan{
			baseRunner: baseRunner{
				region:     "eu-west-1",
				runnerType: ebsSnapshot,
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
		runners []runner
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
				runners:     tt.runners,
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

func TestApp_getAccountID(t *testing.T) {
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
			if err := a.getAccountID(t.Context()); (err != nil) != tt.wantErr {
				t.Errorf("getAccountID() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
