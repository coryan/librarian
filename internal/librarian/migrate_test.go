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

package librarian

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/yaml"
)

func TestMigrateCppConfig_Stdout(t *testing.T) {
	tempDir := t.TempDir()
	textprotoPath := filepath.Join(tempDir, "test.textproto")
	content := `
service {
  service_proto_path: "google/cloud/secretmanager/v1/service.proto"
  product_path: "google/cloud/secretmanager/v1"
  forwarding_product_path: "google/cloud/secretmanager"
  initial_copyright_year: "2021"
}
`
	if err := os.WriteFile(textprotoPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	if err := runMigrateCppConfig(&buf, textprotoPath, ""); err != nil {
		t.Fatalf("runMigrateCppConfig failed: %v", err)
	}

	got := buf.String()
	if !strings.Contains(got, "language: cpp") {
		t.Errorf("expected language: cpp in output, got:\n%s", got)
	}
	if !strings.Contains(got, "google-cloud-secretmanager-v1") {
		t.Errorf("expected library name in output, got:\n%s", got)
	}
}

func TestMigrateCppConfig_OutputFile(t *testing.T) {
	tempDir := t.TempDir()
	textprotoPath := filepath.Join(tempDir, "test.textproto")
	yamlPath := filepath.Join(tempDir, "librarian.yaml")
	content := `
service {
  service_proto_path: "google/cloud/secretmanager/v1/service.proto"
  product_path: "google/cloud/secretmanager/v1"
  forwarding_product_path: "google/cloud/secretmanager"
  initial_copyright_year: "2021"
}
`
	if err := os.WriteFile(textprotoPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	if err := runMigrateCppConfig(&buf, textprotoPath, yamlPath); err != nil {
		t.Fatalf("runMigrateCppConfig failed: %v", err)
	}

	cfg, err := yaml.Read[config.Config](yamlPath)
	if err != nil {
		t.Fatalf("failed to read generated librarian.yaml: %v", err)
	}
	if cfg.Language != config.LanguageCpp {
		t.Errorf("expected language cpp, got %s", cfg.Language)
	}
	if len(cfg.Libraries) != 1 {
		t.Fatalf("expected 1 library, got %d", len(cfg.Libraries))
	}
	if cfg.Libraries[0].Name != "google-cloud-secretmanager-v1" {
		t.Errorf("expected google-cloud-secretmanager-v1, got %s", cfg.Libraries[0].Name)
	}
}

func TestMigrateCppConfig_CLI(t *testing.T) {
	tempDir := t.TempDir()
	textprotoPath := filepath.Join(tempDir, "test.textproto")
	yamlPath := filepath.Join(tempDir, "librarian.yaml")
	content := `
service {
  service_proto_path: "google/cloud/secretmanager/v1/service.proto"
  product_path: "google/cloud/secretmanager/v1"
  initial_copyright_year: "2021"
}
`
	if err := os.WriteFile(textprotoPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	args := []string{"librarian", "migrate", "cpp-config", textprotoPath, "-o", yamlPath}
	if err := Run(t.Context(), args...); err != nil {
		t.Fatalf("librarian migrate cpp-config CLI failed: %v", err)
	}

	if _, err := os.Stat(yamlPath); err != nil {
		t.Fatalf("expected generated file %s to exist: %v", yamlPath, err)
	}
}

func TestMigrateCppConfig_MissingInput(t *testing.T) {
	args := []string{"librarian", "migrate", "cpp-config"}
	if err := Run(t.Context(), args...); err == nil {
		t.Errorf("expected error when missing input textproto, got nil")
	}
}
