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

//go:build integration

package cpp_test

import (
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/librarian"
	"github.com/googleapis/librarian/internal/librarian/cpp"
)

const (
	googleapisCommit = "46403a9acec0719c130b33eb38b2ee62a45f9f6c"
	googleapisSHA256 = "8286d42d466aee4caa13891fee42b66c3208069ce797bd7445275884b8d47c02"
)

func locateGoogleCloudCpp(t *testing.T) string {
	t.Helper()
	if dir := os.Getenv("GOOGLE_CLOUD_CPP_DIR"); dir != "" {
		if _, err := os.Stat(dir); err == nil {
			return dir
		}
	}
	home, err := os.UserHomeDir()
	if err == nil {
		for _, candidate := range []string{
			filepath.Join(home, "google-cloud-cpp", "main"),
			filepath.Join(home, "google-cloud-cpp"),
		} {
			if _, err := os.Stat(candidate); err == nil {
				return candidate
			}
		}
	}
	for _, candidate := range []string{
		filepath.Join("..", "..", "..", "..", "google-cloud-cpp", "main"),
		filepath.Join("..", "..", "..", "..", "google-cloud-cpp"),
	} {
		if abs, err := filepath.Abs(candidate); err == nil {
			if _, err := os.Stat(abs); err == nil {
				return abs
			}
		}
	}
	return ""
}

func TestSecretManager_PilotParity(t *testing.T) {
	cppRepo := locateGoogleCloudCpp(t)
	if cppRepo == "" {
		t.Skip("skipping integration test: google-cloud-cpp repo not found")
	}

	srcs, err := librarian.LoadSources(t.Context(), &config.Sources{
		Googleapis: &config.Source{
			Commit: googleapisCommit,
			SHA256: googleapisSHA256,
		},
	})
	if err != nil {
		t.Fatalf("LoadSources failed: %v", err)
	}

	tmpDir := t.TempDir()

	// Copy root .clang-format so emitted files match google-cloud-cpp formatting conventions.
	clangFormatSrc := filepath.Join(cppRepo, ".clang-format")
	if data, err := os.ReadFile(clangFormatSrc); err == nil {
		if err := os.WriteFile(filepath.Join(tmpDir, ".clang-format"), data, 0o644); err != nil {
			t.Fatalf("failed to copy .clang-format: %v", err)
		}
	}

	outDir := filepath.Join(tmpDir, "google", "cloud", "secretmanager", "v1")
	lib := &config.Library{
		Name:   "google-cloud-secretmanager-v1",
		Output: outDir,
		APIs: []*config.API{
			{Path: "google/cloud/secretmanager/v1/service.proto"},
		},
		Cpp: &config.CppLibrary{
			ProductPath:           "google/cloud/secretmanager/v1",
			ForwardingProductPath: "google/cloud/secretmanager",
			InitialCopyrightYear:  "2021",
			RetryableStatusCodes:  []string{"kUnavailable"},
		},
	}
	cfg := &config.Config{
		Language: config.LanguageCpp,
	}

	if err := cpp.Generate(t.Context(), cfg, lib, srcs); err != nil {
		t.Fatalf("cpp.Generate failed: %v", err)
	}

	if err := cpp.Format(t.Context(), cfg, lib); err != nil {
		t.Fatalf("cpp.Format failed: %v", err)
	}

	// Verify all 28 versioned files
	var generatedVersionedFiles []string
	err = filepath.WalkDir(outDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d == nil || d.IsDir() {
			return nil
		}
		ext := filepath.Ext(path)
		if ext == ".h" || ext == ".cc" {
			rel, _ := filepath.Rel(outDir, path)
			generatedVersionedFiles = append(generatedVersionedFiles, rel)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walking generated files: %v", err)
	}

	if len(generatedVersionedFiles) != 28 {
		t.Fatalf("expected 28 generated versioned files, got %d:\n%v", len(generatedVersionedFiles), generatedVersionedFiles)
	}

	refV1Dir := filepath.Join(cppRepo, "google", "cloud", "secretmanager", "v1")
	for _, relPath := range generatedVersionedFiles {
		t.Run("v1/"+relPath, func(t *testing.T) {
			gotBytes, err := os.ReadFile(filepath.Join(outDir, relPath))
			if err != nil {
				t.Fatalf("failed to read generated file: %v", err)
			}
			refPath := filepath.Join(refV1Dir, relPath)
			wantBytes, err := os.ReadFile(refPath)
			if err != nil {
				t.Fatalf("failed to read reference file %s: %v", refPath, err)
			}

			if diff := cmp.Diff(string(wantBytes), string(gotBytes)); diff != "" {
				t.Errorf("byte-for-byte mismatch (-want +got):\n%s", diff)
			}
		})
	}

	// Verify forwarding headers
	fwdDir := filepath.Join(tmpDir, "google", "cloud", "secretmanager")
	refFwdDir := filepath.Join(cppRepo, "google", "cloud", "secretmanager")

	expectedForwarding := []string{
		"secret_manager_client.h",
		"secret_manager_connection.h",
		"secret_manager_connection_idempotency_policy.h",
		"secret_manager_options.h",
		filepath.Join("mocks", "mock_secret_manager_connection.h"),
	}

	for _, relPath := range expectedForwarding {
		t.Run("forwarding/"+relPath, func(t *testing.T) {
			gotBytes, err := os.ReadFile(filepath.Join(fwdDir, relPath))
			if err != nil {
				t.Fatalf("failed to read generated forwarding file %s: %v", relPath, err)
			}
			refPath := filepath.Join(refFwdDir, relPath)
			wantBytes, err := os.ReadFile(refPath)
			if err != nil {
				t.Fatalf("failed to read reference forwarding file %s: %v", refPath, err)
			}

			if diff := cmp.Diff(string(wantBytes), string(gotBytes)); diff != "" {
				t.Errorf("byte-for-byte mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
