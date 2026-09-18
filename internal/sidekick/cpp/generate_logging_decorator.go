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

func generateLoggingDecoratorHeader(svc *api.Service, serviceVars map[string]string, methods, asyncMethods []*api.Method, lib *config.Library, model *api.API) (string, string) {
	headerPath := serviceVars["logging_header_path"]
	guard := formatHeaderIncludeGuard(headerPath)

	localIncludes := []string{
		serviceVars["stub_header_path"],
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
		"copyright_year":             serviceVars["copyright_year"],
		"proto_file_name":            serviceVars["proto_file_name"],
		"product_internal_namespace": serviceVars["product_internal_namespace"],
		"logging_class_name":         serviceVars["logging_class_name"],
		"stub_class_name":            serviceVars["stub_class_name"],
		"local_includes":             localIncludes,
		"proto_includes":             protoIncludes,
		"methods":                    buildDecoratorMethodList(svc, methods, serviceVars, lib, model),
		"async_methods":              buildDecoratorAsyncMethodList(svc, asyncMethods, serviceVars, lib, model),
		"has_lro":                    hasLongrunningMethod(methods),
		"has_streaming":              hasStreamingMethod(methods),
	}

	content, err := renderTemplate("templates/internal/logging_decorator.h.mustache", data)
	if err != nil {
		panic(err)
	}

	return filepath.Clean(headerPath), content
}

func generateLoggingDecoratorCc(svc *api.Service, serviceVars map[string]string, methods, asyncMethods []*api.Method, lib *config.Library, model *api.API) (string, string) {
	ccPath := serviceVars["logging_cc_path"]

	localIncludes := []string{
		serviceVars["logging_header_path"],
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
	if serviceVars["proto_grpc_header_path"] != "" {
		pbIncludes = append(pbIncludes, serviceVars["proto_grpc_header_path"])
	}

	data := map[string]any{
		"copyright_year":             serviceVars["copyright_year"],
		"proto_file_name":            serviceVars["proto_file_name"],
		"product_internal_namespace": serviceVars["product_internal_namespace"],
		"logging_class_name":         serviceVars["logging_class_name"],
		"stub_class_name":            serviceVars["stub_class_name"],
		"local_includes":             localIncludes,
		"proto_includes":             pbIncludes,
		"methods":                    buildDecoratorMethodList(svc, methods, serviceVars, lib, model),
		"async_methods":              buildDecoratorAsyncMethodList(svc, asyncMethods, serviceVars, lib, model),
		"has_lro":                    hasLongrunningMethod(methods),
		"has_streaming":              hasStreamingMethod(methods),
	}

	content, err := renderTemplate("templates/internal/logging_decorator.cc.mustache", data)
	if err != nil {
		panic(err)
	}

	return filepath.Clean(ccPath), content
}
