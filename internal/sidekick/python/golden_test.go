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

package python

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/sidekick/parser"
	"github.com/googleapis/librarian/internal/sources"
)

func TestGoldenParity(t *testing.T) {
	if _, err := exec.LookPath("protoc"); err != nil {
		t.Skip("skipping test because protoc is not installed")
	}

	testdataDir, err := filepath.Abs("testdata")
	if err != nil {
		t.Fatal(err)
	}
	protosDir := filepath.Join(testdataDir, "protos")
	goldenBaseDir := filepath.Join(testdataDir, "golden")

	for _, tc := range []struct {
		name           string
		specSource     string
		serviceConfig  string
		libraryName    string
		packageVersion string
		defaultVersion string
		goldenRelDir   string
	}{
		{
			name:           "credentials",
			specSource:     "google/iam/credentials/v1",
			serviceConfig:  "google/iam/credentials/v1/iamcredentials_v1.yaml",
			libraryName:    "google-iam-credentials",
			packageVersion: "0.0.0",
			defaultVersion: "v1",
			goldenRelDir:   "credentials",
		},
		{
			name:           "redis",
			specSource:     "google/cloud/redis/v1",
			serviceConfig:  "google/cloud/redis/v1/redis_v1.yaml",
			libraryName:    "google-cloud-redis",
			packageVersion: "0.0.0",
			defaultVersion: "v1",
			goldenRelDir:   "redis",
		},
		{
			name:           "asset",
			specSource:     "google/cloud/asset/v1",
			serviceConfig:  "google/cloud/asset/v1/cloudasset_v1.yaml",
			libraryName:    "google-cloud-asset",
			packageVersion: "1.2.99",
			defaultVersion: "v1",
			goldenRelDir:   "asset",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			outDir := t.TempDir()

			cfg := &parser.ModelConfig{
				Language:            config.LanguagePython,
				SpecificationFormat: config.SpecProtobuf,
				ServiceConfig:       tc.serviceConfig,
				SpecificationSource: tc.specSource,
				Source: &sources.SourceConfig{
					Sources: &sources.Sources{
						Googleapis: protosDir,
					},
					ActiveRoots: []string{"googleapis"},
				},
			}

			model, err := parser.CreateModel(cfg)
			if err != nil {
				t.Fatalf("parser.CreateModel failed: %v", err)
			}

			lib := &config.Library{
				Name:          tc.libraryName,
				Output:        outDir,
				Version:       tc.packageVersion,
				CopyrightYear: "2026",
				APIs: []*config.API{
					{Path: tc.specSource},
				},
				Python: &config.PythonPackage{
					DefaultVersion: tc.defaultVersion,
					PythonDefault: config.PythonDefault{
						Generator: config.PythonGeneratorSidekick,
					},
				},
			}

			if err := Generate(t.Context(), model, outDir, lib); err != nil {
				t.Fatalf("Generate failed: %v", err)
			}

			goldenDir := filepath.Join(goldenBaseDir, tc.goldenRelDir)
			emittedCount := 0
			err = filepath.WalkDir(outDir, func(path string, d fs.DirEntry, err error) error {
				if err != nil {
					return err
				}
				if d.IsDir() {
					return nil
				}
				relPath, err := filepath.Rel(outDir, path)
				if err != nil {
					return err
				}
				emittedCount++

				gotBytes, err := os.ReadFile(path)
				if err != nil {
					return fmt.Errorf("reading emitted file %s: %w", relPath, err)
				}

				goldenPath := filepath.Join(goldenDir, relPath)
				wantBytes, err := os.ReadFile(goldenPath)
				if err != nil {
					if errors.Is(err, os.ErrNotExist) {
						t.Errorf("emitted file %s does not exist in golden directory %s", relPath, goldenDir)
						return nil
					}
					return fmt.Errorf("reading golden file %s: %w", goldenPath, err)
				}

				if diff := cmp.Diff(string(wantBytes), string(gotBytes)); diff != "" {
					t.Errorf("diff in emitted file %s (-golden +got):\n%s", relPath, diff)
				}
				return nil
			})
			if err != nil {
				t.Fatalf("WalkDir failed: %v", err)
			}
			if emittedCount < 6 {
				t.Errorf("expected at least 6 emitted files for %s, got %d", tc.name, emittedCount)
			}
			t.Logf("[%s] Successfully verified %d emitted files against golden", tc.name, emittedCount)
		})
	}
}
