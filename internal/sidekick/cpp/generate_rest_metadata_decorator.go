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

func buildRestMetadataDecoratorMethodList(methods []*api.Method) []map[string]any {
	var list []map[string]any
	for _, m := range getRestMethods(methods) {
		mann := m.Codec.(*methodAnnotations)
		entry := map[string]any{
			"method_name":   mann.MethodName(),
			"request_type":  mann.RequestType(),
			"response_type": mann.ResponseType(),
			"return_type":   mann.ReturnType(),
		}
		if isLongrunning(m) {
			entry["is_longrunning"] = true
			entry["async_set_metadata"] = setRestMetadataText(m, true, mann.RequestType())
			entry["sync_set_metadata"] = setRestMetadataText(m, false, mann.RequestType())
		} else {
			entry["sync_set_metadata"] = setRestMetadataText(m, false, mann.RequestType())
		}
		list = append(list, entry)
	}
	return list
}

func buildRestMetadataDecoratorAsyncMethodList(asyncMethods []*api.Method) []map[string]any {
	var list []map[string]any
	for _, m := range getRestAsyncMethods(asyncMethods) {
		mann := m.Codec.(*methodAnnotations)
		entry := map[string]any{
			"method_name":        mann.MethodName(),
			"request_type":       mann.RequestType(),
			"response_type":      mann.ResponseType(),
			"return_type":        mann.ReturnType(),
			"async_set_metadata": setRestMetadataText(m, true, mann.RequestType()),
		}
		list = append(list, entry)
	}
	return list
}

func generateRestMetadataDecoratorHeader(_ *api.Service, ann *serviceAnnotations, methods, asyncMethods []*api.Method, _ *config.Library, _ *api.API) (string, string) {
	headerPath := ann.MetadataRestHeaderPath()
	guard := formatHeaderIncludeGuard(headerPath)

	localIncludes := []string{
		ann.StubRestHeaderPath(),
		"google/cloud/future.h",
		"google/cloud/rest_options.h",
		"google/cloud/version.h",
	}
	slices.Sort(localIncludes)

	var protoIncludes []string
	protoIncludes = append(protoIncludes, ann.ProtoHeaderPath())
	if hasLongrunningMethod(methods) {
		protoIncludes = append(protoIncludes, "google/longrunning/operations.pb.h")
	}
	slices.Sort(protoIncludes)

	data := map[string]any{
		"header_include_guard":       guard,
		"copyright_year":             ann.CopyrightYear,
		"proto_file_name":            ann.ProtoFileName,
		"product_internal_namespace": ann.InternalNamespace(),
		"metadata_rest_class_name":   ann.MetadataRestClassName(),
		"stub_rest_class_name":       ann.StubRestClassName(),
		"local_includes":             localIncludes,
		"proto_includes":             protoIncludes,
		"methods":                    buildRestMetadataDecoratorMethodList(methods),
		"async_methods":              buildRestMetadataDecoratorAsyncMethodList(asyncMethods),
		"has_lro":                    hasLongrunningMethod(methods),
	}

	content, err := renderTemplate("templates/internal/rest_metadata_decorator.h.mustache", data)
	if err != nil {
		panic(err)
	}

	return filepath.Clean(headerPath), content
}

func generateRestMetadataDecoratorCc(_ *api.Service, ann *serviceAnnotations, methods, asyncMethods []*api.Method, _ *config.Library, _ *api.API) (string, string) {
	ccPath := ann.MetadataRestCcPath()

	localIncludes := []string{
		ann.MetadataRestHeaderPath(),
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
		"copyright_year":             ann.CopyrightYear,
		"proto_file_name":            ann.ProtoFileName,
		"product_internal_namespace": ann.InternalNamespace(),
		"metadata_rest_class_name":   ann.MetadataRestClassName(),
		"stub_rest_class_name":       ann.StubRestClassName(),
		"local_includes":             localIncludes,
		"methods":                    buildRestMetadataDecoratorMethodList(methods),
		"async_methods":              buildRestMetadataDecoratorAsyncMethodList(asyncMethods),
		"has_lro":                    hasLongrunningMethod(methods),
		"api_version":                ann.APIVersion,
	}


	content, err := renderTemplate("templates/internal/rest_metadata_decorator.cc.mustache", data)
	if err != nil {
		panic(err)
	}

	return filepath.Clean(ccPath), content
}

