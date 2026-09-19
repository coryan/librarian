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
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/sidekick/api"
)

func TestGenerate(t *testing.T) {
	outdir := t.TempDir()

	msg := api.NewTestMessage("Secret").WithFields(
		api.NewTestField("name").WithType(api.TypezString),
	)
	svc := api.NewTestService("SecretManagerService")
	model := api.NewTestAPI([]*api.Message{msg}, nil, []*api.Service{svc})
	model.Name = "google-cloud-secretmanager"

	lib := &config.Library{
		Name:          "google-cloud-secretmanager",
		Version:       "2.1.0",
		CopyrightYear: "2026",
		APIs: []*config.API{
			{Path: "google/cloud/secretmanager/v1"},
		},
		Python: &config.PythonPackage{
			DefaultVersion: "v1",
		},
	}

	if err := Generate(t.Context(), model, outdir, lib); err != nil {
		t.Fatalf("Generate() failed: %v", err)
	}

	// Verify gapic_version.py
	versionFile := filepath.Join(outdir, "google", "cloud", "secretmanager_v1", "gapic_version.py")
	versionBytes, err := os.ReadFile(versionFile)
	if err != nil {
		t.Fatalf("reading gapic_version.py failed: %v", err)
	}
	wantVersion := `# -*- coding: utf-8 -*-
# Copyright 2026 Google LLC
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#
__version__ = "2.1.0"  # {x-release-please-version}
`
	if diff := cmp.Diff(wantVersion, string(versionBytes)); diff != "" {
		t.Errorf("gapic_version.py mismatch (-want +got):\n%s", diff)
	}

	// Verify root package gapic_version.py
	rootVersionFile := filepath.Join(outdir, "google", "cloud", "secretmanager", "gapic_version.py")
	rootVersionBytes, err := os.ReadFile(rootVersionFile)
	if err != nil {
		t.Fatalf("reading root gapic_version.py failed: %v", err)
	}
	if diff := cmp.Diff(wantVersion, string(rootVersionBytes)); diff != "" {
		t.Errorf("root gapic_version.py mismatch (-want +got):\n%s", diff)
	}

	// Verify py.typed
	typedFile := filepath.Join(outdir, "google", "cloud", "secretmanager_v1", "py.typed")
	typedBytes, err := os.ReadFile(typedFile)
	if err != nil {
		t.Fatalf("reading py.typed failed: %v", err)
	}
	wantTyped := `# Marker file for PEP 561.
# The google-cloud-secretmanager package uses inline types.
`
	if diff := cmp.Diff(wantTyped, string(typedBytes)); diff != "" {
		t.Errorf("py.typed mismatch (-want +got):\n%s", diff)
	}

	// Verify root package py.typed
	rootTypedFile := filepath.Join(outdir, "google", "cloud", "secretmanager", "py.typed")
	rootTypedBytes, err := os.ReadFile(rootTypedFile)
	if err != nil {
		t.Fatalf("reading root py.typed failed: %v", err)
	}
	if diff := cmp.Diff(wantTyped, string(rootTypedBytes)); diff != "" {
		t.Errorf("root py.typed mismatch (-want +got):\n%s", diff)
	}

	// Verify _compat.py
	compatFile := filepath.Join(outdir, "google", "cloud", "secretmanager_v1", "_compat.py")
	compatBytes, err := os.ReadFile(compatFile)
	if err != nil {
		t.Fatalf("reading _compat.py failed: %v", err)
	}
	if len(compatBytes) == 0 {
		t.Errorf("_compat.py is empty")
	}
}
