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
	"strings"
	"unicode"
)

// camelCaseToSnakeCase converts a CamelCase identifier to snake_case,
// matching the transformation rules used in google-cloud-cpp generator.
func camelCaseToSnakeCase(input string) string {
	var output strings.Builder
	for i := 0; i < len(input); i++ {
		ch := input[i]
		lower := byte(unicode.ToLower(rune(ch)))
		if ch != '_' && i+2 < len(input) {
			if unicode.IsUpper(rune(input[i+1])) && unicode.IsLower(rune(input[i+2])) {
				output.WriteByte(lower)
				output.WriteByte('_')
				continue
			}
		}
		if ch != '_' && i+1 < len(input) {
			if (unicode.IsLower(rune(input[i])) || unicode.IsDigit(rune(input[i]))) && unicode.IsUpper(rune(input[i+1])) {
				output.WriteByte(lower)
				output.WriteByte('_')
				continue
			}
		}
		output.WriteByte(lower)
	}
	res := output.String()
	res = strings.ReplaceAll(res, "big_query", "bigquery")
	return res
}

// serviceNameToFileName converts a service name to a file path component,
// stripping any trailing "Service" suffix from the service name component.
func serviceNameToFileName(serviceName string) string {
	components := strings.Split(serviceName, ".")
	last := components[len(components)-1]
	last = strings.TrimSuffix(last, "Service")
	components[len(components)-1] = last
	for i, c := range components {
		components[i] = camelCaseToSnakeCase(c)
	}
	return strings.Join(components, "/")
}

// protoNameToCppName converts a fully-qualified Protobuf type name to a C++ type name
// with "::" namespace delimiters, stripping any leading dot.
func protoNameToCppName(name string) string {
	name = strings.TrimPrefix(name, ".")
	return strings.ReplaceAll(name, ".", "::")
}

var cppKeywords = map[string]bool{
	"alignas": true, "alignof": true, "and": true, "and_eq": true, "asm": true, "auto": true,
	"bitand": true, "bitor": true, "bool": true, "break": true, "case": true, "catch": true,
	"char": true, "char16_t": true, "char32_t": true, "class": true, "compl": true, "const": true,
	"constexpr": true, "const_cast": true, "continue": true, "decltype": true, "default": true,
	"delete": true, "do": true, "double": true, "dynamic_cast": true, "else": true, "enum": true,
	"explicit": true, "export": true, "extern": true, "false": true, "float": true, "for": true,
	"friend": true, "goto": true, "if": true, "inline": true, "int": true, "long": true,
	"mutable": true, "namespace": true, "new": true, "noexcept": true, "not": true, "not_eq": true,
	"nullptr": true, "operator": true, "or": true, "or_eq": true, "private": true, "protected": true,
	"public": true, "register": true, "reinterpret_cast": true, "return": true, "short": true,
	"signed": true, "sizeof": true, "static": true, "static_assert": true, "static_cast": true,
	"struct": true, "switch": true, "template": true, "this": true, "thread_local": true,
	"throw": true, "true": true, "try": true, "typedef": true, "typeid": true, "typename": true,
	"union": true, "unsigned": true, "using": true, "virtual": true, "void": true, "volatile": true,
	"wchar_t": true, "while": true, "xor": true, "xor_eq": true,
}

// cppFieldName escapes C++ keywords by appending a trailing underscore.
func cppFieldName(name string) string {
	if cppKeywords[name] {
		return name + "_"
	}
	return name
}
