// Copyright 2025 variHQ OÜ
// SPDX-License-Identifier: BSD-3-Clause

package main

import (
	"bytes"
	"log/slog"
	"reflect"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
)

func Test_getLogger(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name             string
		verbose          *bool
		testMessage      string
		useLevel         slog.Level
		expectInOutput   bool
		testOutputSearch string
	}{
		{
			name:             "returns configured logger with verbose=false",
			verbose:          aws.Bool(false),
			testMessage:      "debug message",
			useLevel:         slog.LevelDebug,
			expectInOutput:   false,
			testOutputSearch: "debug message",
		},
		{
			name:             "enables debug logging when verbose=true",
			verbose:          aws.Bool(true),
			testMessage:      "debug message",
			useLevel:         slog.LevelDebug,
			expectInOutput:   true,
			testOutputSearch: "debug message",
		},
		{
			name:             "info messages are always logged",
			verbose:          aws.Bool(false),
			testMessage:      "info message",
			useLevel:         slog.LevelInfo,
			expectInOutput:   true,
			testOutputSearch: "info message",
		},
		{
			name:             "handles nil verbose flag",
			verbose:          nil,
			testMessage:      "info message",
			useLevel:         slog.LevelInfo,
			expectInOutput:   true,
			testOutputSearch: "info message",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var buf bytes.Buffer

			logger := getLogger(&buf, tt.verbose)

			if logger == nil {
				t.Fatal("Expected logger to not be nil")
			}

			switch tt.useLevel {
			case slog.LevelDebug:
				logger.Debug(tt.testMessage)
			case slog.LevelInfo:
				logger.Info(tt.testMessage)
			case slog.LevelWarn:
				logger.Warn(tt.testMessage)
			case slog.LevelError:
				logger.Error(tt.testMessage)
			}

			logOutput := buf.String()
			containsMessage := strings.Contains(logOutput, tt.testOutputSearch)

			if containsMessage != tt.expectInOutput {
				if tt.expectInOutput {
					t.Errorf("Expected log output to contain '%s', but it did not. Got: %s",
						tt.testOutputSearch, logOutput)
				} else {
					t.Errorf("Expected log output to NOT contain '%s', but it did. Got: %s",
						tt.testOutputSearch, logOutput)
				}
			}
		})
	}
}

func Test_prepareOutput(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		output  []Result
		want    []byte
		wantErr bool
	}{
		{
			name:   "empty results list",
			output: []Result{},
			want: []byte{
				123,
				10,
				32,
				32,
				34,
				114,
				101,
				115,
				117,
				108,
				116,
				115,
				34,
				58,
				32,
				91,
				93,
				10,
				125,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := prepareOutput(tt.output)
			if (err != nil) != tt.wantErr {
				t.Errorf("prepareOutput() error = %v, wantErr %v", err, tt.wantErr)

				return
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("prepareOutput() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_getRunners(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input []string
		want  []runnerType
	}{
		{
			name:  "empty input",
			input: []string{},
			want:  []runnerType{},
		},
		{
			name: "check that the result is a uniq slice",
			input: []string{
				"ami",
				"AMi",
				"ssmDocument",
				"snapshotsEBS",
				"snapshotsRDS",
				"42",
			},
			want: []runnerType{amiImage, ebsSnapshot, rdsSnapshot, ssmDocument},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := getRunners(tt.input); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getRunners() = %v, want %v", got, tt.want)
			}
		})
	}
}
