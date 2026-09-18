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
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/sidekick/api"
	"github.com/googleapis/librarian/internal/sidekick/parser"
	"github.com/googleapis/librarian/internal/sources"
)

func TestLibraryToModelConfig(t *testing.T) {
	lib := &config.Library{
		Name:          "golden_kitchen_sink",
		TitleOverride: "Golden Kitchen Sink API",
		Cpp: &config.CppLibrary{
			CppDefault: config.CppDefault{
				ProductPath: "generator/integration_tests/golden/v1",
			},
			OverrideServiceConfigYamlName: "generator/integration_tests/test.yaml",
			AdditionalProtoFiles: []string{
				"generator/integration_tests/backup.proto",
			},
			OmittedServices: []string{
				"DeprecatedService",
			},
		},
	}
	apiCfg := &config.API{
		Path: "generator/integration_tests/test.proto",
	}

	got := libraryToModelConfig(lib, apiCfg, nil, nil)

	want := &parser.ModelConfig{
		Language:            config.LanguageCpp,
		SpecificationFormat: config.SpecProtobuf,
		SpecificationSource: "generator/integration_tests",
		ServiceConfig:       "generator/integration_tests/test.yaml",
		Override: api.ModelOverride{
			Title: "Golden Kitchen Sink API",
			SkippedIDs: []string{
				"DeprecatedService",
			},
		},
	}

	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestFormat(t *testing.T) {
	tempDir := t.TempDir()
	hPath := filepath.Join(tempDir, "test.h")
	unformatted := "class   Bar   { public:   void foo ( ) ; } ;"
	if err := os.WriteFile(hPath, []byte(unformatted), 0644); err != nil {
		t.Fatal(err)
	}

	lib := &config.Library{
		Name:   "test_format",
		Output: tempDir,
	}

	if err := Format(t.Context(), lib); err != nil {
		t.Fatalf("Format failed: %v", err)
	}

	content, err := os.ReadFile(hPath)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(content), "class   Bar") {
		t.Errorf("expected file to be formatted, got:\n%s", string(content))
	}
}

func TestGenerate_NoAPIs(t *testing.T) {
	lib := &config.Library{
		Name: "empty_api_lib",
		APIs: nil,
	}
	if err := Generate(t.Context(), nil, lib, nil); err == nil {
		t.Errorf("expected error when library has no APIs, got nil")
	}
}

func TestGenerate_WithProtobuf(t *testing.T) {
	if _, err := exec.LookPath("protoc"); err != nil {
		t.Skip("skipping test because protoc is not installed")
	}

	testdataDir, err := filepath.Abs("../../sidekick/cpp/testdata")
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

	outDir := t.TempDir()
	lib := &config.Library{
		Name:   "golden_rest_only",
		Output: outDir,
		APIs: []*config.API{
			{Path: "generator/integration_tests/test2.proto"},
		},
		Roots: []string{"googleapis", "showcase"},
		Cpp: &config.CppLibrary{
			CppDefault: config.CppDefault{
				ProductPath: "generator/integration_tests/golden/v1",
			},
		},
	}

	cfg := &config.Config{
		Language: config.LanguageCpp,
	}

	if err := Generate(t.Context(), cfg, lib, src); err != nil {
		t.Fatalf("Generate failed on test2.proto: %v", err)
	}
}
