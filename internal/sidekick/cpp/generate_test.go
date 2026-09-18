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
	"github.com/googleapis/librarian/internal/sidekick/api"
)

func TestGenerate_EmitsCMakeLists(t *testing.T) {
	outdir := t.TempDir()
	model := api.NewTestAPI(nil, nil, nil)
	lib := &config.Library{
		Name:   "test-library",
		Output: outdir,
		Cpp:    &config.CppLibrary{},
	}

	ctx := context.Background()
	if err := Generate(ctx, model, outdir, lib); err != nil {
		t.Fatalf("Generate() failed: %v", err)
	}

	cmakeFile := filepath.Join(outdir, "CMakeLists.txt")
	info, err := os.Stat(cmakeFile)
	if err != nil {
		t.Fatalf("expected CMakeLists.txt to exist: %v", err)
	}
	if info.Size() != 0 {
		t.Errorf("expected empty CMakeLists.txt, got %d bytes", info.Size())
	}
}

func TestGenerate_EmitsServiceFiles(t *testing.T) {
	outdir := t.TempDir()
	svc := api.NewTestService("ExampleService")
	model := api.NewTestAPI(nil, nil, []*api.Service{svc})
	trueVal := true
	lib := &config.Library{
		Name:   "example",
		Output: outdir,
		Cpp: &config.CppLibrary{
			ProductPath:           outdir,
			GenerateGrpcTransport: &trueVal,
		},
	}

	ctx := context.Background()
	if err := Generate(ctx, model, outdir, lib); err != nil {
		t.Fatalf("Generate() failed: %v", err)
	}

	expectedFiles := []string{
		"example_client.h",
		"example_client.cc",
		"example_connection.h",
		"example_connection.cc",
		"example_connection_idempotency_policy.h",
		"example_connection_idempotency_policy.cc",
		"example_options.h",
		filepath.Join("mocks", "mock_example_connection.h"),
		filepath.Join("internal", "example_connection_impl.h"),
		filepath.Join("internal", "example_connection_impl.cc"),
		filepath.Join("internal", "example_stub_factory.h"),
		filepath.Join("internal", "example_stub_factory.cc"),
		filepath.Join("internal", "example_auth_decorator.h"),
		filepath.Join("internal", "example_auth_decorator.cc"),
		filepath.Join("internal", "example_logging_decorator.h"),
		filepath.Join("internal", "example_logging_decorator.cc"),
		filepath.Join("internal", "example_metadata_decorator.h"),
		filepath.Join("internal", "example_metadata_decorator.cc"),
		filepath.Join("internal", "example_stub.h"),
		filepath.Join("internal", "example_stub.cc"),
		filepath.Join("internal", "example_tracing_stub.h"),
		filepath.Join("internal", "example_tracing_stub.cc"),
		filepath.Join("internal", "example_option_defaults.h"),
		filepath.Join("internal", "example_option_defaults.cc"),
		filepath.Join("internal", "example_tracing_connection.h"),
		filepath.Join("internal", "example_tracing_connection.cc"),
		filepath.Join("internal", "example_sources.cc"),
	}

	for _, rel := range expectedFiles {
		fullPath := filepath.Join(outdir, rel)
		info, err := os.Stat(fullPath)
		if err != nil {
			t.Errorf("expected file %s to exist: %v", rel, err)
			continue
		}
		if info.Size() == 0 {
			t.Errorf("expected non-empty file for %s, got 0 bytes", rel)
		}
	}
}
