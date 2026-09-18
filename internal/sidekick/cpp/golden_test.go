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
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
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

func TestGoldenServices_GenerateEmitsAllFiles(t *testing.T) {
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

	configPath := filepath.Join("testdata", "golden_librarian.yaml")
	cfg, err := yaml.Read[config.Config](configPath)
	if err != nil {
		t.Fatalf("failed to read golden_librarian.yaml: %v", err)
	}

	goldenRoot := t.TempDir()
	outdir := filepath.Join(goldenRoot, "v1")
	ctx := context.Background()

	// 1. GoldenKitchenSink and GoldenThingAdmin (Library 0: golden)
	sinkModel, err := loadTestModel(t, protosDir, googleapisDir, "test.yaml", "test.proto", "backup.proto", "common.proto")
	if err != nil {
		t.Fatalf("failed to parse test.proto: %v", err)
	}
	if err := Generate(ctx, sinkModel, outdir, cfg.Libraries[0]); err != nil {
		t.Fatalf("Generate(lib0) failed: %v", err)
	}

	// 2. GoldenRestOnly (Library 1: golden-test2)
	restModel, err := loadTestModel(t, protosDir, googleapisDir, "", "test2.proto")
	if err != nil {
		t.Fatalf("failed to parse test2.proto: %v", err)
	}
	if err := Generate(ctx, restModel, outdir, cfg.Libraries[1]); err != nil {
		t.Fatalf("Generate(lib1) failed: %v", err)
	}

	// 3. RequestIdService (Library 2: test-request-id)
	reqModel, err := loadTestModel(t, protosDir, googleapisDir, "test_request_id.yaml", "test_request_id.proto")
	if err != nil {
		t.Fatalf("failed to parse test_request_id.proto: %v", err)
	}
	if err := Generate(ctx, reqModel, outdir, cfg.Libraries[2]); err != nil {
		t.Fatalf("Generate(lib2) failed: %v", err)
	}

	// 4. DeprecatedService (Library 3: test-deprecated)
	depModel, err := loadTestModel(t, protosDir, googleapisDir, "", "test_deprecated.proto")
	if err != nil {
		t.Fatalf("failed to parse test_deprecated.proto: %v", err)
	}
	if err := Generate(ctx, depModel, outdir, cfg.Libraries[3]); err != nil {
		t.Fatalf("Generate(lib3) failed: %v", err)
	}

	// Collect generated relative paths
	var gotFiles []string
	err = filepath.WalkDir(goldenRoot, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			rel, err := filepath.Rel(goldenRoot, p)
			if err != nil {
				return err
			}
			gotFiles = append(gotFiles, rel)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walking goldenRoot: %v", err)
	}
	slices.Sort(gotFiles)

	// Collect expected relative paths from testdata/golden
	goldenDir := filepath.Join("testdata", "golden")
	var wantFiles []string
	err = filepath.WalkDir(goldenDir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			rel, err := filepath.Rel(goldenDir, p)
			if err != nil {
				return err
			}
			wantFiles = append(wantFiles, rel)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walking goldenDir: %v", err)
	}
	slices.Sort(wantFiles)

	if diff := cmp.Diff(wantFiles, gotFiles); diff != "" {
		t.Errorf("emitted files mismatch (-want +got):\n%s", diff)
	}
	if len(gotFiles) != 188 {
		t.Errorf("expected 188 files, got %d", len(gotFiles))
	}
}

func extractBlock(t *testing.T, content, startStr, endStr string) string {
	t.Helper()
	startIdx := strings.Index(content, startStr)
	if startIdx == -1 {
		t.Fatalf("missing expected block start %q\n\n%s", startStr, content)
	}
	endIdx := strings.Index(content[startIdx:], endStr)
	if endIdx == -1 {
		t.Fatalf("missing expected block end %q\n\n%s", endStr, content)
	}
	return content[startIdx : startIdx+endIdx+len(endStr)]
}

func TestGoldenServices_Layer32_IncludesAndGuards(t *testing.T) {
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

	configPath := filepath.Join("testdata", "golden_librarian.yaml")
	cfg, err := yaml.Read[config.Config](configPath)
	if err != nil {
		t.Fatalf("failed to read golden_librarian.yaml: %v", err)
	}

	goldenRoot := t.TempDir()
	outdir := filepath.Join(goldenRoot, "v1")
	ctx := context.Background()

	sinkModel, err := loadTestModel(t, protosDir, googleapisDir, "test.yaml", "test.proto", "backup.proto", "common.proto")
	if err != nil {
		t.Fatalf("failed to parse test.proto: %v", err)
	}
	if err := Generate(ctx, sinkModel, outdir, cfg.Libraries[0]); err != nil {
		t.Fatalf("Generate(lib0) failed: %v", err)
	}

	reqModel, err := loadTestModel(t, protosDir, googleapisDir, "test_request_id.yaml", "test_request_id.proto")
	if err != nil {
		t.Fatalf("failed to parse test_request_id.proto: %v", err)
	}
	if err := Generate(ctx, reqModel, outdir, cfg.Libraries[2]); err != nil {
		t.Fatalf("Generate(lib2) failed: %v", err)
	}

	readFile := func(path string) string {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("failed to read %s: %v", path, err)
		}
		return string(data)
	}

	goldenDir := filepath.Join("testdata", "golden")

	t.Run("Copyright and Banner", func(t *testing.T) {
		clientH := readFile(filepath.Join(goldenRoot, "v1", "golden_kitchen_sink_client.h"))
		gotCopyright := extractBlock(t, clientH, "// Copyright ", "Google LLC")
		if want := "// Copyright 2022 Google LLC"; gotCopyright != want {
			t.Errorf("copyright header mismatch: want %q, got %q", want, gotCopyright)
		}

		gotBanner := extractBlock(t, clientH, "// Generated by the Codegen C++ plugin.", "they will be lost.")
		wantBanner := "// Generated by the Codegen C++ plugin.\n// If you make any local changes, they will be lost."
		if gotBanner != wantBanner {
			t.Errorf("banner comment mismatch: want %q, got %q", wantBanner, gotBanner)
		}

		reqH := readFile(filepath.Join(goldenRoot, "v1", "request_id_client.h"))
		gotReqCopyright := extractBlock(t, reqH, "// Copyright ", "Google LLC")
		if want := "// Copyright 2024 Google LLC"; gotReqCopyright != want {
			t.Errorf("req copyright header mismatch: want %q, got %q", want, gotReqCopyright)
		}
	})

	t.Run("Header Guards", func(t *testing.T) {
		guards := []struct {
			path      string
			wantGuard string
		}{
			{
				path:      filepath.Join("v1", "golden_kitchen_sink_client.h"),
				wantGuard: "GOOGLE_CLOUD_CPP_GENERATOR_INTEGRATION_TESTS_GOLDEN_V1_GOLDEN_KITCHEN_SINK_CLIENT_H",
			},
			{
				path:      filepath.Join("v1", "golden_kitchen_sink_connection.h"),
				wantGuard: "GOOGLE_CLOUD_CPP_GENERATOR_INTEGRATION_TESTS_GOLDEN_V1_GOLDEN_KITCHEN_SINK_CONNECTION_H",
			},
			{
				path:      filepath.Join("v1", "golden_kitchen_sink_options.h"),
				wantGuard: "GOOGLE_CLOUD_CPP_GENERATOR_INTEGRATION_TESTS_GOLDEN_V1_GOLDEN_KITCHEN_SINK_OPTIONS_H",
			},
			{
				path:      filepath.Join("v1", "mocks", "mock_golden_kitchen_sink_connection.h"),
				wantGuard: "GOOGLE_CLOUD_CPP_GENERATOR_INTEGRATION_TESTS_GOLDEN_V1_MOCKS_MOCK_GOLDEN_KITCHEN_SINK_CONNECTION_H",
			},
			{
				path:      "golden_kitchen_sink_client.h",
				wantGuard: "GOOGLE_CLOUD_CPP_GENERATOR_INTEGRATION_TESTS_GOLDEN_GOLDEN_KITCHEN_SINK_CLIENT_H",
			},
			{
				path:      filepath.Join("mocks", "mock_golden_kitchen_sink_connection.h"),
				wantGuard: "GOOGLE_CLOUD_CPP_GENERATOR_INTEGRATION_TESTS_GOLDEN_MOCKS_MOCK_GOLDEN_KITCHEN_SINK_CONNECTION_H",
			},
			{
				path:      filepath.Join("v1", "internal", "golden_kitchen_sink_stub.h"),
				wantGuard: "GOOGLE_CLOUD_CPP_GENERATOR_INTEGRATION_TESTS_GOLDEN_V1_INTERNAL_GOLDEN_KITCHEN_SINK_STUB_H",
			},
			{
				path:      filepath.Join("v1", "internal", "golden_kitchen_sink_rest_stub.h"),
				wantGuard: "GOOGLE_CLOUD_CPP_GENERATOR_INTEGRATION_TESTS_GOLDEN_V1_INTERNAL_GOLDEN_KITCHEN_SINK_REST_STUB_H",
			},
		}

		for _, tc := range guards {
			content := readFile(filepath.Join(goldenRoot, tc.path))
			gotIfndef := extractBlock(t, content, "#ifndef ", "\n")
			wantIfndef := "#ifndef " + tc.wantGuard + "\n"
			if gotIfndef != wantIfndef {
				t.Errorf("%s #ifndef mismatch: want %q, got %q", tc.path, wantIfndef, gotIfndef)
			}

			gotDefine := extractBlock(t, content, "#define ", "\n")
			wantDefine := "#define " + tc.wantGuard + "\n"
			if gotDefine != wantDefine {
				t.Errorf("%s #define mismatch: want %q, got %q", tc.path, wantDefine, gotDefine)
			}

			gotEndif := extractBlock(t, content, "#endif", "\n")
			wantEndif := "#endif  // " + tc.wantGuard + "\n"
			if gotEndif != wantEndif {
				t.Errorf("%s #endif mismatch: want %q, got %q", tc.path, wantEndif, gotEndif)
			}
		}
	})

	t.Run("Matching Header Included First in CC", func(t *testing.T) {
		sources := []struct {
			ccPath     string
			wantHeader string
		}{
			{
				ccPath:     filepath.Join("v1", "golden_kitchen_sink_client.cc"),
				wantHeader: "generator/integration_tests/golden/v1/golden_kitchen_sink_client.h",
			},
			{
				ccPath:     filepath.Join("v1", "golden_kitchen_sink_connection.cc"),
				wantHeader: "generator/integration_tests/golden/v1/golden_kitchen_sink_connection.h",
			},
			{
				ccPath:     filepath.Join("v1", "golden_kitchen_sink_connection_idempotency_policy.cc"),
				wantHeader: "generator/integration_tests/golden/v1/golden_kitchen_sink_connection_idempotency_policy.h",
			},
			{
				ccPath:     filepath.Join("v1", "golden_kitchen_sink_rest_connection.cc"),
				wantHeader: "generator/integration_tests/golden/v1/golden_kitchen_sink_rest_connection.h",
			},
			{
				ccPath:     filepath.Join("v1", "internal", "golden_kitchen_sink_stub.cc"),
				wantHeader: "generator/integration_tests/golden/v1/internal/golden_kitchen_sink_stub.h",
			},
			{
				ccPath:     filepath.Join("v1", "internal", "golden_kitchen_sink_connection_impl.cc"),
				wantHeader: "generator/integration_tests/golden/v1/internal/golden_kitchen_sink_connection_impl.h",
			},
			{
				ccPath:     filepath.Join("v1", "internal", "golden_kitchen_sink_rest_stub.cc"),
				wantHeader: "generator/integration_tests/golden/v1/internal/golden_kitchen_sink_rest_stub.h",
			},
		}

		for _, tc := range sources {
			content := readFile(filepath.Join(goldenRoot, tc.ccPath))
			gotFirstInclude := extractBlock(t, content, `#include "`, `"`+"\n")
			wantFirstInclude := `#include "` + tc.wantHeader + `"` + "\n"
			if gotFirstInclude != wantFirstInclude {
				t.Errorf("%s first include mismatch: want %q, got %q", tc.ccPath, wantFirstInclude, gotFirstInclude)
			}
		}
	})

	t.Run("Sources CC Includes Block Parity", func(t *testing.T) {
		gotSources := readFile(filepath.Join(goldenRoot, "v1", "internal", "golden_kitchen_sink_sources.cc"))
		wantSources := readFile(filepath.Join(goldenDir, "v1", "internal", "golden_kitchen_sink_sources.cc"))

		const startStr = "// NOLINTBEGIN(bugprone-suspicious-include)\n"
		const endStr = "// NOLINTEND(bugprone-suspicious-include)\n"

		gotBlock := extractBlock(t, gotSources, startStr, endStr)
		wantBlock := extractBlock(t, wantSources, startStr, endStr)

		if diff := cmp.Diff(wantBlock, gotBlock); diff != "" {
			t.Errorf("sources.cc include block mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("Forwarding Header Includes Parity", func(t *testing.T) {
		forwardingHeaders := []string{
			"golden_kitchen_sink_client.h",
			"golden_kitchen_sink_options.h",
			filepath.Join("mocks", "mock_golden_kitchen_sink_connection.h"),
		}

		for _, rel := range forwardingHeaders {
			gotContent := readFile(filepath.Join(goldenRoot, rel))
			wantContent := readFile(filepath.Join(goldenDir, rel))

			gotBlock := extractBlock(t, gotContent, `#include "`, `"`+"\n\n")
			wantBlock := extractBlock(t, wantContent, `#include "`, `"`+"\n\n")

			if diff := cmp.Diff(wantBlock, gotBlock); diff != "" {
				t.Errorf("%s forwarding includes mismatch (-want +got):\n%s", rel, diff)
			}
		}
	})
}

func TestGoldenServices_Layer33_Namespaces(t *testing.T) {
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

	configPath := filepath.Join("testdata", "golden_librarian.yaml")
	cfg, err := yaml.Read[config.Config](configPath)
	if err != nil {
		t.Fatalf("failed to read golden_librarian.yaml: %v", err)
	}

	goldenRoot := t.TempDir()
	outdir := filepath.Join(goldenRoot, "v1")
	ctx := context.Background()

	sinkModel, err := loadTestModel(t, protosDir, googleapisDir, "test.yaml", "test.proto", "backup.proto", "common.proto")
	if err != nil {
		t.Fatalf("failed to parse test.proto: %v", err)
	}
	if err := Generate(ctx, sinkModel, outdir, cfg.Libraries[0]); err != nil {
		t.Fatalf("Generate(lib0) failed: %v", err)
	}

	reqModel, err := loadTestModel(t, protosDir, googleapisDir, "test_request_id.yaml", "test_request_id.proto")
	if err != nil {
		t.Fatalf("failed to parse test_request_id.proto: %v", err)
	}
	if err := Generate(ctx, reqModel, outdir, cfg.Libraries[2]); err != nil {
		t.Fatalf("Generate(lib2) failed: %v", err)
	}

	readFile := func(path string) string {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("failed to read %s: %v", path, err)
		}
		return string(data)
	}

	goldenDir := filepath.Join("testdata", "golden")

	t.Run("Standard Public Namespaces", func(t *testing.T) {
		publicFiles := []string{
			filepath.Join("v1", "golden_kitchen_sink_client.h"),
			filepath.Join("v1", "golden_kitchen_sink_client.cc"),
			filepath.Join("v1", "golden_kitchen_sink_connection.h"),
			filepath.Join("v1", "golden_kitchen_sink_connection.cc"),
			filepath.Join("v1", "golden_kitchen_sink_connection_idempotency_policy.h"),
			filepath.Join("v1", "golden_kitchen_sink_connection_idempotency_policy.cc"),
			filepath.Join("v1", "golden_kitchen_sink_options.h"),
			filepath.Join("v1", "golden_kitchen_sink_rest_connection.h"),
			filepath.Join("v1", "golden_kitchen_sink_rest_connection.cc"),
			filepath.Join("v1", "request_id_client.h"),
		}

		for _, rel := range publicFiles {
			gotContent := readFile(filepath.Join(goldenRoot, rel))
			wantContent := readFile(filepath.Join(goldenDir, rel))

			gotOpen := extractBlock(t, gotContent, "namespace google {\nnamespace cloud {\n", "GOOGLE_CLOUD_CPP_INLINE_NAMESPACE_BEGIN\n")
			wantOpen := extractBlock(t, wantContent, "namespace google {\nnamespace cloud {\n", "GOOGLE_CLOUD_CPP_INLINE_NAMESPACE_BEGIN\n")
			if diff := cmp.Diff(wantOpen, gotOpen); diff != "" {
				t.Errorf("%s namespace opening mismatch (-want +got):\n%s", rel, diff)
			}

			gotClose := extractBlock(t, gotContent, "GOOGLE_CLOUD_CPP_INLINE_NAMESPACE_END\n", "}  // namespace google\n")
			wantClose := extractBlock(t, wantContent, "GOOGLE_CLOUD_CPP_INLINE_NAMESPACE_END\n", "}  // namespace google\n")
			if diff := cmp.Diff(wantClose, gotClose); diff != "" {
				t.Errorf("%s namespace closing mismatch (-want +got):\n%s", rel, diff)
			}
		}
	})

	t.Run("Internal Namespaces", func(t *testing.T) {
		internalFiles := []string{
			filepath.Join("v1", "internal", "golden_kitchen_sink_auth_decorator.h"),
			filepath.Join("v1", "internal", "golden_kitchen_sink_auth_decorator.cc"),
			filepath.Join("v1", "internal", "golden_kitchen_sink_connection_impl.h"),
			filepath.Join("v1", "internal", "golden_kitchen_sink_connection_impl.cc"),
			filepath.Join("v1", "internal", "golden_kitchen_sink_logging_decorator.h"),
			filepath.Join("v1", "internal", "golden_kitchen_sink_logging_decorator.cc"),
			filepath.Join("v1", "internal", "golden_kitchen_sink_metadata_decorator.h"),
			filepath.Join("v1", "internal", "golden_kitchen_sink_metadata_decorator.cc"),
			filepath.Join("v1", "internal", "golden_kitchen_sink_option_defaults.h"),
			filepath.Join("v1", "internal", "golden_kitchen_sink_option_defaults.cc"),
			filepath.Join("v1", "internal", "golden_kitchen_sink_rest_connection_impl.h"),
			filepath.Join("v1", "internal", "golden_kitchen_sink_rest_connection_impl.cc"),
			filepath.Join("v1", "internal", "golden_kitchen_sink_rest_logging_decorator.h"),
			filepath.Join("v1", "internal", "golden_kitchen_sink_rest_logging_decorator.cc"),
			filepath.Join("v1", "internal", "golden_kitchen_sink_rest_metadata_decorator.h"),
			filepath.Join("v1", "internal", "golden_kitchen_sink_rest_metadata_decorator.cc"),
			filepath.Join("v1", "internal", "golden_kitchen_sink_rest_stub.h"),
			filepath.Join("v1", "internal", "golden_kitchen_sink_rest_stub.cc"),
			filepath.Join("v1", "internal", "golden_kitchen_sink_rest_stub_factory.h"),
			filepath.Join("v1", "internal", "golden_kitchen_sink_rest_stub_factory.cc"),
			filepath.Join("v1", "internal", "golden_kitchen_sink_retry_traits.h"),
			filepath.Join("v1", "internal", "golden_kitchen_sink_round_robin_decorator.h"),
			filepath.Join("v1", "internal", "golden_kitchen_sink_round_robin_decorator.cc"),
			filepath.Join("v1", "internal", "golden_kitchen_sink_stub.h"),
			filepath.Join("v1", "internal", "golden_kitchen_sink_stub.cc"),
			filepath.Join("v1", "internal", "golden_kitchen_sink_stub_factory.h"),
			filepath.Join("v1", "internal", "golden_kitchen_sink_stub_factory.cc"),
			filepath.Join("v1", "internal", "golden_kitchen_sink_tracing_connection.h"),
			filepath.Join("v1", "internal", "golden_kitchen_sink_tracing_connection.cc"),
			filepath.Join("v1", "internal", "golden_kitchen_sink_tracing_stub.h"),
			filepath.Join("v1", "internal", "golden_kitchen_sink_tracing_stub.cc"),
		}

		for _, rel := range internalFiles {
			gotContent := readFile(filepath.Join(goldenRoot, rel))
			wantContent := readFile(filepath.Join(goldenDir, rel))

			gotOpen := extractBlock(t, gotContent, "namespace google {\nnamespace cloud {\n", "GOOGLE_CLOUD_CPP_INLINE_NAMESPACE_BEGIN\n")
			wantOpen := extractBlock(t, wantContent, "namespace google {\nnamespace cloud {\n", "GOOGLE_CLOUD_CPP_INLINE_NAMESPACE_BEGIN\n")
			if diff := cmp.Diff(wantOpen, gotOpen); diff != "" {
				t.Errorf("%s namespace opening mismatch (-want +got):\n%s", rel, diff)
			}

			gotClose := extractBlock(t, gotContent, "GOOGLE_CLOUD_CPP_INLINE_NAMESPACE_END\n", "}  // namespace google\n")
			wantClose := extractBlock(t, wantContent, "GOOGLE_CLOUD_CPP_INLINE_NAMESPACE_END\n", "}  // namespace google\n")
			if diff := cmp.Diff(wantClose, gotClose); diff != "" {
				t.Errorf("%s namespace closing mismatch (-want +got):\n%s", rel, diff)
			}
		}
	})

	t.Run("Mock Namespaces", func(t *testing.T) {
		mockFiles := []string{
			filepath.Join("v1", "mocks", "mock_golden_kitchen_sink_connection.h"),
		}

		for _, rel := range mockFiles {
			gotContent := readFile(filepath.Join(goldenRoot, rel))
			wantContent := readFile(filepath.Join(goldenDir, rel))

			gotOpen := extractBlock(t, gotContent, "namespace google {\nnamespace cloud {\n", "GOOGLE_CLOUD_CPP_INLINE_NAMESPACE_BEGIN\n")
			wantOpen := extractBlock(t, wantContent, "namespace google {\nnamespace cloud {\n", "GOOGLE_CLOUD_CPP_INLINE_NAMESPACE_BEGIN\n")
			if diff := cmp.Diff(wantOpen, gotOpen); diff != "" {
				t.Errorf("%s namespace opening mismatch (-want +got):\n%s", rel, diff)
			}

			gotClose := extractBlock(t, gotContent, "GOOGLE_CLOUD_CPP_INLINE_NAMESPACE_END\n", "}  // namespace google\n")
			wantClose := extractBlock(t, wantContent, "GOOGLE_CLOUD_CPP_INLINE_NAMESPACE_END\n", "}  // namespace google\n")
			if diff := cmp.Diff(wantClose, gotClose); diff != "" {
				t.Errorf("%s namespace closing mismatch (-want +got):\n%s", rel, diff)
			}
		}
	})

	t.Run("Forwarding Client Namespaces", func(t *testing.T) {
		rel := "golden_kitchen_sink_client.h"
		gotContent := readFile(filepath.Join(goldenRoot, rel))
		wantContent := readFile(filepath.Join(goldenDir, rel))

		gotOpen := extractBlock(t, gotContent, "namespace google {\nnamespace cloud {\n", "GOOGLE_CLOUD_CPP_INLINE_NAMESPACE_BEGIN\n")
		wantOpen := extractBlock(t, wantContent, "namespace google {\nnamespace cloud {\n", "GOOGLE_CLOUD_CPP_INLINE_NAMESPACE_BEGIN\n")
		if diff := cmp.Diff(wantOpen, gotOpen); diff != "" {
			t.Errorf("%s forwarding client namespace opening mismatch (-want +got):\n%s", rel, diff)
		}

		gotClose := extractBlock(t, gotContent, "GOOGLE_CLOUD_CPP_INLINE_NAMESPACE_END\n", "}  // namespace google\n")
		wantClose := extractBlock(t, wantContent, "GOOGLE_CLOUD_CPP_INLINE_NAMESPACE_END\n", "}  // namespace google\n")
		if diff := cmp.Diff(wantClose, gotClose); diff != "" {
			t.Errorf("%s forwarding client namespace closing mismatch (-want +got):\n%s", rel, diff)
		}
	})

	t.Run("Forwarding Mock Namespaces", func(t *testing.T) {
		rel := filepath.Join("mocks", "mock_golden_kitchen_sink_connection.h")
		gotContent := readFile(filepath.Join(goldenRoot, rel))
		wantContent := readFile(filepath.Join(goldenDir, rel))

		gotOpen := extractBlock(t, gotContent, "namespace google {\nnamespace cloud {\n", "GOOGLE_CLOUD_CPP_INLINE_NAMESPACE_BEGIN\n")
		wantOpen := extractBlock(t, wantContent, "namespace google {\nnamespace cloud {\n", "GOOGLE_CLOUD_CPP_INLINE_NAMESPACE_BEGIN\n")
		if diff := cmp.Diff(wantOpen, gotOpen); diff != "" {
			t.Errorf("%s forwarding mock namespace opening mismatch (-want +got):\n%s", rel, diff)
		}

		gotClose := extractBlock(t, gotContent, "GOOGLE_CLOUD_CPP_INLINE_NAMESPACE_END\n", "}  // namespace google\n")
		wantClose := extractBlock(t, wantContent, "GOOGLE_CLOUD_CPP_INLINE_NAMESPACE_END\n", "}  // namespace google\n")
		if diff := cmp.Diff(wantClose, gotClose); diff != "" {
			t.Errorf("%s forwarding mock namespace closing mismatch (-want +got):\n%s", rel, diff)
		}
	})

	t.Run("Forwarding Connection Namespaces", func(t *testing.T) {
		forwardingFiles := []string{
			"golden_kitchen_sink_connection.h",
			"golden_kitchen_sink_connection_idempotency_policy.h",
			"golden_kitchen_sink_options.h",
		}

		for _, rel := range forwardingFiles {
			gotContent := readFile(filepath.Join(goldenRoot, rel))
			wantContent := readFile(filepath.Join(goldenDir, rel))

			gotOpen := extractBlock(t, gotContent, "namespace google {\nnamespace cloud {\n", "GOOGLE_CLOUD_CPP_INLINE_NAMESPACE_BEGIN\n")
			wantOpen := extractBlock(t, wantContent, "namespace google {\nnamespace cloud {\n", "GOOGLE_CLOUD_CPP_INLINE_NAMESPACE_BEGIN\n")
			if diff := cmp.Diff(wantOpen, gotOpen); diff != "" {
				t.Errorf("%s forwarding namespace opening mismatch (-want +got):\n%s", rel, diff)
			}

			gotClose := extractBlock(t, gotContent, "GOOGLE_CLOUD_CPP_INLINE_NAMESPACE_END\n", "}  // namespace google\n")
			wantClose := extractBlock(t, wantContent, "GOOGLE_CLOUD_CPP_INLINE_NAMESPACE_END\n", "}  // namespace google\n")
			if diff := cmp.Diff(wantClose, gotClose); diff != "" {
				t.Errorf("%s forwarding namespace closing mismatch (-want +got):\n%s", rel, diff)
			}
		}
	})

	t.Run("Sources CC Has No Namespaces", func(t *testing.T) {
		rel := filepath.Join("v1", "internal", "golden_kitchen_sink_sources.cc")
		gotContent := readFile(filepath.Join(goldenRoot, rel))
		wantContent := readFile(filepath.Join(goldenDir, rel))

		if strings.Contains(gotContent, "namespace") {
			t.Errorf("%s unexpectedly contains 'namespace'", rel)
		}
		if strings.Contains(wantContent, "namespace") {
			t.Errorf("%s (golden reference) unexpectedly contains 'namespace'", rel)
		}
	})
}
