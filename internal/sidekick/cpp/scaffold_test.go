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
	"path/filepath"
	"strings"
	"testing"
)

func TestScaffold(t *testing.T) {
	outDir := t.TempDir()
	vars := &ScaffoldVars{
		CopyrightYear:       "2026",
		Title:               "Secret Manager API",
		Description:         "Manages secrets and operations using those secrets.",
		DocumentationURI:    "https://cloud.google.com/secret-manager",
		Library:             "secretmanager",
		ProductNamespace:    "secretmanager_v1",
		ServiceSubdirectory: "v1/",
		Directory:           "google/cloud/secretmanager/v1",
	}

	if err := Scaffold(outDir, vars, false); err != nil {
		t.Fatalf("Scaffold failed: %v", err)
	}

	expectedFiles := []string{
		"CMakeLists.txt",
		"BUILD.bazel",
		"README.md",
		"quickstart/quickstart.cc",
		"quickstart/CMakeLists.txt",
		"quickstart/BUILD.bazel",
		"quickstart/Makefile",
		"quickstart/README.md",
		"quickstart/.bazelrc",
	}

	for _, rel := range expectedFiles {
		path := filepath.Join(outDir, rel)
		data, err := os.ReadFile(path)
		if err != nil {
			t.Errorf("expected file %s to exist: %v", rel, err)
			continue
		}
		if len(data) == 0 {
			t.Errorf("file %s is empty", rel)
		}
	}

	// Verify content of CMakeLists.txt
	cmakeContent, _ := os.ReadFile(filepath.Join(outDir, "CMakeLists.txt"))
	if !strings.Contains(string(cmakeContent), "google_cloud_cpp_add_gapic_library(secretmanager \"Secret Manager API\"") {
		t.Errorf("CMakeLists.txt missing expected gapic library declaration:\n%s", string(cmakeContent))
	}

	// Verify content of BUILD.bazel
	bazelContent, _ := os.ReadFile(filepath.Join(outDir, "BUILD.bazel"))
	if !strings.Contains(string(bazelContent), "name = \"secretmanager\"") {
		t.Errorf("BUILD.bazel missing library target:\n%s", string(bazelContent))
	}

	// Verify non-overwrite behavior
	modified := "# Custom content"
	_ = os.WriteFile(filepath.Join(outDir, "CMakeLists.txt"), []byte(modified), 0644)
	if err := Scaffold(outDir, vars, false); err != nil {
		t.Fatalf("Scaffold with overwrite=false failed: %v", err)
	}
	contentAfter, _ := os.ReadFile(filepath.Join(outDir, "CMakeLists.txt"))
	if string(contentAfter) != modified {
		t.Errorf("expected CMakeLists.txt to not be overwritten when overwrite=false")
	}
}
