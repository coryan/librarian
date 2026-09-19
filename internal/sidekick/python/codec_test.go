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
	"testing"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/sidekick/api"
)

func newTestCodec(t *testing.T, model *api.API, library *config.Library) *codec {
	t.Helper()
	c, err := newCodec(model, library, "")
	if err != nil {
		t.Fatalf("newCodec() failed: %v", err)
	}
	return c
}

func TestNewCodec(t *testing.T) {
	model := api.NewTestAPI(nil, nil, nil)
	model.Name = "google-cloud-secretmanager"

	for _, test := range []struct {
		name    string
		library *config.Library
		want    *codec
	}{
		{
			name: "derived from library configuration",
			library: &config.Library{
				Name:          "google-cloud-secretmanager",
				Version:       "2.1.0",
				CopyrightYear: "2026",
				APIs: []*config.API{
					{Path: "google/cloud/secretmanager/v1"},
				},
				Python: &config.PythonPackage{
					DefaultVersion: "v1",
				},
			},
			want: &codec{
				GenerationYear: "2026",
				PackageName:    "google-cloud-secretmanager",
				PackageVersion: "2.1.0",
				DefaultVersion: "v1",
				GAPICNamespace: "google.cloud",
				GAPICName:      "secretmanager",
				CurrentVersion: "v1",
			},
		},
		{
			name: "fallback to model name when library name is empty",
			library: &config.Library{
				CopyrightYear: "2026",
			},
			want: &codec{
				GenerationYear: "2026",
				PackageName:    "google-cloud-secretmanager",
				PackageVersion: "0.0.0",
			},
		},
		{
			name: "multi-API library resolves matching API",
			library: &config.Library{
				Name: "google-cloud-speech",
				APIs: []*config.API{
					{Path: "google/cloud/speech/v1"},
					{Path: "google/cloud/speech/v1p1beta1"},
				},
				Python: &config.PythonPackage{
					DefaultVersion: "v1",
				},
			},
			want: &codec{
				GenerationYear: "2026",
				PackageName:    "google-cloud-speech",
				PackageVersion: "0.0.0",
				DefaultVersion: "v1",
				CurrentVersion: "v1p1beta1",
				GAPICNamespace: "google.cloud",
				GAPICName:      "speech",
			},
		},
		{
			name:    "fallback to model.PackageName when library is nil",
			library: nil,
			want: &codec{
				GenerationYear: "2026",
				PackageName:    "google-cloud-speech",
				PackageVersion: "0.0.0",
				DefaultVersion: "v1p1beta1",
				CurrentVersion: "v1p1beta1",
				GAPICNamespace: "google.cloud",
				GAPICName:      "speech",
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			m := api.NewTestAPI(nil, nil, nil)
			m.Name = "google-cloud-secretmanager"
			if test.name == "multi-API library resolves matching API" || test.name == "fallback to model.PackageName when library is nil" {
				m.Name = "google-cloud-speech"
				m.PackageName = "google.cloud.speech.v1p1beta1"
			}
			got, err := newCodec(m, test.library, "")
			if err != nil {
				t.Fatalf("newCodec() error = %v", err)
			}
			if test.want.PackageName != got.PackageName {
				t.Errorf("PackageName = %q, want %q", got.PackageName, test.want.PackageName)
			}
			if test.want.PackageVersion != got.PackageVersion {
				t.Errorf("PackageVersion = %q, want %q", got.PackageVersion, test.want.PackageVersion)
			}
			if test.want.GenerationYear != got.GenerationYear {
				t.Errorf("GenerationYear = %q, want %q", got.GenerationYear, test.want.GenerationYear)
			}
			if test.want.GAPICNamespace != got.GAPICNamespace {
				t.Errorf("GAPICNamespace = %q, want %q", got.GAPICNamespace, test.want.GAPICNamespace)
			}
			if test.want.GAPICName != got.GAPICName {
				t.Errorf("GAPICName = %q, want %q", got.GAPICName, test.want.GAPICName)
			}
			if test.want.CurrentVersion != got.CurrentVersion {
				t.Errorf("CurrentVersion = %q, want %q", got.CurrentVersion, test.want.CurrentVersion)
			}
		})
	}
}

func TestPackageDir(t *testing.T) {
	for _, test := range []struct {
		name      string
		namespace string
		gapicName string
		version   string
		wantPkg   string
		wantRoot  string
	}{
		{
			name:      "secretmanager v1",
			namespace: "google.cloud",
			gapicName: "secretmanager",
			version:   "v1",
			wantPkg:   "google/cloud/secretmanager_v1",
			wantRoot:  "google/cloud/secretmanager",
		},
		{
			name:      "iam credentials v1",
			namespace: "google.iam",
			gapicName: "credentials",
			version:   "v1",
			wantPkg:   "google/iam/credentials_v1",
			wantRoot:  "google/iam/credentials",
		},
		{
			name:      "empty",
			namespace: "",
			gapicName: "",
			wantPkg:   "",
			wantRoot:  "",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			c := &codec{
				GAPICNamespace: test.namespace,
				GAPICName:      test.gapicName,
				DefaultVersion: test.version,
			}
			if got := c.packageDir(); got != test.wantPkg {
				t.Errorf("packageDir() = %q, want %q", got, test.wantPkg)
			}
			if got := c.rootPackageDir(); got != test.wantRoot {
				t.Errorf("rootPackageDir() = %q, want %q", got, test.wantRoot)
			}
		})
	}
}

func TestNewCodec_Error(t *testing.T) {
	_, err := newCodec(nil, nil, "")
	if err == nil {
		t.Errorf("newCodec(nil) error = nil, want non-nil")
	}
}
