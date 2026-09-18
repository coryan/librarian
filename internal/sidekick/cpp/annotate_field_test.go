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

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/sidekick/api"
)

func TestAnnotateField_Scalars(t *testing.T) {
	for _, test := range []struct {
		name         string
		typez        api.Typez
		wantCppType  string
		wantConstRef string
	}{
		{"int32_field", api.TypezInt32, "std::int32_t", "std::int32_t"},
		{"int64_field", api.TypezInt64, "std::int64_t", "std::int64_t"},
		{"uint32_field", api.TypezUint32, "std::uint32_t", "std::uint32_t"},
		{"uint64_field", api.TypezUint64, "std::uint64_t", "std::uint64_t"},
		{"float_field", api.TypezFloat, "float", "float"},
		{"double_field", api.TypezDouble, "double", "double"},
		{"bool_field", api.TypezBool, "bool", "bool"},
		{"string_field", api.TypezString, "std::string", "std::string const&"},
		{"bytes_field", api.TypezBytes, "std::string", "std::string const&"},
	} {
		t.Run(test.name, func(t *testing.T) {
			field := api.NewTestField(test.name).WithType(test.typez)
			ann := annotateField(field)
			if diff := cmp.Diff(test.wantCppType, ann.CppType()); diff != "" {
				t.Errorf("CppType mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(test.wantConstRef, ann.ConstRefType()); diff != "" {
				t.Errorf("ConstRefType mismatch (-want +got):\n%s", diff)
			}
			if field.Codec != ann {
				t.Errorf("field.Codec not set to annotations")
			}
		})
	}
}

func TestAnnotateField_Repeated(t *testing.T) {
	strField := api.NewTestField("names").WithType(api.TypezString).WithRepeated()
	ann := annotateField(strField)
	if !ann.IsRepeated() {
		t.Errorf("expected IsRepeated()=true")
	}
	if diff := cmp.Diff("std::vector<std::string>", ann.CppType()); diff != "" {
		t.Errorf("CppType mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff("std::vector<std::string> const&", ann.ConstRefType()); diff != "" {
		t.Errorf("ConstRefType mismatch (-want +got):\n%s", diff)
	}
}

func TestAnnotateField_Map(t *testing.T) {
	keyField := api.NewTestField("key").WithType(api.TypezString)
	valField := api.NewTestField("value").WithType(api.TypezString)
	mapEntryMsg := api.NewTestMessage("LabelsEntry").WithFields(keyField, valField)

	mapField := api.NewTestField("labels").WithMap()
	mapField.MessageType = mapEntryMsg

	ann := annotateField(mapField)
	if !ann.IsMap() {
		t.Errorf("expected IsMap()=true")
	}
	if diff := cmp.Diff("std::string", ann.KeyType); diff != "" {
		t.Errorf("KeyType mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff("std::string", ann.ValueType); diff != "" {
		t.Errorf("ValueType mismatch (-want +got):\n%s", diff)
	}

	if diff := cmp.Diff("std::map<std::string, std::string>", ann.CppType()); diff != "" {
		t.Errorf("CppType mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff("std::map<std::string, std::string> const&", ann.ConstRefType()); diff != "" {
		t.Errorf("ConstRefType mismatch (-want +got):\n%s", diff)
	}
}

func TestAnnotateField_Message(t *testing.T) {
	msg := api.NewTestMessage("Resource").WithPackage("google.test.v1")
	field := api.NewTestField("resource").WithType(api.TypezMessage)
	field.TypezID = msg.ID

	ann := annotateField(field)
	if !ann.IsMessage() {
		t.Errorf("expected IsMessage()=true")
	}
	if diff := cmp.Diff("google::test::v1::Resource", ann.CppType()); diff != "" {
		t.Errorf("CppType mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff("google::test::v1::Resource const&", ann.ConstRefType()); diff != "" {
		t.Errorf("ConstRefType mismatch (-want +got):\n%s", diff)
	}
}
