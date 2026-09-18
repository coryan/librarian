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

// FieldAnnotations contains C++-specific metadata for a protobuf field.
type FieldAnnotations struct {
	// CppType is the standard C++ type name for the field.
	CppType string

	// ConstRefType is the C++ parameter type passed by const reference where appropriate.
	ConstRefType string

	// ByValueType is the C++ type passed by value.
	ByValueType string

	// IsMap indicates whether the field is a map.
	IsMap bool

	// IsRepeated indicates whether the field is repeated (vector).
	IsRepeated bool

	// IsMessage indicates whether the field is a message.
	IsMessage bool

	// KeyType is the map key type, if IsMap is true.
	KeyType string

	// ValueType is the map value type, if IsMap is true.
	ValueType string
}

// annotateField enriches an api.Field with C++-specific type information.
func annotateField(f *api.Field) *FieldAnnotations {
	if f == nil {
		return nil
	}

	cppType := cppTypeToString(f)
	ann := &FieldAnnotations{
		CppType:    cppType,
		IsMap:      f.Map,
		IsRepeated: f.Repeated,
		IsMessage:  f.Typez == api.TypezMessage,
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
		ann.CppType = fmt.Sprintf("std::map<%s, %s>", keyType, valType)
		ann.ConstRefType = fmt.Sprintf("std::map<%s, %s> const&", keyType, valType)
		ann.ByValueType = ann.CppType
	} else if f.Repeated {
		ann.CppType = fmt.Sprintf("std::vector<%s>", cppType)
		ann.ConstRefType = fmt.Sprintf("std::vector<%s> const&", cppType)
		ann.ByValueType = ann.CppType
	} else {
		switch f.Typez {
		case api.TypezString, api.TypezBytes, api.TypezMessage:
			ann.ConstRefType = ann.CppType + " const&"
			ann.ByValueType = ann.CppType
		default:
			ann.ConstRefType = ann.CppType
			ann.ByValueType = ann.CppType
		}
	}

	f.Codec = ann
	return ann
}
