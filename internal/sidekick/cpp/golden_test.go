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
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/sidekick/api"
	"github.com/googleapis/librarian/internal/sidekick/parser"
	"github.com/googleapis/librarian/internal/sources"
	"github.com/googleapis/librarian/internal/yaml"
)

func TestGoldenFixtures_Intact(t *testing.T) {
	requiredFixtures := []string{
		"backup.proto",
		"common.proto",
		"test.proto",
		"test2.proto",
		"test_deprecated.proto",
		"test_request_id.proto",
		"test.yaml",
		"test_request_id.yaml",
	}
	for _, name := range requiredFixtures {
		path := filepath.Join("testdata", "protos", name)
		info, err := os.Stat(path)
		if err != nil {
			t.Errorf("expected fixture %s to exist: %v", path, err)
			continue
		}
		if info.Size() == 0 {
			t.Errorf("fixture %s is empty", path)
		}
	}
}

func TestGoldenFixtures_GoldenFileCount(t *testing.T) {
	const wantCount = 188
	goldenDir := filepath.Join("testdata", "golden")

	var gotCount int
	err := filepath.WalkDir(goldenDir, func(_ string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			gotCount++
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walking golden dir: %v", err)
	}

	if gotCount != wantCount {
		t.Fatalf("golden file count mismatch: want %d, got %d", wantCount, gotCount)
	}

	// Verify key structural files exist in reference golden.
	keyFiles := []string{
		"golden_kitchen_sink_client.h",
		"golden_thing_admin_client.h",
		filepath.Join("mocks", "mock_golden_kitchen_sink_connection.h"),
		filepath.Join("mocks", "mock_golden_thing_admin_connection.h"),
		filepath.Join("v1", "golden_kitchen_sink_client.h"),
		filepath.Join("v1", "golden_kitchen_sink_client.cc"),
		filepath.Join("v1", "golden_thing_admin_client.h"),
		filepath.Join("v1", "golden_rest_only_client.h"),
		filepath.Join("v1", "request_id_client.h"),
		filepath.Join("v1", "deprecated_client.h"),
		filepath.Join("v1", "mocks", "mock_golden_kitchen_sink_connection.h"),
		filepath.Join("v1", "internal", "golden_kitchen_sink_stub.h"),
	}
	for _, keyFile := range keyFiles {
		path := filepath.Join(goldenDir, keyFile)
		if _, err := os.Stat(path); err != nil {
			t.Errorf("expected key golden file %s to exist: %v", keyFile, err)
		}
	}
}

func TestGoldenFixtures_LibrarianConfig(t *testing.T) {
	configPath := filepath.Join("testdata", "golden_librarian.yaml")
	cfg, err := yaml.Read[config.Config](configPath)
	if err != nil {
		t.Fatalf("failed to read golden_librarian.yaml: %v", err)
	}

	if cfg.Language != config.LanguageCpp {
		t.Errorf("language mismatch: want %q, got %q", config.LanguageCpp, cfg.Language)
	}
	if len(cfg.Libraries) != 4 {
		t.Fatalf("library count mismatch: want 4, got %d", len(cfg.Libraries))
	}

	// Check library 0: golden
	lib0 := cfg.Libraries[0]
	if lib0.Name != "golden" {
		t.Errorf("lib0 Name: want 'golden', got %q", lib0.Name)
	}
	if len(lib0.APIs) != 1 || lib0.APIs[0].Path != "generator/integration_tests/test.proto" {
		t.Errorf("lib0 APIs mismatch: %+v", lib0.APIs)
	}
	if lib0.Cpp == nil {
		t.Fatalf("lib0 Cpp config is nil")
	}
	if lib0.Cpp.InitialCopyrightYear != "2022" {
		t.Errorf("lib0 InitialCopyrightYear: want '2022', got %q", lib0.Cpp.InitialCopyrightYear)
	}
	if !lib0.Cpp.GenerateRestTransport {
		t.Errorf("lib0 GenerateRestTransport: want true")
	}
	if !lib0.Cpp.GenerateRoundRobinDecorator {
		t.Errorf("lib0 GenerateRoundRobinDecorator: want true")
	}
	if lib0.Cpp.ServiceEndpointEnvVar != "GOLDEN_KITCHEN_SINK_ENDPOINT" {
		t.Errorf("lib0 ServiceEndpointEnvVar: want 'GOLDEN_KITCHEN_SINK_ENDPOINT', got %q", lib0.Cpp.ServiceEndpointEnvVar)
	}
	if len(lib0.Cpp.IdempotencyOverrides) != 2 {
		t.Errorf("lib0 IdempotencyOverrides: want 2, got %d", len(lib0.Cpp.IdempotencyOverrides))
	}

	// Check library 1: golden-test2 (REST-only)
	lib1 := cfg.Libraries[1]
	if lib1.Cpp == nil {
		t.Fatalf("lib1 Cpp config is nil")
	}
	if lib1.Cpp.GenerateGrpcTransport == nil || *lib1.Cpp.GenerateGrpcTransport {
		t.Errorf("lib1 GenerateGrpcTransport: want false")
	}
	if lib1.Cpp.EndpointLocationStyle != "LOCATION_OPTIONALLY_DEPENDENT" {
		t.Errorf("lib1 EndpointLocationStyle: want LOCATION_OPTIONALLY_DEPENDENT, got %q", lib1.Cpp.EndpointLocationStyle)
	}

	// Check library 2: test-request-id
	lib2 := cfg.Libraries[2]
	if lib2.Cpp == nil {
		t.Fatalf("lib2 Cpp config is nil")
	}
	if len(lib2.Cpp.GenAsyncRPCs) != 1 || lib2.Cpp.GenAsyncRPCs[0] != "CreateFoo" {
		t.Errorf("lib2 GenAsyncRPCs mismatch: %+v", lib2.Cpp.GenAsyncRPCs)
	}

	// Check library 3: test-deprecated
	lib3 := cfg.Libraries[3]
	if lib3.Cpp == nil {
		t.Fatalf("lib3 Cpp config is nil")
	}
	if lib3.Cpp.GenerateGrpcTransport == nil || !*lib3.Cpp.GenerateGrpcTransport {
		t.Errorf("lib3 GenerateGrpcTransport: want true")
	}
}

func TestGoldenFixtures_ParseProtobuf(t *testing.T) {
	if _, err := exec.LookPath("protoc"); err != nil {
		t.Skip("skipping test because protoc is not installed")
	}

	protosDir, err := filepath.Abs(filepath.Join("testdata", "protos"))
	if err != nil {
		t.Fatal(err)
	}
	googleapisDir, err := filepath.Abs(filepath.Join("..", "..", "testdata", "googleapis"))
	if err != nil {
		t.Fatal(err)
	}

	t.Run("GoldenKitchenSink and GoldenThingAdmin", func(t *testing.T) {
		model, err := loadTestModel(t, protosDir, googleapisDir, "test.yaml", "test.proto", "backup.proto", "common.proto")
		if err != nil {
			t.Fatalf("failed to parse test protos into model: %v", err)
		}
		if model == nil {
			t.Fatal("expected non-nil model")
		}
		sinkSvc := model.Service(".google.test.admin.database.v1.GoldenKitchenSink")
		if sinkSvc == nil {
			t.Errorf("missing service .google.test.admin.database.v1.GoldenKitchenSink in model")
		}
		adminSvc := model.Service(".google.test.admin.database.v1.GoldenThingAdmin")
		if adminSvc == nil {
			t.Errorf("missing service .google.test.admin.database.v1.GoldenThingAdmin in model")
		}
	})

	t.Run("GoldenRestOnly", func(t *testing.T) {
		model, err := loadTestModel(t, protosDir, googleapisDir, "", "test2.proto")
		if err != nil {
			t.Fatalf("failed to parse test2.proto into model: %v", err)
		}
		if model == nil {
			t.Fatal("expected non-nil model")
		}
		restSvc := model.Service(".google.test.rest.only.v1.GoldenRestOnly")
		if restSvc == nil {
			t.Errorf("missing service .google.test.rest.only.v1.GoldenRestOnly in model")
		}
	})

	t.Run("RequestIdService", func(t *testing.T) {
		model, err := loadTestModel(t, protosDir, googleapisDir, "test_request_id.yaml", "test_request_id.proto")
		if err != nil {
			t.Fatalf("failed to parse test_request_id.proto into model: %v", err)
		}
		if model == nil {
			t.Fatal("expected non-nil model")
		}
		reqSvc := model.Service(".google.test.requestid.v1.RequestIdService")
		if reqSvc == nil {
			t.Errorf("missing service .google.test.requestid.v1.RequestIdService in model")
		}
	})

	t.Run("DeprecatedService", func(t *testing.T) {
		model, err := loadTestModel(t, protosDir, googleapisDir, "", "test_deprecated.proto")
		if err != nil {
			t.Fatalf("failed to parse test_deprecated.proto into model: %v", err)
		}
		if model == nil {
			t.Fatal("expected non-nil model")
		}
		depSvc := model.Service(".google.test.deprecated.v1.DeprecatedService")
		if depSvc == nil {
			t.Errorf("missing service .google.test.deprecated.v1.DeprecatedService in model")
		}
	})
}

// loadTestModel loads the test protos into an api.API model by setting up a hermetic
// source tree matching the generator/integration_tests directory structure.
func loadTestModel(t *testing.T, protosDir, googleapisDir, serviceConfigFile string, protoFiles ...string) (*api.API, error) {
	tempDir := t.TempDir()
	genDir := filepath.Join(tempDir, "generator", "integration_tests")
	if err := os.MkdirAll(genDir, 0o755); err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(protosDir)
	if err != nil {
		return nil, err
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		src := filepath.Join(protosDir, entry.Name())
		dst := filepath.Join(genDir, entry.Name())
		if err := os.Symlink(src, dst); err != nil {
			data, err := os.ReadFile(src)
			if err != nil {
				return nil, err
			}
			if err := os.WriteFile(dst, data, 0o644); err != nil {
				return nil, err
			}
		}
	}

	var svcConfig string
	if serviceConfigFile != "" {
		svcConfig = filepath.Join(genDir, serviceConfigFile)
	}

	cfg := &parser.ModelConfig{
		SpecificationFormat: config.SpecProtobuf,
		ServiceConfig:       svcConfig,
		SpecificationSource: "generator/integration_tests",
		Source: &sources.SourceConfig{
			Sources: &sources.Sources{
				Googleapis:  googleapisDir,
				Conformance: tempDir,
			},
			ActiveRoots: []string{"conformance", "googleapis"},
			IncludeList: protoFiles,
		},
	}
	return parser.CreateModel(cfg)
}
