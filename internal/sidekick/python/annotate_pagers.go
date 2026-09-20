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
	"slices"
	"strings"

	"github.com/googleapis/librarian/internal/sidekick/api"
)

// PagersAnnotations contains template annotations for pagers.py.
type PagersAnnotations struct {
	VersionPackage string
	TypeImports    []string
	Pagers         []*PagerAnnotations
}

// PagerAnnotations contains template annotations for a single pager pair.
type PagerAnnotations struct {
	MethodName      string
	MethodSnakeName string
	RequestIdent    string
	ResponseIdent   string
	RequestDocType  string
	ResponseDocType string
	ItemFieldName   string
	ItemTypeIdent   string
	IsMap           bool
	HasNext         bool
}

func (c *codec) annotatePagers(service *api.Service) *PagersAnnotations {
	if service == nil {
		return nil
	}
	versionPackage := strings.ReplaceAll(c.packageDir(), "/", ".")
	var pagers []*PagerAnnotations
	importSet := make(map[string]bool)

	for _, m := range service.Methods {
		if isMixinMethod(m, service) {
			continue
		}
		if m.Pagination == nil {
			continue
		}

		var respMsg *api.Message
		if c.Model != nil {
			respMsg = c.Model.Message(m.OutputTypeID)
		}
		if respMsg == nil && m.OutputType != nil {
			respMsg = m.OutputType
		}

		pageableItem := getPageableItem(respMsg)
		if pageableItem == nil {
			continue
		}

		var reqMsg *api.Message
		if c.Model != nil {
			reqMsg = c.Model.Message(m.InputTypeID)
		}
		if reqMsg == nil && m.InputType != nil {
			reqMsg = m.InputType
		}

		reqRelName := resolveMessageRelativeName(reqMsg)
		if reqRelName == "" {
			reqRelName = typeNameWithFallback(m.InputTypeID, m.Name+"Request")
		}
		reqMod := c.resolveTypeModule(typeIDOrMsgID(m.InputTypeID, reqMsg), service)

		respRelName := resolveMessageRelativeName(respMsg)
		if respRelName == "" {
			respRelName = typeNameWithFallback(m.OutputTypeID, m.Name+"Response")
		}
		respMod := c.resolveTypeModule(typeIDOrMsgID(m.OutputTypeID, respMsg), service)

		itemTypeIdent, itemMod, isMap := c.resolvePagerItemType(pageableItem, service)

		if reqMod != "" {
			importSet[reqMod] = true
		}
		if respMod != "" {
			importSet[respMod] = true
		}
		if itemMod != "" {
			importSet[itemMod] = true
		}

		reqIdent := reqRelName
		if reqMod != "" {
			reqIdent = reqMod + "." + reqRelName
		}
		respIdent := respRelName
		if respMod != "" {
			respIdent = respMod + "." + respRelName
		}

		pagers = append(pagers, &PagerAnnotations{
			MethodName:      m.Name,
			MethodSnakeName: snakeCase(m.Name),
			RequestIdent:    reqIdent,
			ResponseIdent:   respIdent,
			RequestDocType:  versionPackage + ".types." + reqRelName,
			ResponseDocType: versionPackage + ".types." + respRelName,
			ItemFieldName:   pageableItem.Name,
			ItemTypeIdent:   itemTypeIdent,
			IsMap:           isMap,
		})
	}

	for i, p := range pagers {
		p.HasNext = (i < len(pagers)-1)
	}

	typeImports := make([]string, 0, len(importSet))
	for mod := range importSet {
		typeImports = append(typeImports, mod)
	}
	slices.Sort(typeImports)

	return &PagersAnnotations{
		VersionPackage: versionPackage,
		TypeImports:    typeImports,
		Pagers:         pagers,
	}
}

func getPageableItem(respMsg *api.Message) *api.Field {
	if respMsg == nil {
		return nil
	}
	if respMsg.Pagination != nil && respMsg.Pagination.PageableItem != nil {
		return respMsg.Pagination.PageableItem
	}
	if idx := slices.IndexFunc(respMsg.Fields, func(f *api.Field) bool {
		return f.Repeated && f.Typez == api.TypezMessage
	}); idx != -1 {
		return respMsg.Fields[idx]
	}
	if idx := slices.IndexFunc(respMsg.Fields, func(f *api.Field) bool {
		return f.Map
	}); idx != -1 {
		return respMsg.Fields[idx]
	}
	if idx := slices.IndexFunc(respMsg.Fields, func(f *api.Field) bool {
		return f.Repeated
	}); idx != -1 {
		return respMsg.Fields[idx]
	}
	return nil
}

func (c *codec) resolvePagerItemType(field *api.Field, service *api.Service) (string, string, bool) {
	if field == nil {
		return "Any", "", false
	}
	if field.Map {
		var valField *api.Field
		if c.Model != nil {
			if entry := c.Model.Message(field.TypezID); entry != nil {
				if idx := slices.IndexFunc(entry.Fields, func(f *api.Field) bool {
					return f.Name == "value"
				}); idx != -1 {
					valField = entry.Fields[idx]
				}
			}
		}
		if valField != nil {
			ident, mod := c.resolveFieldTypeIdent(valField, service)
			return ident, mod, true
		}
		return "Any", "", true
	}
	ident, mod := c.resolveFieldTypeIdent(field, service)
	return ident, mod, false
}

func (c *codec) resolveFieldTypeIdent(field *api.Field, service *api.Service) (string, string) {
	switch field.Typez {
	case api.TypezMessage:
		var msg *api.Message
		if c.Model != nil {
			msg = c.Model.Message(field.TypezID)
		}
		if msg == nil && field.MessageType != nil {
			msg = field.MessageType
		}
		if msg != nil {
			mod := c.resolveTypeModule(msg.ID, service)
			relName := resolveMessageRelativeName(msg)
			if mod != "" {
				return mod + "." + relName, mod
			}
			return relName, ""
		}
		mod := c.resolveTypeModule(field.TypezID, service)
		relName := typeNameWithFallback(field.TypezID, "Any")
		if relName == "Any" {
			return "Any", ""
		}
		if mod != "" {
			return mod + "." + relName, mod
		}
		return relName, ""
	case api.TypezEnum:
		var enum *api.Enum
		if c.Model != nil {
			enum = c.Model.Enum(field.TypezID)
		}
		if enum == nil && field.EnumType != nil {
			enum = field.EnumType
		}
		if enum != nil {
			mod := c.resolveTypeModule(enum.ID, service)
			relName := resolveEnumRelativeName(enum)
			if mod != "" {
				return mod + "." + relName, mod
			}
			return relName, ""
		}
		mod := c.resolveTypeModule(field.TypezID, service)
		relName := typeNameWithFallback(field.TypezID, "Any")
		if relName == "Any" {
			return "Any", ""
		}
		if mod != "" {
			return mod + "." + relName, mod
		}
		return relName, ""
	default:
		return primitiveTypeName(field.Typez), ""
	}
}

func primitiveTypeName(t api.Typez) string {
	switch t {
	case api.TypezString:
		return "str"
	case api.TypezBytes:
		return "bytes"
	case api.TypezBool:
		return "bool"
	case api.TypezFloat, api.TypezDouble:
		return "float"
	case api.TypezInt32, api.TypezInt64, api.TypezUint32, api.TypezUint64,
		api.TypezSint32, api.TypezSint64, api.TypezFixed32, api.TypezFixed64,
		api.TypezSfixed32, api.TypezSfixed64:
		return "int"
	default:
		return "Any"
	}
}

func resolveMessageRelativeName(m *api.Message) string {
	if m == nil {
		return ""
	}
	var parts []string
	for curr := m; curr != nil; curr = curr.Parent {
		parts = append(parts, curr.Name)
	}
	slices.Reverse(parts)
	return strings.Join(parts, ".")
}

func resolveEnumRelativeName(e *api.Enum) string {
	if e == nil {
		return ""
	}
	var parts []string
	parts = append(parts, e.Name)
	for curr := e.Parent; curr != nil; curr = curr.Parent {
		parts = append(parts, curr.Name)
	}
	slices.Reverse(parts)
	return strings.Join(parts, ".")
}

func typeIDOrMsgID(typeID string, msg *api.Message) string {
	if msg != nil && msg.ID != "" {
		return msg.ID
	}
	return typeID
}

func typeNameWithFallback(typeID, fallback string) string {
	name := typeNameFromID(typeID)
	if name == "" {
		return fallback
	}
	return name
}
