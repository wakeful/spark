// Copyright 2025 variHQ OÜ
// SPDX-License-Identifier: BSD-3-Clause

package main

import (
	"context"
	"encoding/json"
	"fmt"
)

//go:generate go tool -modfile=tools/go.mod stringer -type=runnerType -linecomment -output=runner_type_string.go
type runnerType int

func (i runnerType) MarshalJSON() ([]byte, error) {
	return json.Marshal(i.String()) //nolint:wrapcheck
}

const (
	amiImage    runnerType = iota + 1 // AMI
	ebsSnapshot                       // snapshotsEBS
	rdsSnapshot                       // snapshotsRDS
	ssmDocument
)

var (
	_ json.Marshaler = (*runnerType)(nil)
	_ fmt.Stringer   = (*runnerType)(nil)
)

type runner interface {
	scan(ctx context.Context, target string) ([]Result, error)
	runType() runnerType
}
