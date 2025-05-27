// Copyright 2025 variHQ OÜ
// SPDX-License-Identifier: BSD-3-Clause

package main

// Result represents the output of a scanning operation, including metadata about the scanned resource.
type Result struct {
	CreationDate string     `json:"creationDate"`
	Identifier   string     `json:"identifier"`
	Region       string     `json:"region"`
	RType        runnerType `json:"type"`
}
