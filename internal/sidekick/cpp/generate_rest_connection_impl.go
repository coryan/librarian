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

func buildRestConnectionImplMethodList(svc *api.Service, methods []*api.Method, serviceVars map[string]string, lib *config.Library, model *api.API) []map[string]any {
	var list []map[string]any
	for _, m := range getRestMethods(methods) {
		mVars := buildMethodVars(svc, m, serviceVars, lib, model)
		entry := map[string]any{
			"method_name":                       mVars["method_name"],
			"request_type":                      mVars["request_type"],
			"response_type":                     mVars["response_type"],
			"return_type":                       mVars["return_type"],
			"range_output_type":                 mVars["range_output_type"],
			"range_output_field_name":           mVars["range_output_field_name"],
			"longrunning_operation_type":        mVars["longrunning_operation_type"],
			"longrunning_deduced_response_type": mVars["longrunning_deduced_response_type"],
			"longrunning_metadata_type":         mVars["longrunning_metadata_type"],
		}

		if isPaginated(m) {
			entry["is_paginated"] = true
		} else if isLongrunning(m) {
			entry["is_longrunning"] = true
			if isResponseTypeEmpty(m) {
				entry["is_response_type_empty"] = true
			}
			extractor := "&google::cloud::internal::ExtractLongRunningResultResponse<" + mVars["longrunning_deduced_response_type"] + ">"
			if isLongrunningMetadataTypeUsedAsResponse(m) {
				extractor = "&google::cloud::internal::ExtractLongRunningResultMetadata<" + mVars["longrunning_deduced_response_type"] + ">"
			}
			entry["lro_extractor"] = extractor
		} else {
			entry["is_plain_unary"] = true
		}
		list = append(list, entry)
	}
	return list
}

func buildRestConnectionImplAsyncMethodList(svc *api.Service, asyncMethods []*api.Method, serviceVars map[string]string, lib *config.Library, model *api.API) []map[string]any {
	var list []map[string]any
	for _, m := range getRestAsyncMethods(asyncMethods) {
		mVars := buildMethodVars(svc, m, serviceVars, lib, model)
		entry := map[string]any{
			"method_name":   mVars["method_name"],
			"request_type":  mVars["request_type"],
			"response_type": mVars["response_type"],
			"return_type":   mVars["return_type"],
		}
		list = append(list, entry)
	}
	return list
}

func generateRestConnectionImplHeader(svc *api.Service, serviceVars map[string]string, methods, asyncMethods []*api.Method, lib *config.Library, model *api.API) (string, string) {
	headerPath := serviceVars["connection_impl_rest_header_path"]
	guard := formatHeaderIncludeGuard(headerPath)

	restMethods := getRestMethods(methods)
	var localIncludes []string
	localIncludes = append(localIncludes,
		serviceVars["connection_header_path"],
		serviceVars["idempotency_policy_header_path"],
		serviceVars["options_header_path"],
		serviceVars["stub_rest_header_path"],
		serviceVars["retry_traits_header_path"],
		"google/cloud/background_threads.h",
		"google/cloud/backoff_policy.h",
		"google/cloud/options.h",
		"google/cloud/status_or.h",
	)
	if hasPaginatedMethod(restMethods) {
		localIncludes = append(localIncludes, "google/cloud/stream_range.h")
	}
	localIncludes = append(localIncludes, "google/cloud/version.h")
	slices.Sort(localIncludes)

	var protoIncludes []string
	if hasLongrunningMethod(restMethods) {
		protoIncludes = append(protoIncludes, "google/longrunning/operations.pb.h")
	}

	data := map[string]any{
		"header_include_guard":            guard,
		"copyright_year":                  serviceVars["copyright_year"],
		"proto_file_name":                 serviceVars["proto_file_name"],
		"product_namespace":               serviceVars["product_namespace"],
		"product_internal_namespace":      serviceVars["product_internal_namespace"],
		"connection_class_name":           serviceVars["connection_class_name"],
		"connection_impl_rest_class_name": serviceVars["connection_impl_rest_class_name"],
		"stub_rest_class_name":            serviceVars["stub_rest_class_name"],
		"service_name":                    serviceVars["service_name"],
		"retry_policy_name":               serviceVars["retry_policy_name"],
		"idempotency_class_name":          serviceVars["idempotency_class_name"],
		"local_includes":                  localIncludes,
		"proto_includes":                  protoIncludes,
		"methods":                         buildRestConnectionImplMethodList(svc, methods, serviceVars, lib, model),
		"async_methods":                   buildRestConnectionImplAsyncMethodList(svc, asyncMethods, serviceVars, lib, model),
		"has_lro":                         hasLongrunningMethod(restMethods),
	}

	content, err := renderTemplate("templates/internal/rest_connection_impl.h.mustache", data)
	if err != nil {
		panic(err)
	}

	return filepath.Clean(headerPath), content
}

func generateRestConnectionImplCc(svc *api.Service, serviceVars map[string]string, methods, asyncMethods []*api.Method, lib *config.Library, model *api.API) (string, string) {
	ccPath := serviceVars["connection_impl_rest_cc_path"]

	restMethods := getRestMethods(methods)
	var localIncludes []string
	localIncludes = append(localIncludes,
		serviceVars["connection_impl_rest_header_path"],
		serviceVars["stub_factory_rest_header_path"],
		"google/cloud/common_options.h",
		"google/cloud/credentials.h",
	)
	if hasLongrunningMethod(restMethods) {
		localIncludes = append(localIncludes, "google/cloud/internal/async_rest_long_running_operation.h")
	}
	if len(asyncMethods) > 0 {
		localIncludes = append(localIncludes, "google/cloud/internal/async_rest_retry_loop.h")
	}
	if hasLongrunningMethod(restMethods) {
		localIncludes = append(localIncludes, "google/cloud/internal/extract_long_running_result.h")
	}
	if hasPaginatedMethod(restMethods) {
		localIncludes = append(localIncludes, "google/cloud/internal/pagination_range.h")
	}
	localIncludes = append(localIncludes,
		"google/cloud/internal/rest_retry_loop.h",
		"google/cloud/rest_options.h",
	)
	slices.Sort(localIncludes)

	data := map[string]any{
		"copyright_year":                  serviceVars["copyright_year"],
		"proto_file_name":                 serviceVars["proto_file_name"],
		"product_namespace":               serviceVars["product_namespace"],
		"product_internal_namespace":      serviceVars["product_internal_namespace"],
		"connection_class_name":           serviceVars["connection_class_name"],
		"connection_impl_rest_class_name": serviceVars["connection_impl_rest_class_name"],
		"stub_rest_class_name":            serviceVars["stub_rest_class_name"],
		"service_name":                    serviceVars["service_name"],
		"retry_policy_name":               serviceVars["retry_policy_name"],
		"idempotency_class_name":          serviceVars["idempotency_class_name"],
		"local_includes":                  localIncludes,
		"methods":                         buildRestConnectionImplMethodList(svc, methods, serviceVars, lib, model),
		"async_methods":                   buildRestConnectionImplAsyncMethodList(svc, asyncMethods, serviceVars, lib, model),
	}

	content, err := renderTemplate("templates/internal/rest_connection_impl.cc.mustache", data)
	if err != nil {
		panic(err)
	}

	return filepath.Clean(ccPath), content
}
