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
	"testing"
)

func TestParseProductPath(t *testing.T) {
	tests := []struct {
		input       string
		wantPrefix  string
		wantLibrary string
		wantSubdir  string
	}{
		{
			input:       "google/cloud/secretmanager/v1",
			wantPrefix:  "google/cloud",
			wantLibrary: "secretmanager",
			wantSubdir:  "v1",
		},
		{
			input:       "google/cloud/test",
			wantPrefix:  "google/cloud",
			wantLibrary: "test",
			wantSubdir:  "",
		},
		{
			input:       "google/cloud/test/",
			wantPrefix:  "google/cloud",
			wantLibrary: "test",
			wantSubdir:  "",
		},
		{
			input:       "google/cloud/test/v1",
			wantPrefix:  "google/cloud",
			wantLibrary: "test",
			wantSubdir:  "v1",
		},
		{
			input:       "google/cloud/test/foo/v1",
			wantPrefix:  "google/cloud",
			wantLibrary: "test",
			wantSubdir:  "foo/v1",
		},
		{
			input:       "google/cloud/compute/addresses/v1",
			wantPrefix:  "google/cloud",
			wantLibrary: "compute",
			wantSubdir:  "addresses/v1",
		},
		{
			input:       "generator/integration_tests/golden/v1",
			wantPrefix:  "generator/integration_tests",
			wantLibrary: "golden",
			wantSubdir:  "v1",
		},
		{
			input:       "generator/integration_tests/golden/v1/",
			wantPrefix:  "generator/integration_tests",
			wantLibrary: "golden",
			wantSubdir:  "v1",
		},
		{
			input:       "generator/integration_tests/golden",
			wantPrefix:  "generator/integration_tests",
			wantLibrary: "golden",
			wantSubdir:  "",
		},
		{
			input:       "blah/golden",
			wantPrefix:  "blah",
			wantLibrary: "golden",
			wantSubdir:  "",
		},
		{
			input:       "blah/golden/v1",
			wantPrefix:  "blah",
			wantLibrary: "golden",
			wantSubdir:  "v1",
		},
		{
			input:       "golden",
			wantPrefix:  "",
			wantLibrary: "golden",
			wantSubdir:  "",
		},
		{
			input:       "foo/bar/service",
			wantPrefix:  "foo/bar",
			wantLibrary: "service",
			wantSubdir:  "",
		},
		{
			input:       "service",
			wantPrefix:  "",
			wantLibrary: "service",
			wantSubdir:  "",
		},
		{
			input:       "/google/cloud/secretmanager/v1/",
			wantPrefix:  "google/cloud",
			wantLibrary: "secretmanager",
			wantSubdir:  "v1",
		},
		{
			input:       `google\cloud\secretmanager\v1`,
			wantPrefix:  "google/cloud",
			wantLibrary: "secretmanager",
			wantSubdir:  "v1",
		},
		{
			input:       `generator\integration_tests\golden\v1`,
			wantPrefix:  "generator/integration_tests",
			wantLibrary: "golden",
			wantSubdir:  "v1",
		},
		{
			input:       "google//cloud///secretmanager/v1",
			wantPrefix:  "google/cloud",
			wantLibrary: "secretmanager",
			wantSubdir:  "v1",
		},
		{
			input:       "",
			wantPrefix:  "",
			wantLibrary: "",
			wantSubdir:  "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			got := parseProductPath(tc.input)
			if got.prefix != tc.wantPrefix {
				t.Errorf("prefix: want %q, got %q", tc.wantPrefix, got.prefix)
			}
			if got.libraryName != tc.wantLibrary {
				t.Errorf("libraryName: want %q, got %q", tc.wantLibrary, got.libraryName)
			}
			if got.serviceSubdirectory != tc.wantSubdir {
				t.Errorf("serviceSubdirectory: want %q, got %q", tc.wantSubdir, got.serviceSubdirectory)
			}
		})
	}
}

func TestDeriveNamespace(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"google/cloud/test", "test"},
		{"google/cloud/test/", "test"},
		{"google/cloud/test/v1", "test_v1"},
		{"google/cloud/test/v1/", "test_v1"},
		{"google/cloud/test/foo/v1", "test_foo_v1"},
		{"generator/integration_tests/golden/v1", "golden_v1"},
		{"generator/integration_tests/golden", "golden"},
		{"blah/golden", "golden"},
		{"blah/golden/v1", "golden_v1"},
		{"foo/bar/service", "service"},
		{"google/cloud/secretmanager/v1", "secretmanager_v1"},
		{"google/cloud/compute/addresses/v1", "compute_addresses_v1"},
		{"", ""},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			got := deriveNamespace(tc.input)
			if got != tc.want {
				t.Errorf("deriveNamespace(%q): want %q, got %q", tc.input, tc.want, got)
			}
		})
	}
}

func TestServiceAnnotations_Namespaces(t *testing.T) {
	ann := &serviceAnnotations{
		ProductPath:    "generator/integration_tests/golden/v1",
		ForwardingPath: "generator/integration_tests/golden",
	}

	if got, want := ann.PackageNamespace(), "golden_v1"; got != want {
		t.Errorf("PackageNamespace: want %q, got %q", want, got)
	}
	if got, want := ann.InternalNamespace(), "golden_v1_internal"; got != want {
		t.Errorf("InternalNamespace: want %q, got %q", want, got)
	}
	if got, want := ann.MocksNamespace(), "golden_v1_mocks"; got != want {
		t.Errorf("MocksNamespace: want %q, got %q", want, got)
	}
	if got, want := ann.ForwardingNamespace(), "golden"; got != want {
		t.Errorf("ForwardingNamespace: want %q, got %q", want, got)
	}
	if got, want := ann.ForwardingMocksNamespace(), "golden_mocks"; got != want {
		t.Errorf("ForwardingMocksNamespace: want %q, got %q", want, got)
	}
}

func TestServiceAnnotations_ProductionPaths(t *testing.T) {
	ann := &serviceAnnotations{
		ProductPath:    "google/cloud/secretmanager/v1",
		ForwardingPath: "google/cloud/secretmanager",
	}

	if got, want := ann.PackageNamespace(), "secretmanager_v1"; got != want {
		t.Errorf("PackageNamespace: want %q, got %q", want, got)
	}
	if got, want := ann.InternalNamespace(), "secretmanager_v1_internal"; got != want {
		t.Errorf("InternalNamespace: want %q, got %q", want, got)
	}
	if got, want := ann.MocksNamespace(), "secretmanager_v1_mocks"; got != want {
		t.Errorf("MocksNamespace: want %q, got %q", want, got)
	}
	if got, want := ann.ForwardingNamespace(), "secretmanager"; got != want {
		t.Errorf("ForwardingNamespace: want %q, got %q", want, got)
	}
	if got, want := ann.ForwardingMocksNamespace(), "secretmanager_mocks"; got != want {
		t.Errorf("ForwardingMocksNamespace: want %q, got %q", want, got)
	}
}

func TestServiceAnnotations_ComputeDeepPaths(t *testing.T) {
	ann := &serviceAnnotations{
		ProductPath: "google/cloud/compute/addresses/v1",
	}

	if got, want := ann.PackageNamespace(), "compute_addresses_v1"; got != want {
		t.Errorf("PackageNamespace: want %q, got %q", want, got)
	}
	if got, want := ann.InternalNamespace(), "compute_addresses_v1_internal"; got != want {
		t.Errorf("InternalNamespace: want %q, got %q", want, got)
	}
	if got, want := ann.MocksNamespace(), "compute_addresses_v1_mocks"; got != want {
		t.Errorf("MocksNamespace: want %q, got %q", want, got)
	}
}

func TestServiceAnnotations_NoForwarding(t *testing.T) {
	ann := &serviceAnnotations{
		ProductPath: "generator/integration_tests/golden/v1",
	}

	if got, want := ann.ForwardingNamespace(), ""; got != want {
		t.Errorf("ForwardingNamespace: want %q, got %q", want, got)
	}
	if got, want := ann.ForwardingMocksNamespace(), ""; got != want {
		t.Errorf("ForwardingMocksNamespace: want %q, got %q", want, got)
	}
}

func TestServiceAnnotations_Empty(t *testing.T) {
	ann := &serviceAnnotations{}

	if got, want := ann.PackageNamespace(), ""; got != want {
		t.Errorf("PackageNamespace: want %q, got %q", want, got)
	}
	if got, want := ann.InternalNamespace(), ""; got != want {
		t.Errorf("InternalNamespace: want %q, got %q", want, got)
	}
	if got, want := ann.MocksNamespace(), ""; got != want {
		t.Errorf("MocksNamespace: want %q, got %q", want, got)
	}
	if got, want := ann.ForwardingNamespace(), ""; got != want {
		t.Errorf("ForwardingNamespace: want %q, got %q", want, got)
	}
	if got, want := ann.ForwardingMocksNamespace(), ""; got != want {
		t.Errorf("ForwardingMocksNamespace: want %q, got %q", want, got)
	}
}
