// Copyright 2025 variHQ OÜ
// SPDX-License-Identifier: BSD-3-Clause

package main

import (
	"context"
	_ "embed"
	"fmt"
	"log/slog"
	"os"
	"sort"
	"text/template"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
)

//go:embed regions.gotpl
var templateContent string

type regionTemplate struct {
	Regions []string
}

const useRegion = "eu-west-1"

func main() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		AddSource: false,
		Level:     slog.LevelDebug,
	})))

	ctx := context.Background()

	cnf, errGetConfig := config.LoadDefaultConfig(ctx, config.WithRegion(useRegion))
	if errGetConfig != nil {
		slog.Error("failed to get AWS config", slog.String("error", errGetConfig.Error()))

		return
	}

	regions, errGetRegions := getActiveRegions(ctx, cnf)
	if errGetRegions != nil {
		slog.Error("failed to get active regions", slog.String("error", errGetRegions.Error()))

		return
	}

	slog.Info("found regions", slog.Int("count", len(regions)))

	target, errCreateFile := os.Create("./gen_regions.go")
	if errCreateFile != nil {
		slog.Error("failed to create target file", slog.String("error", errCreateFile.Error()))

		return
	}

	defer func(target *os.File) {
		errCloseFile := target.Close()
		if errCloseFile != nil {
			slog.Error("failed to close target file", slog.String("error", errCloseFile.Error()))
		}
	}(target)

	supportedRegionsTemplate := template.Must(template.New("").Parse(templateContent))

	if errTemplate := supportedRegionsTemplate.Execute(target, regionTemplate{Regions: regions}); errTemplate != nil {
		slog.Error("failed to execute template", slog.String("error", errTemplate.Error()))

		return
	}

	slog.Warn("generated file with regions")
}

// getActiveRegions return a list of active regions in the current AWS account.
func getActiveRegions(ctx context.Context, cfg aws.Config) ([]string, error) {
	client := ec2.NewFromConfig(cfg)

	regions, err := client.DescribeRegions(ctx, &ec2.DescribeRegionsInput{
		AllRegions: aws.Bool(false),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to describe regions: %w", err)
	}

	activeRegions := make([]string, 0, len(regions.Regions))

	for _, region := range regions.Regions {
		slog.Debug("found region", slog.String("name", *region.RegionName))
		activeRegions = append(activeRegions, *region.RegionName)
	}

	slog.Debug("found regions", slog.Int("count", len(activeRegions)))

	sort.Strings(activeRegions)

	return activeRegions, nil
}
