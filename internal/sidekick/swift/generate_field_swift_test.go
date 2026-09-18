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

func TestGenerateField_InitFromDecoder(t *testing.T) {
	outDir := t.TempDir()

	// Field 1: Normal field with JSONName override to trigger CustomSerialization
	field1 := api.NewTestField("normal_field").
		WithType(api.TypezString)
	field1.Documentation = "A normal field."
	field1.JSONName = "normal_field" // Differs from camelCase "normalField"

	// Field 2: Optional field with JSONName override
	field2 := api.NewTestField("optional_field").
		WithType(api.TypezString).
		WithOptional()
	field2.Documentation = "An optional field."
	field2.JSONName = "optional_field" // Differs from camelCase "optionalField"

	msg := api.NewTestMessage("TestMessage").
		WithPackage("google.cloud.test.v1").
		WithFields(field1, field2)

	model := api.NewTestAPI([]*api.Message{msg}, nil, nil)
	model.PackageName = "google.cloud.test.v1"

	library := &config.Library{
		Swift: swiftConfig(t, nil),
	}
	if err := Generate(t.Context(), model, outDir, library, nil); err != nil {
		t.Fatal(err)
	}

	expectedDir := filepath.Join(outDir, "Sources", "GoogleCloudTestV1")
	filename := filepath.Join(expectedDir, "TestMessage.swift")

	content, err := os.ReadFile(filename)
	if err != nil {
		t.Fatal(err)
	}
	contentStr := string(content)

	// Verify init(from decoder: Decoder) content
	gotBlock := extractBlock(t, contentStr, "  public init(from decoder: Decoder) throws {", "\n  }")
	wantBlock := `  public init(from decoder: Decoder) throws {
    let container = try decoder.container(keyedBy: CodingKeys.self)
    if let value = try container.decodeIfPresent(Swift.String.self, forKey: .normalField) {
      self.normalField = value
    }
    self.optionalField = try container.decodeIfPresent(Swift.String.self, forKey: .optionalField)
    for key in container.allKeys where !CodingKeys._knownKeys.contains(key.stringValue) {
      self._unknownFields.json[key.stringValue] = try container.decode(
        GoogleWKT.Value.self, forKey: key)
    }
  }`

	if diff := cmp.Diff(wantBlock, gotBlock); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestGenerateField_DocComments(t *testing.T) {
	outDir := t.TempDir()

	field1 := api.NewTestField("normal_field").
		WithType(api.TypezString)
	field1.Documentation = "Documentation for normal_field."

	field2 := api.NewTestField("oneof_field").
		WithType(api.TypezString)
	field2.Documentation = "Documentation for oneof_field."

	oneof := api.NewTestOneOf("my_oneof").
		WithFields(field2)
	oneof.Documentation = "Documentation for my_oneof."

	msg := api.NewTestMessage("TestMessage").
		WithPackage("google.cloud.test.v1").
		WithFields(field1).
		WithOneOfs(oneof)

	model := api.NewTestAPI([]*api.Message{msg}, nil, nil)
	model.PackageName = "google.cloud.test.v1"

	library := &config.Library{}
	if err := Generate(t.Context(), model, outDir, library, nil); err != nil {
		t.Fatal(err)
	}

	filename := filepath.Join(outDir, "Sources", "GoogleCloudTestV1", "TestMessage.swift")
	content, err := os.ReadFile(filename)
	if err != nil {
		t.Fatal(err)
	}
	contentStr := string(content)

	// Verify normal field documentation
	want := "  /// Documentation for normal_field.\n  public var normalField: Swift.String"
	got := extractBlock(t, contentStr, "  /// Documentation for normal_field.", "public var normalField: Swift.String")
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}

	// Verify oneof documentation in the message
	want = "  /// Documentation for my_oneof.\n  public var myOneof: OneOf_MyOneof?"
	got = extractBlock(t, contentStr, "  /// Documentation for my_oneof.", "public var myOneof: OneOf_MyOneof?")
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}

	// Verify field documentation in the oneof enum
	want = "    /// Documentation for oneof_field.\n    case oneofField(Swift.String)"
	got = extractBlock(t, contentStr, "    /// Documentation for oneof_field.", "case oneofField(Swift.String)")
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}
