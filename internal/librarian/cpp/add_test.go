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

package cpp

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
)

func TestAdd(t *testing.T) {
	lib := &config.Library{
		Name:          "google-cloud-secretmanager-v1",
		CopyrightYear: "2026",
		APIs: []*config.API{
			{Path: "google/cloud/secretmanager/v1/service.proto"},
		},
	}

	got := Add(lib)

	if got.Output != "google/cloud/secretmanager/v1" {
		t.Errorf("expected output google/cloud/secretmanager/v1, got %s", got.Output)
	}
	if got.Cpp == nil {
		t.Fatal("expected Cpp configuration to be populated")
	}
	if got.Cpp.ProductPath != "google/cloud/secretmanager/v1" {
		t.Errorf("expected product path google/cloud/secretmanager/v1, got %s", got.Cpp.ProductPath)
	}
	if got.Cpp.InitialCopyrightYear != "2026" {
		t.Errorf("expected initial copyright year 2026, got %s", got.Cpp.InitialCopyrightYear)
	}
	wantCodes := []string{"kUnavailable"}
	if diff := cmp.Diff(wantCodes, got.Cpp.RetryableStatusCodes); diff != "" {
		t.Errorf("retryable status codes mismatch (-want +got):\n%s", diff)
	}
}
