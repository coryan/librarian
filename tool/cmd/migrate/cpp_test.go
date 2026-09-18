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

package main

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/googleapis/librarian/internal/config"
)

func TestRunCppMigration(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	fetchSource = func(ctx context.Context) (*config.Source, error) {
		return &config.Source{
			Commit: "abcd123",
			SHA256: "sha123",
			Dir:    filepath.Join(wd, "../../internal/testdata/googleapis"),
		}, nil
	}
	for _, test := range []struct {
		name     string
		repoPath string
		wantErr  error
	}{
		{
			name:     "success",
			repoPath: "testdata/run/success-cpp",
		},
		{
			name:     "missing_file",
			repoPath: "testdata/run/no-config",
			wantErr:  fs.ErrNotExist,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			dir := t.TempDir()
			if err := os.CopyFS(dir, os.DirFS(test.repoPath)); err != nil {
				t.Fatal(err)
			}
			err := runCppMigration(t.Context(), dir)
			if !errors.Is(err, test.wantErr) {
				t.Fatalf("expected error %v, got %v", test.wantErr, err)
			}
			if test.wantErr == nil {
				yamlPath := filepath.Join(dir, config.LibrarianYAML)
				if _, err := os.Stat(yamlPath); err != nil {
					t.Errorf("librarian.yaml was not created: %v", err)
				}
			}
		})
	}
}

func TestRunCppMigration_Production(t *testing.T) {
	prodPath := filepath.Join("..", "..", "..", "internal", "sidekick", "cpp", "testdata", "generator_config.textproto")
	if _, err := os.Stat(prodPath); err != nil {
		t.Skip("skipping production migration test; generator_config.textproto not found")
	}

	fetchSource = func(ctx context.Context) (*config.Source, error) {
		return &config.Source{
			Commit: "46403a9acec0719c130b33eb38b2ee62a45f9f6c",
			SHA256: "sha123",
		}, nil
	}

	dir := t.TempDir()
	genDir := filepath.Join(dir, "generator")
	if err := os.MkdirAll(genDir, 0755); err != nil {
		t.Fatal(err)
	}
	content, err := os.ReadFile(prodPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(genDir, "generator_config.textproto"), content, 0644); err != nil {
		t.Fatal(err)
	}

	if err := runCppMigration(t.Context(), dir); err != nil {
		t.Fatalf("runCppMigration failed: %v", err)
	}

	yamlPath := filepath.Join(dir, config.LibrarianYAML)
	if _, err := os.Stat(yamlPath); err != nil {
		t.Fatalf("librarian.yaml was not created: %v", err)
	}
}
