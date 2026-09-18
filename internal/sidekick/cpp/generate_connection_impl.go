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

func isLongrunningMetadataTypeUsedAsResponse(m *api.Method) bool {
	if m.OperationInfo == nil {
		return false
	}
	return m.OperationInfo.ResponseTypeID == "google.protobuf.Empty" || m.OperationInfo.ResponseTypeID == ".google.protobuf.Empty"
}

func hasRequestIDService(methods []*api.Method) bool {
	return slices.ContainsFunc(methods, hasRequestID)
}

func buildStreamingUpdatersList(svc *api.Service, methods []*api.Method, serviceVars map[string]string, lib *config.Library, model *api.API) []map[string]string {
	var list []map[string]string
	for _, m := range methods {
		if isStreamingRead(m) {
			mVars := buildMethodVars(svc, m, serviceVars, lib, model)
			list = append(list, map[string]string{
				"service_name":  mVars["service_name"],
				"method_name":   mVars["method_name"],
				"request_type":  mVars["request_type"],
				"response_type": mVars["response_type"],
			})
		}
	}
	return list
}

func buildConnectionImplMethodList(svc *api.Service, methods []*api.Method, serviceVars map[string]string, lib *config.Library, model *api.API) []map[string]any {
	var list []map[string]any
	for _, m := range methods {
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
			"request_id_field_name":             mVars["request_id_field_name"],
			"has_request_id":                    hasRequestID(m),
		}

		if isBidiStreaming(m) {
			entry["is_bidi_streaming"] = true
		} else if isStreamingRead(m) {
			entry["is_streaming_read"] = true
		} else if isStreamingWrite(m) {
			continue
		} else if isPaginated(m) {
			entry["is_paginated"] = true
		} else if isLongrunning(m) {
			entry["is_longrunning"] = true
			returnFragment := "future<StatusOr<" + mVars["longrunning_deduced_response_type"] + ">>"
			if isResponseTypeEmpty(m) {
				returnFragment = "future<Status>"
				entry["is_response_type_empty"] = true
			}

			requestIdFragment := ""
			if hasRequestID(m) {
				requestIdFragment = "\n  if (request_copy." + mVars["request_id_field_name"] + "().empty()) {\n    request_copy.set_" + mVars["request_id_field_name"] + "(invocation_id_generator_->MakeInvocationId());\n  }"
			}

			extractValueFragment := "    &google::cloud::internal::ExtractLongRunningResultResponse<" + mVars["longrunning_deduced_response_type"] + ">,"
			if isLongrunningMetadataTypeUsedAsResponse(m) {
				extractValueFragment = "    &google::cloud::internal::ExtractLongRunningResultMetadata<" + mVars["longrunning_deduced_response_type"] + ">,"
			}

			elideProtobufEmptyFragment := ""
			if isResponseTypeEmpty(m) {
				elideProtobufEmptyFragment = "\n    .then([](future<StatusOr<google::protobuf::Empty>> f) {\n      return f.get().status();\n    }))"
			}

			entry["lro_return_fragment"] = returnFragment
			entry["lro_request_id_fragment"] = requestIdFragment
			entry["lro_extract_value_fragment"] = extractValueFragment
			entry["lro_elide_fragment"] = elideProtobufEmptyFragment
		} else {
			entry["is_plain_unary"] = true
		}
		list = append(list, entry)
	}
	return list
}

func buildConnectionImplAsyncMethodList(svc *api.Service, asyncMethods []*api.Method, serviceVars map[string]string, lib *config.Library, model *api.API) []map[string]any {
	var list []map[string]any
	for _, m := range asyncMethods {
		if isStreamingRead(m) || isStreamingWrite(m) {
			continue
		}
		mVars := buildMethodVars(svc, m, serviceVars, lib, model)
		requestIdFragment := ""
		if hasRequestID(m) {
			requestIdFragment = "\n  if (request_copy." + mVars["request_id_field_name"] + "().empty()) {\n    request_copy.set_" + mVars["request_id_field_name"] + "(invocation_id_generator_->MakeInvocationId());\n  }"
		}
		entry := map[string]any{
			"method_name":               mVars["method_name"],
			"request_type":              mVars["request_type"],
			"response_type":             mVars["response_type"],
			"return_type":               mVars["return_type"],
			"async_request_id_fragment": requestIdFragment,
		}
		list = append(list, entry)
	}
	return list
}

func generateConnectionImplHeader(svc *api.Service, serviceVars map[string]string, methods, asyncMethods []*api.Method, lib *config.Library, model *api.API) (string, string) {
	headerPath := serviceVars["connection_impl_header_path"]
	guard := formatHeaderIncludeGuard(headerPath)

	var localIncludes []string
	localIncludes = append(localIncludes,
		serviceVars["idempotency_policy_header_path"],
		serviceVars["options_header_path"],
		serviceVars["stub_header_path"],
		serviceVars["connection_header_path"],
		serviceVars["retry_traits_header_path"],
		"google/cloud/background_threads.h",
		"google/cloud/backoff_policy.h",
		"google/cloud/options.h",
		"google/cloud/status_or.h",
		"google/cloud/version.h",
	)
	if hasBidiStreamingMethod(methods) {
		localIncludes = append(localIncludes, "google/cloud/async_streaming_read_write_rpc.h")
	}
	if hasLongrunningMethod(methods) {
		localIncludes = append(localIncludes, "google/cloud/future.h", "google/cloud/polling_policy.h")
	}
	if hasRequestIDService(methods) {
		localIncludes = append(localIncludes, "google/cloud/internal/invocation_id_generator.h")
	}
	if hasStreamingReadMethod(methods) || hasPaginatedMethod(methods) {
		localIncludes = append(localIncludes, "google/cloud/stream_range.h")
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
		"product_namespace":          serviceVars["product_namespace"],
		"product_internal_namespace": serviceVars["product_internal_namespace"],
		"connection_class_name":      serviceVars["connection_class_name"],
		"stub_class_name":            serviceVars["stub_class_name"],
		"local_includes":             localIncludes,
		"proto_includes":             protoIncludes,
		"streaming_updaters":         buildStreamingUpdatersList(svc, methods, serviceVars, lib, model),
		"methods":                    buildConnectionImplMethodList(svc, methods, serviceVars, lib, model),
		"async_methods":              buildConnectionImplAsyncMethodList(svc, asyncMethods, serviceVars, lib, model),
		"has_request_id":             hasRequestIDService(methods),
	}

	content, err := renderTemplate("templates/internal/connection_impl.h.mustache", data)
	if err != nil {
		panic(err)
	}

	return filepath.Clean(headerPath), content
}

func generateConnectionImplCc(svc *api.Service, serviceVars map[string]string, methods, asyncMethods []*api.Method, lib *config.Library, model *api.API) (string, string) {
	ccPath := serviceVars["connection_impl_cc_path"]

	var localIncludes []string
	localIncludes = append(localIncludes,
		serviceVars["connection_impl_header_path"],
		serviceVars["option_defaults_header_path"],
		"google/cloud/background_threads.h",
		"google/cloud/common_options.h",
		"google/cloud/grpc_options.h",
		"google/cloud/internal/retry_loop.h",
	)
	if hasPaginatedMethod(methods) {
		localIncludes = append(localIncludes, "google/cloud/internal/pagination_range.h")
	}
	if hasLongrunningMethod(methods) {
		localIncludes = append(localIncludes, "google/cloud/internal/async_long_running_operation.h")
	}
	if len(asyncMethods) > 0 {
		localIncludes = append(localIncludes, "google/cloud/internal/async_retry_loop.h")
	}
	if hasStreamingReadMethod(methods) {
		localIncludes = append(localIncludes,
			"google/cloud/internal/resumable_streaming_read_rpc.h",
			"google/cloud/internal/streaming_read_rpc_logging.h",
		)
	}
	slices.Sort(localIncludes)

	data := map[string]any{
		"copyright_year":             serviceVars["copyright_year"],
		"proto_file_name":            serviceVars["proto_file_name"],
		"product_namespace":          serviceVars["product_namespace"],
		"product_internal_namespace": serviceVars["product_internal_namespace"],
		"connection_class_name":      serviceVars["connection_class_name"],
		"stub_class_name":            serviceVars["stub_class_name"],
		"service_name":               serviceVars["service_name"],
		"retry_policy_name":          serviceVars["retry_policy_name"],
		"idempotency_class_name":     serviceVars["idempotency_class_name"],
		"local_includes":             localIncludes,
		"has_lro":                    hasLongrunningMethod(methods),
		"streaming_updaters":         buildStreamingUpdatersList(svc, methods, serviceVars, lib, model),
		"methods":                    buildConnectionImplMethodList(svc, methods, serviceVars, lib, model),
		"async_methods":              buildConnectionImplAsyncMethodList(svc, asyncMethods, serviceVars, lib, model),
	}

	content, err := renderTemplate("templates/internal/connection_impl.cc.mustache", data)
	if err != nil {
		panic(err)
	}

	return filepath.Clean(ccPath), content
}
