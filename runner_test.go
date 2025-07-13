// Copyright 2025 variHQ OÜ
// SPDX-License-Identifier: BSD-3-Clause

package spark_test

import (
	"reflect"
	"testing"

	"github.com/wakeful/spark"
)

func Test_runnerType_MarshalJSON(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		i       spark.RunnerType
		want    []byte
		wantErr bool
	}{
		{
			name: "runType(0)",
			i:    0,
			want: []byte{
				34, 82, 117, 110, 110, 101, 114, 84, 121, 112, 101, 40, 48, 41, 34,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := tt.i.MarshalJSON()
			if (err != nil) != tt.wantErr {
				t.Errorf("MarshalJSON() error = %v, wantErr %v", err, tt.wantErr)

				return
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("MarshalJSON() got = %v, want %v", got, tt.want)
			}
		})
	}
}
