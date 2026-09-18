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

func generateMockConnectionHeader(svc *api.Service, serviceVars map[string]string, methods []*api.Method, asyncMethods []*api.Method, lib *config.Library, model *api.API) (string, string) {
	headerPath := serviceVars["mock_connection_header_path"]
	guard := formatHeaderIncludeGuard(headerPath)

	var methodList []map[string]any
	for _, m := range methods {
		mVars := buildMethodVars(svc, m, serviceVars, lib, model)
		entry := map[string]any{
			"method_name":                       mVars["method_name"],
			"request_type":                      mVars["request_type"],
			"response_type":                     mVars["response_type"],
			"return_type":                       mVars["return_type"],
			"range_output_type":                 mVars["range_output_type"],
			"longrunning_deduced_response_type": mVars["longrunning_deduced_response_type"],
			"longrunning_operation_type":        mVars["longrunning_operation_type"],
		}

		if isBidiStreaming(m) {
			entry["is_bidi_streaming"] = true
		} else if isNonStreaming(m) && !isLongrunning(m) && !isPaginated(m) {
			entry["is_plain"] = true
		} else if isNonStreaming(m) && isLongrunning(m) && !isPaginated(m) {
			entry["is_longrunning"] = true
		} else if isNonStreaming(m) && !isLongrunning(m) && isPaginated(m) {
			entry["is_paginated"] = true
		} else if isStreamingRead(m) {
			entry["is_streaming_read"] = true
		}

		methodList = append(methodList, entry)
	}

	var asyncMethodList []map[string]any
	for _, m := range asyncMethods {
		if isNonStreaming(m) && !isLongrunning(m) && !isPaginated(m) {
			mVars := buildMethodVars(svc, m, serviceVars, lib, model)
			asyncMethodList = append(asyncMethodList, map[string]any{
				"method_name":  mVars["method_name"],
				"request_type": mVars["request_type"],
				"return_type":  mVars["return_type"],
			})
		}
	}

	data := map[string]any{
		"header_include_guard":       guard,
		"copyright_year":             serviceVars["copyright_year"],
		"proto_file_name":            serviceVars["proto_file_name"],
		"product_mocks_namespace":    serviceVars["product_mocks_namespace"],
		"product_namespace":          serviceVars["product_namespace"],
		"connection_class_name":      serviceVars["connection_class_name"],
		"client_class_name":          serviceVars["client_class_name"],
		"mock_connection_class_name": serviceVars["mock_connection_class_name"],
		"local_includes":             []string{serviceVars["connection_header_path"]},
		"methods":                    methodList,
		"async_methods":              asyncMethodList,
	}

	content, err := renderTemplate("templates/mocks/mock_connection.h.mustache", data)
	if err != nil {
		panic(err)
	}

	return filepath.Clean(headerPath), content
}
