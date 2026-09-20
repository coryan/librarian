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
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/sidekick/api"
	"github.com/googleapis/librarian/internal/sidekick/language"
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

	// Verify services/__init__.py
	servicesInitFile := filepath.Join(outdir, "google", "cloud", "secretmanager_v1", "services", "__init__.py")
	servicesInitBytes, err := os.ReadFile(servicesInitFile)
	if err != nil {
		t.Fatalf("reading services/__init__.py failed: %v", err)
	}
	wantServicesInit := `# -*- coding: utf-8 -*-
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
`
	if diff := cmp.Diff(wantServicesInit, string(servicesInitBytes)); diff != "" {
		t.Errorf("services/__init__.py mismatch (-want +got):\n%s", diff)
	}

	// Verify services/secret_manager_service/__init__.py
	svcInitFile := filepath.Join(outdir, "google", "cloud", "secretmanager_v1", "services", "secret_manager_service", "__init__.py")
	svcInitBytes, err := os.ReadFile(svcInitFile)
	if err != nil {
		t.Fatalf("reading service __init__.py failed: %v", err)
	}
	wantSvcInit := `# -*- coding: utf-8 -*-
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
from .client import SecretManagerServiceClient
from .async_client import SecretManagerServiceAsyncClient

__all__ = (
    'SecretManagerServiceClient',
    'SecretManagerServiceAsyncClient',
)
`
	if diff := cmp.Diff(wantSvcInit, string(svcInitBytes)); diff != "" {
		t.Errorf("service __init__.py mismatch (-want +got):\n%s", diff)
	}

	// Verify services/secret_manager_service/transports/README.rst
	readmeFile := filepath.Join(outdir, "google", "cloud", "secretmanager_v1", "services", "secret_manager_service", "transports", "README.rst")
	readmeBytes, err := os.ReadFile(readmeFile)
	if err != nil {
		t.Fatalf("reading transports/README.rst failed: %v", err)
	}
	wantReadme := `
transport inheritance structure
_______________________________

` + "``SecretManagerServiceTransport`` is the ABC for all transports." + `

- public child ` + "``SecretManagerServiceGrpcTransport`` for sync gRPC transport (defined in ``grpc.py``)." + `
- public child ` + "``SecretManagerServiceGrpcAsyncIOTransport`` for async gRPC transport (defined in ``grpc_asyncio.py``)." + `
- private child ` + "``_BaseSecretManagerServiceRestTransport`` for base REST transport with inner classes ``_BaseMETHOD`` (defined in ``rest_base.py``)." + `
- public child ` + "``SecretManagerServiceRestTransport`` for sync REST transport with inner classes ``METHOD`` derived from the parent's corresponding ``_BaseMETHOD`` classes (defined in ``rest.py``)." + `
`
	if diff := cmp.Diff(wantReadme, string(readmeBytes)); diff != "" {
		t.Errorf("transports/README.rst mismatch (-want +got):\n%s", diff)
	}
}

func TestValidateOutputContainment(t *testing.T) {
	outdir := t.TempDir()
	files := []language.GeneratedFile{
		{OutputPath: "google/cloud/secretmanager_v1/gapic_version.py"},
		{OutputPath: "google/cloud/secretmanager_v1/py.typed"},
	}
	if err := validateOutputContainment(outdir, files); err != nil {
		t.Errorf("validateOutputContainment() error = %v, want nil", err)
	}
}

func TestValidateOutputContainment_Error(t *testing.T) {
	outdir := t.TempDir()

	for _, test := range []struct {
		name    string
		files   []language.GeneratedFile
		wantErr error
	}{
		{
			name: "empty output path",
			files: []language.GeneratedFile{
				{OutputPath: ""},
			},
			wantErr: ErrInvalidOutputPath,
		},
		{
			name: "absolute output path",
			files: []language.GeneratedFile{
				{OutputPath: filepath.Join(outdir, "escaped.py")},
			},
			wantErr: ErrInvalidOutputPath,
		},
		{
			name: "file escaping outdir via ..",
			files: []language.GeneratedFile{
				{OutputPath: "../escaped.py"},
			},
			wantErr: ErrEscapeOutputDir,
		},
		{
			name: "file escaping outdir deeply",
			files: []language.GeneratedFile{
				{OutputPath: "google/../../../escaped.py"},
			},
			wantErr: ErrEscapeOutputDir,
		},
		{
			name: "duplicate output path",
			files: []language.GeneratedFile{
				{OutputPath: "google/cloud/secretmanager_v1/py.typed"},
				{OutputPath: "google/cloud/secretmanager_v1/py.typed"},
			},
			wantErr: ErrDuplicateOutputPath,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			err := validateOutputContainment(outdir, test.files)
			if !errors.Is(err, test.wantErr) {
				t.Errorf("validateOutputContainment() error = %v, want %v", err, test.wantErr)
			}
		})
	}
}
