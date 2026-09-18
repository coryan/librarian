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

package swift

import (
	"github.com/googleapis/librarian/internal/config"

	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/sidekick/api"
)

func TestGenerateEnum_Files(t *testing.T) {
	outDir := t.TempDir()

	color := api.NewTestEnum("Color").
		WithPackage("google.cloud.test.v1").
		WithValues(api.NewTestEnumValue("COLOR_UNSPECIFIED", 0))
	color.WithUniqueNumberValues(color.Values...)

	kind := api.NewTestEnum("Kind").
		WithPackage("google.cloud.test.v1").
		WithValues(api.NewTestEnumValue("KIND_UNSPECIFIED", 0))
	kind.WithUniqueNumberValues(kind.Values...)

	clash0 := api.NewTestEnum("ClashName").
		WithPackage("google.cloud.test.v1").
		WithValues(api.NewTestEnumValue("CLASH_UNSPECIFIED", 0))
	clash0.WithUniqueNumberValues(clash0.Values...)

	clash1 := api.NewTestEnum("clashName").
		WithPackage("google.cloud.test.v1").
		WithValues(api.NewTestEnumValue("CLASH_UNSPECIFIED", 0))
	clash1.WithUniqueNumberValues(clash1.Values...)

	model := api.NewTestAPI(nil, []*api.Enum{color, kind, clash0, clash1}, nil)
	model.PackageName = "google.cloud.test.v1"
	library := &config.Library{}
	if err := Generate(t.Context(), model, outDir, library, nil); err != nil {
		t.Fatal(err)
	}

	expectedDir := filepath.Join(outDir, "Sources", "GoogleCloudTestV1")
	want := []string{
		"Color.swift",
		"Kind.swift",
		"ClashName.swift",
		"clashName+000.swift",
	}
	for _, expected := range want {
		filename := filepath.Join(expectedDir, expected)
		if _, err := os.Stat(filename); err != nil {
			t.Error(err)
		}
	}
}

func TestGenerateEnum_UniqueNumbers(t *testing.T) {
	outDir := t.TempDir()

	val0 := api.NewTestEnumValue("KIND_UNSPECIFIED", 0)
	val1 := api.NewTestEnumValue("KIND_TEST", 0)
	val2 := api.NewTestEnumValue("KIND_OTHER_TEST", 1)

	kind := api.NewTestEnum("Kind").
		WithPackage("google.cloud.test.v1").
		WithValues(val0, val1, val2).
		WithUniqueNumberValues(val1, val2)

	model := api.NewTestAPI(nil, []*api.Enum{kind}, nil)
	model.PackageName = "google.cloud.test.v1"
	library := &config.Library{}
	if err := Generate(t.Context(), model, outDir, library, nil); err != nil {
		t.Fatal(err)
	}

	contentsB, err := os.ReadFile(filepath.Join(outDir, "Sources", "GoogleCloudTestV1", "Kind.swift"))
	if err != nil {
		t.Fatal(err)
	}
	got := extractBlock(t, string(contentsB), "/// Initialize from an integer value.", "\n  }")
	want := `/// Initialize from an integer value.
  ///
  /// If the value is unknown, this initializes to ` + "[`unknownIntValue`](doc:Kind/unknownIntValue(_:))." + `
  public init(intValue: Int) {
    switch intValue {
    case 0: self = .test
    case 1: self = .otherTest
    default: self = .unknownIntValue(intValue)
    }
  }`
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}

	gotEncode := extractBlock(t, string(contentsB), "public func encode(to encoder: Encoder) throws {", "\n  }")
	wantEncode := `public func encode(to encoder: Encoder) throws {
    var container = encoder.singleValueContainer()
    switch self {
    case .test: return try container.encode("KIND_TEST")
    case .otherTest: return try container.encode("KIND_OTHER_TEST")
    case .unknownIntValue(let v): return try container.encode(v)
    case .unknownStringValue(let v): return try container.encode(v)
    }
  }`
	if diff := cmp.Diff(wantEncode, gotEncode); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestGenerateEnum_DocComments(t *testing.T) {
	outDir := t.TempDir()

	val := api.NewTestEnumValue("COLOR_UNSPECIFIED", 0).
		WithDocumentation("Documentation for the COLOR_UNSPECIFIED value.")
	color := api.NewTestEnum("Color").
		WithPackage("google.cloud.test.v1").
		WithDocumentation("Documentation for the Color enum.").
		WithValues(val)
	color.WithUniqueNumberValues(color.Values...)

	model := api.NewTestAPI(nil, []*api.Enum{color}, nil)
	model.PackageName = "google.cloud.test.v1"
	library := &config.Library{}
	if err := Generate(t.Context(), model, outDir, library, nil); err != nil {
		t.Fatal(err)
	}

	filename := filepath.Join(outDir, "Sources", "GoogleCloudTestV1", "Color.swift")
	content, err := os.ReadFile(filename)
	if err != nil {
		t.Fatal(err)
	}
	contentStr := string(content)

	want := "/// Documentation for the Color enum.\npublic enum Color"
	got := extractBlock(t, contentStr, "/// Documentation for the Color enum.", "public enum Color")
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}

	want = "/// Documentation for the COLOR_UNSPECIFIED value.\n  case unspecified"
	got = extractBlock(t, contentStr, "/// Documentation for the COLOR_UNSPECIFIED value.", "case unspecified")
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}
