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

func generateTracingStubHeader(svc *api.Service, serviceVars map[string]string, methods, asyncMethods []*api.Method, lib *config.Library, model *api.API) (string, string) {
	headerPath := serviceVars["tracing_stub_header_path"]
	guard := formatHeaderIncludeGuard(headerPath)

	localIncludes := []string{
		serviceVars["stub_header_path"],
		"google/cloud/internal/trace_propagator.h",
		"google/cloud/options.h",
		"google/cloud/version.h",
	}
	slices.Sort(localIncludes)

	data := map[string]any{
		"header_include_guard":       guard,
		"copyright_year":             serviceVars["copyright_year"],
		"proto_file_name":            serviceVars["proto_file_name"],
		"product_internal_namespace": serviceVars["product_internal_namespace"],
		"tracing_stub_class_name":    serviceVars["tracing_stub_class_name"],
		"stub_class_name":            serviceVars["stub_class_name"],
		"local_includes":             localIncludes,
		"methods":                    buildDecoratorMethodList(svc, methods, serviceVars, lib, model),
		"async_methods":              buildDecoratorAsyncMethodList(svc, asyncMethods, serviceVars, lib, model),
		"has_lro":                    hasLongrunningMethod(methods),
	}

	content, err := renderTemplate("templates/internal/tracing_stub.h.mustache", data)
	if err != nil {
		panic(err)
	}

	return filepath.Clean(headerPath), content
}

func buildTracingStubMethodList(svc *api.Service, methods []*api.Method, serviceVars map[string]string, lib *config.Library, model *api.API) []map[string]any {
	var list []map[string]any
	for _, m := range methods {
		mVars := buildMethodVars(svc, m, serviceVars, lib, model)
		reqIdFragment := ""
		if len(m.AutoPopulated) > 0 {
			reqIdFragment = "\n  span->SetAttribute(\"gl-cpp.request_id\", request." + mVars["request_id_field_name"] + "());"
		}
		entry := map[string]any{
			"method_name":         mVars["method_name"],
			"request_type":        mVars["request_type"],
			"response_type":       mVars["response_type"],
			"return_type":         mVars["return_type"],
			"grpc_service":        mVars["grpc_service"],
			"request_id_fragment": reqIdFragment,
		}
		if isStreamingWrite(m) {
			entry["is_streaming_write"] = true
		} else if isBidiStreaming(m) {
			entry["is_bidi_streaming"] = true
		} else if isLongrunning(m) {
			entry["is_longrunning"] = true
		} else if isStreamingRead(m) {
			entry["is_streaming_read"] = true
		} else {
			entry["is_plain_unary"] = true
		}
		list = append(list, entry)
	}
	return list
}

func buildTracingStubAsyncMethodList(svc *api.Service, asyncMethods []*api.Method, serviceVars map[string]string, lib *config.Library, model *api.API) []map[string]any {
	var list []map[string]any
	for _, m := range asyncMethods {
		if isBidiStreaming(m) || isLongrunning(m) {
			continue
		}
		mVars := buildMethodVars(svc, m, serviceVars, lib, model)
		reqIdFragment := ""
		if len(m.AutoPopulated) > 0 {
			reqIdFragment = "\n  span->SetAttribute(\"gl-cpp.request_id\", request." + mVars["request_id_field_name"] + "());"
		}
		entry := map[string]any{
			"method_name":         mVars["method_name"],
			"request_type":        mVars["request_type"],
			"response_type":       mVars["response_type"],
			"return_type":         mVars["return_type"],
			"grpc_service":        mVars["grpc_service"],
			"request_id_fragment": reqIdFragment,
		}
		if isStreamingRead(m) {
			entry["is_streaming_read"] = true
		} else if isStreamingWrite(m) {
			entry["is_streaming_write"] = true
		} else {
			entry["is_unary"] = true
		}
		list = append(list, entry)
	}
	return list
}

func generateTracingStubCc(svc *api.Service, serviceVars map[string]string, methods, asyncMethods []*api.Method, lib *config.Library, model *api.API) (string, string) {
	ccPath := serviceVars["tracing_stub_cc_path"]

	localIncludes := []string{serviceVars["tracing_stub_header_path"]}
	if hasAsynchronousStreamingReadMethod(asyncMethods) {
		localIncludes = append(localIncludes, "google/cloud/internal/async_streaming_read_rpc_tracing.h")
	}
	if hasAsynchronousStreamingWriteMethod(asyncMethods) {
		localIncludes = append(localIncludes, "google/cloud/internal/async_streaming_write_rpc_tracing.h")
	}
	if hasBidiStreamingMethod(methods) {
		localIncludes = append(localIncludes, "google/cloud/internal/async_read_write_stream_tracing.h")
	}
	if hasStreamingReadMethod(methods) {
		localIncludes = append(localIncludes, "google/cloud/internal/streaming_read_rpc_tracing.h")
	}
	if hasStreamingWriteMethod(methods) {
		localIncludes = append(localIncludes, "google/cloud/internal/streaming_write_rpc_tracing.h")
	}
	localIncludes = append(localIncludes, "google/cloud/internal/grpc_opentelemetry.h")
	if len(localIncludes) > 1 {
		slices.Sort(localIncludes[1:])
	}

	data := map[string]any{
		"copyright_year":             serviceVars["copyright_year"],
		"proto_file_name":            serviceVars["proto_file_name"],
		"product_internal_namespace": serviceVars["product_internal_namespace"],
		"tracing_stub_class_name":    serviceVars["tracing_stub_class_name"],
		"stub_class_name":            serviceVars["stub_class_name"],
		"local_includes":             localIncludes,
		"methods":                    buildTracingStubMethodList(svc, methods, serviceVars, lib, model),
		"async_methods":              buildTracingStubAsyncMethodList(svc, asyncMethods, serviceVars, lib, model),
		"has_lro":                    hasLongrunningMethod(methods),
	}

	content, err := renderTemplate("templates/internal/tracing_stub.cc.mustache", data)
	if err != nil {
		panic(err)
	}

	return filepath.Clean(ccPath), content
}
