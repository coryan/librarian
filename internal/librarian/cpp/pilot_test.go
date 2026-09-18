//go:build integration

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

package cpp_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/librarian"
	"github.com/googleapis/librarian/internal/librarian/cpp"
	"github.com/googleapis/librarian/internal/yaml"
)

func findProductionRoot() string {
	if dir := os.Getenv("GOOGLE_CLOUD_CPP_DIR"); dir != "" {
		if _, err := os.Stat(dir); err == nil {
			return dir
		}
	}
	candidates := []string{
		"../../../../../google-cloud-cpp/main",
		"../../../../../google-cloud-cpp",
		"../../../../google-cloud-cpp/main",
		"../../../../google-cloud-cpp",
		"../../../google-cloud-cpp/main",
	}
	for _, c := range candidates {
		if _, err := os.Stat(filepath.Join(c, "google/cloud/secretmanager")); err == nil {
			abs, err := filepath.Abs(c)
			if err == nil {
				return abs
			}
			return c
		}
	}
	return ""
}

func TestPilotSecretManagerParity(t *testing.T) {
	if _, err := exec.LookPath("protoc"); err != nil {
		t.Skip("skipping test because protoc is not installed")
	}

	productionRoot := findProductionRoot()
	if productionRoot == "" {
		t.Skip("skipping test because production google-cloud-cpp not found")
	}

	cfg, err := yaml.Read[config.Config]("testdata/librarian.yaml")
	if err != nil {
		t.Fatalf("failed to read testdata/librarian.yaml: %v", err)
	}

	src, err := librarian.LoadSources(t.Context(), cfg.Sources)
	if err != nil {
		t.Fatalf("failed to load sources: %v", err)
	}

	if len(cfg.Libraries) == 0 {
		t.Fatal("expected at least one library in testdata/librarian.yaml")
	}

	tempDir := t.TempDir()
	outDir := filepath.Join(tempDir, "google/cloud/secretmanager")

	lib := cfg.Libraries[0]
	lib.Output = outDir

	if err := cpp.Generate(t.Context(), cfg, lib, src); err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Copy .clang-format from production to tempDir so clang-format uses project rules
	clangFormatSrc := filepath.Join(productionRoot, ".clang-format")
	if data, err := os.ReadFile(clangFormatSrc); err == nil {
		_ = os.WriteFile(filepath.Join(tempDir, ".clang-format"), data, 0644)
	}

	if err := cpp.Format(t.Context(), lib); err != nil {
		t.Fatalf("Format failed: %v", err)
	}

	// Compare generated files against production
	var generatedFiles []string
	err = filepath.Walk(tempDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		rel, err := filepath.Rel(tempDir, path)
		if err != nil {
			return err
		}
		// Skip CMakeLists, BUILD, etc. if comparing generated C++ code
		if strings.HasSuffix(rel, ".h") || strings.HasSuffix(rel, ".cc") {
			generatedFiles = append(generatedFiles, rel)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	if len(generatedFiles) == 0 {
		t.Fatal("no C++ files generated")
	}

	var mismatches []string
	for _, rel := range generatedFiles {
		gotPath := filepath.Join(tempDir, rel)
		gotBytes, err := os.ReadFile(gotPath)
		if err != nil {
			t.Errorf("failed reading generated %s: %v", rel, err)
			continue
		}

		prodPath := filepath.Join(productionRoot, rel)
		prodBytes, err := os.ReadFile(prodPath)
		if err != nil {
			mismatches = append(mismatches, rel+" (missing in production)")
			continue
		}

		if diff := cmp.Diff(string(prodBytes), string(gotBytes)); diff != "" {
			mismatches = append(mismatches, rel)
			t.Logf("diff in %s (-production +generated):\n%s", rel, diff)
		}
	}

	if len(mismatches) > 0 {
		t.Errorf("found %d mismatches between generated and production secretmanager:\n%s",
			len(mismatches), strings.Join(mismatches, "\n"))
	}
}
