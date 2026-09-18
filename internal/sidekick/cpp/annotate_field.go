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
	"fmt"

	"github.com/googleapis/librarian/internal/sidekick/api"
)

// fieldAnnotations contains C++-specific metadata for a protobuf field.
// Annotations are private to the package to encapsulate implementation details and enforce
// accessor method usage for derived properties.
type fieldAnnotations struct {
	Field     *api.Field
	KeyType   string
	ValueType string
}

func (f *fieldAnnotations) IsMap() bool {
	if f == nil || f.Field == nil {
		return false
	}
	return f.Field.Map
}

func (f *fieldAnnotations) IsRepeated() bool {
	if f == nil || f.Field == nil {
		return false
	}
	return f.Field.Repeated
}

func (f *fieldAnnotations) IsMessage() bool {
	if f == nil || f.Field == nil {
		return false
	}
	return f.Field.Typez == api.TypezMessage
}

func (f *fieldAnnotations) CppType() string {
	if f == nil || f.Field == nil {
		return ""
	}
	if f.IsMap() {
		return fmt.Sprintf("std::map<%s, %s>", f.KeyType, f.ValueType)
	}
	base := cppTypeToString(f.Field)
	if f.IsRepeated() {
		return fmt.Sprintf("std::vector<%s>", base)
	}
	return base
}

func (f *fieldAnnotations) ConstRefType() string {
	if f == nil || f.Field == nil {
		return ""
	}
	if f.IsMap() || f.IsRepeated() || f.Field.Typez == api.TypezMessage || f.Field.Typez == api.TypezString || f.Field.Typez == api.TypezBytes {
		return f.CppType() + " const&"
	}
	return f.CppType()
}

func (f *fieldAnnotations) ByValueType() string {
	return f.CppType()
}

// annotateField enriches an api.Field with C++-specific type information.
func annotateField(f *api.Field) *fieldAnnotations {
	if f == nil {
		return nil
	}

	ann := &fieldAnnotations{
		Field: f,
	}

	if f.Map {
		keyType := "std::string"
		valType := "std::string"
		if f.MessageType != nil && len(f.MessageType.Fields) >= 2 {
			keyType = cppTypeToString(f.MessageType.Fields[0])
			valType = cppTypeToString(f.MessageType.Fields[1])
		}
		ann.KeyType = keyType
		ann.ValueType = valType
	}

	f.Codec = ann
	return ann
}
