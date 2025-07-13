// Copyright 2025 variHQ OÜ
// SPDX-License-Identifier: BSD-3-Clause

package spark

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"golang.org/x/sync/errgroup"
)

// App represents a struct that provides functionality for interacting with the AWS services.
type App struct {
	accountID   string
	Runners     []Runner
	stsClient   stsClient
	workerLimit int
}

// NewApp initializes and returns a new App with the given settings and runners.
func NewApp(
	ctx context.Context,
	check []RunnerType,
	regions []string,
	workerLimit int,
) (*App, error) {
	if len(regions) == 0 {
		return nil, ErrEmptyRegion
	}

	if len(check) == 0 {
		return nil, ErrEmptyCheck
	}

	regions = uniqRegions(regions)

	if workerLimit < 1 {
		workerLimit = 1
	}

	baseCfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to load aws config, %w", err)
	}

	stsCfg := baseCfg.Copy()
	stsCfg.Region = regions[0]

	runners := setUpRunners(baseCfg, check, regions)

	return &App{
		accountID:   "",
		Runners:     runners,
		stsClient:   sts.NewFromConfig(stsCfg),
		workerLimit: workerLimit,
	}, nil
}

// setUpRunners initializes and returns a list of runners based on the specified configuration, checks, and regions.
func setUpRunners(baseCfg aws.Config, check []RunnerType, regions []string) []Runner {
	runners := make([]Runner, 0)

	for _, region := range regions {
		cfg := baseCfg.Copy()
		cfg.Region = region

		for _, scan := range check {
			switch scan {
			case ImageAMI:
				runners = append(runners, NewAMIScan(cfg))
			case SnapshotEBS:
				runners = append(runners, NewEBSSnapshotRunner(cfg))
			case SnapshotRDS:
				runners = append(
					runners,
					NewRDSSnapshotRunner(cfg, isRDSSnapshotOwner),
					NewRDSClusterSnapshotRunner(cfg, isRDSClusterSnapshotOwner),
				)
			case DocumentSSM:
				runners = append(runners, NewSSMDocumentScan(cfg, isSSMDocumentOwner))
			}
		}
	}

	return runners
}

// Run scans the target using all runners and returns the results or an error.
func (a *App) Run(ctx context.Context, target string) ([]Result, error) { //nolint:funlen
	group, gCtx := errgroup.WithContext(ctx)
	group.SetLimit(a.workerLimit)

	if strings.EqualFold(target, "self") {
		slog.Debug("replacing self with account ID",
			slog.String("accountID", a.accountID),
		)

		target = a.accountID
	}

	buffer := make(chan []Result, len(a.Runners))
	for _, scanRunner := range a.Runners {
		group.Go(func() error {
			select {
			case <-gCtx.Done():
				return gCtx.Err()
			default:
				if target == "" {
					return fmt.Errorf("%w: target account ID is required", ErrEmptyTarget)
				}

				slog.Debug(
					"starting scan",
					slog.String("region", scanRunner.getRegion()),
					slog.String("target", target),
					slog.String("type", scanRunner.RunType().String()),
				)

				scanResults, err := scanRunner.Scan(ctx, target)
				if err != nil {
					return fmt.Errorf(
						"failed to scan %s, in region %s, %w",
						scanRunner.RunType().String(),
						scanRunner.getRegion(),
						err,
					)
				}

				slog.Debug("finished scan",
					slog.Int("count", len(scanResults)),
					slog.String("region", scanRunner.getRegion()),
					slog.String("target", target),
					slog.String("type", scanRunner.RunType().String()),
				)

				buffer <- scanResults
			}

			return nil
		})
	}

	err := group.Wait()
	if err != nil {
		return nil, fmt.Errorf("failed to run all checks, %w", err)
	}

	close(buffer)

	var output []Result
	for item := range buffer {
		output = append(output, item...)
	}

	return output, nil
}

// GetAccountID fetches the AWS account ID and sets it in App.
func (a *App) GetAccountID(ctx context.Context) error {
	output, err := a.stsClient.GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
	if err != nil {
		return fmt.Errorf("failed to get caller identity, %w", err)
	}

	if *output.Account == "" {
		return ErrEmptyAccountID
	}

	a.accountID = *output.Account

	return nil
}
