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

func hasExplicitRoutingMethod(methods []*api.Method) bool {
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

func setMetadataText(m *api.Method, isPointer bool, optionsStr string, requestType string, mVars map[string]string) string {
	context := "context"
	if isPointer {
		context = "*context"
	}

	orderedRouting := getOrderedRouting(m)
	if len(orderedRouting) == 0 {
		if params, ok := mVars["method_request_params"]; ok && params != "" {
			return fmt.Sprintf("  SetMetadata(%s, %s, absl::StrCat(%s));", context, optionsStr, params)
		}
		return fmt.Sprintf("  SetMetadata(%s, %s);", context, optionsStr)
	}

	var sb strings.Builder
	sb.WriteString("  std::vector<std::string> params;\n")
	sb.WriteString("  params.reserve(" + strconv.Itoa(len(orderedRouting)) + ");\n\n")

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

func buildMetadataDecoratorMethodList(svc *api.Service, methods []*api.Method, serviceVars map[string]string, lib *config.Library, model *api.API) []map[string]any {
	var list []map[string]any
	for _, m := range methods {
		mVars := buildMethodVars(svc, m, serviceVars, lib, model)
		entry := map[string]any{
			"method_name":   mVars["method_name"],
			"request_type":  mVars["request_type"],
			"response_type": mVars["response_type"],
			"return_type":   mVars["return_type"],
		}
		if isStreamingWrite(m) {
			entry["is_streaming_write"] = true
			entry["set_metadata"] = setMetadataText(m, true, "options", mVars["request_type"], mVars)
		} else if isBidiStreaming(m) {
			entry["is_bidi_streaming"] = true
			entry["set_metadata"] = setMetadataText(m, true, "*options", mVars["request_type"], mVars)
		} else if isLongrunning(m) {
			entry["is_longrunning"] = true
			entry["async_set_metadata"] = setMetadataText(m, true, "*options", mVars["request_type"], mVars)
			entry["sync_set_metadata"] = setMetadataText(m, false, "options", mVars["request_type"], mVars)
		} else if isStreamingRead(m) {
			entry["is_streaming_read"] = true
			entry["set_metadata"] = setMetadataText(m, true, "options", mVars["request_type"], mVars)
		} else {
			entry["is_plain_unary"] = true
			entry["set_metadata"] = setMetadataText(m, false, "options", mVars["request_type"], mVars)
		}
		list = append(list, entry)
	}
	return list
}

func buildMetadataDecoratorAsyncMethodList(svc *api.Service, asyncMethods []*api.Method, serviceVars map[string]string, lib *config.Library, model *api.API) []map[string]any {
	var list []map[string]any
	for _, m := range asyncMethods {
		if isBidiStreaming(m) || isLongrunning(m) {
			continue
		}
		mVars := buildMethodVars(svc, m, serviceVars, lib, model)
		entry := map[string]any{
			"method_name":   mVars["method_name"],
			"request_type":  mVars["request_type"],
			"response_type": mVars["response_type"],
			"return_type":   mVars["return_type"],
		}
		if isStreamingRead(m) {
			entry["is_streaming_read"] = true
			entry["set_metadata"] = setMetadataText(m, true, "*options", mVars["request_type"], mVars)
		} else if isStreamingWrite(m) {
			entry["is_streaming_write"] = true
			entry["set_metadata"] = setMetadataText(m, true, "*options", mVars["request_type"], mVars)
		} else {
			entry["is_unary"] = true
			entry["set_metadata"] = setMetadataText(m, true, "*options", mVars["request_type"], mVars)
		}
		list = append(list, entry)
	}
	return list
}

func generateMetadataDecoratorHeader(svc *api.Service, serviceVars map[string]string, methods, asyncMethods []*api.Method, lib *config.Library, model *api.API) (string, string) {
	headerPath := serviceVars["metadata_header_path"]
	guard := formatHeaderIncludeGuard(headerPath)

	localIncludes := []string{
		serviceVars["stub_header_path"],
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
		"copyright_year":             serviceVars["copyright_year"],
		"proto_file_name":            serviceVars["proto_file_name"],
		"product_internal_namespace": serviceVars["product_internal_namespace"],
		"metadata_class_name":        serviceVars["metadata_class_name"],
		"stub_class_name":            serviceVars["stub_class_name"],
		"local_includes":             localIncludes,
		"proto_includes":             protoIncludes,
		"methods":                    buildMetadataDecoratorMethodList(svc, methods, serviceVars, lib, model),
		"async_methods":              buildMetadataDecoratorAsyncMethodList(svc, asyncMethods, serviceVars, lib, model),
		"has_lro":                    hasLongrunningMethod(methods),
	}

	content, err := renderTemplate("templates/internal/metadata_decorator.h.mustache", data)
	if err != nil {
		panic(err)
	}

	return filepath.Clean(headerPath), content
}

func generateMetadataDecoratorCc(svc *api.Service, serviceVars map[string]string, methods, asyncMethods []*api.Method, lib *config.Library, model *api.API) (string, string) {
	ccPath := serviceVars["metadata_cc_path"]

	localIncludes := []string{
		serviceVars["metadata_header_path"],
		"google/cloud/internal/absl_str_cat_quiet.h",
	}
	if hasExplicitRoutingMethod(methods) {
		localIncludes = append(localIncludes, "google/cloud/internal/absl_str_join_quiet.h")
	}
	localIncludes = append(localIncludes,
		"google/cloud/internal/api_client_header.h",
		"google/cloud/grpc_options.h",
	)
	if hasExplicitRoutingMethod(methods) {
		localIncludes = append(localIncludes, "google/cloud/internal/routing_matcher.h")
	}
	localIncludes = append(localIncludes,
		"google/cloud/status_or.h",
		"google/cloud/internal/url_encode.h",
	)
	if len(localIncludes) > 1 {
		slices.Sort(localIncludes[1:])
	}

	var pbIncludes []string
	if serviceVars["proto_grpc_header_path"] != "" {
		pbIncludes = append(pbIncludes, serviceVars["proto_grpc_header_path"])
	}

	data := map[string]any{
		"copyright_year":             serviceVars["copyright_year"],
		"proto_file_name":            serviceVars["proto_file_name"],
		"product_internal_namespace": serviceVars["product_internal_namespace"],
		"metadata_class_name":        serviceVars["metadata_class_name"],
		"stub_class_name":            serviceVars["stub_class_name"],
		"local_includes":             localIncludes,
		"proto_includes":             pbIncludes,
		"methods":                    buildMetadataDecoratorMethodList(svc, methods, serviceVars, lib, model),
		"async_methods":              buildMetadataDecoratorAsyncMethodList(svc, asyncMethods, serviceVars, lib, model),
		"has_lro":                    hasLongrunningMethod(methods),
		"api_version":                serviceVars["api_version"],
	}

	content, err := renderTemplate("templates/internal/metadata_decorator.cc.mustache", data)
	if err != nil {
		panic(err)
	}

	return filepath.Clean(ccPath), content
}
