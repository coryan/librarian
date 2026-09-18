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
