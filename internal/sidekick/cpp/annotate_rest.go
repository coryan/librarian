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
	"regexp"
	"strings"

	"github.com/googleapis/librarian/internal/sidekick/api"
)

var apiVersionRegex = regexp.MustCompile(`^v\d+$`)

func isRestMethod(m *api.Method) bool {
	return !isStreaming(m) && m.PathInfo != nil && len(m.PathInfo.Bindings) > 0
}

func getRestMethods(methods []*api.Method) []*api.Method {
	var result []*api.Method
	for _, m := range methods {
		if isRestMethod(m) {
			result = append(result, m)
		}
	}
	return result
}

func getRestAsyncMethods(asyncMethods []*api.Method) []*api.Method {
	var result []*api.Method
	for _, m := range asyncMethods {
		if isRestMethod(m) && !isLongrunning(m) {
			result = append(result, m)
		}
	}
	return result
}

func httpVerb(verb string) string {
	switch strings.ToUpper(verb) {
	case "GET":
		return "Get"
	case "POST":
		return "Post"
	case "PUT":
		return "Put"
	case "PATCH":
		return "Patch"
	case "DELETE":
		return "Delete"
	default:
		if len(verb) > 0 {
			return strings.ToUpper(verb[:1]) + strings.ToLower(verb[1:])
		}
		return verb
	}
}

func formatRequestResource(m *api.Method) string {
	if m.PathInfo == nil || m.PathInfo.BodyFieldPath == "" || m.PathInfo.BodyFieldPath == "*" {
		return "request"
	}
	return "request." + m.PathInfo.BodyFieldPath + "()"
}

func formatRestPath(m *api.Method, isAsync bool) string {
	if m.PathInfo == nil || len(m.PathInfo.Bindings) == 0 {
		return `absl::StrCat("/")`
	}
	binding := m.PathInfo.Bindings[0]
	template := binding.PathTemplate
	if template == nil {
		return `absl::StrCat("/")`
	}

	// Detect API version from literal segments (e.g. "v1")
	var apiVersion string
	for _, seg := range template.Segments {
		if seg.Literal != "" && apiVersionRegex.MatchString(seg.Literal) {
			apiVersion = seg.Literal
			break
		}
	}

	var pieces []string
	for _, seg := range template.Segments {
		if seg.Literal != "" {
			if apiVersion != "" && seg.Literal == apiVersion {
				if isAsync {
					pieces = append(pieces, fmt.Sprintf(`rest_internal::DetermineApiVersion("%s", *options)`, apiVersion))
				} else {
					pieces = append(pieces, fmt.Sprintf(`rest_internal::DetermineApiVersion("%s", options)`, apiVersion))
				}
			} else {
				pieces = append(pieces, fmt.Sprintf(`"%s"`, seg.Literal))
			}
		} else if seg.Variable != nil {
			accessor := formatFieldAccessor(seg.Variable.FieldPath)
			pieces = append(pieces, fmt.Sprintf("request.%s()", accessor))
		}
	}

	var sb strings.Builder
	sb.WriteString(`absl::StrCat("/"`)
	for _, piece := range pieces {
		sb.WriteString(`, `)
		sb.WriteString(piece)
		sb.WriteString(`, "/"`)
	}
	// Note: trailing "/" from the last piece needs to be adjusted or replaced by verb
	result := sb.String()
	if len(pieces) > 0 {
		// Remove the trailing `, "/"`
		result = strings.TrimSuffix(result, `, "/"`)
	}
	if template.Verb != "" {
		result += fmt.Sprintf(`, ":%s")`, template.Verb)
	} else {
		result += ")"
	}
	return result
}

var supportedWKTWrappers = map[string]string{
	".google.protobuf.BoolValue":   "bool",
	".google.protobuf.DoubleValue": "double",
	".google.protobuf.FloatValue":  "float",
	".google.protobuf.Int32Value":  "int32",
	".google.protobuf.Int64Value":  "int64",
	".google.protobuf.StringValue": "string",
	".google.protobuf.UInt32Value": "uint32",
	".google.protobuf.UInt64Value": "uint64",
}

func formatHTTPQueryParameters(m *api.Method, model *api.API) string {
	if m.PathInfo == nil || len(m.PathInfo.Bindings) == 0 {
		return ""
	}
	binding := m.PathInfo.Bindings[0]
	if m.PathInfo.BodyFieldPath == "*" {
		return ""
	}

	// Path variables must not be included in query params
	pathVars := make(map[string]bool)
	if binding.PathTemplate != nil {
		for _, seg := range binding.PathTemplate.Segments {
			if seg.Variable != nil && len(seg.Variable.FieldPath) > 0 {
				pathVars[seg.Variable.FieldPath[0]] = true
			}
		}
	}

	inputMsg := model.Message(m.InputTypeID)
	if inputMsg == nil {
		return ""
	}

	var lines []string
	for _, f := range inputMsg.Fields {
		if f.Repeated || f.Deprecated {
			continue
		}
		if pathVars[f.Name] {
			continue
		}
		if f.Name == m.PathInfo.BodyFieldPath {
			continue
		}

		if f.Typez == api.TypezMessage {
			wrapperKind, isWKT := supportedWKTWrappers[f.TypezID]
			if !isWKT {
				continue
			}
			var fieldAccess string
			switch wrapperKind {
			case "string":
				fieldAccess = fmt.Sprintf("request.%s().value()", f.Name)
			case "bool":
				fieldAccess = fmt.Sprintf(`(request.%s().value() ? "1" : "0")`, f.Name)
			default:
				fieldAccess = fmt.Sprintf("std::to_string(request.%s().value())", f.Name)
			}
			lines = append(lines, fmt.Sprintf(`  query_params.push_back({"%s", (request.has_%s() ? %s : "")});`, f.Name, f.Name, fieldAccess))
		} else {
			var fieldAccess string
			switch f.Typez {
			case api.TypezString, api.TypezBytes:
				fieldAccess = fmt.Sprintf("request.%s()", f.Name)
			case api.TypezBool:
				fieldAccess = fmt.Sprintf(`(request.%s() ? "1" : "0")`, f.Name)
			default:
				fieldAccess = fmt.Sprintf("std::to_string(request.%s())", f.Name)
			}
			lines = append(lines, fmt.Sprintf(`  query_params.push_back({"%s", %s});`, f.Name, fieldAccess))
		}
	}

	if len(lines) == 0 {
		return ""
	}

	lines = append(lines, `  query_params = rest_internal::TrimEmptyQueryParameters(std::move(query_params));`)
	return "\n" + strings.Join(lines, "\n")
}
