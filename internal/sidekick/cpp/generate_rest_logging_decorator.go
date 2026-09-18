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
	"path/filepath"
	"slices"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/sidekick/api"
)

func buildRestLoggingDecoratorMethodList(svc *api.Service, methods []*api.Method, serviceVars map[string]string, lib *config.Library, model *api.API) []map[string]any {
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
		}
		list = append(list, entry)
	}
	return list
}

func buildRestLoggingDecoratorAsyncMethodList(svc *api.Service, asyncMethods []*api.Method, serviceVars map[string]string, lib *config.Library, model *api.API) []map[string]any {
	var list []map[string]any
	for _, m := range getRestAsyncMethods(asyncMethods) {
		mVars := buildMethodVars(svc, m, serviceVars, lib, model)
		entry := map[string]any{
			"method_name":   mVars["method_name"],
			"request_type":  mVars["request_type"],
			"response_type": mVars["response_type"],
			"return_type":   mVars["return_type"],
		}
		list = append(list, entry)
	}
	return list
}

func generateRestLoggingDecoratorHeader(svc *api.Service, serviceVars map[string]string, methods, asyncMethods []*api.Method, lib *config.Library, model *api.API) (string, string) {
	headerPath := serviceVars["logging_rest_header_path"]
	guard := formatHeaderIncludeGuard(headerPath)

	localIncludes := []string{
		serviceVars["stub_rest_header_path"],
		"google/cloud/future.h",
		"google/cloud/internal/rest_context.h",
		"google/cloud/tracing_options.h",
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
		"logging_rest_class_name":    serviceVars["logging_rest_class_name"],
		"stub_rest_class_name":       serviceVars["stub_rest_class_name"],
		"local_includes":             localIncludes,
		"proto_includes":             protoIncludes,
		"methods":                    buildRestLoggingDecoratorMethodList(svc, methods, serviceVars, lib, model),
		"async_methods":              buildRestLoggingDecoratorAsyncMethodList(svc, asyncMethods, serviceVars, lib, model),
		"has_lro":                    hasLongrunningMethod(methods),
	}

	content, err := renderTemplate("templates/internal/rest_logging_decorator.h.mustache", data)
	if err != nil {
		panic(err)
	}

	return filepath.Clean(headerPath), content
}

func generateRestLoggingDecoratorCc(svc *api.Service, serviceVars map[string]string, methods, asyncMethods []*api.Method, lib *config.Library, model *api.API) (string, string) {
	ccPath := serviceVars["logging_rest_cc_path"]

	localIncludes := []string{
		serviceVars["logging_rest_header_path"],
		"google/cloud/internal/log_wrapper.h",
		"google/cloud/status_or.h",
	}
	slices.Sort(localIncludes)

	data := map[string]any{
		"copyright_year":             serviceVars["copyright_year"],
		"proto_file_name":            serviceVars["proto_file_name"],
		"product_internal_namespace": serviceVars["product_internal_namespace"],
		"logging_rest_class_name":    serviceVars["logging_rest_class_name"],
		"stub_rest_class_name":       serviceVars["stub_rest_class_name"],
		"local_includes":             localIncludes,
		"methods":                    buildRestLoggingDecoratorMethodList(svc, methods, serviceVars, lib, model),
		"async_methods":              buildRestLoggingDecoratorAsyncMethodList(svc, asyncMethods, serviceVars, lib, model),
		"has_lro":                    hasLongrunningMethod(methods),
	}

	content, err := renderTemplate("templates/internal/rest_logging_decorator.cc.mustache", data)
	if err != nil {
		panic(err)
	}

	return filepath.Clean(ccPath), content
}
