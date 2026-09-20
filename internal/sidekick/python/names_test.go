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

	"github.com/google/go-cmp/cmp"
)

func TestSnakeCase(t *testing.T) {
	for _, test := range []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "PascalCase",
			input: "SecretManagerService",
			want:  "secret_manager_service",
		},
		{
			name:  "camelCase",
			input: "secretManagerService",
			want:  "secret_manager_service",
		},
		{
			name:  "already snake_case",
			input: "secret_manager",
			want:  "secret_manager",
		},
		{
			name:  "empty",
			input: "",
			want:  "",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := snakeCase(test.input)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestPascalCase(t *testing.T) {
	for _, test := range []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "snake_case",
			input: "secret_manager_service",
			want:  "SecretManagerService",
		},
		{
			name:  "camelCase",
			input: "secretManagerService",
			want:  "SecretManagerService",
		},
		{
			name:  "already PascalCase",
			input: "SecretManager",
			want:  "SecretManager",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := pascalCase(test.input)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestPythonIdentifier(t *testing.T) {
	for _, test := range []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "keyword from",
			input: "from",
			want:  "from_",
		},
		{
			name:  "keyword class",
			input: "class",
			want:  "class_",
		},
		{
			name:  "keyword def",
			input: "def",
			want:  "def_",
		},
		{
			name:  "non-keyword",
			input: "project_id",
			want:  "project_id",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := pythonIdentifier(test.input)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
