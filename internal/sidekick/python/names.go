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
	"strings"
	"unicode"

	"github.com/iancoleman/strcase"
)

// pythonKeywords contains keywords and names reserved in Python/proto-plus.
var pythonKeywords = map[string]bool{
	"False":                 true,
	"None":                  true,
	"True":                  true,
	"__peg_parser__":        true,
	"all":                   true,
	"and":                   true,
	"any":                   true,
	"as":                    true,
	"assert":                true,
	"async":                 true,
	"await":                 true,
	"break":                 true,
	"breakpoint":            true,
	"class":                 true,
	"cls":                   true,
	"continue":              true,
	"def":                   true,
	"del":                   true,
	"dir":                   true,
	"elif":                  true,
	"else":                  true,
	"except":                true,
	"exec":                  true,
	"finally":               true,
	"for":                   true,
	"format":                true,
	"from":                  true,
	"global":                true,
	"hash":                  true,
	"help":                  true,
	"if":                    true,
	"ignore_unknown_fields": true,
	"import":                true,
	"in":                    true,
	"is":                    true,
	"lambda":                true,
	"license":               true,
	"list":                  true,
	"locals":                true,
	"mapping":               true,
	"max":                   true,
	"min":                   true,
	"next":                  true,
	"nonlocal":              true,
	"not":                   true,
	"object":                true,
	"open":                  true,
	"or":                    true,
	"pass":                  true,
	"raise":                 true,
	"range":                 true,
	"return":                true,
	"self":                  true,
	"slice":                 true,
	"try":                   true,
	"type":                  true,
	"while":                 true,
	"with":                  true,
	"yield":                 true,
	"zip":                   true,
}

// snakeCase converts a string to snake_case.
func snakeCase(s string) string {
	if !strings.ContainsFunc(s, unicode.IsUpper) {
		return s
	}
	return strcase.ToSnake(s)
}

// pascalCase converts a string to PascalCase.
func pascalCase(s string) string {
	return strcase.ToCamel(s)
}

// pythonIdentifier escapes keywords with a trailing underscore.
func pythonIdentifier(s string) string {
	if pythonKeywords[s] {
		return s + "_"
	}
	return s
}
