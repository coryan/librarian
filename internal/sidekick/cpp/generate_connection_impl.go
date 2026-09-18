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

func buildStreamingUpdatersList(ann *serviceAnnotations, methods []*api.Method) []map[string]string {
	var list []map[string]string
	for _, m := range methods {
		mann := m.Codec.(*methodAnnotations)
		if mann.IsStreamingRead() {
			list = append(list, map[string]string{
				"service_name":  ann.ServiceName,
				"method_name":   mann.MethodName(),
				"request_type":  mann.RequestType(),
				"response_type": mann.ResponseType(),
			})
		}
	}
	return list
}

func buildConnectionImplMethodList(methods []*api.Method) []map[string]any {
	var list []map[string]any
	for _, m := range methods {
		mann := m.Codec.(*methodAnnotations)
		entry := map[string]any{
			"method_name":                       mann.MethodName(),
			"request_type":                      mann.RequestType(),
			"response_type":                     mann.ResponseType(),
			"return_type":                       mann.ReturnType(),
			"range_output_type":                 mann.RangeOutputType(),
			"range_output_field_name":           mann.RangeOutputFieldName(),
			"longrunning_operation_type":        mann.LongrunningOperationType(),
			"longrunning_deduced_response_type": mann.LongrunningDeducedResponseType(),
			"longrunning_metadata_type":         mann.LongrunningMetadataType(),
			"request_id_field_name":             mann.RequestIDFieldName(),
			"has_request_id":                    mann.HasRequestID(),
		}

		if mann.IsBidiStreaming() {
			entry["is_bidi_streaming"] = true
		} else if mann.IsStreamingRead() {
			entry["is_streaming_read"] = true
		} else if mann.IsStreamingWrite() {
			continue
		} else if mann.IsPaginated() {
			entry["is_paginated"] = true
		} else if mann.IsLongrunning() {
			entry["is_longrunning"] = true
			returnFragment := "future<StatusOr<" + mann.LongrunningDeducedResponseType() + ">>"
			if isResponseTypeEmpty(m) {
				returnFragment = "future<Status>"
				entry["is_response_type_empty"] = true
			}

			requestIdFragment := ""
			if mann.HasRequestID() {
				requestIdFragment = "\n  if (request_copy." + mann.RequestIDFieldName() + "().empty()) {\n    request_copy.set_" + mann.RequestIDFieldName() + "(invocation_id_generator_->MakeInvocationId());\n  }"
			}

			extractValueFragment := "    &google::cloud::internal::ExtractLongRunningResultResponse<" + mann.LongrunningDeducedResponseType() + ">,"
			if isLongrunningMetadataTypeUsedAsResponse(m) {
				extractValueFragment = "    &google::cloud::internal::ExtractLongRunningResultMetadata<" + mann.LongrunningDeducedResponseType() + ">,"
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

func buildConnectionImplAsyncMethodList(asyncMethods []*api.Method) []map[string]any {
	var list []map[string]any
	for _, m := range asyncMethods {
		mann := m.Codec.(*methodAnnotations)
		if mann.IsStreamingRead() || mann.IsStreamingWrite() {
			continue
		}
		requestIdFragment := ""
		if mann.HasRequestID() {
			requestIdFragment = "\n  if (request_copy." + mann.RequestIDFieldName() + "().empty()) {\n    request_copy.set_" + mann.RequestIDFieldName() + "(invocation_id_generator_->MakeInvocationId());\n  }"
		}
		entry := map[string]any{
			"method_name":               mann.MethodName(),
			"request_type":              mann.RequestType(),
			"response_type":             mann.ResponseType(),
			"return_type":               mann.ReturnType(),
			"async_request_id_fragment": requestIdFragment,
		}
		list = append(list, entry)
	}
	return list
}

func generateConnectionImplHeader(_ *api.Service, ann *serviceAnnotations, methods, asyncMethods []*api.Method, _ *config.Library, _ *api.API) (string, string) {
	headerPath := ann.ConnectionImplHeaderPath()
	guard := ann.ConnectionImplHeaderIncludeGuard()

	var localIncludes []string
	localIncludes = append(localIncludes,
		ann.IdempotencyHeaderPath(),
		ann.OptionsHeaderPath(),
		ann.StubHeaderPath(),
		ann.ConnectionHeaderPath(),
		ann.RetryTraitsHeaderPath(),
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
		"copyright_year":             ann.CopyrightYear,
		"proto_file_name":            ann.ProtoFileName,
		"product_namespace":          ann.Namespace(),
		"product_internal_namespace": ann.InternalNamespace(),
		"connection_class_name":      ann.ConnectionClassName(),
		"stub_class_name":            ann.StubClassName(),
		"local_includes":             localIncludes,
		"proto_includes":             protoIncludes,
		"streaming_updaters":         buildStreamingUpdatersList(ann, methods),
		"methods":                    buildConnectionImplMethodList(methods),
		"async_methods":              buildConnectionImplAsyncMethodList(asyncMethods),
		"has_request_id":             hasRequestIDService(methods),
	}

	content, err := renderTemplate("templates/internal/connection_impl.h.mustache", data)
	if err != nil {
		panic(err)
	}

	return filepath.Clean(headerPath), content
}

func generateConnectionImplCc(_ *api.Service, ann *serviceAnnotations, methods, asyncMethods []*api.Method, _ *config.Library, _ *api.API) (string, string) {
	ccPath := ann.ConnectionImplCcPath()

	var localIncludes []string
	localIncludes = append(localIncludes,
		ann.ConnectionImplHeaderPath(),
		ann.OptionDefaultsHeaderPath(),
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
		"copyright_year":             ann.CopyrightYear,
		"proto_file_name":            ann.ProtoFileName,
		"product_namespace":          ann.Namespace(),
		"product_internal_namespace": ann.InternalNamespace(),
		"connection_class_name":      ann.ConnectionClassName(),
		"stub_class_name":            ann.StubClassName(),
		"service_name":               ann.ServiceName,
		"retry_policy_name":          ann.RetryPolicyName(),
		"idempotency_class_name":     ann.IdempotencyClassName(),
		"local_includes":             localIncludes,
		"has_lro":                    hasLongrunningMethod(methods),
		"streaming_updaters":         buildStreamingUpdatersList(ann, methods),
		"methods":                    buildConnectionImplMethodList(methods),
		"async_methods":              buildConnectionImplAsyncMethodList(asyncMethods),
	}

	content, err := renderTemplate("templates/internal/connection_impl.cc.mustache", data)
	if err != nil {
		panic(err)
	}

	return filepath.Clean(ccPath), content
}
