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

func buildTracingConnectionMethodList(methods []*api.Method) []map[string]any {
	var list []map[string]any
	for _, m := range methods {
		if isStreamingWrite(m) {
			continue
		}
		mann := m.Codec.(*methodAnnotations)
		entry := map[string]any{
			"method_name":   mann.MethodName(),
			"request_type":  mann.RequestType(),
			"response_type": mann.ResponseType(),
			"return_type":   mann.ReturnType(),
		}
		if isBidiStreaming(m) {
			entry["is_bidi_streaming"] = true
		} else if isStreamingRead(m) {
			entry["is_streaming_read"] = true
		} else if isPaginated(m) {
			entry["is_paginated"] = true
			entry["range_output_type"] = mann.RangeOutputType()
		} else if isLongrunning(m) {
			entry["is_longrunning"] = true
			entry["longrunning_operation_type"] = mann.LongrunningOperationType()
			if isResponseTypeEmpty(m) {
				entry["is_response_type_empty"] = true
				entry["lro_return_type"] = "future<Status>"
			} else {
				entry["longrunning_deduced_response_type"] = mann.LongrunningDeducedResponseType()
				entry["lro_return_type"] = "future<StatusOr<" + mann.LongrunningDeducedResponseType() + ">>"
			}
		} else {
			entry["is_plain_unary"] = true
		}
		list = append(list, entry)
	}
	return list
}

func buildTracingConnectionAsyncMethodList(asyncMethods []*api.Method) []map[string]any {
	var list []map[string]any
	for _, m := range asyncMethods {
		if isStreamingRead(m) || isStreamingWrite(m) {
			continue
		}
		mann := m.Codec.(*methodAnnotations)
		list = append(list, map[string]any{
			"method_name":  mann.MethodName(),
			"request_type": mann.RequestType(),
			"return_type":  mann.ReturnType(),
		})
	}
	return list
}

func generateTracingConnectionHeader(_ *api.Service, ann *serviceAnnotations, methods, asyncMethods []*api.Method, _ *config.Library, _ *api.API) (string, string) {
	headerPath := ann.TracingConnectionHeaderPath()
	guard := formatHeaderIncludeGuard(headerPath)

	localIncludes := []string{
		ann.ConnectionHeaderPath(),
		"google/cloud/version.h",
	}
	slices.Sort(localIncludes)

	data := map[string]any{
		"header_include_guard":          guard,
		"copyright_year":                ann.CopyrightYear,
		"proto_file_name":               ann.ProtoFileName,
		"product_internal_namespace":    ann.InternalNamespace(),
		"product_namespace":             ann.Namespace(),
		"connection_class_name":         ann.ConnectionClassName(),
		"tracing_connection_class_name": ann.TracingConnectionClassName(),
		"local_includes":                localIncludes,
		"system_includes":               []string{"memory"},
		"methods":                       buildTracingConnectionMethodList(methods),
		"async_methods":                 buildTracingConnectionAsyncMethodList(asyncMethods),
	}

	content, err := renderTemplate("templates/internal/tracing_connection.h.mustache", data)
	if err != nil {
		panic(err)
	}

	return filepath.Clean(headerPath), content
}

func generateTracingConnectionCc(_ *api.Service, ann *serviceAnnotations, methods, asyncMethods []*api.Method, _ *config.Library, _ *api.API) (string, string) {
	ccPath := ann.TracingConnectionCcPath()

	localIncludes := []string{
		ann.TracingConnectionHeaderPath(),
		"google/cloud/internal/opentelemetry.h",
	}
	if hasPaginatedMethod(methods) || hasStreamingReadMethod(methods) {
		localIncludes = append(localIncludes, "google/cloud/internal/traced_stream_range.h")
	}
	if len(localIncludes) > 1 {
		slices.Sort(localIncludes[1:])
	}

	data := map[string]any{
		"copyright_year":                ann.CopyrightYear,
		"proto_file_name":               ann.ProtoFileName,
		"product_internal_namespace":    ann.InternalNamespace(),
		"product_namespace":             ann.Namespace(),
		"connection_class_name":         ann.ConnectionClassName(),
		"tracing_connection_class_name": ann.TracingConnectionClassName(),
		"local_includes":                localIncludes,
		"system_includes":               []string{"memory", "utility"},
		"methods":                       buildTracingConnectionMethodList(methods),
		"async_methods":                 buildTracingConnectionAsyncMethodList(asyncMethods),
	}

	content, err := renderTemplate("templates/internal/tracing_connection.cc.mustache", data)
	if err != nil {
		panic(err)
	}

	return filepath.Clean(ccPath), content
}

