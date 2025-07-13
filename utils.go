// Copyright 2025 variHQ OÜ
// SPDX-License-Identifier: BSD-3-Clause

package spark

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"sort"
	"strings"
	"time"
)

var (
	// ErrEmptyAccountID is returned when the AWS caller identity has no account ID.
	ErrEmptyAccountID = errors.New("failed to get caller identity, empty account ID")
	// ErrCtxCancelled indicates the scan was canceled via context.
	ErrCtxCancelled = errors.New("scan was cancelled")
	// ErrEmptyCheck is returned when no resource types are specified for scanning.
	ErrEmptyCheck = errors.New(
		"no resource types specified; use -list-scanners, -scan <type>, or -scan-all",
	)
	// ErrEmptyRegion is returned when no AWS regions are specified.
	ErrEmptyRegion = errors.New("no AWS regions specified; use -region <name> or -region-all")
	// ErrEmptyTarget indicates a missing target AWS account ID.
	ErrEmptyTarget = errors.New("empty target AWS account ID")
)

// GetLogger returns a slog.Logger configured with the given output and log level.
// If verbose is true, the log level is set to debug; otherwise, it defaults to info.
func GetLogger(output io.Writer, verbose *bool) *slog.Logger {
	logLevel := slog.LevelInfo
	if verbose != nil && *verbose {
		logLevel = slog.LevelDebug
	}

	logger := slog.New(
		slog.NewTextHandler(output, &slog.HandlerOptions{
			AddSource:   false,
			Level:       logLevel,
			ReplaceAttr: nil,
		}),
	)

	return logger
}

// PrepareOutput returns a pretty-printed JSON byte slice from a list of Result objects.
func PrepareOutput(output []Result) ([]byte, error) {
	marshal, err := json.MarshalIndent(struct {
		Results []Result `json:"results"`
	}{
		Results: output,
	}, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal output, %w", err)
	}

	return marshal, nil
}

// GetSupportedScanners returns supported AWS scanner names.
func GetSupportedScanners() []string {
	return []string{
		ImageAMI.String(),
		SnapshotEBS.String(),
		DocumentSSM.String(),
		SnapshotRDS.String(),
	}
}

// GetRunners maps input strings to unique RunnerType values, ignoring case and invalid entries.
func GetRunners(input []string) []RunnerType {
	uniq := make(map[RunnerType]struct{})

	for _, scan := range input {
		switch {
		case strings.EqualFold(scan, ImageAMI.String()):
			uniq[ImageAMI] = struct{}{}
		case strings.EqualFold(scan, SnapshotEBS.String()):
			uniq[SnapshotEBS] = struct{}{}
		case strings.EqualFold(scan, DocumentSSM.String()):
			uniq[DocumentSSM] = struct{}{}
		case strings.EqualFold(scan, SnapshotRDS.String()):
			uniq[SnapshotRDS] = struct{}{}
		default:
			slog.Debug("invalid scan type", slog.String("type", scan))
		}
	}

	output := make([]RunnerType, 0, len(uniq))
	for scan := range uniq {
		output = append(output, scan)
	}

	sort.Slice(output, func(left, right int) bool {
		return output[left].String() < output[right].String()
	})

	return output
}

func uniqRegions(input []string) []string {
	uniq := make(map[string]struct{})
	for _, region := range input {
		uniq[region] = struct{}{}
	}

	output := make([]string, 0, len(uniq))
	for region := range uniq {
		output = append(output, region)
	}

	sort.Strings(output)

	return output
}

// Spinner writes a rotating animation to the writer, updating on each tick until the context is canceled.
func Spinner(ctx context.Context, writer io.Writer, ticker <-chan time.Time) {
	spinChars := []rune{
		'|',
		'/',
		'-',
		'\\',
	}

	for position := 0; ; position = (position + 1) % len(spinChars) {
		select {
		case <-ctx.Done():
			_, _ = fmt.Fprintln(writer, "\r\033[K")

			return
		case <-ticker:
			_, _ = fmt.Fprintf(writer, "\r\033[K%s %c", "scanning...", spinChars[position])
		}
	}
}
