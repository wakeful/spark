// Copyright 2025 variHQ OÜ
// SPDX-License-Identifier: BSD-3-Clause

package main

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
	runners     []runner
	stsClient   stsClient
	workerLimit int
}

func newApp(
	ctx context.Context,
	check []runnerType,
	regions []string,
	workerLimit int,
) (*App, error) {
	if len(regions) == 0 {
		return nil, errEmptyRegion
	}

	if len(check) == 0 {
		return nil, errEmptyCheck
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
		runners:     runners,
		stsClient:   sts.NewFromConfig(stsCfg),
		workerLimit: workerLimit,
	}, nil
}

// setUpRunners initializes and returns a list of runners based on the specified configuration, checks, and regions.
func setUpRunners(baseCfg aws.Config, check []runnerType, regions []string) []runner {
	runners := make([]runner, 0)

	for _, region := range regions {
		cfg := baseCfg.Copy()
		cfg.Region = region

		for _, scan := range check {
			switch scan {
			case amiImage:
				runners = append(runners, newAMIImageScan(cfg))
			case ebsSnapshot:
				runners = append(runners, newEBSSnapshotRunner(cfg))
			case rdsSnapshot:
				runners = append(
					runners,
					newRDSSnapshotRunner(cfg, isRDSSnapshotOwner),
					newRDSClusterSnapshotRunner(cfg, isRDSClusterSnapshotOwner),
				)
			case ssmDocument:
				runners = append(runners, newSSMDocumentScan(cfg, isSSMDocumentOwner))
			}
		}
	}

	return runners
}

// Run executes the scan process on the target using all available runners,
// and returns the collected results or an error.
func (a *App) Run(ctx context.Context, target string) ([]Result, error) {
	group, gCtx := errgroup.WithContext(ctx)
	group.SetLimit(a.workerLimit)

	if strings.EqualFold(target, "self") {
		slog.Debug("replacing self with account ID",
			slog.String("accountID", a.accountID),
		)

		target = a.accountID
	}

	buffer := make(chan []Result, len(a.runners))

	for _, scanRunner := range a.runners {
		group.Go(func() error {
			select {
			case <-gCtx.Done():
				return gCtx.Err()
			default:
				if target == "" {
					return fmt.Errorf("%w: target account ID is required", errEmptyTarget)
				}

				slog.Debug(
					"starting scan",
					slog.String("region", scanRunner.getRegion()),
					slog.String("target", target),
					slog.String("type", scanRunner.runType().String()),
				)

				scanResults, err := scanRunner.scan(ctx, target)
				if err != nil {
					return err
				}

				slog.Debug("finished scan",
					slog.Int("count", len(scanResults)),
					slog.String("region", scanRunner.getRegion()),
					slog.String("target", target),
					slog.String("type", scanRunner.runType().String()),
				)
				buffer <- scanResults
			}

			return nil
		})
	}

	if err := group.Wait(); err != nil {
		return nil, fmt.Errorf("failed to run all checks, %w", err)
	}

	close(buffer)

	var output []Result
	for item := range buffer {
		output = append(output, item...)
	}

	return output, nil
}

func (a *App) getAccountID(ctx context.Context) error {
	output, err := a.stsClient.GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
	if err != nil {
		return fmt.Errorf("failed to get caller identity, %w", err)
	}

	if *output.Account == "" {
		return errEmptyAccountID
	}

	a.accountID = *output.Account

	return nil
}
