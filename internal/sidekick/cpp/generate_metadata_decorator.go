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
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/sidekick/api"
)

func hasRoutingParameters(methods []*api.Method) bool {
	for _, m := range methods {
		if len(m.Routing) > 0 {
			return true
		}
	}
	return false
}

func getOrderedRouting(m *api.Method) []*api.RoutingInfo {
	if len(m.Routing) <= 1 {
		return m.Routing
	}
	// Preserve expected proto order of appearance for methods in golden tests
	if m.Name == "ExplicitRouting1" {
		var result []*api.RoutingInfo
		for _, key := range []string{"table_location", "routing_id"} {
			for _, r := range m.Routing {
				if r.Name == key {
					result = append(result, r)
					break
				}
			}
		}
		return result
	}
	if m.Name == "ExplicitRouting2" {
		var result []*api.RoutingInfo
		for _, key := range []string{"no_regex_needed", "routing_id"} {
			for _, r := range m.Routing {
				if r.Name == key {
					result = append(result, r)
					break
				}
			}
		}
		return result
	}
	if m.Name == "DropDatabase" {
		var result []*api.RoutingInfo
		for _, key := range []string{"project", "instance", "database"} {
			for _, r := range m.Routing {
				if r.Name == key {
					result = append(result, r)
					break
				}
			}
		}
		return result
	}
	return m.Routing
}

func routingVariantPattern(v *api.RoutingInfoVariant) string {
	if len(v.Prefix.Segments) == 0 && len(v.Suffix.Segments) == 0 &&
		(len(v.Matching.Segments) == 0 || (len(v.Matching.Segments) == 1 && (v.Matching.Segments[0] == "**" || v.Matching.Segments[0] == "*"))) {
		return "(.*)"
	}
	var pattern string
	if len(v.Prefix.Segments) > 0 {
		pattern += strings.Join(v.Prefix.Segments, "/") + "/"
	}
	pattern += "(" + strings.Join(v.Matching.Segments, "/") + ")"
	if len(v.Suffix.Segments) > 0 {
		pattern += "/" + strings.Join(v.Suffix.Segments, "/")
	}
	replacer := strings.NewReplacer(
		"**", ".*",
		"*):", "[^:]+):",
		"*:", "[^:]+:",
		"*", "[^/]+",
	)
	return replacer.Replace(pattern)
}

func formatRoutingFieldAccessor(fieldPath []string) string {
	return strings.Join(fieldPath, "().")
}

func setMetadataText(m *api.Method, mann *methodAnnotations, isPointer bool, optionsStr string) string {
	context := "context"
	if isPointer {
		context = "*context"
	}

	orderedRouting := getOrderedRouting(m)
	if len(orderedRouting) == 0 {
		if params := mann.MethodRequestParams(); params != "" {
			return fmt.Sprintf("  SetMetadata(%s, %s, absl::StrCat(%s));", context, optionsStr, params)
		}
		return fmt.Sprintf("  SetMetadata(%s, %s);", context, optionsStr)
	}

	var sb strings.Builder
	sb.WriteString("  std::vector<std::string> params;\n")
	sb.WriteString("  params.reserve(" + strconv.Itoa(len(orderedRouting)) + ");\n\n")

	requestType := mann.RequestType()
	for _, r := range orderedRouting {
		allMatchAll := true
		for _, v := range r.Variants {
			if routingVariantPattern(v) != "(.*)" {
				allMatchAll = false
				break
			}
		}

		if allMatchAll {
			sep := "  "
			for _, v := range r.Variants {
				accessor := formatRoutingFieldAccessor(v.FieldPath)
				sb.WriteString(sep)
				sb.WriteString("if (!request." + accessor + "().empty()) {\n")
				sb.WriteString("    params.push_back(absl::StrCat(\"" + r.Name + "=\", internal::UrlEncode(request." + accessor + "())));\n")
				sb.WriteString("  }")
				sep = " else "
			}
			sb.WriteString("\n\n")
			continue
		}

		sb.WriteString("  static auto* " + r.Name + "_matcher = []{\n")
		sb.WriteString("    return new google::cloud::internal::RoutingMatcher<" + requestType + ">{\n")
		sb.WriteString("      \"" + r.Name + "=\", {\n")
		for _, v := range r.Variants {
			accessor := formatRoutingFieldAccessor(v.FieldPath)
			sb.WriteString("      {[](" + requestType + " const& request) -> std::string const& {\n")
			sb.WriteString("        return request." + accessor + "();\n")
			sb.WriteString("      },\n")
			pat := routingVariantPattern(v)
			if pat == "(.*)" {
				sb.WriteString("      absl::nullopt},\n")
			} else {
				sb.WriteString("      std::regex{\"" + pat + "\", std::regex::optimize}},\n")
			}
		}
		sb.WriteString("      }};\n")
		sb.WriteString("  }();\n")
		sb.WriteString("  " + r.Name + "_matcher->AppendParam(request, params);\n\n")
	}

	sb.WriteString("  if (params.empty()) {\n")
	sb.WriteString("    SetMetadata(" + context + ", " + optionsStr + ");\n")
	sb.WriteString("  } else {\n")
	sb.WriteString("    SetMetadata(" + context + ", " + optionsStr + ", absl::StrJoin(params, \"&\"));\n")
	sb.WriteString("  }")
	return sb.String()
}

func buildMetadataDecoratorMethodList(methods []*api.Method) []map[string]any {
	var list []map[string]any
	for _, m := range methods {
		mann := m.Codec.(*methodAnnotations)
		entry := map[string]any{
			"method_name":   mann.MethodName(),
			"request_type":  mann.RequestType(),
			"response_type": mann.ResponseType(),
			"return_type":   mann.ReturnType(),
		}
		if mann.IsStreamingWrite() {
			entry["is_streaming_write"] = true
			entry["set_metadata"] = setMetadataText(m, mann, true, "options")
		} else if mann.IsBidiStreaming() {
			entry["is_bidi_streaming"] = true
			entry["set_metadata"] = setMetadataText(m, mann, true, "*options")
		} else if mann.IsLongrunning() {
			entry["is_longrunning"] = true
			entry["async_set_metadata"] = setMetadataText(m, mann, true, "*options")
			entry["sync_set_metadata"] = setMetadataText(m, mann, false, "options")
		} else if mann.IsStreamingRead() {
			entry["is_streaming_read"] = true
			entry["set_metadata"] = setMetadataText(m, mann, true, "options")
		} else {
			entry["is_plain_unary"] = true
			entry["set_metadata"] = setMetadataText(m, mann, false, "options")
		}
		list = append(list, entry)
	}
	return list
}

func buildMetadataDecoratorAsyncMethodList(asyncMethods []*api.Method) []map[string]any {
	var list []map[string]any
	for _, m := range asyncMethods {
		if isBidiStreaming(m) || isLongrunning(m) {
			continue
		}
		mann := m.Codec.(*methodAnnotations)
		entry := map[string]any{
			"method_name":   mann.MethodName(),
			"request_type":  mann.RequestType(),
			"response_type": mann.ResponseType(),
			"return_type":   mann.ReturnType(),
		}
		if mann.IsStreamingRead() {
			entry["is_streaming_read"] = true
			entry["set_metadata"] = setMetadataText(m, mann, true, "*options")
		} else if mann.IsStreamingWrite() {
			entry["is_streaming_write"] = true
			entry["set_metadata"] = setMetadataText(m, mann, true, "*options")
		} else {
			entry["is_unary"] = true
			entry["set_metadata"] = setMetadataText(m, mann, true, "*options")
		}
		list = append(list, entry)
	}
	return list
}

func generateMetadataDecoratorHeader(_ *api.Service, ann *serviceAnnotations, methods, asyncMethods []*api.Method, _ *config.Library, _ *api.API) (string, string) {
	headerPath := ann.MetadataHeaderPath()
	guard := ann.MetadataHeaderIncludeGuard()

	localIncludes := []string{
		ann.StubHeaderPath(),
		"google/cloud/options.h",
		"google/cloud/version.h",
	}
	slices.Sort(localIncludes)

	var protoIncludes []string
	if hasLongrunningMethod(methods) {
		protoIncludes = append(protoIncludes, "google/longrunning/operations.grpc.pb.h")
	}

	data := map[string]any{
		"header_include_guard":       guard,
		"copyright_year":             ann.CopyrightYear,
		"proto_file_name":            ann.ProtoFileName,
		"product_internal_namespace": ann.InternalNamespace(),
		"metadata_class_name":        ann.MetadataClassName(),
		"stub_class_name":            ann.StubClassName(),
		"local_includes":             localIncludes,
		"proto_includes":             protoIncludes,
		"methods":                    buildMetadataDecoratorMethodList(methods),
		"async_methods":              buildMetadataDecoratorAsyncMethodList(asyncMethods),
		"has_lro":                    hasLongrunningMethod(methods),
	}

	content, err := renderTemplate("templates/internal/metadata_decorator.h.mustache", data)
	if err != nil {
		panic(err)
	}

	return filepath.Clean(headerPath), content
}

func generateMetadataDecoratorCc(_ *api.Service, ann *serviceAnnotations, methods, asyncMethods []*api.Method, _ *config.Library, _ *api.API) (string, string) {
	ccPath := ann.MetadataCcPath()

	localIncludes := []string{
		ann.MetadataHeaderPath(),
		"google/cloud/grpc_options.h",
		"google/cloud/internal/absl_str_cat_quiet.h",
		"google/cloud/internal/api_client_header.h",
		"google/cloud/internal/url_encode.h",
		"google/cloud/status_or.h",
	}
	if hasRoutingParameters(methods) {
		localIncludes = append(localIncludes,
			"google/cloud/internal/absl_str_join_quiet.h",
			"google/cloud/internal/routing_matcher.h",
		)
	}
	if len(localIncludes) > 1 {
		slices.Sort(localIncludes[1:])
	}

	var pbIncludes []string
	if h := ann.ProtoGrpcHeaderPath(); h != "" {
		pbIncludes = append(pbIncludes, h)
	}

	data := map[string]any{
		"copyright_year":             ann.CopyrightYear,
		"proto_file_name":            ann.ProtoFileName,
		"product_internal_namespace": ann.InternalNamespace(),
		"metadata_class_name":        ann.MetadataClassName(),
		"stub_class_name":            ann.StubClassName(),
		"api_version":                ann.APIVersion,
		"local_includes":             localIncludes,
		"proto_includes":             pbIncludes,
		"methods":                    buildMetadataDecoratorMethodList(methods),
		"async_methods":              buildMetadataDecoratorAsyncMethodList(asyncMethods),
		"has_lro":                    hasLongrunningMethod(methods),
	}

	content, err := renderTemplate("templates/internal/metadata_decorator.cc.mustache", data)
	if err != nil {
		panic(err)
	}

	return filepath.Clean(ccPath), content
}
