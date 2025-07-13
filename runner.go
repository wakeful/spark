// Copyright 2025 variHQ OÜ
// SPDX-License-Identifier: BSD-3-Clause

package spark

import (
	"context"
	"encoding/json"
	"fmt"
)

// RunnerType represents the type of runner used in specific operations or processes.
//
//go:generate go tool -modfile=tools/go.mod stringer -type=RunnerType -linecomment -output=runner_type_string.go
type RunnerType int

// MarshalJSON customizes the JSON marshaling of RunnerType by returning its string representation.
func (i RunnerType) MarshalJSON() ([]byte, error) {
	return json.Marshal(i.String()) //nolint:wrapcheck
}

const (
	// ImageAMI represents a scanner for Amazon Machine Images (AMIs).
	ImageAMI RunnerType = iota + 1 // AMI
	// SnapshotEBS represents a scanner for EBS snapshots.
	SnapshotEBS // snapshotsEBS
	// SnapshotRDS represents a scanner for RDS snapshots.
	SnapshotRDS // snapshotsRDS
	// DocumentSSM represents a scanner for SSM documents.
	DocumentSSM
)

var (
	_ json.Marshaler = (*RunnerType)(nil)
	_ fmt.Stringer   = (*RunnerType)(nil)
)

// Runner defines an interface for scanning and retrieving runner metadata.
type Runner interface {
	Scan(ctx context.Context, target string) ([]Result, error)
	RunType() RunnerType
	getRegion() string
}

type baseRunner struct {
	region     string
	runnerType RunnerType
}

func (b baseRunner) RunType() RunnerType {
	return b.runnerType
}

func (b baseRunner) getRegion() string {
	return b.region
}
