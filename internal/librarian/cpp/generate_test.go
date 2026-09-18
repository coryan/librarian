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
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/googleapis/librarian/internal/config"
)

func TestGenerate(t *testing.T) {
	outdir := t.TempDir()
	lib := &config.Library{
		Name:   "test-cpp-library",
		Output: outdir,
		Cpp:    &config.CppLibrary{},
	}
	ctx := context.Background()
	if err := Generate(ctx, nil, lib, nil); err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	cmakeFile := filepath.Join(outdir, "CMakeLists.txt")
	if _, err := os.Stat(cmakeFile); err != nil {
		t.Fatalf("CMakeLists.txt does not exist: %v", err)
	}
}

func TestDefaultOutput(t *testing.T) {
	for _, test := range []struct {
		name       string
		api        string
		defaultOut string
		want       string
	}{
		{
			name:       "with default out",
			api:        "google/cloud/secretmanager/v1",
			defaultOut: "google/cloud",
			want:       filepath.Join("google/cloud", "google/cloud/secretmanager/v1"),
		},
		{
			name:       "without default out",
			api:        "google/cloud/secretmanager/v1",
			defaultOut: "",
			want:       "google/cloud/secretmanager/v1",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := DefaultOutput(test.api, test.defaultOut)
			if got != test.want {
				t.Errorf("DefaultOutput() = %v, want %v", got, test.want)
			}
		})
	}
}

func TestAdd(t *testing.T) {
	lib := &config.Library{
		Output: "google/cloud/secretmanager/v1",
	}
	got := Add(lib, nil)
	if got.Cpp == nil {
		t.Fatal("expected Cpp config to be initialized")
	}
	if got.Cpp.ProductPath != "google/cloud/secretmanager/v1" {
		t.Errorf("got ProductPath = %v, want %v", got.Cpp.ProductPath, "google/cloud/secretmanager/v1")
	}
}
