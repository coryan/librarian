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

func TestGenerate(t *testing.T) {
	reqMsg := api.NewTestMessage("GetDatabaseRequest")
	respMsg := api.NewTestMessage("Database")
	method := api.NewTestMethod("GetDatabase").WithInput(reqMsg).WithOutput(respMsg)
	svc := api.NewTestService("GoldenKitchenSink").WithMethods(method)
	model := api.NewTestAPI([]*api.Message{reqMsg, respMsg}, nil, []*api.Service{svc})
	if err := api.CrossReference(model); err != nil {
		t.Fatalf("api.CrossReference failed: %v", err)
	}

	tempDir := t.TempDir()
	lib := &config.Library{
		Name: "golden_kitchen_sink",
		Cpp: &config.CppLibrary{
			CppDefault: config.CppDefault{
				ProductPath: "generator/integration_tests/golden/v1",
			},
		},
	}

	if err := Generate(t.Context(), model, tempDir, lib); err != nil {
		t.Fatalf("Generate failed: %v", err)
	}
}

func TestTestDataParitySetup(t *testing.T) {
	testdataDir, err := filepath.Abs("testdata")
	if err != nil {
		t.Fatal(err)
	}

	// 1. Verify golden_librarian.yaml
	yamlPath := filepath.Join(testdataDir, "golden_librarian.yaml")
	content, err := os.ReadFile(yamlPath)
	if err != nil {
		t.Fatalf("reading golden_librarian.yaml: %v", err)
	}

	cfg, err := yaml.Unmarshal[config.Config](content)
	if err != nil {
		t.Fatalf("unmarshaling golden_librarian.yaml: %v", err)
	}

	if cfg.Language != config.LanguageCpp {
		t.Errorf("got language %q, want %q", cfg.Language, config.LanguageCpp)
	}
	if len(cfg.Libraries) != 4 {
		t.Fatalf("got %d libraries, want 4", len(cfg.Libraries))
	}

	expectedLibs := map[string]struct {
		hasRest bool
		hasGrpc bool
	}{
		"golden_kitchen_sink": {hasRest: true, hasGrpc: true},
		"golden_rest_only":    {hasRest: true, hasGrpc: false},
		"request_id":          {hasRest: false, hasGrpc: true},
		"deprecated":          {hasRest: true, hasGrpc: true},
	}

	for _, lib := range cfg.Libraries {
		expected, ok := expectedLibs[lib.Name]
		if !ok {
			t.Errorf("unexpected library in config: %s", lib.Name)
			continue
		}
		if lib.Cpp == nil {
			t.Errorf("library %s has nil Cpp config", lib.Name)
			continue
		}
		if lib.Cpp.GenerateRestTransport != expected.hasRest {
			t.Errorf("library %s GenerateRestTransport got %v, want %v", lib.Name, lib.Cpp.GenerateRestTransport, expected.hasRest)
		}
		if expected.hasGrpc && !lib.Cpp.HasGrpcTransport() {
			t.Errorf("library %s HasGrpcTransport should be true", lib.Name)
		}
		if !expected.hasGrpc && lib.Cpp.HasGrpcTransport() {
			t.Errorf("library %s HasGrpcTransport should be false", lib.Name)
		}
	}

	// 2. Verify Proto fixtures (both loose and in generator/integration_tests directory structure)
	requiredProtos := []string{
		"test.proto",
		"test2.proto",
		"test_request_id.proto",
		"test_deprecated.proto",
		"backup.proto",
		"common.proto",
		"test.yaml",
		"test_request_id.yaml",
	}
	for _, proto := range requiredProtos {
		protoPath := filepath.Join(testdataDir, "protos", proto)
		if _, err := os.Stat(protoPath); err != nil {
			t.Errorf("required proto fixture missing: %s", protoPath)
		}
		nestedPath := filepath.Join(testdataDir, "protos", "generator", "integration_tests", proto)
		if _, err := os.Stat(nestedPath); err != nil {
			t.Errorf("required nested proto fixture missing: %s", nestedPath)
		}
	}

	// 3. Verify Golden output directory and .clang-format
	clangFormatPath := filepath.Join(testdataDir, "golden", ".clang-format")
	if _, err := os.Stat(clangFormatPath); err != nil {
		t.Errorf("golden .clang-format missing: %s", clangFormatPath)
	}

	goldenForwardingHeaders := []string{
		"golden_kitchen_sink_client.h",
		"golden_kitchen_sink_connection.h",
		"golden_kitchen_sink_connection_idempotency_policy.h",
		"golden_kitchen_sink_options.h",
		"golden_thing_admin_client.h",
		"golden_thing_admin_connection.h",
		"golden_thing_admin_connection_idempotency_policy.h",
		"golden_thing_admin_options.h",
	}
	for _, header := range goldenForwardingHeaders {
		p := filepath.Join(testdataDir, "golden", header)
		if _, err := os.Stat(p); err != nil {
			t.Errorf("golden forwarding header missing: %s", p)
		}
	}

	goldenV1Files := []string{
		"golden_kitchen_sink_client.h",
		"golden_kitchen_sink_client.cc",
		"golden_kitchen_sink_connection.h",
		"golden_kitchen_sink_connection.cc",
		"golden_thing_admin_client.h",
		"golden_thing_admin_client.cc",
		"golden_rest_only_client.h",
		"golden_rest_only_client.cc",
		"deprecated_client.h",
		"deprecated_client.cc",
		"request_id_client.h",
		"request_id_client.cc",
	}
	for _, f := range goldenV1Files {
		p := filepath.Join(testdataDir, "golden", "v1", f)
		if _, err := os.Stat(p); err != nil {
			t.Errorf("golden v1 file missing: %s", p)
		}
	}
}

func TestFromProtobuf_GoldenProtos(t *testing.T) {
	if _, err := exec.LookPath("protoc"); err != nil {
		t.Skip("skipping test because protoc is not installed")
	}

	testdataDir, err := filepath.Abs("testdata")
	if err != nil {
		t.Fatal(err)
	}
	rootTestdata, err := filepath.Abs("../../testdata")
	if err != nil {
		t.Fatal(err)
	}

	src := &sources.Sources{
		Googleapis: filepath.Join(rootTestdata, "googleapis"),
		Showcase:   filepath.Join(testdataDir, "protos"),
	}

	cases := []struct {
		name          string
		includeList   []string
		serviceConfig string
		wantServices  []string
	}{
		{
			name:         "golden_rest_only (test2.proto)",
			includeList:  []string{"test2.proto"},
			wantServices: []string{"GoldenRestOnly"},
		},
		{
			name:          "request_id (test_request_id.proto)",
			includeList:   []string{"test_request_id.proto"},
			serviceConfig: "generator/integration_tests/test_request_id.yaml",
			wantServices:  []string{"RequestIdService"},
		},
		{
			name:         "deprecated (test_deprecated.proto)",
			includeList:  []string{"test_deprecated.proto"},
			wantServices: []string{"DeprecatedService"},
		},
		{
			name:          "golden_kitchen_sink (test.proto + backup.proto)",
			includeList:   []string{"test.proto", "backup.proto"},
			serviceConfig: "generator/integration_tests/test.yaml",
			wantServices:  []string{"GoldenKitchenSink", "GoldenThingAdmin"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			sourceCfg := &sources.SourceConfig{
				Sources:     src,
				ActiveRoots: []string{"googleapis", "showcase"},
				IncludeList: tc.includeList,
			}

			cfg := &parser.ModelConfig{
				Language:            config.LanguageCpp,
				SpecificationFormat: config.SpecProtobuf,
				SpecificationSource: "generator/integration_tests",
				Source:              sourceCfg,
				ServiceConfig:       tc.serviceConfig,
			}

			model, err := parser.CreateModel(cfg)
			if err != nil {
				t.Fatalf("parser.CreateModel failed: %v", err)
			}

			if len(model.Services) != len(tc.wantServices) {
				var names []string
				for _, s := range model.Services {
					names = append(names, s.Name)
				}
				t.Fatalf("expected %d services %v, got %d %v", len(tc.wantServices), tc.wantServices, len(model.Services), names)
			}

			gotNames := make(map[string]bool)
			for _, s := range model.Services {
				gotNames[s.Name] = true
			}
			for _, want := range tc.wantServices {
				if !gotNames[want] {
					t.Errorf("expected service %s not found in model", want)
				}
			}

			outDir := t.TempDir()
			lib := &config.Library{
				Name: tc.name,
				Cpp: &config.CppLibrary{
					CppDefault: config.CppDefault{
						ProductPath: "generator/integration_tests/golden/v1",
					},
				},
			}
			if err := Generate(t.Context(), model, outDir, lib); err != nil {
				t.Fatalf("Generate failed on parsed model: %v", err)
			}
		})
	}
}
