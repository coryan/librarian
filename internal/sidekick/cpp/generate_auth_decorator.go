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

func buildDecoratorMethodList(svc *api.Service, methods []*api.Method, serviceVars map[string]string, lib *config.Library, model *api.API) []map[string]any {
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

func buildDecoratorAsyncMethodList(svc *api.Service, asyncMethods []*api.Method, serviceVars map[string]string, lib *config.Library, model *api.API) []map[string]any {
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
		} else if isStreamingWrite(m) {
			entry["is_streaming_write"] = true
		} else {
			entry["is_unary"] = true
			if isResponseTypeEmpty(m) {
				entry["is_plain_empty"] = true
			} else {
				entry["is_plain_non_empty"] = true
			}
		}
		list = append(list, entry)
	}
	return list
}

func generateAuthDecoratorHeader(svc *api.Service, serviceVars map[string]string, methods, asyncMethods []*api.Method, lib *config.Library, model *api.API) (string, string) {
	headerPath := serviceVars["auth_header_path"]
	guard := formatHeaderIncludeGuard(headerPath)

	localIncludes := []string{
		serviceVars["stub_header_path"],
		"google/cloud/internal/unified_grpc_credentials.h",
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
		"auth_class_name":            serviceVars["auth_class_name"],
		"stub_class_name":            serviceVars["stub_class_name"],
		"local_includes":             localIncludes,
		"proto_includes":             protoIncludes,
		"methods":                    buildDecoratorMethodList(svc, methods, serviceVars, lib, model),
		"async_methods":              buildDecoratorAsyncMethodList(svc, asyncMethods, serviceVars, lib, model),
		"has_lro":                    hasLongrunningMethod(methods),
	}

	content, err := renderTemplate("templates/internal/auth_decorator.h.mustache", data)
	if err != nil {
		panic(err)
	}

	return filepath.Clean(headerPath), content
}

func generateAuthDecoratorCc(svc *api.Service, serviceVars map[string]string, methods, asyncMethods []*api.Method, lib *config.Library, model *api.API) (string, string) {
	ccPath := serviceVars["auth_cc_path"]

	localIncludes := []string{serviceVars["auth_header_path"]}
	if hasBidiStreamingMethod(methods) {
		localIncludes = append(localIncludes, "google/cloud/internal/async_read_write_stream_auth.h")
	}
	if hasAsynchronousStreamingReadMethod(asyncMethods) {
		localIncludes = append(localIncludes, "google/cloud/internal/async_streaming_read_rpc_auth.h")
	}
	if hasAsynchronousStreamingWriteMethod(asyncMethods) {
		localIncludes = append(localIncludes, "google/cloud/internal/async_streaming_write_rpc_auth.h")
	}
	if hasStreamingWriteMethod(methods) {
		localIncludes = append(localIncludes, "google/cloud/internal/streaming_write_rpc_impl.h")
	}
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
		"auth_class_name":            serviceVars["auth_class_name"],
		"stub_class_name":            serviceVars["stub_class_name"],
		"local_includes":             localIncludes,
		"proto_includes":             pbIncludes,
		"methods":                    buildDecoratorMethodList(svc, methods, serviceVars, lib, model),
		"async_methods":              buildDecoratorAsyncMethodList(svc, asyncMethods, serviceVars, lib, model),
		"has_lro":                    hasLongrunningMethod(methods),
	}

	content, err := renderTemplate("templates/internal/auth_decorator.cc.mustache", data)
	if err != nil {
		panic(err)
	}

	return filepath.Clean(ccPath), content
}
