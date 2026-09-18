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

func hasStreamingMethod(methods []*api.Method) bool {
	return hasStreamingReadMethod(methods) || hasStreamingWriteMethod(methods) || hasBidiStreamingMethod(methods)
}

func generateLoggingDecoratorHeader(_ *api.Service, ann *serviceAnnotations, methods, asyncMethods []*api.Method, _ *config.Library, _ *api.API) (string, string) {
	headerPath := ann.LoggingHeaderPath()
	guard := ann.LoggingHeaderIncludeGuard()

	localIncludes := []string{
		ann.StubHeaderPath(),
		"google/cloud/tracing_options.h",
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
		"logging_class_name":         ann.LoggingClassName(),
		"stub_class_name":            ann.StubClassName(),
		"local_includes":             localIncludes,
		"proto_includes":             protoIncludes,
		"methods":                    buildDecoratorMethodList(methods),
		"async_methods":              buildDecoratorAsyncMethodList(asyncMethods),
		"has_lro":                    hasLongrunningMethod(methods),
		"has_streaming":              hasStreamingMethod(methods),
	}

	content, err := renderTemplate("templates/internal/logging_decorator.h.mustache", data)
	if err != nil {
		panic(err)
	}

	return filepath.Clean(headerPath), content
}

func generateLoggingDecoratorCc(_ *api.Service, ann *serviceAnnotations, methods, asyncMethods []*api.Method, _ *config.Library, _ *api.API) (string, string) {
	ccPath := ann.LoggingCcPath()

	localIncludes := []string{
		ann.LoggingHeaderPath(),
		"google/cloud/internal/log_wrapper.h",
	}
	if hasStreamingReadMethod(methods) {
		localIncludes = append(localIncludes, "google/cloud/internal/streaming_read_rpc_logging.h")
	}
	if hasStreamingWriteMethod(methods) {
		localIncludes = append(localIncludes, "google/cloud/internal/streaming_write_rpc_logging.h")
	}
	if hasBidiStreamingMethod(methods) {
		localIncludes = append(localIncludes, "google/cloud/internal/async_read_write_stream_logging.h")
	}
	if hasAsynchronousStreamingReadMethod(asyncMethods) {
		localIncludes = append(localIncludes, "google/cloud/internal/async_streaming_read_rpc_logging.h")
	}
	if hasAsynchronousStreamingWriteMethod(asyncMethods) {
		localIncludes = append(localIncludes, "google/cloud/internal/async_streaming_write_rpc_logging.h")
	}
	localIncludes = append(localIncludes, "google/cloud/status_or.h")
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
		"logging_class_name":         ann.LoggingClassName(),
		"stub_class_name":            ann.StubClassName(),
		"local_includes":             localIncludes,
		"proto_includes":             pbIncludes,
		"methods":                    buildDecoratorMethodList(methods),
		"async_methods":              buildDecoratorAsyncMethodList(asyncMethods),
		"has_lro":                    hasLongrunningMethod(methods),
		"has_streaming":              hasStreamingMethod(methods),
	}

	content, err := renderTemplate("templates/internal/logging_decorator.cc.mustache", data)
	if err != nil {
		panic(err)
	}

	return filepath.Clean(ccPath), content
}
