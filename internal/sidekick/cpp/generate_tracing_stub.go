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

func generateTracingStubHeader(_ *api.Service, ann *serviceAnnotations, methods, asyncMethods []*api.Method, _ *config.Library, _ *api.API) (string, string) {
	headerPath := ann.TracingStubHeaderPath()
	guard := ann.TracingStubHeaderIncludeGuard()

	localIncludes := []string{
		ann.StubHeaderPath(),
		"google/cloud/internal/trace_propagator.h",
		"google/cloud/options.h",
		"google/cloud/version.h",
	}
	slices.Sort(localIncludes)

	data := map[string]any{
		"header_include_guard":       guard,
		"copyright_year":             ann.CopyrightYear,
		"proto_file_name":            ann.ProtoFileName,
		"product_internal_namespace": ann.InternalNamespace(),
		"tracing_stub_class_name":    ann.TracingStubClassName(),
		"stub_class_name":            ann.StubClassName(),
		"local_includes":             localIncludes,
		"methods":                    buildDecoratorMethodList(methods),
		"async_methods":              buildDecoratorAsyncMethodList(asyncMethods),
		"has_lro":                    hasLongrunningMethod(methods),
	}

	content, err := renderTemplate("templates/internal/tracing_stub.h.mustache", data)
	if err != nil {
		panic(err)
	}

	return filepath.Clean(headerPath), content
}

func buildTracingStubMethodList(ann *serviceAnnotations, methods []*api.Method) []map[string]any {
	var list []map[string]any
	for _, m := range methods {
		mann := m.Codec.(*methodAnnotations)
		reqIdFragment := ""
		if mann.HasRequestID() {
			reqIdFragment = "\n  span->SetAttribute(\"gl-cpp.request_id\", request." + mann.RequestIDFieldName() + "());"
		}
		entry := map[string]any{
			"method_name":         mann.MethodName(),
			"request_type":        mann.RequestType(),
			"response_type":       mann.ResponseType(),
			"return_type":         mann.ReturnType(),
			"grpc_service":        ann.GrpcService,
			"request_id_fragment": reqIdFragment,
		}
		if mann.IsStreamingWrite() {
			entry["is_streaming_write"] = true
		} else if mann.IsBidiStreaming() {
			entry["is_bidi_streaming"] = true
		} else if mann.IsLongrunning() {
			entry["is_longrunning"] = true
		} else if mann.IsStreamingRead() {
			entry["is_streaming_read"] = true
		} else {
			entry["is_plain_unary"] = true
		}
		list = append(list, entry)
	}
	return list
}

func buildTracingStubAsyncMethodList(ann *serviceAnnotations, asyncMethods []*api.Method) []map[string]any {
	var list []map[string]any
	for _, m := range asyncMethods {
		if isBidiStreaming(m) || isLongrunning(m) {
			continue
		}
		mann := m.Codec.(*methodAnnotations)
		reqIdFragment := ""
		if mann.HasRequestID() {
			reqIdFragment = "\n  span->SetAttribute(\"gl-cpp.request_id\", request." + mann.RequestIDFieldName() + "());"
		}
		entry := map[string]any{
			"method_name":         mann.MethodName(),
			"request_type":        mann.RequestType(),
			"response_type":       mann.ResponseType(),
			"return_type":         mann.ReturnType(),
			"grpc_service":        ann.GrpcService,
			"request_id_fragment": reqIdFragment,
		}
		if mann.IsStreamingRead() {
			entry["is_streaming_read"] = true
		} else if mann.IsStreamingWrite() {
			entry["is_streaming_write"] = true
		} else {
			entry["is_unary"] = true
		}
		list = append(list, entry)
	}
	return list
}

func generateTracingStubCc(_ *api.Service, ann *serviceAnnotations, methods, asyncMethods []*api.Method, _ *config.Library, _ *api.API) (string, string) {
	ccPath := ann.TracingStubCcPath()

	localIncludes := []string{ann.TracingStubHeaderPath()}
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
		"copyright_year":             ann.CopyrightYear,
		"proto_file_name":            ann.ProtoFileName,
		"product_internal_namespace": ann.InternalNamespace(),
		"tracing_stub_class_name":    ann.TracingStubClassName(),
		"stub_class_name":            ann.StubClassName(),
		"local_includes":             localIncludes,
		"methods":                    buildTracingStubMethodList(ann, methods),
		"async_methods":              buildTracingStubAsyncMethodList(ann, asyncMethods),
		"has_lro":                    hasLongrunningMethod(methods),
	}

	content, err := renderTemplate("templates/internal/tracing_stub.cc.mustache", data)
	if err != nil {
		panic(err)
	}

	return filepath.Clean(ccPath), content
}
