// Copyright 2025 variHQ OÜ
// SPDX-License-Identifier: BSD-3-Clause

package spark

import "testing"

func Test_runnerType_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		rType RunnerType
		want  string
	}{
		{
			name:  "AMI",
			rType: ImageAMI,
			want:  "AMI",
		},
		{
			name:  "unknown",
			rType: RunnerType(-1),
			want:  "RunnerType(-1)",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := tt.rType.String(); got != tt.want {
				t.Errorf("String() = %v, want %v", got, tt.want)
			}
		})
	}
}
