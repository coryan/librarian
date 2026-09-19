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

package config

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/yaml"
)

func TestPythonConfig_Unmarshal(t *testing.T) {
	for _, test := range []struct {
		name string
		yaml string
		want *PythonPackage
	}{
		{
			name: "package generator override",
			yaml: `
generator: sidekick
default_version: v1
`,
			want: &PythonPackage{
				PythonDefault: PythonDefault{
					Generator: PythonGeneratorSidekick,
				},
				DefaultVersion: "v1",
			},
		},
		{
			name: "default with generator",
			yaml: `
generator: legacy
library_type: GAPIC_AUTO
allowed_namespaces:
  - google.cloud
`,
			want: &PythonPackage{
				PythonDefault: PythonDefault{
					Generator:   PythonGeneratorLegacy,
					LibraryType: "GAPIC_AUTO",
					AllowedNamespaces: []string{
						"google.cloud",
					},
				},
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := yaml.Unmarshal[PythonPackage]([]byte(test.yaml))
			if err != nil {
				t.Fatalf("yaml.Unmarshal() failed: %v", err)
			}
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
