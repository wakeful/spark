// Copyright 2025 variHQ OÜ
// SPDX-License-Identifier: BSD-3-Clause

// SPARK (Seeking Public AWS Resources and Kernels) is a command-line tool for identifying public
// AWS resources — including backups, AMIs, snapshots, and more—associated with specific AWS accounts.
package main

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"time"

	"github.com/wakeful/spark"
)

var version = "dev"

func main() { //nolint:cyclop
	const numberOfWorkers = 2

	var (
		target         = flag.String("target", "self", "target AWS account ID")
		listScanners   = flag.Bool("list-scanners", false, "list available resource types")
		showVersion    = flag.Bool("version", false, "show version")
		verbose        = flag.Bool("verbose", false, "verbose log output")
		scanAllRegions = flag.Bool("region-all", false, "scan all regions")
		scannersAll    = flag.Bool("scan-all", false, "scan all resource types")
		workerCount    = flag.Int("workers", numberOfWorkers, "number of workers used for scanning")
		regionVars     spark.StringSlice
		scannersVars   spark.StringSlice
	)

	flag.Var(
		&regionVars,
		"region",
		"AWS region to scan (can be specified multiple times)",
	)
	flag.Var(
		&scannersVars,
		"scan",
		"AWS resource type to scan (can be specified multiple times)",
	)
	flag.Parse()

	slog.SetDefault(spark.GetLogger(os.Stderr, verbose))

	if *showVersion {
		slog.Info(
			"spark",
			slog.String("repo", "https://github.com/wakeful/spark"),
			slog.String("version", version),
		)

		return
	}

	types := spark.GetSupportedScanners()

	if *listScanners {
		slog.Info("available resource types")

		for _, rType := range types {
			_, _ = os.Stdout.Write([]byte(rType + "\n"))
		}

		return
	}

	if *scanAllRegions {
		slog.Debug("scan all regions")

		regionVars = spark.SupportedRegions
	}

	if *scannersAll {
		slog.Debug("scan all resource types")

		scannersVars = types
	}

	const tickerInterval = 100

	ticker := time.NewTicker(tickerInterval * time.Millisecond)

	ctx := context.Background()
	if !*verbose {
		go spark.Spinner(ctx, os.Stderr, ticker.C)
	}

	app, err := spark.NewApp(
		ctx,
		spark.GetRunners(scannersVars),
		regionVars,
		*workerCount,
	)
	if err != nil {
		slog.Error("failed to initialize app", slog.String("error", err.Error()))

		return
	}

	errGetAccountID := app.GetAccountID(ctx)
	if errGetAccountID != nil {
		slog.Error(
			"failed to obtain current aws account ID",
			slog.String("error", errGetAccountID.Error()),
		)

		return
	}

	output, err := app.Run(ctx, *target)
	if err != nil {
		slog.Error("failed to run checks", slog.String("error", err.Error()))

		return
	}

	marshal, err := spark.PrepareOutput(output)
	if err != nil {
		slog.Error("failed to marshal output", slog.String("error", err.Error()))

		return
	}

	_, _ = os.Stdout.Write(marshal)
}
