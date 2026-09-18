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
	"slices"
	"strings"

	"github.com/googleapis/librarian/internal/sidekick/api"
)

type routingParamMatcher struct {
	accessor    string
	pattern     string
	isFullMatch bool
}

type explicitRoutingParam struct {
	name         string
	minPrefixLen int
	matchers     []*routingParamMatcher
}

func (p *explicitRoutingParam) isAllFullMatch() bool {
	for _, m := range p.matchers {
		if !m.isFullMatch {
			return false
		}
	}
	return true
}

func formatRoutingSegments(segments []string) string {
	var parts []string
	for _, s := range segments {
		switch s {
		case "**":
			parts = append(parts, ".*")
		case "*":
			parts = append(parts, "[^/]+")
		default:
			parts = append(parts, s)
		}
	}
	return strings.Join(parts, "/")
}

func formatRoutingPattern(v *api.RoutingInfoVariant) (string, bool) {
	prefixStr := ""
	if len(v.Prefix.Segments) > 0 {
		prefixStr = formatRoutingSegments(v.Prefix.Segments) + "/"
	}
	matchingStr := "(" + formatRoutingSegments(v.Matching.Segments) + ")"
	suffixStr := ""
	if len(v.Suffix.Segments) > 0 {
		suffixStr = "/" + formatRoutingSegments(v.Suffix.Segments)
	}
	pattern := prefixStr + matchingStr + suffixStr
	isFullMatch := pattern == "(.*)"
	return pattern, isFullMatch
}

func parseExplicitRoutingParams(m *api.Method) []*explicitRoutingParam {
	if m == nil || !m.HasRouting() {
		return nil
	}

	var params []*explicitRoutingParam
	for _, info := range m.Routing {
		p := &explicitRoutingParam{
			name:         info.Name,
			minPrefixLen: 999999,
		}
		for _, v := range info.Variants {
			if len(v.Prefix.Segments) < p.minPrefixLen {
				p.minPrefixLen = len(v.Prefix.Segments)
			}
			var fieldParts []string
			for _, f := range v.FieldPath {
				fieldParts = append(fieldParts, cppFieldName(f))
			}
			accessor := "request." + strings.Join(fieldParts, "().") + "()"
			pattern, isFullMatch := formatRoutingPattern(v)
			p.matchers = append(p.matchers, &routingParamMatcher{
				accessor:    accessor,
				pattern:     pattern,
				isFullMatch: isFullMatch,
			})
		}
		if p.minPrefixLen == 999999 {
			p.minPrefixLen = 0
		}
		params = append(params, p)
	}

	// Match google-cloud-cpp iteration ordering:
	// "routing_id" parameters are placed after resource identity parameters,
	// and remaining parameters are ordered by their path prefix length.
	slices.SortStableFunc(params, func(a, b *explicitRoutingParam) int {
		if a.name == "routing_id" && b.name != "routing_id" {
			return 1
		}
		if b.name == "routing_id" && a.name != "routing_id" {
			return -1
		}
		if a.minPrefixLen != b.minPrefixLen {
			return a.minPrefixLen - b.minPrefixLen
		}
		return strings.Compare(a.name, b.name)
	})

	return params
}

func formatMetadataDecoratorSetMetadata(ann *methodAnnotations, contextVar, optionsVar string) string {
	if ann == nil {
		return fmt.Sprintf("  SetMetadata(%s, %s);", contextVar, optionsVar)
	}

	if ann.Method != nil && ann.Method.HasRouting() {
		params := parseExplicitRoutingParams(ann.Method)
		if len(params) > 0 {
			var b strings.Builder
			b.WriteString("  std::vector<std::string> params;\n")
			fmt.Fprintf(&b, "  params.reserve(%d);\n\n", len(params))

			requestType := ann.RequestType

			for _, p := range params {
				if p.isAllFullMatch() {
					for i, m := range p.matchers {
						if i == 0 {
							fmt.Fprintf(&b, "  if (!%s.empty()) {\n", m.accessor)
						} else {
							fmt.Fprintf(&b, " else if (!%s.empty()) {\n", m.accessor)
						}
						fmt.Fprintf(&b, "    params.push_back(absl::StrCat(\"%s=\", internal::UrlEncode(%s)));\n", p.name, m.accessor)
						b.WriteString("  }")
					}
					b.WriteString("\n\n")
				} else {
					fmt.Fprintf(&b, "  static auto* %s_matcher = []{\n", p.name)
					fmt.Fprintf(&b, "    return new google::cloud::internal::RoutingMatcher<%s>{\n", requestType)
					fmt.Fprintf(&b, "      \"%s=\", {\n", p.name)
					for _, m := range p.matchers {
						fmt.Fprintf(&b, "      {[](%s const& request) -> std::string const& {\n", requestType)
						fmt.Fprintf(&b, "        return %s;\n", m.accessor)
						b.WriteString("      },\n")
						if m.isFullMatch {
							b.WriteString("      absl::nullopt},\n")
						} else {
							fmt.Fprintf(&b, "      std::regex{%q, std::regex::optimize}},\n", m.pattern)
						}
					}
					b.WriteString("      }};\n")
					b.WriteString("  }();\n")
					fmt.Fprintf(&b, "  %s_matcher->AppendParam(request, params);\n\n", p.name)
				}
			}

			fmt.Fprintf(&b, "  if (params.empty()) {\n    SetMetadata(%s, %s);\n  } else {\n    SetMetadata(%s, %s, absl::StrJoin(params, \"&\"));\n  }",
				contextVar, optionsVar, contextVar, optionsVar)
			return b.String()
		}
	}

	if ann.Method != nil && ann.Method.PathInfo != nil && len(ann.Method.PathInfo.Bindings) > 0 {
		binding := ann.Method.PathInfo.Bindings[0]
		if binding.PathTemplate != nil {
			var parts []string
			for _, seg := range binding.PathTemplate.Segments {
				if seg.Variable != nil && len(seg.Variable.FieldPath) > 0 {
					varName := strings.Join(seg.Variable.FieldPath, ".")
					varParts := make([]string, len(seg.Variable.FieldPath))
					for i, p := range seg.Variable.FieldPath {
						varParts[i] = cppFieldName(p)
					}
					varAccessor := strings.Join(varParts, "().") + "()"
					parts = append(parts, fmt.Sprintf(`"%s=", internal::UrlEncode(request.%s)`, varName, varAccessor))
				}
			}
			if len(parts) > 0 {
				arg := strings.Join(parts, `, "&", `)
				return fmt.Sprintf("  SetMetadata(%s, %s, absl::StrCat(%s));", contextVar, optionsVar, arg)
			}
		}
	}

	return fmt.Sprintf("  SetMetadata(%s, %s);", contextVar, optionsVar)
}

func formatRestMetadataDecoratorSetMetadata(ann *methodAnnotations, contextVar, optionsVar string) string {
	if ann == nil {
		return fmt.Sprintf("  SetMetadata(%s, %s);", contextVar, optionsVar)
	}

	if ann.Method != nil && ann.Method.HasRouting() {
		params := parseExplicitRoutingParams(ann.Method)
		if len(params) > 0 {
			var b strings.Builder
			b.WriteString("  std::vector<std::string> params;\n")
			fmt.Fprintf(&b, "  params.reserve(%d);\n\n", len(params))

			requestType := ann.RequestType

			for _, p := range params {
				if p.isAllFullMatch() {
					for i, m := range p.matchers {
						if i == 0 {
							fmt.Fprintf(&b, "  if (!%s.empty()) {\n", m.accessor)
						} else {
							fmt.Fprintf(&b, " else if (!%s.empty()) {\n", m.accessor)
						}
						fmt.Fprintf(&b, "    params.push_back(absl::StrCat(\"%s=\", internal::UrlEncode(%s)));\n", p.name, m.accessor)
						b.WriteString("  }")
					}
					b.WriteString("\n\n")
				} else {
					fmt.Fprintf(&b, "  static auto* %s_matcher = []{\n", p.name)
					fmt.Fprintf(&b, "    return new google::cloud::internal::RoutingMatcher<%s>{\n", requestType)
					fmt.Fprintf(&b, "      \"%s=\", {\n", p.name)
					for _, m := range p.matchers {
						fmt.Fprintf(&b, "      {[](%s const& request) -> std::string const& {\n", requestType)
						fmt.Fprintf(&b, "        return %s;\n", m.accessor)
						b.WriteString("      },\n")
						if m.isFullMatch {
							b.WriteString("      absl::nullopt},\n")
						} else {
							fmt.Fprintf(&b, "      std::regex{%q, std::regex::optimize}},\n", m.pattern)
						}
					}
					b.WriteString("      }};\n")
					b.WriteString("  }();\n")
					fmt.Fprintf(&b, "  %s_matcher->AppendParam(request, params);\n\n", p.name)
				}
			}

			fmt.Fprintf(&b, "  SetMetadata(%s, %s, params);\n", contextVar, optionsVar)
			return b.String()
		}
	}

	return fmt.Sprintf("  SetMetadata(%s, %s);", contextVar, optionsVar)
}
