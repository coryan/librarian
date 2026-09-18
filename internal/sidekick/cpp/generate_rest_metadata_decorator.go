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

func setRestMetadataText(m *api.Method, isAsync bool, requestType string) string {
	context := "rest_context"
	optionsStr := "options"
	if isAsync {
		context = "*rest_context"
		optionsStr = "*options"
	}

	orderedRouting := getOrderedRouting(m)
	if len(orderedRouting) == 0 {
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

	sb.WriteString("  SetMetadata(" + context + ", " + optionsStr + ", params);\n")
	return sb.String()
}

func buildRestMetadataDecoratorMethodList(svc *api.Service, methods []*api.Method, serviceVars map[string]string, lib *config.Library, model *api.API) []map[string]any {
	var list []map[string]any
	for _, m := range getRestMethods(methods) {
		mVars := buildMethodVars(svc, m, serviceVars, lib, model)
		entry := map[string]any{
			"method_name":   mVars["method_name"],
			"request_type":  mVars["request_type"],
			"response_type": mVars["response_type"],
			"return_type":   mVars["return_type"],
		}
		if isLongrunning(m) {
			entry["is_longrunning"] = true
			entry["async_set_metadata"] = setRestMetadataText(m, true, mVars["request_type"])
			entry["sync_set_metadata"] = setRestMetadataText(m, false, mVars["request_type"])
		} else {
			entry["sync_set_metadata"] = setRestMetadataText(m, false, mVars["request_type"])
		}
		list = append(list, entry)
	}
	return list
}

func buildRestMetadataDecoratorAsyncMethodList(svc *api.Service, asyncMethods []*api.Method, serviceVars map[string]string, lib *config.Library, model *api.API) []map[string]any {
	var list []map[string]any
	for _, m := range getRestAsyncMethods(asyncMethods) {
		mVars := buildMethodVars(svc, m, serviceVars, lib, model)
		entry := map[string]any{
			"method_name":        mVars["method_name"],
			"request_type":       mVars["request_type"],
			"response_type":      mVars["response_type"],
			"return_type":        mVars["return_type"],
			"async_set_metadata": setRestMetadataText(m, true, mVars["request_type"]),
		}
		list = append(list, entry)
	}
	return list
}

func generateRestMetadataDecoratorHeader(svc *api.Service, serviceVars map[string]string, methods, asyncMethods []*api.Method, lib *config.Library, model *api.API) (string, string) {
	headerPath := serviceVars["metadata_rest_header_path"]
	guard := formatHeaderIncludeGuard(headerPath)

	localIncludes := []string{
		serviceVars["stub_rest_header_path"],
		"google/cloud/future.h",
		"google/cloud/rest_options.h",
		"google/cloud/version.h",
	}
	slices.Sort(localIncludes)

	var protoIncludes []string
	protoIncludes = append(protoIncludes, serviceVars["proto_header_path"])
	if hasLongrunningMethod(methods) {
		protoIncludes = append(protoIncludes, "google/longrunning/operations.pb.h")
	}
	slices.Sort(protoIncludes)

	data := map[string]any{
		"header_include_guard":       guard,
		"copyright_year":             serviceVars["copyright_year"],
		"proto_file_name":            serviceVars["proto_file_name"],
		"product_internal_namespace": serviceVars["product_internal_namespace"],
		"metadata_rest_class_name":   serviceVars["metadata_rest_class_name"],
		"stub_rest_class_name":       serviceVars["stub_rest_class_name"],
		"local_includes":             localIncludes,
		"proto_includes":             protoIncludes,
		"methods":                    buildRestMetadataDecoratorMethodList(svc, methods, serviceVars, lib, model),
		"async_methods":              buildRestMetadataDecoratorAsyncMethodList(svc, asyncMethods, serviceVars, lib, model),
		"has_lro":                    hasLongrunningMethod(methods),
	}

	content, err := renderTemplate("templates/internal/rest_metadata_decorator.h.mustache", data)
	if err != nil {
		panic(err)
	}

	return filepath.Clean(headerPath), content
}

func generateRestMetadataDecoratorCc(svc *api.Service, serviceVars map[string]string, methods, asyncMethods []*api.Method, lib *config.Library, model *api.API) (string, string) {
	ccPath := serviceVars["metadata_rest_cc_path"]

	localIncludes := []string{
		serviceVars["metadata_rest_header_path"],
		"absl/strings/str_format.h",
		"google/cloud/internal/absl_str_cat_quiet.h",
		"google/cloud/internal/api_client_header.h",
		"google/cloud/internal/rest_set_metadata.h",
		"google/cloud/status_or.h",
	}
	if hasExplicitRoutingMethod(methods) {
		localIncludes = append(localIncludes, "google/cloud/internal/routing_matcher.h")
	}
	slices.Sort(localIncludes[1:])

	data := map[string]any{
		"copyright_year":             serviceVars["copyright_year"],
		"proto_file_name":            serviceVars["proto_file_name"],
		"product_internal_namespace": serviceVars["product_internal_namespace"],
		"metadata_rest_class_name":   serviceVars["metadata_rest_class_name"],
		"stub_rest_class_name":       serviceVars["stub_rest_class_name"],
		"local_includes":             localIncludes,
		"methods":                    buildRestMetadataDecoratorMethodList(svc, methods, serviceVars, lib, model),
		"async_methods":              buildRestMetadataDecoratorAsyncMethodList(svc, asyncMethods, serviceVars, lib, model),
		"has_lro":                    hasLongrunningMethod(methods),
		"api_version":                serviceVars["api_version"],
	}

	content, err := renderTemplate("templates/internal/rest_metadata_decorator.cc.mustache", data)
	if err != nil {
		panic(err)
	}

	return filepath.Clean(ccPath), content
}
