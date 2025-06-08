// Copyright 2025 variHQ OÜ
// SPDX-License-Identifier: BSD-3-Clause

package main

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
	errEmptyAccountID = errors.New("failed to get caller identity, empty account ID")
	errCtxCancelled   = errors.New("scan was cancelled")
	errEmptyCheck     = errors.New("no resource types specified")
	errEmptyRegion    = errors.New("no AWS regions specified")
	errEmptyTarget    = errors.New("empty target AWS account ID")
)

// getLogger returns a slog.Logger configured with the given output and log level.
// If verbose is true, the log level is set to debug; otherwise, it defaults to info.
func getLogger(output io.Writer, verbose *bool) *slog.Logger {
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

func prepareOutput(output []Result) ([]byte, error) {
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

func getRunners(input []string) []runnerType {
	uniq := make(map[runnerType]struct{})

	for _, scan := range input {
		switch {
		case strings.EqualFold(scan, amiImage.String()):
			uniq[amiImage] = struct{}{}
		case strings.EqualFold(scan, ebsSnapshot.String()):
			uniq[ebsSnapshot] = struct{}{}
		case strings.EqualFold(scan, ssmDocument.String()):
			uniq[ssmDocument] = struct{}{}
		case strings.EqualFold(scan, rdsSnapshot.String()):
			uniq[rdsSnapshot] = struct{}{}
		default:
			slog.Debug("invalid scan type", slog.String("type", scan))
		}
	}

	output := make([]runnerType, 0, len(uniq))
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

func spinner(ctx context.Context, writer io.Writer, ticker <-chan time.Time) {
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
