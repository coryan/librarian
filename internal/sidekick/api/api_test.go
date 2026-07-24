// Copyright 2026 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package api

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func TestWithPackageNames(t *testing.T) {
	for _, test := range []struct {
		name        string
		input       string
		inputPascal string
		want        *API
	}{
		{
			"disco",
			"google.cloud.compute.v1",
			"",
			&API{
				PackageName:           "google.cloud.compute.v1",
				PackageNamePascalCase: "Google.Cloud.Compute.V1",
			},
		},
		{
			"proto simple",
			"google.cloud.secretmanager.v1",
			"",
			&API{
				PackageName:           "google.cloud.secretmanager.v1",
				PackageNamePascalCase: "Google.Cloud.Secretmanager.V1",
			},
		},
		{
			"proto with override",
			"google.cloud.secretmanager.v1",
			"Google.Cloud.SecretManager.V1",
			&API{
				PackageName:           "google.cloud.secretmanager.v1",
				PackageNamePascalCase: "Google.Cloud.SecretManager.V1",
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := &API{}
			got.WithPackageNames(test.input, test.inputPascal)
			if diff := cmp.Diff(test.want, got, cmpopts.IgnoreUnexported(API{})); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
