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
		if !d.IsDir() && !strings.HasPrefix(d.Name(), ".") {
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
		if !d.IsDir() && !strings.HasPrefix(d.Name(), ".") {
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
		if !d.IsDir() && !strings.HasPrefix(d.Name(), ".") {
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
			const disableWarnings = "#include \"google/cloud/internal/disable_deprecation_warnings.inc\"\n"
			if idx := strings.Index(content, disableWarnings); idx != -1 {
				content = content[idx+len(disableWarnings):]
			}
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

func TestGoldenServices_Layer34_ClassSkeletons(t *testing.T) {
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

	readFile := func(path string) string {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("failed to read %s: %v", path, err)
		}
		return string(data)
	}

	goldenDir := filepath.Join("testdata", "golden")

	compareBlock := func(t *testing.T, rel, startStr, endStr string) {
		t.Helper()
		gotContent := readFile(filepath.Join(goldenRoot, rel))
		wantContent := readFile(filepath.Join(goldenDir, rel))

		gotBlock := extractBlock(t, gotContent, startStr, endStr)
		wantBlock := extractBlock(t, wantContent, startStr, endStr)
		if diff := cmp.Diff(wantBlock, gotBlock); diff != "" {
			t.Errorf("%s block [%q ... %q] mismatch (-want +got):\n%s", rel, startStr, endStr, diff)
		}
	}

	t.Run("Client Skeletons", func(t *testing.T) {
		// GoldenKitchenSink Client class declaration and copy/move/equality operators
		compareBlock(t, filepath.Join("v1", "golden_kitchen_sink_client.h"),
			"class GoldenKitchenSinkClient {\n",
			"  ///@}\n")
		// GoldenKitchenSink Client private members
		compareBlock(t, filepath.Join("v1", "golden_kitchen_sink_client.h"),
			" private:\n  std::shared_ptr<GoldenKitchenSinkConnection> connection_;\n",
			"  Options options_;\n};\n")
		// GoldenKitchenSink Client CC constructor and destructor
		compareBlock(t, filepath.Join("v1", "golden_kitchen_sink_client.cc"),
			"GoldenKitchenSinkClient::GoldenKitchenSinkClient(\n",
			"GoldenKitchenSinkClient::~GoldenKitchenSinkClient() = default;\n")

		// GoldenThingAdmin Client constructor
		compareBlock(t, filepath.Join("v1", "golden_thing_admin_client.h"),
			"class GoldenThingAdminClient {\n",
			"  explicit GoldenThingAdminClient(std::shared_ptr<GoldenThingAdminConnection> connection, Options opts = {});\n")

		// GoldenRestOnly Client constructor
		compareBlock(t, filepath.Join("v1", "golden_rest_only_client.h"),
			"class GoldenRestOnlyClient {\n",
			"  explicit GoldenRestOnlyClient(std::shared_ptr<GoldenRestOnlyConnection> connection, Options opts = {});\n")

		// RequestIdService Client constructor
		compareBlock(t, filepath.Join("v1", "request_id_client.h"),
			"class RequestIdServiceClient {\n",
			"  explicit RequestIdServiceClient(std::shared_ptr<RequestIdServiceConnection> connection, Options opts = {});\n")

		// DeprecatedService Client deprecation annotation
		compareBlock(t, filepath.Join("v1", "deprecated_client.h"),
			"class\n GOOGLE_CLOUD_CPP_DEPRECATED(\n",
			"DeprecatedServiceClient {\n")
	})

	t.Run("Connection Skeletons", func(t *testing.T) {
		// RetryPolicy interface
		compareBlock(t, filepath.Join("v1", "golden_kitchen_sink_connection.h"),
			"class GoldenKitchenSinkRetryPolicy : public ::google::cloud::RetryPolicy {\n",
			"  virtual std::unique_ptr<GoldenKitchenSinkRetryPolicy> clone() const = 0;\n};\n")

		// LimitedErrorCountRetryPolicy class, constructor, BaseType, impl
		compareBlock(t, filepath.Join("v1", "golden_kitchen_sink_connection.h"),
			"class GoldenKitchenSinkLimitedErrorCountRetryPolicy : public GoldenKitchenSinkRetryPolicy {\n",
			"class GoldenKitchenSinkLimitedErrorCountRetryPolicy : public GoldenKitchenSinkRetryPolicy {\n")
		compareBlock(t, filepath.Join("v1", "golden_kitchen_sink_connection.h"),
			"  explicit GoldenKitchenSinkLimitedErrorCountRetryPolicy(int maximum_failures)\n",
			"    : impl_(maximum_failures) {}\n")
		compareBlock(t, filepath.Join("v1", "golden_kitchen_sink_connection.h"),
			"  using BaseType = GoldenKitchenSinkRetryPolicy;\n",
			"  using BaseType = GoldenKitchenSinkRetryPolicy;\n")
		compareBlock(t, filepath.Join("v1", "golden_kitchen_sink_connection.h"),
			"  google::cloud::internal::LimitedErrorCountRetryPolicy<golden_v1_internal::GoldenKitchenSinkRetryTraits> impl_;\n",
			"  google::cloud::internal::LimitedErrorCountRetryPolicy<golden_v1_internal::GoldenKitchenSinkRetryTraits> impl_;\n")

		// LimitedTimeRetryPolicy class, constructor, BaseType, impl
		compareBlock(t, filepath.Join("v1", "golden_kitchen_sink_connection.h"),
			"class GoldenKitchenSinkLimitedTimeRetryPolicy : public GoldenKitchenSinkRetryPolicy {\n",
			"class GoldenKitchenSinkLimitedTimeRetryPolicy : public GoldenKitchenSinkRetryPolicy {\n")
		compareBlock(t, filepath.Join("v1", "golden_kitchen_sink_connection.h"),
			"  template <typename DurationRep, typename DurationPeriod>\n  explicit GoldenKitchenSinkLimitedTimeRetryPolicy(\n",
			"    : impl_(maximum_duration) {}\n")
		compareBlock(t, filepath.Join("v1", "golden_kitchen_sink_connection.h"),
			"  using BaseType = GoldenKitchenSinkRetryPolicy;\n",
			"  using BaseType = GoldenKitchenSinkRetryPolicy;\n")
		compareBlock(t, filepath.Join("v1", "golden_kitchen_sink_connection.h"),
			"  google::cloud::internal::LimitedTimeRetryPolicy<golden_v1_internal::GoldenKitchenSinkRetryTraits> impl_;\n",
			"  google::cloud::internal::LimitedTimeRetryPolicy<golden_v1_internal::GoldenKitchenSinkRetryTraits> impl_;\n")

		// Connection class pure virtual dtor and options
		compareBlock(t, filepath.Join("v1", "golden_kitchen_sink_connection.h"),
			"class GoldenKitchenSinkConnection {\n public:\n",
			"  virtual Options options() { return Options{}; }\n")

		// MakeConnection factory declaration
		compareBlock(t, filepath.Join("v1", "golden_kitchen_sink_connection.h"),
			"std::shared_ptr<GoldenKitchenSinkConnection> MakeGoldenKitchenSinkConnection(\n",
			"    Options options = {});\n")

		// Connection CC dtor and MakeConnection factory definition
		compareBlock(t, filepath.Join("v1", "golden_kitchen_sink_connection.cc"),
			"GoldenKitchenSinkConnection::~GoldenKitchenSinkConnection() = default;\n",
			"GoldenKitchenSinkConnection::~GoldenKitchenSinkConnection() = default;\n")
		compareBlock(t, filepath.Join("v1", "golden_kitchen_sink_connection.cc"),
			"std::shared_ptr<GoldenKitchenSinkConnection> MakeGoldenKitchenSinkConnection(\n",
			"      std::move(background), std::move(stub), std::move(options)));\n}\n")
	})

	t.Run("Connection Idempotency Policy Skeletons", func(t *testing.T) {
		compareBlock(t, filepath.Join("v1", "golden_kitchen_sink_connection_idempotency_policy.h"),
			"class GoldenKitchenSinkConnectionIdempotencyPolicy {\n public:\n",
			"  virtual std::unique_ptr<GoldenKitchenSinkConnectionIdempotencyPolicy> clone() const;\n")
		compareBlock(t, filepath.Join("v1", "golden_kitchen_sink_connection_idempotency_policy.h"),
			"std::unique_ptr<GoldenKitchenSinkConnectionIdempotencyPolicy>\n",
			"    MakeDefaultGoldenKitchenSinkConnectionIdempotencyPolicy();\n")

		compareBlock(t, filepath.Join("v1", "golden_kitchen_sink_connection_idempotency_policy.cc"),
			"GoldenKitchenSinkConnectionIdempotencyPolicy::~GoldenKitchenSinkConnectionIdempotencyPolicy() = default;\n",
			"  return std::make_unique<GoldenKitchenSinkConnectionIdempotencyPolicy>(*this);\n}\n")
		compareBlock(t, filepath.Join("v1", "golden_kitchen_sink_connection_idempotency_policy.cc"),
			"std::unique_ptr<GoldenKitchenSinkConnectionIdempotencyPolicy>\n    MakeDefaultGoldenKitchenSinkConnectionIdempotencyPolicy() {\n",
			"  return std::make_unique<GoldenKitchenSinkConnectionIdempotencyPolicy>();\n}\n")
	})

	t.Run("Options Skeletons", func(t *testing.T) {
		compareBlock(t, filepath.Join("v1", "golden_kitchen_sink_options.h"),
			"struct GoldenKitchenSinkRetryPolicyOption {\n",
			"  using Type = std::shared_ptr<GoldenKitchenSinkRetryPolicy>;\n};\n")
		compareBlock(t, filepath.Join("v1", "golden_kitchen_sink_options.h"),
			"struct GoldenKitchenSinkBackoffPolicyOption {\n",
			"  using Type = std::shared_ptr<BackoffPolicy>;\n};\n")
		compareBlock(t, filepath.Join("v1", "golden_kitchen_sink_options.h"),
			"struct GoldenKitchenSinkConnectionIdempotencyPolicyOption {\n",
			"  using Type = std::shared_ptr<GoldenKitchenSinkConnectionIdempotencyPolicy>;\n};\n")
		compareBlock(t, filepath.Join("v1", "golden_kitchen_sink_options.h"),
			"using GoldenKitchenSinkPolicyOptionList =\n",
			"               GoldenKitchenSinkConnectionIdempotencyPolicyOption>;\n")
	})

	t.Run("Mock Skeletons", func(t *testing.T) {
		compareBlock(t, filepath.Join("v1", "mocks", "mock_golden_kitchen_sink_connection.h"),
			"class MockGoldenKitchenSinkConnection : public golden_v1::GoldenKitchenSinkConnection {\n",
			"  MOCK_METHOD(Options, options, (), (override));\n")

		compareBlock(t, filepath.Join("v1", "mocks", "mock_golden_thing_admin_connection.h"),
			"class MockGoldenThingAdminConnection : public golden_v1::GoldenThingAdminConnection {\n",
			"  MOCK_METHOD(Options, options, (), (override));\n")
	})

	t.Run("Internal Decorators and Stubs (gRPC)", func(t *testing.T) {
		// RetryTraits
		compareBlock(t, filepath.Join("v1", "internal", "golden_kitchen_sink_retry_traits.h"),
			"struct GoldenKitchenSinkRetryTraits {\n",
			"  }\n};\n")
		compareBlock(t, filepath.Join("v1", "internal", "golden_thing_admin_retry_traits.h"),
			"struct GoldenThingAdminRetryTraits {\n",
			"  }\n};\n")

		// Stub & DefaultStub
		compareBlock(t, filepath.Join("v1", "internal", "golden_kitchen_sink_stub.h"),
			"class GoldenKitchenSinkStub {\n public:\n",
			"  virtual ~GoldenKitchenSinkStub() = 0;\n")
		compareBlock(t, filepath.Join("v1", "internal", "golden_kitchen_sink_stub.h"),
			"class DefaultGoldenKitchenSinkStub : public GoldenKitchenSinkStub {\n",
			"class DefaultGoldenKitchenSinkStub : public GoldenKitchenSinkStub {\n")
		compareBlock(t, filepath.Join("v1", "internal", "deprecated_stub.h"),
			"class DefaultDeprecatedServiceStub : public DeprecatedServiceStub {\n",
			": grpc_stub_(std::move(grpc_stub)) {}\n")
		compareBlock(t, filepath.Join("v1", "internal", "deprecated_stub.h"),
			" private:\n  std::unique_ptr<google::test::deprecated::v1::DeprecatedService::StubInterface> grpc_stub_;\n};\n",
			" private:\n  std::unique_ptr<google::test::deprecated::v1::DeprecatedService::StubInterface> grpc_stub_;\n};\n")
		compareBlock(t, filepath.Join("v1", "internal", "golden_kitchen_sink_stub.cc"),
			"GoldenKitchenSinkStub::~GoldenKitchenSinkStub() = default;\n",
			"GoldenKitchenSinkStub::~GoldenKitchenSinkStub() = default;\n")

		// Auth decorator
		compareBlock(t, filepath.Join("v1", "internal", "golden_kitchen_sink_auth_decorator.h"),
			"class GoldenKitchenSinkAuth : public GoldenKitchenSinkStub {\n",
			"      std::shared_ptr<GoldenKitchenSinkStub> child);\n")
		compareBlock(t, filepath.Join("v1", "internal", "golden_kitchen_sink_auth_decorator.h"),
			" private:\n  std::shared_ptr<google::cloud::internal::GrpcAuthenticationStrategy> auth_;\n",
			"  std::shared_ptr<GoldenKitchenSinkStub> child_;\n};\n")
		compareBlock(t, filepath.Join("v1", "internal", "golden_kitchen_sink_auth_decorator.cc"),
			"GoldenKitchenSinkAuth::GoldenKitchenSinkAuth(\n",
			": auth_(std::move(auth)), child_(std::move(child)) {}\n")

		// Logging decorator
		compareBlock(t, filepath.Join("v1", "internal", "golden_kitchen_sink_logging_decorator.h"),
			"class GoldenKitchenSinkLogging : public GoldenKitchenSinkStub {\n",
			"                       std::set<std::string> const& components);\n")
		compareBlock(t, filepath.Join("v1", "internal", "golden_kitchen_sink_logging_decorator.h"),
			" private:\n  std::shared_ptr<GoldenKitchenSinkStub> child_;\n",
			"  bool stream_logging_;\n};  // GoldenKitchenSinkLogging\n")
		compareBlock(t, filepath.Join("v1", "internal", "golden_kitchen_sink_logging_decorator.cc"),
			"GoldenKitchenSinkLogging::GoldenKitchenSinkLogging(\n",
			"stream_logging_(components.find(\"rpc-streams\") != components.end()) {}\n")

		// Metadata decorator
		compareBlock(t, filepath.Join("v1", "internal", "golden_kitchen_sink_metadata_decorator.h"),
			"class GoldenKitchenSinkMetadata : public GoldenKitchenSinkStub {\n",
			"      std::string api_client_header = \"\");\n")
		compareBlock(t, filepath.Join("v1", "internal", "golden_kitchen_sink_metadata_decorator.h"),
			" private:\n  void SetMetadata(grpc::ClientContext& context,\n",
			"  std::string api_client_header_;\n};\n")
		compareBlock(t, filepath.Join("v1", "internal", "golden_kitchen_sink_metadata_decorator.cc"),
			"GoldenKitchenSinkMetadata::GoldenKitchenSinkMetadata(\n",
			"              : std::move(api_client_header)) {}\n")

		// Round robin decorator
		compareBlock(t, filepath.Join("v1", "internal", "golden_kitchen_sink_round_robin_decorator.h"),
			"class GoldenKitchenSinkRoundRobin : public GoldenKitchenSinkStub {\n",
			"      std::vector<std::shared_ptr<GoldenKitchenSinkStub>> children);\n")
		compareBlock(t, filepath.Join("v1", "internal", "golden_kitchen_sink_round_robin_decorator.h"),
			"  std::vector<std::shared_ptr<GoldenKitchenSinkStub>> const children_;\n",
			"  std::size_t current_ = 0;\n};\n")
		compareBlock(t, filepath.Join("v1", "internal", "golden_kitchen_sink_round_robin_decorator.cc"),
			"GoldenKitchenSinkRoundRobin::GoldenKitchenSinkRoundRobin(\n",
			": children_(std::move(children)) {}\n")

		// Tracing stub
		compareBlock(t, filepath.Join("v1", "internal", "golden_kitchen_sink_tracing_stub.h"),
			"class GoldenKitchenSinkTracingStub : public GoldenKitchenSinkStub {\n",
			"  explicit GoldenKitchenSinkTracingStub(std::shared_ptr<GoldenKitchenSinkStub> child);\n")
		compareBlock(t, filepath.Join("v1", "internal", "golden_kitchen_sink_tracing_stub.h"),
			" private:\n  std::shared_ptr<GoldenKitchenSinkStub> child_;\n",
			"  std::shared_ptr<opentelemetry::context::propagation::TextMapPropagator> propagator_;\n};\n")
		compareBlock(t, filepath.Join("v1", "internal", "golden_kitchen_sink_tracing_stub.h"),
			"std::shared_ptr<GoldenKitchenSinkStub> MakeGoldenKitchenSinkTracingStub(\n",
			"    std::shared_ptr<GoldenKitchenSinkStub> stub);\n")
		compareBlock(t, filepath.Join("v1", "internal", "golden_kitchen_sink_tracing_stub.cc"),
			"GoldenKitchenSinkTracingStub::GoldenKitchenSinkTracingStub(\n",
			": child_(std::move(child)), propagator_(internal::MakePropagator()) {}\n")

		// Connection Impl
		compareBlock(t, filepath.Join("v1", "internal", "golden_kitchen_sink_connection_impl.h"),
			"class GoldenKitchenSinkConnectionImpl\n",
			"    Options options);\n")
		compareBlock(t, filepath.Join("v1", "internal", "golden_kitchen_sink_connection_impl.h"),
			"  Options options() override { return options_; }\n",
			"  Options options() override { return options_; }\n")
		compareBlock(t, filepath.Join("v1", "internal", "golden_kitchen_sink_connection_impl.h"),
			" private:\n  std::unique_ptr<google::cloud::BackgroundThreads> background_;\n",
			"  Options options_;\n};\n")
		compareBlock(t, filepath.Join("v1", "internal", "golden_kitchen_sink_connection_impl.cc"),
			"GoldenKitchenSinkConnectionImpl::GoldenKitchenSinkConnectionImpl(\n",
			"        GoldenKitchenSinkConnection::options())) {}\n")

		// Tracing Connection
		compareBlock(t, filepath.Join("v1", "internal", "golden_kitchen_sink_tracing_connection.h"),
			"class GoldenKitchenSinkTracingConnection\n",
			"  Options options() override { return child_->options(); }\n")
		compareBlock(t, filepath.Join("v1", "internal", "golden_kitchen_sink_tracing_connection.h"),
			" private:\n  std::shared_ptr<golden_v1::GoldenKitchenSinkConnection> child_;\n",
			"  std::shared_ptr<golden_v1::GoldenKitchenSinkConnection> child_;\n};\n")
		compareBlock(t, filepath.Join("v1", "internal", "golden_kitchen_sink_tracing_connection.cc"),
			"GoldenKitchenSinkTracingConnection::GoldenKitchenSinkTracingConnection(\n",
			": child_(std::move(child)) {}\n")
		compareBlock(t, filepath.Join("v1", "internal", "golden_kitchen_sink_tracing_connection.cc"),
			"std::shared_ptr<golden_v1::GoldenKitchenSinkConnection>\nMakeGoldenKitchenSinkTracingConnection(\n",
			"  return conn;\n}\n")

		// Option Defaults & Stub Factory
		compareBlock(t, filepath.Join("v1", "internal", "golden_kitchen_sink_option_defaults.h"),
			"Options GoldenKitchenSinkDefaultOptions(Options options);\n",
			"Options GoldenKitchenSinkDefaultOptions(Options options);\n")
		compareBlock(t, filepath.Join("v1", "internal", "golden_kitchen_sink_stub_factory.h"),
			"std::shared_ptr<GoldenKitchenSinkStub> CreateDefaultGoldenKitchenSinkStub(\n",
			"    Options const& options);\n")
	})

	t.Run("REST Skeletons", func(t *testing.T) {
		// REST Connection Class in golden_rest_only_connection.h
		compareBlock(t, filepath.Join("v1", "golden_rest_only_connection.h"),
			"class GoldenRestOnlyConnection {\n public:\n",
			"  virtual Options options() { return Options{}; }\n")

		// REST Connection Factory in golden_kitchen_sink_rest_connection.h & golden_kitchen_sink_rest_connection.cc
		compareBlock(t, filepath.Join("v1", "golden_kitchen_sink_rest_connection.h"),
			"std::shared_ptr<GoldenKitchenSinkConnection> MakeGoldenKitchenSinkConnectionRest(\n",
			"    Options options = {});\n")
		compareBlock(t, filepath.Join("v1", "golden_kitchen_sink_rest_connection.cc"),
			"std::shared_ptr<GoldenKitchenSinkConnection> MakeGoldenKitchenSinkConnectionRest(\n",
			"      std::move(background), std::move(stub), std::move(options)));\n}\n")

		// REST Stub & DefaultRESTStub
		compareBlock(t, filepath.Join("v1", "internal", "golden_rest_only_rest_stub.h"),
			"class GoldenRestOnlyRestStub {\n public:\n",
			"  virtual ~GoldenRestOnlyRestStub() = default;\n")
		compareBlock(t, filepath.Join("v1", "internal", "golden_rest_only_rest_stub.h"),
			"class DefaultGoldenRestOnlyRestStub : public GoldenRestOnlyRestStub {\n",
			"      Options options);\n")
		compareBlock(t, filepath.Join("v1", "internal", "golden_rest_only_rest_stub.h"),
			" private:\n  std::shared_ptr<rest_internal::RestClient> service_;\n",
			"  Options options_;\n};\n")
		compareBlock(t, filepath.Join("v1", "internal", "golden_rest_only_rest_stub.cc"),
			"DefaultGoldenRestOnlyRestStub::DefaultGoldenRestOnlyRestStub(Options options)\n",
			"      options_(std::move(options)) {}\n")

		// REST Logging
		compareBlock(t, filepath.Join("v1", "internal", "golden_rest_only_rest_logging_decorator.h"),
			"class GoldenRestOnlyRestLogging : public GoldenRestOnlyRestStub {\n",
			"                       std::set<std::string> components);\n")
		compareBlock(t, filepath.Join("v1", "internal", "golden_rest_only_rest_logging_decorator.h"),
			" private:\n  std::shared_ptr<GoldenRestOnlyRestStub> child_;\n",
			"  std::set<std::string> components_;\n};\n")
		compareBlock(t, filepath.Join("v1", "internal", "golden_rest_only_rest_logging_decorator.cc"),
			"GoldenRestOnlyRestLogging::GoldenRestOnlyRestLogging(\n",
			"      components_(std::move(components)) {}\n")

		// REST Metadata
		compareBlock(t, filepath.Join("v1", "internal", "golden_rest_only_rest_metadata_decorator.h"),
			"class GoldenRestOnlyRestMetadata : public GoldenRestOnlyRestStub {\n",
			"      std::string api_client_header = \"\");\n")
		compareBlock(t, filepath.Join("v1", "internal", "golden_rest_only_rest_metadata_decorator.h"),
			" private:\n  void SetMetadata(rest_internal::RestContext& rest_context,\n",
			"  std::string api_client_header_;\n};\n")
		compareBlock(t, filepath.Join("v1", "internal", "golden_rest_only_rest_metadata_decorator.cc"),
			"GoldenRestOnlyRestMetadata::GoldenRestOnlyRestMetadata(\n",
			": std::move(api_client_header)) {}\n")
		compareBlock(t, filepath.Join("v1", "internal", "golden_rest_only_rest_metadata_decorator.cc"),
			"void GoldenRestOnlyRestMetadata::SetMetadata(\n",
			"api_client_header_);\n}\n")

		// REST Connection Impl
		compareBlock(t, filepath.Join("v1", "internal", "golden_rest_only_rest_connection_impl.h"),
			"class GoldenRestOnlyRestConnectionImpl\n",
			"  Options options() override { return options_; }\n")
		compareBlock(t, filepath.Join("v1", "internal", "golden_rest_only_rest_connection_impl.h"),
			" private:\n  static std::unique_ptr<golden_v1::GoldenRestOnlyRetryPolicy>\n  retry_policy(Options const& options) {\n",
			"  Options options_;\n};\n")
		compareBlock(t, filepath.Join("v1", "internal", "golden_rest_only_rest_connection_impl.cc"),
			"GoldenRestOnlyRestConnectionImpl::GoldenRestOnlyRestConnectionImpl(\n",
			"        GoldenRestOnlyConnection::options())) {}\n")

		// REST Stub Factory
		compareBlock(t, filepath.Join("v1", "internal", "golden_rest_only_rest_stub_factory.h"),
			"std::shared_ptr<GoldenRestOnlyRestStub> CreateDefaultGoldenRestOnlyRestStub(\n",
			"    Options const& options);\n")
		compareBlock(t, filepath.Join("v1", "internal", "golden_rest_only_rest_stub_factory.cc"),
			"std::shared_ptr<GoldenRestOnlyRestStub>\nCreateDefaultGoldenRestOnlyRestStub(Options const& options) {\n",
			"  return stub;\n}\n")
	})

	t.Run("Forwarding Headers", func(t *testing.T) {
		compareBlock(t, "golden_kitchen_sink_client.h",
			"using ::google::cloud::golden_v1::GoldenKitchenSinkClient;\n",
			"using ::google::cloud::golden_v1::GoldenKitchenSinkClient;\n")
		compareBlock(t, "golden_kitchen_sink_connection.h",
			"using ::google::cloud::golden_v1::MakeGoldenKitchenSinkConnection;\n",
			"using ::google::cloud::golden_v1::GoldenKitchenSinkRetryPolicy;\n")
		compareBlock(t, "golden_kitchen_sink_connection_idempotency_policy.h",
			"using ::google::cloud::golden_v1::MakeDefaultGoldenKitchenSinkConnectionIdempotencyPolicy;\n",
			"using ::google::cloud::golden_v1::GoldenKitchenSinkConnectionIdempotencyPolicy;\n")
		compareBlock(t, "golden_kitchen_sink_options.h",
			"using ::google::cloud::golden_v1::GoldenKitchenSinkBackoffPolicyOption;\n",
			"using ::google::cloud::golden_v1::GoldenKitchenSinkRetryPolicyOption;\n")
		compareBlock(t, filepath.Join("mocks", "mock_golden_kitchen_sink_connection.h"),
			"using ::google::cloud::golden_v1_mocks::MockGoldenKitchenSinkConnection;\n",
			"using ::google::cloud::golden_v1_mocks::MockGoldenKitchenSinkConnection;\n")
	})
}

func TestGoldenServices_Layer35_MethodDeclarations(t *testing.T) {
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

	readFile := func(path string) string {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("failed to read %s: %v", path, err)
		}
		return string(data)
	}

	goldenDir := filepath.Join("testdata", "golden")

	compareBlock := func(t *testing.T, rel, startStr, endStr string) {
		t.Helper()
		gotContent := readFile(filepath.Join(goldenRoot, rel))
		wantContent := readFile(filepath.Join(goldenDir, rel))

		gotBlock := extractBlock(t, gotContent, startStr, endStr)
		wantBlock := extractBlock(t, wantContent, startStr, endStr)
		if diff := cmp.Diff(wantBlock, gotBlock); diff != "" {
			t.Errorf("%s block [%q ... %q] mismatch (-want +got):\n%s", rel, startStr, endStr, diff)
		}
	}

	t.Run("Client Method Declarations", func(t *testing.T) {
		// GoldenKitchenSinkClient: GenerateAccessToken overloads + full request
		compareBlock(t, filepath.Join("v1", "golden_kitchen_sink_client.h"),
			"  // clang-format off\n  ///\n  /// Generates an OAuth 2.0 access token for a service account.\n",
			"GenerateAccessToken(google::test::admin::database::v1::GenerateAccessTokenRequest const& request, Options opts = {});\n")

		// GoldenKitchenSinkClient: DoNothing (void/Empty return type)
		compareBlock(t, filepath.Join("v1", "golden_kitchen_sink_client.h"),
			"  // clang-format off\n  ///\n  /// Does Nothing.\n",
			"DoNothing(google::protobuf::Empty const& request, Options opts = {});\n")

		// GoldenKitchenSinkClient: StreamingRead
		compareBlock(t, filepath.Join("v1", "golden_kitchen_sink_client.h"),
			"  StreamRange<google::test::admin::database::v1::Response>\n  StreamingRead(",
			"StreamingRead(google::test::admin::database::v1::Request const& request, Options opts = {});\n")

		// GoldenKitchenSinkClient: AsyncStreamingReadWrite (bidi streaming)
		compareBlock(t, filepath.Join("v1", "golden_kitchen_sink_client.h"),
			"  std::unique_ptr<::google::cloud::AsyncStreamingReadWriteRpc<\n",
			"AsyncStreamingReadWrite(Options opts = {});\n")

		// GoldenKitchenSinkClient: ExplicitRouting
		compareBlock(t, filepath.Join("v1", "golden_kitchen_sink_client.h"),
			"  Status\n  ExplicitRouting1(",
			"ExplicitRouting2(google::test::admin::database::v1::ExplicitRoutingRequest const& request, Options opts = {});\n")

		// GoldenThingAdminClient: CreateDatabase (LRO future, start with NoAwaitTag, await with Operation)
		compareBlock(t, filepath.Join("v1", "golden_thing_admin_client.h"),
			"  future<StatusOr<google::test::admin::database::v1::Database>>\n  CreateDatabase(std::string const& parent",
			"CreateDatabase(google::longrunning::Operation const& operation, Options opts = {});\n")

		// GoldenThingAdminClient: SetIamPolicy (with IAM updater)
		compareBlock(t, filepath.Join("v1", "golden_thing_admin_client.h"),
			"  SetIamPolicy(std::string const& resource, google::iam::v1::Policy const& policy, Options opts = {});\n",
			"SetIamPolicy(google::iam::v1::SetIamPolicyRequest const& request, Options opts = {});\n")

		// GoldenThingAdminClient: AsyncGetDatabase
		compareBlock(t, filepath.Join("v1", "golden_thing_admin_client.h"),
			"  future<StatusOr<google::test::admin::database::v1::Database>>\n  AsyncGetDatabase(",
			"AsyncGetDatabase(google::test::admin::database::v1::GetDatabaseRequest const& request, Options opts = {});\n")

		// RequestIdServiceClient: CreateFoo, RenameFoo, ListFoos, AsyncCreateFoo
		compareBlock(t, filepath.Join("v1", "request_id_client.h"),
			"  StatusOr<google::test::requestid::v1::Foo>\n  CreateFoo(",
			"AsyncCreateFoo(google::test::requestid::v1::CreateFooRequest const& request, Options opts = {});\n")

		// GoldenKitchenSinkClient: Deprecated2 method
		compareBlock(t, filepath.Join("v1", "golden_kitchen_sink_client.h"),
			"  GOOGLE_CLOUD_CPP_DEPRECATED(\"This RPC is deprecated.\")\n  Status\n  Deprecated2(Options opts = {});\n",
			"Deprecated2(google::test::admin::database::v1::GenerateAccessTokenRequest const& request, Options opts = {});\n")

		// DeprecatedServiceClient: Noop
		compareBlock(t, filepath.Join("v1", "deprecated_client.h"),
			"  Status\n  Noop(",
			"Noop(google::test::deprecated::v1::DeprecatedServiceRequest const& request, Options opts = {});\n")
	})

	t.Run("Connection Method Declarations", func(t *testing.T) {
		// GoldenKitchenSinkConnection: pure virtual method declarations
		compareBlock(t, filepath.Join("v1", "golden_kitchen_sink_connection.h"),
			"  virtual StatusOr<google::test::admin::database::v1::GenerateAccessTokenResponse>\n  GenerateAccessToken(",
			"ListOperations(google::longrunning::ListOperationsRequest request);\n")

		// GoldenThingAdminConnection: LRO, IAM, Async method declarations
		compareBlock(t, filepath.Join("v1", "golden_thing_admin_connection.h"),
			"  virtual StreamRange<google::test::admin::database::v1::Database>\n  ListDatabases(",
			"AsyncDropDatabase(google::test::admin::database::v1::DropDatabaseRequest const& request);\n")
	})

	t.Run("Idempotency Policy Declarations", func(t *testing.T) {
		// GoldenKitchenSink Idempotency Policy
		compareBlock(t, filepath.Join("v1", "golden_kitchen_sink_connection_idempotency_policy.h"),
			"  virtual google::cloud::Idempotency\n  GenerateAccessToken(",
			"ListOperations(google::longrunning::ListOperationsRequest request);\n")

		// GoldenThingAdmin Idempotency Policy
		compareBlock(t, filepath.Join("v1", "golden_thing_admin_connection_idempotency_policy.h"),
			"  virtual google::cloud::Idempotency\n  ListDatabases(",
			"ListOperations(google::longrunning::ListOperationsRequest request);\n")
	})

	t.Run("Mock Connection Declarations", func(t *testing.T) {
		// MockGoldenKitchenSinkConnection
		compareBlock(t, filepath.Join("v1", "mocks", "mock_golden_kitchen_sink_connection.h"),
			"  MOCK_METHOD(StatusOr<google::test::admin::database::v1::GenerateAccessTokenResponse>,\n  GenerateAccessToken,\n",
			"ListOperations,\n  (google::longrunning::ListOperationsRequest request), (override));\n")

		// MockGoldenThingAdminConnection (including LRO disambiguation comments)
		compareBlock(t, filepath.Join("v1", "mocks", "mock_golden_thing_admin_connection.h"),
			"  /// To disambiguate calls, use:\n  ///\n  /// @code\n  /// using ::testing::_;",
			"AsyncDropDatabase,\n  (google::test::admin::database::v1::DropDatabaseRequest const& request), (override));\n")
	})

	t.Run("Stub Declarations", func(t *testing.T) {
		// GoldenKitchenSinkStub pure virtuals
		compareBlock(t, filepath.Join("v1", "internal", "golden_kitchen_sink_stub.h"),
			"  virtual StatusOr<google::test::admin::database::v1::GenerateAccessTokenResponse> GenerateAccessToken(\n",
			"AsyncStreamingWrite(\n      google::cloud::CompletionQueue const& cq,\n      std::shared_ptr<grpc::ClientContext> context,\n    google::cloud::internal::ImmutableOptions options) = 0;\n")

		// DefaultGoldenKitchenSinkStub overrides
		compareBlock(t, filepath.Join("v1", "internal", "golden_kitchen_sink_stub.h"),
			"  StatusOr<google::test::admin::database::v1::GenerateAccessTokenResponse> GenerateAccessToken(\n      grpc::ClientContext& context,\n      Options const& options,\n      google::test::admin::database::v1::GenerateAccessTokenRequest const& request) override;\n",
			"AsyncStreamingWrite(\n      google::cloud::CompletionQueue const& cq,\n      std::shared_ptr<grpc::ClientContext> context,\n      google::cloud::internal::ImmutableOptions options) override;\n")
	})

	t.Run("Decorator Declarations", func(t *testing.T) {
		// Auth decorator
		compareBlock(t, filepath.Join("v1", "internal", "golden_kitchen_sink_auth_decorator.h"),
			"  StatusOr<google::test::admin::database::v1::GenerateAccessTokenResponse> GenerateAccessToken(\n",
			"AsyncStreamingWrite(\n      google::cloud::CompletionQueue const& cq,\n      std::shared_ptr<grpc::ClientContext> context,\n      google::cloud::internal::ImmutableOptions options) override;\n")

		// Logging decorator
		compareBlock(t, filepath.Join("v1", "internal", "golden_kitchen_sink_logging_decorator.h"),
			"  StatusOr<google::test::admin::database::v1::GenerateAccessTokenResponse> GenerateAccessToken(\n",
			"AsyncStreamingWrite(\n      google::cloud::CompletionQueue const& cq,\n      std::shared_ptr<grpc::ClientContext> context,\n      google::cloud::internal::ImmutableOptions options) override;\n")

		// Metadata decorator
		compareBlock(t, filepath.Join("v1", "internal", "golden_kitchen_sink_metadata_decorator.h"),
			"  StatusOr<google::test::admin::database::v1::GenerateAccessTokenResponse> GenerateAccessToken(\n",
			"AsyncStreamingWrite(\n      google::cloud::CompletionQueue const& cq,\n      std::shared_ptr<grpc::ClientContext> context,\n      google::cloud::internal::ImmutableOptions options) override;\n")

		// Round robin decorator
		compareBlock(t, filepath.Join("v1", "internal", "golden_kitchen_sink_round_robin_decorator.h"),
			"  StatusOr<google::test::admin::database::v1::GenerateAccessTokenResponse> GenerateAccessToken(\n",
			"AsyncStreamingWrite(\n      google::cloud::CompletionQueue const& cq,\n      std::shared_ptr<grpc::ClientContext> context,\n      google::cloud::internal::ImmutableOptions options) override;\n")

		// Tracing stub
		compareBlock(t, filepath.Join("v1", "internal", "golden_kitchen_sink_tracing_stub.h"),
			"  StatusOr<google::test::admin::database::v1::GenerateAccessTokenResponse> GenerateAccessToken(\n",
			"AsyncStreamingWrite(\n      google::cloud::CompletionQueue const& cq,\n      std::shared_ptr<grpc::ClientContext> context,\n      google::cloud::internal::ImmutableOptions options) override;\n")
	})

	t.Run("Connection Implementation and Tracing Connection Declarations", func(t *testing.T) {
		// GoldenKitchenSinkConnectionImpl (StreamingUpdater + method overrides)
		compareBlock(t, filepath.Join("v1", "internal", "golden_kitchen_sink_connection_impl.h"),
			"void GoldenKitchenSinkStreamingReadStreamingUpdater(\n",
			"ListOperations(google::longrunning::ListOperationsRequest request) override;\n")

		// GoldenKitchenSinkTracingConnection
		compareBlock(t, filepath.Join("v1", "internal", "golden_kitchen_sink_tracing_connection.h"),
			"  StatusOr<google::test::admin::database::v1::GenerateAccessTokenResponse>\n  GenerateAccessToken(",
			"ListOperations(google::longrunning::ListOperationsRequest request) override;\n")

		// GoldenThingAdminConnectionImpl
		compareBlock(t, filepath.Join("v1", "internal", "golden_thing_admin_connection_impl.h"),
			"  StreamRange<google::test::admin::database::v1::Database>\n  ListDatabases(",
			"AsyncDropDatabase(google::test::admin::database::v1::DropDatabaseRequest const& request) override;\n")

		// GoldenThingAdminTracingConnection
		compareBlock(t, filepath.Join("v1", "internal", "golden_thing_admin_tracing_connection.h"),
			"  StreamRange<google::test::admin::database::v1::Database>\n  ListDatabases(",
			"AsyncDropDatabase(google::test::admin::database::v1::DropDatabaseRequest const& request) override;\n")

		// RequestIdServiceConnectionImpl (includes invocation_id_generator_)
		compareBlock(t, filepath.Join("v1", "internal", "request_id_connection_impl.h"),
			"  StatusOr<google::test::requestid::v1::Foo>\n  CreateFoo(",
			"invocation_id_generator_ =\n          std::make_shared<google::cloud::internal::InvocationIdGenerator>();\n")
	})
}

func TestGoldenServices_Layer36_MethodDefinitions(t *testing.T) {
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

	// 1. GoldenKitchenSink & GoldenThingAdmin (Library 0: golden)
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

	readFile := func(path string) string {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("failed to read %s: %v", path, err)
		}
		return string(data)
	}

	goldenDir := filepath.Join("testdata", "golden")

	compareBlock := func(t *testing.T, rel, startStr, endStr string) {
		t.Helper()
		gotContent := readFile(filepath.Join(goldenRoot, rel))
		wantContent := readFile(filepath.Join(goldenDir, rel))

		gotBlock := extractBlock(t, gotContent, startStr, endStr)
		wantBlock := extractBlock(t, wantContent, startStr, endStr)
		if diff := cmp.Diff(wantBlock, gotBlock); diff != "" {
			t.Errorf("%s block [%q ... %q] mismatch (-want +got):\n%s", rel, startStr, endStr, diff)
		}
	}

	t.Run("Client Method Definitions", func(t *testing.T) {
		// GoldenKitchenSinkClient: GenerateAccessToken method
		compareBlock(t, filepath.Join("v1", "golden_kitchen_sink_client.cc"),
			"StatusOr<google::test::admin::database::v1::GenerateAccessTokenResponse>\nGoldenKitchenSinkClient::GenerateAccessToken(std::string const& name, std::string const& not_used_anymore, Options opts) {\n",
			"  return connection_->GenerateAccessToken(request);\n}\n")

		// GoldenKitchenSinkClient: DoNothing
		compareBlock(t, filepath.Join("v1", "golden_kitchen_sink_client.cc"),
			"Status\nGoldenKitchenSinkClient::DoNothing(Options opts) {\n",
			"  return connection_->DoNothing(request);\n}\n")

		// GoldenThingAdminClient: CreateDatabase (LRO future)
		compareBlock(t, filepath.Join("v1", "golden_thing_admin_client.cc"),
			"future<StatusOr<google::test::admin::database::v1::Database>>\nGoldenThingAdminClient::CreateDatabase(std::string const& parent, std::string const& create_statement, Options opts) {\n",
			"  return connection_->CreateDatabase(request);\n}\n")

		// GoldenThingAdminClient: SetIamPolicy (with IAM updater)
		compareBlock(t, filepath.Join("v1", "golden_thing_admin_client.cc"),
			"StatusOr<google::iam::v1::Policy>\nGoldenThingAdminClient::SetIamPolicy(std::string const& resource, IamUpdater const& updater, Options opts) {\n",
			"    std::this_thread::sleep_for(backoff_policy->OnCompletion());\n  }\n}\n")

		// RequestIdServiceClient: CreateFoo
		compareBlock(t, filepath.Join("v1", "request_id_client.cc"),
			"StatusOr<google::test::requestid::v1::Foo>\nRequestIdServiceClient::CreateFoo(std::string const& parent, std::string const& foo_id, Options opts) {\n",
			"  return connection_->CreateFoo(request);\n}\n")

		// DeprecatedServiceClient: Noop
		compareBlock(t, filepath.Join("v1", "deprecated_client.cc"),
			"Status\nDeprecatedServiceClient::Noop(google::test::deprecated::v1::DeprecatedServiceRequest const& request, Options opts) {\n",
			"  return connection_->Noop(request);\n}\n")
	})

	t.Run("Connection Method Definitions", func(t *testing.T) {
		// GoldenKitchenSinkConnection: AsyncStreamingReadWrite
		compareBlock(t, filepath.Join("v1", "golden_kitchen_sink_connection.cc"),
			"std::unique_ptr<::google::cloud::AsyncStreamingReadWriteRpc<\n    google::test::admin::database::v1::Request,\n    google::test::admin::database::v1::Response>>\nGoldenKitchenSinkConnection::AsyncStreamingReadWrite() {\n",
			"      Status(StatusCode::kUnimplemented, \"not implemented\"));\n}\n")

		// GoldenThingAdminConnection: CreateDatabase (LRO defaults)
		compareBlock(t, filepath.Join("v1", "golden_thing_admin_connection.cc"),
			"future<StatusOr<google::test::admin::database::v1::Database>>\nGoldenThingAdminConnection::CreateDatabase(\n    google::test::admin::database::v1::CreateDatabaseRequest const&) {\n",
			"      Status(StatusCode::kUnimplemented, \"not implemented\"));\n}\n")
	})

	t.Run("Connection Idempotency Policy Definitions", func(t *testing.T) {
		// GoldenKitchenSinkConnectionIdempotencyPolicy: GenerateAccessToken
		compareBlock(t, filepath.Join("v1", "golden_kitchen_sink_connection_idempotency_policy.cc"),
			"Idempotency GoldenKitchenSinkConnectionIdempotencyPolicy::GenerateAccessToken(google::test::admin::database::v1::GenerateAccessTokenRequest const&) {\n",
			"  return Idempotency::kNonIdempotent;\n}\n")

		// GoldenThingAdminConnectionIdempotencyPolicy: MakeDefault
		compareBlock(t, filepath.Join("v1", "golden_thing_admin_connection_idempotency_policy.cc"),
			"std::unique_ptr<GoldenThingAdminConnectionIdempotencyPolicy>\n    MakeDefaultGoldenThingAdminConnectionIdempotencyPolicy() {\n",
			"  return std::make_unique<GoldenThingAdminConnectionIdempotencyPolicy>();\n}\n")
	})

	t.Run("Stub Definitions", func(t *testing.T) {
		// DefaultGoldenKitchenSinkStub: GenerateAccessToken
		compareBlock(t, filepath.Join("v1", "internal", "golden_kitchen_sink_stub.cc"),
			"StatusOr<google::test::admin::database::v1::GenerateAccessTokenResponse>\nDefaultGoldenKitchenSinkStub::GenerateAccessToken(\n",
			"  return response;\n}\n")

		// DefaultGoldenThingAdminStub: AsyncCreateDatabase & CreateDatabase
		compareBlock(t, filepath.Join("v1", "internal", "golden_thing_admin_stub.cc"),
			"future<StatusOr<google::longrunning::Operation>>\nDefaultGoldenThingAdminStub::AsyncCreateDatabase(\n",
			"  return response;\n}\n")

		// DefaultRequestIdServiceStub: CreateFoo
		compareBlock(t, filepath.Join("v1", "internal", "request_id_stub.cc"),
			"StatusOr<google::test::requestid::v1::Foo>\nDefaultRequestIdServiceStub::CreateFoo(\n",
			"  return response;\n}\n")
	})

	t.Run("Stub Factory Definitions", func(t *testing.T) {
		// CreateDefaultGoldenKitchenSinkStub
		compareBlock(t, filepath.Join("v1", "internal", "golden_kitchen_sink_stub_factory.cc"),
			"std::shared_ptr<GoldenKitchenSinkStub>\nCreateDefaultGoldenKitchenSinkStub(\n",
			"  return stub;\n}\n")

		// CreateDefaultGoldenThingAdminStub
		compareBlock(t, filepath.Join("v1", "internal", "golden_thing_admin_stub_factory.cc"),
			"std::shared_ptr<GoldenThingAdminStub>\nCreateDefaultGoldenThingAdminStub(\n",
			"  return stub;\n}\n")
	})

	t.Run("Connection Implementation Definitions", func(t *testing.T) {
		// GoldenKitchenSinkConnectionImpl: StreamingUpdater & GenerateAccessToken
		compareBlock(t, filepath.Join("v1", "internal", "golden_kitchen_sink_connection_impl.cc"),
			"void GoldenKitchenSinkStreamingReadStreamingUpdater(\n    google::test::admin::database::v1::Response const&,\n    google::test::admin::database::v1::Request&) {}\n",
			"      *current, request, __func__);\n}\n")

		// GoldenKitchenSinkConnectionImpl: StreamingRead
		compareBlock(t, filepath.Join("v1", "internal", "golden_kitchen_sink_connection_impl.cc"),
			"StreamRange<google::test::admin::database::v1::Response>\nGoldenKitchenSinkConnectionImpl::StreamingRead(google::test::admin::database::v1::Request const& request) {\n",
			"        return response;\n      });\n}\n")

		// GoldenThingAdminConnectionImpl: CreateDatabase (LRO overloads)
		compareBlock(t, filepath.Join("v1", "internal", "golden_thing_admin_connection_impl.cc"),
			"future<StatusOr<google::test::admin::database::v1::Database>>\nGoldenThingAdminConnectionImpl::CreateDatabase(google::test::admin::database::v1::CreateDatabaseRequest const& request) {\n",
			"    polling_policy(*current), __func__);\n}\n")

		// RequestIdServiceConnectionImpl: CreateFoo with invocation_id_generator
		compareBlock(t, filepath.Join("v1", "internal", "request_id_connection_impl.cc"),
			"StatusOr<google::test::requestid::v1::Foo>\nRequestIdServiceConnectionImpl::CreateFoo(google::test::requestid::v1::CreateFooRequest const& request) {\n",
			"      *current, request_copy, __func__);\n}\n")
	})

	t.Run("Auth Decorator Definitions", func(t *testing.T) {
		// GoldenKitchenSinkAuth: GenerateAccessToken
		compareBlock(t, filepath.Join("v1", "internal", "golden_kitchen_sink_auth_decorator.cc"),
			"StatusOr<google::test::admin::database::v1::GenerateAccessTokenResponse> GoldenKitchenSinkAuth::GenerateAccessToken(\n",
			"  return child_->GenerateAccessToken(context, options, request);\n}\n")

		// GoldenThingAdminAuth: AsyncCreateDatabase & CreateDatabase
		compareBlock(t, filepath.Join("v1", "internal", "golden_thing_admin_auth_decorator.cc"),
			"future<StatusOr<google::longrunning::Operation>>\nGoldenThingAdminAuth::AsyncCreateDatabase(\n",
			"  return child_->CreateDatabase(context, options, request);\n}\n")

		// GoldenThingAdminAuth: AsyncGetOperation & AsyncCancelOperation
		compareBlock(t, filepath.Join("v1", "internal", "golden_thing_admin_auth_decorator.cc"),
			"future<StatusOr<google::longrunning::Operation>>\nGoldenThingAdminAuth::AsyncGetOperation(\n",
			"        return child->AsyncCancelOperation(\n            cq, *std::move(context), std::move(options), request);\n      });\n}\n")
	})

	t.Run("Logging Decorator Definitions", func(t *testing.T) {
		// GoldenKitchenSinkLogging: GenerateAccessToken
		compareBlock(t, filepath.Join("v1", "internal", "golden_kitchen_sink_logging_decorator.cc"),
			"StatusOr<google::test::admin::database::v1::GenerateAccessTokenResponse>\nGoldenKitchenSinkLogging::GenerateAccessToken(\n",
			"      context, options, request, __func__, tracing_options_);\n}\n")

		// GoldenThingAdminLogging: AsyncCreateDatabase & CreateDatabase
		compareBlock(t, filepath.Join("v1", "internal", "golden_thing_admin_logging_decorator.cc"),
			"future<StatusOr<google::longrunning::Operation>>\nGoldenThingAdminLogging::AsyncCreateDatabase(\n",
			"      context, options, request, __func__, tracing_options_);\n}\n")
	})

	t.Run("Metadata Decorator Definitions", func(t *testing.T) {
		// GoldenKitchenSinkMetadata: GenerateAccessToken (url-encoded routing)
		compareBlock(t, filepath.Join("v1", "internal", "golden_kitchen_sink_metadata_decorator.cc"),
			"StatusOr<google::test::admin::database::v1::GenerateAccessTokenResponse>\nGoldenKitchenSinkMetadata::GenerateAccessToken(\n",
			"  return child_->GenerateAccessToken(context, options, request);\n}\n")

		// GoldenKitchenSinkMetadata: SetMetadata overloads
		compareBlock(t, filepath.Join("v1", "internal", "golden_kitchen_sink_metadata_decorator.cc"),
			"void GoldenKitchenSinkMetadata::SetMetadata(grpc::ClientContext& context,\n",
			"      context, options, fixed_metadata_, api_client_header_);\n}\n")

		// GoldenThingAdminMetadata: AsyncGetOperation & AsyncCancelOperation
		compareBlock(t, filepath.Join("v1", "internal", "golden_thing_admin_metadata_decorator.cc"),
			"future<StatusOr<google::longrunning::Operation>>\nGoldenThingAdminMetadata::AsyncGetOperation(\n",
			"  return child_->AsyncCancelOperation(\n      cq, std::move(context), std::move(options), request);\n}\n")
	})

	t.Run("Round Robin Decorator Definitions", func(t *testing.T) {
		// GoldenKitchenSinkRoundRobin: GenerateAccessToken
		compareBlock(t, filepath.Join("v1", "internal", "golden_kitchen_sink_round_robin_decorator.cc"),
			"StatusOr<google::test::admin::database::v1::GenerateAccessTokenResponse> GoldenKitchenSinkRoundRobin::GenerateAccessToken(\n",
			"  return Child()->GenerateAccessToken(context, options, request);\n}\n")

		// GoldenKitchenSinkRoundRobin: Child helper
		compareBlock(t, filepath.Join("v1", "internal", "golden_kitchen_sink_round_robin_decorator.cc"),
			"std::shared_ptr<GoldenKitchenSinkStub>\nGoldenKitchenSinkRoundRobin::Child() {\n",
			"  return children_[current];\n}\n")
	})

	t.Run("Tracing Stub Definitions", func(t *testing.T) {
		// GoldenKitchenSinkTracingStub: GenerateAccessToken
		compareBlock(t, filepath.Join("v1", "internal", "golden_kitchen_sink_tracing_stub.cc"),
			"StatusOr<google::test::admin::database::v1::GenerateAccessTokenResponse> GoldenKitchenSinkTracingStub::GenerateAccessToken(\n",
			"                           child_->GenerateAccessToken(context, options, request));\n}\n")

		// RequestIdServiceTracingStub: CreateFoo (request_id attribute)
		compareBlock(t, filepath.Join("v1", "internal", "request_id_tracing_stub.cc"),
			"StatusOr<google::test::requestid::v1::Foo> RequestIdServiceTracingStub::CreateFoo(\n",
			"                           child_->CreateFoo(context, options, request));\n}\n")

		// MakeGoldenKitchenSinkTracingStub factory
		compareBlock(t, filepath.Join("v1", "internal", "golden_kitchen_sink_tracing_stub.cc"),
			"std::shared_ptr<GoldenKitchenSinkStub> MakeGoldenKitchenSinkTracingStub(\n",
			"#endif  // GOOGLE_CLOUD_CPP_HAVE_OPENTELEMETRY\n}\n")
	})

	t.Run("Tracing Connection Definitions", func(t *testing.T) {
		// GoldenKitchenSinkTracingConnection: GenerateAccessToken
		compareBlock(t, filepath.Join("v1", "internal", "golden_kitchen_sink_tracing_connection.cc"),
			"StatusOr<google::test::admin::database::v1::GenerateAccessTokenResponse>\nGoldenKitchenSinkTracingConnection::GenerateAccessToken(google::test::admin::database::v1::GenerateAccessTokenRequest const& request) {\n",
			"  return internal::EndSpan(*span, child_->GenerateAccessToken(request));\n}\n")

		// GoldenThingAdminTracingConnection: CreateDatabase (LRO)
		compareBlock(t, filepath.Join("v1", "internal", "golden_thing_admin_tracing_connection.cc"),
			"future<StatusOr<google::test::admin::database::v1::Database>>\nGoldenThingAdminTracingConnection::CreateDatabase(google::test::admin::database::v1::CreateDatabaseRequest const& request) {\n",
			"  return internal::EndSpan(std::move(span), child_->CreateDatabase(request));\n}\n")

		// MakeGoldenKitchenSinkTracingConnection factory
		compareBlock(t, filepath.Join("v1", "internal", "golden_kitchen_sink_tracing_connection.cc"),
			"std::shared_ptr<golden_v1::GoldenKitchenSinkConnection>\nMakeGoldenKitchenSinkTracingConnection(\n",
			"  return conn;\n}\n")
	})

	t.Run("Option Defaults Definitions", func(t *testing.T) {
		// GoldenKitchenSinkDefaultOptions
		compareBlock(t, filepath.Join("v1", "internal", "golden_kitchen_sink_option_defaults.cc"),
			"Options GoldenKitchenSinkDefaultOptions(Options options) {\n",
			"  return options;\n}\n")

		// GoldenThingAdminDefaultOptions (with PollingPolicy)
		compareBlock(t, filepath.Join("v1", "internal", "golden_thing_admin_option_defaults.cc"),
			"Options GoldenThingAdminDefaultOptions(Options options) {\n",
			"  return options;\n}\n")

		// RequestIdServiceDefaultOptions
		compareBlock(t, filepath.Join("v1", "internal", "request_id_option_defaults.cc"),
			"Options RequestIdServiceDefaultOptions(Options options) {\n",
			"  return options;\n}\n")
	})
}

// TestGoldenServices_Layer37_PureGrpcParity verifies 100% byte-for-byte parity for pure gRPC
// services (test-request-id) against the upstream golden files.
//
// Note: Legacy golden files in generator/integration_tests/golden/ have DisableFormat: true
// from the upstream generator (.clang-format). Clang-format respects this configuration and leaves
// formatting unperturbed to preserve legacy golden parity. Active formatting will be verified in
// production oracles (Phases 6–9) where clang-format is standard.
func TestGoldenServices_Layer37_PureGrpcParity(t *testing.T) {
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

	// Copy .clang-format to goldenRoot to ensure clang-format respects the golden configuration.
	// Legacy golden files in generator/integration_tests/golden/ have DisableFormat: true
	// from the upstream generator. Active formatting will be verified in production oracles
	// (Phases 6–9) where clang-format is standard.
	dotClangFormatSrc := filepath.Join("testdata", "golden", ".clang-format")
	if _, err := os.Stat(dotClangFormatSrc); os.IsNotExist(err) {
		dotClangFormatSrc = filepath.Join("testdata", ".clang-format")
	}
	if data, err := os.ReadFile(dotClangFormatSrc); err == nil {
		_ = os.WriteFile(filepath.Join(goldenRoot, ".clang-format"), data, 0o644)
	}

	// Generate pure gRPC service: test-request-id (Library 2)
	reqModel, err := loadTestModel(t, protosDir, googleapisDir, "test_request_id.yaml", "test_request_id.proto")
	if err != nil {
		t.Fatalf("failed to parse test_request_id.proto: %v", err)
	}
	if err := Generate(ctx, reqModel, outdir, cfg.Libraries[2]); err != nil {
		t.Fatalf("Generate(lib2) failed: %v", err)
	}

	goldenDir := filepath.Join("testdata", "golden")

	// Find all 28 request_id files in goldenDir
	var wantRelPaths []string
	err = filepath.WalkDir(goldenDir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && strings.Contains(filepath.Base(p), "request_id") {
			rel, err := filepath.Rel(goldenDir, p)
			if err != nil {
				return err
			}
			wantRelPaths = append(wantRelPaths, rel)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walking goldenDir: %v", err)
	}
	slices.Sort(wantRelPaths)

	if len(wantRelPaths) != 28 {
		t.Fatalf("expected 28 request_id files in golden, found %d", len(wantRelPaths))
	}

	// Format emitted files with clang-format -i
	formatter := "clang-format"
	if _, err := exec.LookPath(formatter); err == nil {
		var emittedFiles []string
		for _, rel := range wantRelPaths {
			emittedFiles = append(emittedFiles, filepath.Join(goldenRoot, rel))
		}
		args := append([]string{"-i"}, emittedFiles...)
		cmd := exec.CommandContext(ctx, formatter, args...)
		if output, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("clang-format failed: %v\nOutput: %s", err, output)
		}
	} else {
		t.Logf("clang-format not found on PATH, skipping formatting")
	}

	// Byte-for-byte comparison
	for _, rel := range wantRelPaths {
		t.Run(rel, func(t *testing.T) {
			gotBytes, err := os.ReadFile(filepath.Join(goldenRoot, rel))
			if err != nil {
				t.Fatalf("failed to read emitted file %s: %v", rel, err)
			}
			wantBytes, err := os.ReadFile(filepath.Join(goldenDir, rel))
			if err != nil {
				t.Fatalf("failed to read golden file %s: %v", rel, err)
			}

			if diff := cmp.Diff(string(wantBytes), string(gotBytes)); diff != "" {
				gotPath := filepath.Join(goldenRoot, rel)
				wantPath := filepath.Join(goldenDir, rel)
				cmd := exec.Command("diff", "-u", wantPath, gotPath)
				out, _ := cmd.CombinedOutput()
				t.Errorf("file %s differs from golden:\n%s", rel, string(out))
			}
		})
	}
}

// TestGoldenServices_Phase4_AllFilesParity asserts 100% byte-for-byte exact zero diff
// across all 188 reference golden files across all 4 test services.
func TestGoldenServices_Phase4_AllFilesParity(t *testing.T) {
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

	dotClangFormatSrc := filepath.Join("testdata", "golden", ".clang-format")
	if _, err := os.Stat(dotClangFormatSrc); os.IsNotExist(err) {
		dotClangFormatSrc = filepath.Join("testdata", ".clang-format")
	}
	if data, err := os.ReadFile(dotClangFormatSrc); err == nil {
		_ = os.WriteFile(filepath.Join(goldenRoot, ".clang-format"), data, 0o644)
	}

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

	goldenDir := filepath.Join("testdata", "golden")

	var wantRelPaths []string
	err = filepath.WalkDir(goldenDir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && !strings.HasPrefix(d.Name(), ".") {
			rel, err := filepath.Rel(goldenDir, p)
			if err != nil {
				return err
			}
			wantRelPaths = append(wantRelPaths, rel)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walking goldenDir: %v", err)
	}
	slices.Sort(wantRelPaths)

	if len(wantRelPaths) != 188 {
		t.Fatalf("expected 188 files in golden, found %d", len(wantRelPaths))
	}

	// Format emitted files with clang-format -i
	formatter := "clang-format"
	if _, err := exec.LookPath(formatter); err == nil {
		var emittedFiles []string
		for _, rel := range wantRelPaths {
			emittedFiles = append(emittedFiles, filepath.Join(goldenRoot, rel))
		}
		// Format in chunks if needed, but 188 files easily fit on command line
		args := append([]string{"-i"}, emittedFiles...)
		cmd := exec.CommandContext(ctx, formatter, args...)
		if output, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("clang-format failed: %v\nOutput: %s", err, output)
		}
	} else {
		t.Logf("clang-format not found on PATH, skipping formatting")
	}

	var differingFiles []string
	for _, rel := range wantRelPaths {
		gotBytes, err := os.ReadFile(filepath.Join(goldenRoot, rel))
		if err != nil {
			t.Errorf("failed to read emitted file %s: %v", rel, err)
			continue
		}
		wantBytes, err := os.ReadFile(filepath.Join(goldenDir, rel))
		if err != nil {
			t.Errorf("failed to read golden file %s: %v", rel, err)
			continue
		}

		if diff := cmp.Diff(string(wantBytes), string(gotBytes)); diff != "" {
			differingFiles = append(differingFiles, rel)
			t.Run(rel, func(t *testing.T) {
				gotPath := filepath.Join(goldenRoot, rel)
				wantPath := filepath.Join(goldenDir, rel)
				cmd := exec.Command("diff", "-u", wantPath, gotPath)
				out, _ := cmd.CombinedOutput()
				t.Errorf("file %s differs from golden:\n%s", rel, string(out))
			})
		}
	}

	t.Logf("Total golden files: %d, Matching: %d, Differing: %d",
		len(wantRelPaths), len(wantRelPaths)-len(differingFiles), len(differingFiles))
	for _, f := range differingFiles {
		t.Logf("DIFFERING: %s", f)
	}
}
