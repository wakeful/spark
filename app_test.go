// Copyright 2025 variHQ OÜ
// SPDX-License-Identifier: BSD-3-Clause

package spark_test

import (
	"reflect"
	"testing"

	"github.com/wakeful/spark"
)

func Test_NewApp(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		check       []spark.RunnerType
		regions     []string
		workerLimit int
		wantApp     bool
		wantErr     bool
		wantWorkers []spark.RunnerType
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
			check:   []spark.RunnerType{spark.ImageAMI, spark.SnapshotEBS, spark.DocumentSSM},
			regions: []string{"eu-west-1"},
			wantApp: true,
			wantErr: false,
		},
		{
			name:    "succeeds with multiple uniq regions",
			check:   []spark.RunnerType{spark.ImageAMI, spark.SnapshotEBS, spark.DocumentSSM},
			regions: []string{"eu-west-1", "eu-west-1"},
			wantApp: true,
			wantErr: false,
			wantWorkers: []spark.RunnerType{
				spark.ImageAMI,
				spark.SnapshotEBS,
				spark.DocumentSSM,
			},
		},
		{
			name: "successes with multiple regions",
			check: []spark.RunnerType{
				spark.ImageAMI,
				spark.SnapshotEBS,
				spark.DocumentSSM,
				spark.SnapshotRDS,
			},
			regions: []string{"eu-west-1", "eu-west-2"},
			wantWorkers: []spark.RunnerType{
				spark.ImageAMI,
				spark.SnapshotEBS,
				spark.DocumentSSM,
				spark.SnapshotRDS,
				spark.SnapshotRDS,
				spark.ImageAMI,
				spark.SnapshotEBS,
				spark.DocumentSSM,
				spark.SnapshotRDS,
				spark.SnapshotRDS,
			},
			wantApp: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := spark.NewApp(t.Context(), tt.check, tt.regions, tt.workerLimit)
			if (err != nil) != tt.wantErr {
				t.Errorf("newApp() error = %v, wantErr %v", err, tt.wantErr)

				return
			}

			if (got != nil) != tt.wantApp {
				t.Errorf("got app = %v, want non-nil: %v", got, tt.wantApp)
			}

			if len(tt.wantWorkers) > 0 {
				var collected []spark.RunnerType
				for _, worker := range got.Runners {
					collected = append(collected, worker.RunType())
				}

				if !reflect.DeepEqual(collected, tt.wantWorkers) {
					t.Errorf("workers got = %v, want %v", collected, tt.wantWorkers)
				}
			}
		})
	}
}
