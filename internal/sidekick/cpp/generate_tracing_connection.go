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

func buildTracingConnectionMethodList(svc *api.Service, methods []*api.Method, serviceVars map[string]string, lib *config.Library, model *api.API) []map[string]any {
	var list []map[string]any
	for _, m := range methods {
		if isStreamingWrite(m) {
			continue
		}
		mVars := buildMethodVars(svc, m, serviceVars, lib, model)
		entry := map[string]any{
			"method_name":   mVars["method_name"],
			"request_type":  mVars["request_type"],
			"response_type": mVars["response_type"],
			"return_type":   mVars["return_type"],
		}
		if isBidiStreaming(m) {
			entry["is_bidi_streaming"] = true
		} else if isStreamingRead(m) {
			entry["is_streaming_read"] = true
		} else if isPaginated(m) {
			entry["is_paginated"] = true
			entry["range_output_type"] = mVars["range_output_type"]
		} else if isLongrunning(m) {
			entry["is_longrunning"] = true
			entry["longrunning_operation_type"] = mVars["longrunning_operation_type"]
			if isResponseTypeEmpty(m) {
				entry["is_response_type_empty"] = true
				entry["lro_return_type"] = "future<Status>"
			} else {
				entry["longrunning_deduced_response_type"] = mVars["longrunning_deduced_response_type"]
				entry["lro_return_type"] = "future<StatusOr<" + mVars["longrunning_deduced_response_type"] + ">>"
			}
		} else {
			entry["is_plain_unary"] = true
		}
		list = append(list, entry)
	}
	return list
}

func buildTracingConnectionAsyncMethodList(svc *api.Service, asyncMethods []*api.Method, serviceVars map[string]string, lib *config.Library, model *api.API) []map[string]any {
	var list []map[string]any
	for _, m := range asyncMethods {
		if isStreamingRead(m) || isStreamingWrite(m) {
			continue
		}
		mVars := buildMethodVars(svc, m, serviceVars, lib, model)
		list = append(list, map[string]any{
			"method_name":  mVars["method_name"],
			"request_type": mVars["request_type"],
			"return_type":  mVars["return_type"],
		})
	}
	return list
}

func generateTracingConnectionHeader(svc *api.Service, serviceVars map[string]string, methods, asyncMethods []*api.Method, lib *config.Library, model *api.API) (string, string) {
	headerPath := serviceVars["tracing_connection_header_path"]
	guard := formatHeaderIncludeGuard(headerPath)

	localIncludes := []string{
		serviceVars["connection_header_path"],
		"google/cloud/version.h",
	}
	slices.Sort(localIncludes)

	data := map[string]any{
		"header_include_guard":          guard,
		"copyright_year":                serviceVars["copyright_year"],
		"proto_file_name":               serviceVars["proto_file_name"],
		"product_internal_namespace":    serviceVars["product_internal_namespace"],
		"product_namespace":             serviceVars["product_namespace"],
		"connection_class_name":         serviceVars["connection_class_name"],
		"tracing_connection_class_name": serviceVars["tracing_connection_class_name"],
		"local_includes":                localIncludes,
		"system_includes":               []string{"memory"},
		"methods":                       buildTracingConnectionMethodList(svc, methods, serviceVars, lib, model),
		"async_methods":                 buildTracingConnectionAsyncMethodList(svc, asyncMethods, serviceVars, lib, model),
	}

	content, err := renderTemplate("templates/internal/tracing_connection.h.mustache", data)
	if err != nil {
		panic(err)
	}

	return filepath.Clean(headerPath), content
}

func generateTracingConnectionCc(svc *api.Service, serviceVars map[string]string, methods, asyncMethods []*api.Method, lib *config.Library, model *api.API) (string, string) {
	ccPath := serviceVars["tracing_connection_cc_path"]

	localIncludes := []string{
		serviceVars["tracing_connection_header_path"],
		"google/cloud/internal/opentelemetry.h",
	}
	if hasPaginatedMethod(methods) || hasStreamingReadMethod(methods) {
		localIncludes = append(localIncludes, "google/cloud/internal/traced_stream_range.h")
	}
	if len(localIncludes) > 1 {
		slices.Sort(localIncludes[1:])
	}

	data := map[string]any{
		"copyright_year":                serviceVars["copyright_year"],
		"proto_file_name":               serviceVars["proto_file_name"],
		"product_internal_namespace":    serviceVars["product_internal_namespace"],
		"product_namespace":             serviceVars["product_namespace"],
		"connection_class_name":         serviceVars["connection_class_name"],
		"tracing_connection_class_name": serviceVars["tracing_connection_class_name"],
		"local_includes":                localIncludes,
		"system_includes":               []string{"memory", "utility"},
		"methods":                       buildTracingConnectionMethodList(svc, methods, serviceVars, lib, model),
		"async_methods":                 buildTracingConnectionAsyncMethodList(svc, asyncMethods, serviceVars, lib, model),
	}

	content, err := renderTemplate("templates/internal/tracing_connection.cc.mustache", data)
	if err != nil {
		panic(err)
	}

	return filepath.Clean(ccPath), content
}
