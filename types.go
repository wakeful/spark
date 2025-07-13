// Copyright 2025 variHQ OÜ
// SPDX-License-Identifier: BSD-3-Clause

package spark

import (
	"context"
	"flag"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/sts"
)

// StringSlice is a type alias representing a slice of strings, commonly used to handle multiple string inputs.
type StringSlice []string

// String returns the StringSlice as a single comma-separated string.
func (s *StringSlice) String() string {
	return strings.Join(*s, ", ")
}

// Set appends the provided string value to the StringSlice. Returns an error if the operation fails.
func (s *StringSlice) Set(value string) error {
	*s = append(*s, value)

	return nil
}

var _ flag.Value = (*StringSlice)(nil)

type stsClient interface {
	GetCallerIdentity(
		ctx context.Context,
		params *sts.GetCallerIdentityInput,
		optFns ...func(*sts.Options),
	) (*sts.GetCallerIdentityOutput, error)
}
