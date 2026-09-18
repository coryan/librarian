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

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/sidekick/api"
)

func generateMockConnectionHeader(_ *api.Service, ann *serviceAnnotations, methods, asyncMethods []*api.Method, _ *config.Library, _ *api.API) (string, string) {
	headerPath := ann.MockConnectionHeaderPath()
	guard := ann.MockConnectionHeaderIncludeGuard()

	var methodList []map[string]any
	for _, m := range methods {
		mann := m.Codec.(*methodAnnotations)
		entry := map[string]any{
			"method_name":                       mann.MethodName(),
			"request_type":                      mann.RequestType(),
			"response_type":                     mann.ResponseType(),
			"return_type":                       mann.ReturnType(),
			"range_output_type":                 mann.RangeOutputType(),
			"longrunning_deduced_response_type": mann.LongrunningDeducedResponseType(),
			"longrunning_operation_type":        mann.LongrunningOperationType(),
		}

		if mann.IsBidiStreaming() {
			entry["is_bidi_streaming"] = true
		} else if mann.IsNonStreaming() && !mann.IsLongrunning() && !mann.IsPaginated() {
			entry["is_plain"] = true
		} else if mann.IsNonStreaming() && mann.IsLongrunning() && !mann.IsPaginated() {
			entry["is_longrunning"] = true
		} else if mann.IsNonStreaming() && !mann.IsLongrunning() && mann.IsPaginated() {
			entry["is_paginated"] = true
		} else if mann.IsStreamingRead() {
			entry["is_streaming_read"] = true
		}

		methodList = append(methodList, entry)
	}

	var asyncMethodList []map[string]any
	for _, m := range asyncMethods {
		mann := m.Codec.(*methodAnnotations)
		if mann.IsNonStreaming() && !mann.IsLongrunning() && !mann.IsPaginated() {
			asyncMethodList = append(asyncMethodList, map[string]any{
				"method_name":  mann.MethodName(),
				"request_type": mann.RequestType(),
				"return_type":  mann.ReturnType(),
			})
		}
	}

	data := map[string]any{
		"header_include_guard":       guard,
		"copyright_year":             ann.CopyrightYear,
		"proto_file_name":            ann.ProtoFileName,
		"product_mocks_namespace":    ann.MocksNamespace(),
		"product_namespace":          ann.Namespace(),
		"connection_class_name":      ann.ConnectionClassName(),
		"client_class_name":          ann.ClientClassName(),
		"mock_connection_class_name": ann.MockConnectionClassName(),
		"local_includes":             []string{ann.ConnectionHeaderPath()},
		"methods":                    methodList,
		"async_methods":              asyncMethodList,
	}

	content, err := renderTemplate("templates/mocks/mock_connection.h.mustache", data)
	if err != nil {
		panic(err)
	}

	return filepath.Clean(headerPath), content
}
