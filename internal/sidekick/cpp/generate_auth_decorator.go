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

func buildDecoratorMethodList(methods []*api.Method) []map[string]any {
	var list []map[string]any
	for _, m := range methods {
		mann := m.Codec.(*methodAnnotations)
		entry := map[string]any{
			"method_name":   mann.MethodName(),
			"request_type":  mann.RequestType(),
			"response_type": mann.ResponseType(),
			"return_type":   mann.ReturnType(),
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

func buildDecoratorAsyncMethodList(asyncMethods []*api.Method) []map[string]any {
	var list []map[string]any
	for _, m := range asyncMethods {
		mann := m.Codec.(*methodAnnotations)
		if mann.IsBidiStreaming() || mann.IsLongrunning() {
			continue
		}
		entry := map[string]any{
			"method_name":   mann.MethodName(),
			"request_type":  mann.RequestType(),
			"response_type": mann.ResponseType(),
			"return_type":   mann.ReturnType(),
		}
		if mann.IsStreamingRead() {
			entry["is_streaming_read"] = true
		} else if mann.IsStreamingWrite() {
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

func generateAuthDecoratorHeader(_ *api.Service, ann *serviceAnnotations, methods, asyncMethods []*api.Method, _ *config.Library, _ *api.API) (string, string) {
	headerPath := ann.AuthHeaderPath()
	guard := ann.AuthHeaderIncludeGuard()

	localIncludes := []string{
		ann.StubHeaderPath(),
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
		"copyright_year":             ann.CopyrightYear,
		"proto_file_name":            ann.ProtoFileName,
		"product_internal_namespace": ann.InternalNamespace(),
		"auth_class_name":            ann.AuthClassName(),
		"stub_class_name":            ann.StubClassName(),
		"local_includes":             localIncludes,
		"proto_includes":             protoIncludes,
		"methods":                    buildDecoratorMethodList(methods),
		"async_methods":              buildDecoratorAsyncMethodList(asyncMethods),
		"has_lro":                    hasLongrunningMethod(methods),
	}

	content, err := renderTemplate("templates/internal/auth_decorator.h.mustache", data)
	if err != nil {
		panic(err)
	}

	return filepath.Clean(headerPath), content
}

func generateAuthDecoratorCc(_ *api.Service, ann *serviceAnnotations, methods, asyncMethods []*api.Method, _ *config.Library, _ *api.API) (string, string) {
	ccPath := ann.AuthCcPath()

	localIncludes := []string{ann.AuthHeaderPath()}
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
	if h := ann.ProtoGrpcHeaderPath(); h != "" {
		pbIncludes = append(pbIncludes, h)
	}

	data := map[string]any{
		"copyright_year":             ann.CopyrightYear,
		"proto_file_name":            ann.ProtoFileName,
		"product_internal_namespace": ann.InternalNamespace(),
		"auth_class_name":            ann.AuthClassName(),
		"stub_class_name":            ann.StubClassName(),
		"local_includes":             localIncludes,
		"proto_includes":             pbIncludes,
		"methods":                    buildDecoratorMethodList(methods),
		"async_methods":              buildDecoratorAsyncMethodList(asyncMethods),
		"has_lro":                    hasLongrunningMethod(methods),
	}

	content, err := renderTemplate("templates/internal/auth_decorator.cc.mustache", data)
	if err != nil {
		panic(err)
	}

	return filepath.Clean(ccPath), content
}
