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
	"github.com/googleapis/librarian/internal/sources"
)

func TestPilotSecretManagerParity(t *testing.T) {
	if _, err := exec.LookPath("protoc"); err != nil {
		t.Skip("skipping test because protoc is not installed")
	}

	googleapisCache := "/usr/local/google/home/coryan/.cache/librarian/github.com/googleapis/googleapis@0db4dc67dd805d20294c6dc34068c37f546d71da"
	if _, err := os.Stat(googleapisCache); err != nil {
		t.Skipf("skipping test because googleapis cache not found: %v", err)
	}

	productionRoot := "/usr/local/google/home/coryan/google-cloud-cpp/main"
	if _, err := os.Stat(productionRoot); err != nil {
		t.Skipf("skipping test because production google-cloud-cpp not found: %v", err)
	}

	tempDir := t.TempDir()
	outDir := filepath.Join(tempDir, "google/cloud/secretmanager")

	lib := &config.Library{
		Name:          "google-cloud-secretmanager-v1",
		CopyrightYear: "2021",
		Output:        outDir,
		APIs: []*config.API{
			{Path: "google/cloud/secretmanager/v1/service.proto"},
		},
		Roots: []string{"googleapis"},
		Cpp: &config.CppLibrary{
			CppDefault: config.CppDefault{
				ProductPath:           "google/cloud/secretmanager/v1",
				ForwardingProductPath: "google/cloud/secretmanager",
				InitialCopyrightYear:  "2021",
				RetryableStatusCodes:  []string{"kUnavailable"},
			},
		},
	}

	cfg := &config.Config{
		Language: config.LanguageCpp,
	}
	src := &sources.Sources{
		Googleapis: googleapisCache,
	}

	if err := Generate(t.Context(), cfg, lib, src); err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Copy .clang-format from production to tempDir so clang-format uses project rules
	clangFormatSrc := filepath.Join(productionRoot, ".clang-format")
	if data, err := os.ReadFile(clangFormatSrc); err == nil {
		_ = os.WriteFile(filepath.Join(tempDir, ".clang-format"), data, 0644)
	}

	if err := Format(t.Context(), lib); err != nil {
		t.Fatalf("Format failed: %v", err)
	}

	// Compare generated files against production
	var generatedFiles []string
	err := filepath.Walk(tempDir, func(path string, info os.FileInfo, err error) error {
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
