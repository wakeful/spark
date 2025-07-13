// Copyright 2025 variHQ OÜ
// SPDX-License-Identifier: BSD-3-Clause

package spark

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

type mockAMIClient struct {
	mockImages   []types.Image
	mockImageErr error
}

func (m *mockAMIClient) DescribeImages(
	_ context.Context,
	_ *ec2.DescribeImagesInput,
	_ ...func(*ec2.Options),
) (*ec2.DescribeImagesOutput, error) {
	return &ec2.DescribeImagesOutput{Images: m.mockImages}, m.mockImageErr
}

var _ amiClient = (*mockAMIClient)(nil)

func Test_amiImageScan_scan(t *testing.T) {
	t.Parallel()
	withTimeout, _ := context.WithTimeout(t.Context(), -time.Minute) //nolint:govet

	tests := []struct {
		name    string
		ctx     context.Context //nolint:containedctx
		client  amiClient
		region  string
		target  string
		want    []Result
		wantErr bool
	}{
		{
			name: "should fail when ctx is cancelled",
			ctx:  withTimeout,
			client: &mockAMIClient{
				mockImages:   nil,
				mockImageErr: nil,
			},
			region:  "eu-west-1",
			target:  "self",
			want:    nil,
			wantErr: true,
		},
		{
			name: "should fail when api returns error",
			ctx:  t.Context(),
			client: &mockAMIClient{
				mockImages:   nil,
				mockImageErr: errors.New("some error"),
			},
			region:  "eu-west-1",
			target:  "self",
			want:    nil,
			wantErr: true,
		},
		{
			name: "should succeed with zero images when no images found",
			ctx:  t.Context(),
			client: &mockAMIClient{
				mockImages:   nil,
				mockImageErr: nil,
			},
			region:  "eu-west-1",
			target:  "self",
			want:    nil,
			wantErr: false,
		},
		{
			name: "should succeed with one AMI",
			ctx:  t.Context(),
			client: &mockAMIClient{
				mockImages: []types.Image{
					{
						CreationDate: aws.String("properly formatted date"),
						ImageId:      aws.String("test-image-id"),
					},
				},
				mockImageErr: nil,
			},
			region: "eu-west-1",
			target: "self",
			want: []Result{
				{
					CreationDate: "properly formatted date",
					Identifier:   "test-image-id",
					Region:       "eu-west-1",
					RType:        ImageAMI,
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			s := &AMIScan{
				baseRunner: baseRunner{
					region:     tt.region,
					runnerType: ImageAMI,
				},
				client: tt.client,
			}

			got, err := s.Scan(tt.ctx, tt.target)
			if (err != nil) != tt.wantErr {
				t.Errorf("scan() error = %v, wantErr %v", err, tt.wantErr)

				return
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("scan() got = %v, want %v", got, tt.want)
			}
		})
	}
}
