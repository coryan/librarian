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
	"fmt"
	"path/filepath"
	"strings"

	"github.com/googleapis/librarian/internal/sidekick/api"
)

// FieldAnnotations decorates api.Field with Python-specific metadata.
type FieldAnnotations struct {
	Field          *api.Field
	Message        *MessageAnnotations
	Name           string
	Number         int32
	ProtoType      string
	TypeAnnotation string
	SphinxType     string
	IsRepeated     bool
	IsMap          bool
	KeyProtoType   string
	ValProtoType   string
	KeyType        string
	ValType        string
	FieldArg       string // "message" or "enum"
	TypeRef        string // relative reference, e.g. "'TlsCertificate'"
	ValFieldArg    string // "message" or "enum" for map values
	ValTypeRef     string // relative reference for map values
	OneOf          string
	OneOfDoc       string
	IsOptional     bool
	DocLines       []string
}

func (c *codec) annotateField(field *api.Field, message *MessageAnnotations) error {
	docLines := formatRSTDocLines(field.Documentation, 72, 12)

	ann := &FieldAnnotations{
		Field:      field,
		Message:    message,
		Name:       pythonIdentifier(snakeCase(field.Name)),
		Number:     field.Number,
		DocLines:   docLines,
		IsRepeated: field.Repeated,
		IsMap:      field.Map,
	}

	// Handle OneOf and Optional
	if field.Optional && field.Typez != api.TypezMessage {
		ann.IsOptional = true
		ann.OneOfDoc = fmt.Sprintf("_%s", field.Name)
	} else if field.Group != nil {
		ann.OneOf = field.Group.Name
		ann.OneOfDoc = field.Group.Name
	}

	if field.Map {
		mapMsg := c.Model.Message(field.TypezID)
		if mapMsg != nil && len(mapMsg.Fields) >= 2 {
			keyField := mapMsg.Fields[0]
			valField := mapMsg.Fields[1]
			ann.KeyProtoType = keyField.Typez.String()
			ann.ValProtoType = valField.Typez.String()

			keyType, keySphinx, _, _ := c.resolveFieldTypeDetails(keyField, message)
			valType, valSphinx, valArg, valRef := c.resolveFieldTypeDetails(valField, message)

			ann.KeyType = keyType
			ann.ValType = valType
			ann.ValFieldArg = valArg
			ann.ValTypeRef = valRef
			ann.SphinxType = fmt.Sprintf("MutableMapping[%s, %s]", keySphinx, valSphinx)
			ann.TypeAnnotation = fmt.Sprintf("MutableMapping[%s, %s]", keyType, valType)
		} else {
			ann.KeyProtoType = "STRING"
			ann.ValProtoType = "STRING"
			ann.KeyType = "str"
			ann.ValType = "str"
			ann.SphinxType = "MutableMapping[str, str]"
			ann.TypeAnnotation = "MutableMapping[str, str]"
		}
	} else {
		ann.ProtoType = field.Typez.String()
		baseType, baseSphinx, fieldArg, typeRef := c.resolveFieldTypeDetails(field, message)
		ann.FieldArg = fieldArg
		ann.TypeRef = typeRef

		if field.Repeated {
			ann.TypeAnnotation = fmt.Sprintf("MutableSequence[%s]", baseType)
			ann.SphinxType = fmt.Sprintf("MutableSequence[%s]", baseSphinx)
		} else {
			ann.TypeAnnotation = baseType
			ann.SphinxType = baseSphinx
		}
	}

	field.Codec = ann
	return nil
}

func (c *codec) resolveFieldTypeDetails(field *api.Field, message *MessageAnnotations) (baseType, sphinxType, fieldArg, typeRef string) {
	switch field.Typez {
	case api.TypezBool:
		return "bool", "bool", "", ""
	case api.TypezString:
		return "str", "str", "", ""
	case api.TypezBytes:
		return "bytes", "bytes", "", ""
	case api.TypezInt32, api.TypezSint32, api.TypezSfixed32,
		api.TypezInt64, api.TypezSint64, api.TypezSfixed64,
		api.TypezUint32, api.TypezFixed32,
		api.TypezUint64, api.TypezFixed64:
		return "int", "int", "", ""
	case api.TypezFloat, api.TypezDouble:
		return "float", "float", "", ""
	case api.TypezEnum:
		fieldArg = "enum"
	case api.TypezMessage:
		fieldArg = "message"
	default:
		return "str", "str", "", ""
	}

	typeID := field.TypezID
	if typeID == "" {
		return "str", "str", "", ""
	}

	targetMsg := c.Model.Message(typeID)
	var targetEnum *api.Enum
	if targetMsg == nil {
		targetEnum = c.Model.Enum(typeID)
	}

	targetPkg := ""
	if targetMsg != nil {
		targetPkg = targetMsg.Package
	} else if targetEnum != nil {
		targetPkg = targetEnum.Package
	} else {
		trimmed := strings.TrimPrefix(typeID, ".")
		if lastDot := strings.LastIndex(trimmed, "."); lastDot != -1 {
			targetPkg = trimmed[:lastDot]
		}
	}

	loc, hasLoc := c.Model.DefinitionLocation(typeID)
	targetFile := ""
	if hasLoc {
		targetFile = loc.Filename
	}

	targetModule := ""
	if targetFile != "" {
		targetModule = strings.TrimSuffix(filepath.Base(targetFile), ".proto")
	} else {
		targetModule = deriveModuleFromTypeID(typeID)
	}

	var nameParts []string
	if targetMsg != nil {
		for p := targetMsg; p != nil; p = p.Parent {
			nameParts = append([]string{p.Name}, nameParts...)
		}
	} else if targetEnum != nil {
		nameParts = []string{targetEnum.Name}
		for p := targetEnum.Parent; p != nil; p = p.Parent {
			nameParts = append([]string{p.Name}, nameParts...)
		}
	} else {
		trimmed := strings.TrimPrefix(typeID, ".")
		parts := strings.Split(trimmed, ".")
		nameParts = []string{parts[len(parts)-1]}
	}
	relativeName := strings.Join(nameParts, ".")

	versionedPkg := strings.ReplaceAll(c.packageDir(), "/", ".")

	// Case 1: External protobuf type
	if targetPkg != c.Model.PackageName {
		typeRef = fmt.Sprintf("%s_pb2.%s", targetModule, relativeName)
		sphinxType = fmt.Sprintf("%s.%s_pb2.%s", targetPkg, targetModule, relativeName)
		return typeRef, sphinxType, fieldArg, typeRef
	}

	// Current message's proto file
	currentProtoFile := ""
	if message != nil && message.Message != nil {
		if loc, ok := c.Model.DefinitionLocation(message.Message.ID); ok {
			currentProtoFile = loc.Filename
		}
	}

	// Case 2: In the same package, but different proto file
	if targetFile != currentProtoFile && currentProtoFile != "" && targetFile != "" {
		modIdentifier := targetModule
		if c.isModuleCollisionInProto(currentProtoFile, targetModule) {
			modIdentifier = computeModuleAlias(strings.Split(targetPkg, "."), c.CurrentVersion, targetModule)
		}
		typeRef = fmt.Sprintf("%s.%s", modIdentifier, relativeName)
		sphinxType = fmt.Sprintf("%s.types.%s", versionedPkg, relativeName)
		return typeRef, sphinxType, fieldArg, typeRef
	}

	// Case 3: In the same proto file
	sphinxType = fmt.Sprintf("%s.types.%s", versionedPkg, relativeName)
	typeRef = computeSameFileTypeRef(targetMsg, targetEnum, message)
	return typeRef, sphinxType, fieldArg, typeRef
}

func computeSameFileTypeRef(targetMsg *api.Message, targetEnum *api.Enum, currentMsg *MessageAnnotations) string {
	var targetParents []string
	targetName := ""
	if targetMsg != nil {
		targetName = targetMsg.Name
		for p := targetMsg.Parent; p != nil; p = p.Parent {
			targetParents = append([]string{p.Name}, targetParents...)
		}
	} else if targetEnum != nil {
		targetName = targetEnum.Name
		for p := targetEnum.Parent; p != nil; p = p.Parent {
			targetParents = append([]string{p.Name}, targetParents...)
		}
	}

	currentName := ""
	if currentMsg != nil && currentMsg.Message != nil {
		currentName = currentMsg.Message.Name
	}

	// If this message references a nested message that it contains,
	// reference relative to this message's namespace:
	if len(targetParents) > 0 && targetParents[0] == currentName {
		relParts := append(targetParents[1:], targetName)
		return strings.Join(relParts, ".")
	}

	// Standard case: send name enclosed in quotes:
	fullPath := strings.Join(append(targetParents, targetName), ".")
	return fmt.Sprintf("'%s'", fullPath)
}

func (c *codec) isModuleCollisionInProto(protoFile, module string) bool {
	for _, msg := range c.Model.Messages {
		loc, ok := c.Model.DefinitionLocation(msg.ID)
		if ok && loc.Filename == protoFile {
			if snakeCase(msg.Name) == module || msg.Name == module {
				return true
			}
			for _, f := range msg.Fields {
				if f.Name == module {
					return true
				}
			}
		}
	}
	for _, enum := range c.Model.Enums {
		loc, ok := c.Model.DefinitionLocation(enum.ID)
		if ok && loc.Filename == protoFile {
			if snakeCase(enum.Name) == module || enum.Name == module {
				return true
			}
		}
	}
	return false
}
