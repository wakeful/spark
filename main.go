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
)

var version = "dev"

func main() {
	var (
		target       = flag.String("target", "self", "target AWS account ID")
		listScanners = flag.Bool("list-scanners", false, "list available resource types")
		showVersion  = flag.Bool("version", false, "show version")
		verbose      = flag.Bool("verbose", false, "verbose log output")
		regionVars   StringSlice
		scannersVars StringSlice
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

	slog.SetDefault(getLogger(os.Stderr, verbose))

	if *showVersion {
		slog.Info(
			"spark",
			slog.String("repo", "https://github.com/wakeful/spark"),
			slog.String("version", version),
		)

		return
	}

	if *listScanners {
		slog.Info("available resource types")

		for _, rType := range []runnerType{amiImage, ebsSnapshot, rdsSnapshot, ssmDocument} {
			_, _ = os.Stdout.Write([]byte(rType.String() + "\n"))
		}

		return
	}

	ctx := context.TODO()

	app, err := newApp(
		ctx,
		getRunners(scannersVars),
		regionVars,
	)
	if err != nil {
		slog.Error("failed to initialize app", slog.String("error", err.Error()))

		return
	}

	if errGetAccountID := app.getAccountID(ctx); errGetAccountID != nil {
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

	marshal, err := prepareOutput(output)
	if err != nil {
		slog.Error("failed to marshal output", slog.String("error", err.Error()))

		return
	}

	_, _ = os.Stdout.Write(marshal)
}
