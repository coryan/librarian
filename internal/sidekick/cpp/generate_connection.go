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
	"strings"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/sidekick/api"
)

func buildConnectionMethodList(svc *api.Service, methods []*api.Method, serviceVars map[string]string, lib *config.Library, model *api.API) []map[string]any {
	var list []map[string]any
	for _, m := range methods {
		if isStreamingWrite(m) {
			continue
		}
		mVars := buildMethodVars(svc, m, serviceVars, lib, model)
		entry := map[string]any{
			"method_name":   mVars["method_name"],
			"request_type":  mVars["request_type"],
			"response_type": mVars["response_type"],
			"return_type":   mVars["return_type"],
		}
		if isBidiStreaming(m) {
			entry["is_bidi_streaming"] = true
		} else if isStreamingRead(m) {
			entry["is_streaming_read"] = true
		} else if isPaginated(m) {
			entry["is_paginated"] = true
			entry["range_output_type"] = mVars["range_output_type"]
		} else if isLongrunning(m) {
			entry["is_longrunning"] = true
			entry["longrunning_operation_type"] = mVars["longrunning_operation_type"]
			if isResponseTypeEmpty(m) {
				entry["is_response_type_empty"] = true
			} else {
				entry["longrunning_deduced_response_type"] = mVars["longrunning_deduced_response_type"]
			}
		} else {
			entry["is_plain_unary"] = true
		}
		list = append(list, entry)
	}
	return list
}

func buildConnectionAsyncMethodList(svc *api.Service, asyncMethods []*api.Method, serviceVars map[string]string, lib *config.Library, model *api.API) []map[string]any {
	var list []map[string]any
	for _, m := range asyncMethods {
		if isStreamingRead(m) || isStreamingWrite(m) {
			continue
		}
		mVars := buildMethodVars(svc, m, serviceVars, lib, model)
		list = append(list, map[string]any{
			"method_name":            mVars["method_name"],
			"request_type":           mVars["request_type"],
			"response_type":          mVars["response_type"],
			"return_type":            mVars["return_type"],
			"is_response_type_empty": isResponseTypeEmpty(m),
		})
	}
	return list
}

func generateConnectionHeader(svc *api.Service, serviceVars map[string]string, methods, asyncMethods []*api.Method, lib *config.Library, model *api.API) (string, string) {
	headerPath := serviceVars["connection_header_path"]
	guard := formatHeaderIncludeGuard(headerPath)

	var localIncludes []string
	localIncludes = append(localIncludes,
		serviceVars["idempotency_policy_header_path"],
		serviceVars["retry_traits_header_path"],
		"google/cloud/backoff_policy.h",
		"google/cloud/internal/retry_policy_impl.h",
		"google/cloud/options.h",
		"google/cloud/status_or.h",
		"google/cloud/version.h",
	)
	if hasLongrunningMethod(methods) {
		localIncludes = append(localIncludes, "google/cloud/no_await_tag.h", "google/cloud/polling_policy.h")
	}
	if hasLongrunningMethod(methods) || hasAsyncMethod(asyncMethods) {
		localIncludes = append(localIncludes, "google/cloud/future.h")
	}
	if hasStreamingReadMethod(methods) || hasPaginatedMethod(methods) {
		localIncludes = append(localIncludes, "google/cloud/stream_range.h")
	}
	if hasBidiStreamingMethod(methods) {
		localIncludes = append(localIncludes, "google/cloud/internal/async_read_write_stream_impl.h")
	}
	slices.Sort(localIncludes)

	var protoIncludes []string
	if addPaths, ok := serviceVars["additional_pb_header_paths"]; ok && addPaths != "" {
		for add := range strings.SplitSeq(addPaths, ",") {
			if add != "" {
				protoIncludes = append(protoIncludes, add)
			}
		}
	}
	protoIncludes = append(protoIncludes, serviceVars["proto_header_path"])
	if hasLongrunningMethod(methods) {
		protoIncludes = append(protoIncludes, "google/longrunning/operations.grpc.pb.h")
	}
	slices.Sort(protoIncludes)

	hasGrpc := lib == nil || lib.Cpp == nil || lib.Cpp.HasGrpcTransport()
	locationStyle := ""
	if lib != nil && lib.Cpp != nil {
		locationStyle = lib.Cpp.EndpointLocationStyle
	}
	isLocationDependent := locationStyle == "LOCATION_DEPENDENT" ||
		locationStyle == "LOCATION_DEPENDENT_COMPAT" ||
		locationStyle == "LOCATION_OPTIONALLY_DEPENDENT"

	sysIncludes := []string{"memory"}
	if hasGrpc && isLocationDependent {
		sysIncludes = append(sysIncludes, "string")
	}
	slices.Sort(sysIncludes)

	locationDoc := ""
	if isLocationDependent {
		locationDoc = "\n * @param location Sets the prefix for the default `EndpointOption` value."
	}

	data := map[string]any{
		"header_include_guard":                  guard,
		"copyright_year":                        serviceVars["copyright_year"],
		"proto_file_name":                       serviceVars["proto_file_name"],
		"product_namespace":                     serviceVars["product_namespace"],
		"product_internal_namespace":            serviceVars["product_internal_namespace"],
		"connection_class_name":                 serviceVars["connection_class_name"],
		"client_class_name":                     serviceVars["client_class_name"],
		"retry_policy_name":                     serviceVars["retry_policy_name"],
		"limited_error_count_retry_policy_name": serviceVars["limited_error_count_retry_policy_name"],
		"limited_time_retry_policy_name":        serviceVars["limited_time_retry_policy_name"],
		"retry_traits_name":                     serviceVars["retry_traits_name"],
		"transient_errors_comment":              serviceVars["transient_errors_comment"],
		"mock_connection_class_name":            serviceVars["mock_connection_class_name"],
		"product_mocks_namespace":               serviceVars["product_mocks_namespace"],
		"service_name":                          serviceVars["service_name"],
		"has_grpc":                              hasGrpc,
		"is_location_dependent":                 isLocationDependent,
		"is_location_dependent_compat":          locationStyle == "LOCATION_DEPENDENT_COMPAT",
		"is_location_optionally_dependent":      locationStyle == "LOCATION_OPTIONALLY_DEPENDENT",
		"location_doc":                          locationDoc,
		"local_includes":                        localIncludes,
		"proto_includes":                        protoIncludes,
		"system_includes":                       sysIncludes,
		"methods":                               buildConnectionMethodList(svc, methods, serviceVars, lib, model),
		"async_methods":                         buildConnectionAsyncMethodList(svc, asyncMethods, serviceVars, lib, model),
	}

	content, err := renderTemplate("templates/connection.h.mustache", data)
	if err != nil {
		panic(err)
	}

	return filepath.Clean(headerPath), content
}

func generateConnectionCc(svc *api.Service, serviceVars map[string]string, methods, asyncMethods []*api.Method, lib *config.Library, model *api.API) (string, string) {
	ccPath := serviceVars["connection_cc_path"]

	hasGrpc := lib == nil || lib.Cpp == nil || lib.Cpp.HasGrpcTransport()

	var localIncludes []string
	localIncludes = append(localIncludes,
		serviceVars["connection_header_path"],
		serviceVars["options_header_path"],
	)
	if hasGrpc {
		localIncludes = append(localIncludes, serviceVars["connection_impl_header_path"])
	}
	localIncludes = append(localIncludes, serviceVars["option_defaults_header_path"])
	if hasGrpc {
		localIncludes = append(localIncludes, serviceVars["stub_factory_header_path"])
	}
	localIncludes = append(localIncludes,
		serviceVars["tracing_connection_header_path"],
		"google/cloud/background_threads.h",
		"google/cloud/common_options.h",
		"google/cloud/credentials.h",
		"google/cloud/grpc_options.h",
		"google/cloud/internal/unified_grpc_credentials.h",
	)
	if hasPaginatedMethod(methods) {
		localIncludes = append(localIncludes, "google/cloud/internal/pagination_range.h")
	}
	if len(localIncludes) > 1 {
		slices.Sort(localIncludes[1:])
	}

	locationStyle := ""
	if lib != nil && lib.Cpp != nil {
		locationStyle = lib.Cpp.EndpointLocationStyle
	}
	isLocationDependent := locationStyle == "LOCATION_DEPENDENT" ||
		locationStyle == "LOCATION_DEPENDENT_COMPAT" ||
		locationStyle == "LOCATION_OPTIONALLY_DEPENDENT"
	hasNonLocationOverload := locationStyle == "LOCATION_DEPENDENT_COMPAT" ||
		locationStyle == "LOCATION_OPTIONALLY_DEPENDENT"

	data := map[string]any{
		"copyright_year":                serviceVars["copyright_year"],
		"proto_file_name":               serviceVars["proto_file_name"],
		"product_namespace":             serviceVars["product_namespace"],
		"product_internal_namespace":    serviceVars["product_internal_namespace"],
		"connection_class_name":         serviceVars["connection_class_name"],
		"service_name":                  serviceVars["service_name"],
		"stub_class_name":               serviceVars["stub_class_name"],
		"tracing_connection_class_name": serviceVars["tracing_connection_class_name"],
		"has_grpc":                      hasGrpc,
		"is_location_dependent":         isLocationDependent,
		"has_non_location_overload":     hasNonLocationOverload,
		"local_includes":                localIncludes,
		"system_includes":               []string{"memory", "utility"},
		"methods":                       buildConnectionMethodList(svc, methods, serviceVars, lib, model),
		"async_methods":                 buildConnectionAsyncMethodList(svc, asyncMethods, serviceVars, lib, model),
	}

	content, err := renderTemplate("templates/connection.cc.mustache", data)
	if err != nil {
		panic(err)
	}

	return filepath.Clean(ccPath), content
}
