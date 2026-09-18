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


func buildConnectionMethodList(methods []*api.Method) []map[string]any {
	var list []map[string]any
	for _, m := range methods {
		if isStreamingWrite(m) {
			continue
		}
		mann := m.Codec.(*methodAnnotations)
		entry := map[string]any{
			"method_name":   mann.MethodName(),
			"request_type":  mann.RequestType(),
			"response_type": mann.ResponseType(),
			"return_type":   mann.ReturnType(),
		}
		if isBidiStreaming(m) {
			entry["is_bidi_streaming"] = true
		} else if isStreamingRead(m) {
			entry["is_streaming_read"] = true
		} else if isPaginated(m) {
			entry["is_paginated"] = true
			entry["range_output_type"] = mann.RangeOutputType()
		} else if isLongrunning(m) {
			entry["is_longrunning"] = true
			entry["longrunning_operation_type"] = mann.LongrunningOperationType()
			if isResponseTypeEmpty(m) {
				entry["is_response_type_empty"] = true
			} else {
				entry["longrunning_deduced_response_type"] = mann.LongrunningDeducedResponseType()
			}
		} else {
			entry["is_plain_unary"] = true
		}
		list = append(list, entry)
	}
	return list
}

func buildConnectionAsyncMethodList(asyncMethods []*api.Method) []map[string]any {
	var list []map[string]any
	for _, m := range asyncMethods {
		if isStreamingRead(m) || isStreamingWrite(m) {
			continue
		}
		mann := m.Codec.(*methodAnnotations)
		list = append(list, map[string]any{
			"method_name":            mann.MethodName(),
			"request_type":           mann.RequestType(),
			"response_type":          mann.ResponseType(),
			"return_type":            mann.ReturnType(),
			"is_response_type_empty": isResponseTypeEmpty(m),
		})
	}
	return list
}

func generateConnectionHeader(_ *api.Service, ann *serviceAnnotations, methods, asyncMethods []*api.Method, _ *config.Library, _ *api.API) (string, string) {
	headerPath := ann.ConnectionHeaderPath()
	guard := formatHeaderIncludeGuard(headerPath)

	var localIncludes []string
	localIncludes = append(localIncludes,
		ann.IdempotencyPolicyHeaderPath(),
		ann.RetryTraitsHeaderPath(),
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
	protoIncludes = append(protoIncludes, ann.AdditionalPbHeaderPaths...)
	protoIncludes = append(protoIncludes, ann.ProtoHeaderPath())
	if hasLongrunningMethod(methods) {
		protoIncludes = append(protoIncludes, "google/longrunning/operations.grpc.pb.h")
	}
	slices.Sort(protoIncludes)

	hasGrpc := ann.HasGrpcTransport
	isLocationDependent := ann.IsLocationDependent()

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
		"copyright_year":                        ann.CopyrightYear,
		"proto_file_name":                       ann.ProtoFileName,
		"product_namespace":                     ann.Namespace(),
		"product_internal_namespace":            ann.InternalNamespace(),
		"connection_class_name":                 ann.ConnectionClassName(),
		"client_class_name":                     ann.ClientClassName(),
		"retry_policy_name":                     ann.RetryPolicyName(),
		"limited_error_count_retry_policy_name": ann.LimitedErrorCountRetryPolicyName(),
		"limited_time_retry_policy_name":        ann.LimitedTimeRetryPolicyName(),
		"retry_traits_name":                     ann.RetryTraitsName(),
		"transient_errors_comment":              ann.TransientErrorsComment(),
		"mock_connection_class_name":            ann.MockConnectionClassName(),
		"product_mocks_namespace":               ann.MocksNamespace(),
		"service_name":                          ann.ServiceName,
		"has_grpc":                              hasGrpc,
		"is_location_dependent":                 isLocationDependent,
		"is_location_dependent_compat":          ann.EndpointLocationStyle == "LOCATION_DEPENDENT_COMPAT",
		"is_location_optionally_dependent":      ann.EndpointLocationStyle == "LOCATION_OPTIONALLY_DEPENDENT",
		"location_doc":                          locationDoc,
		"local_includes":                        localIncludes,
		"proto_includes":                        protoIncludes,
		"system_includes":                       sysIncludes,
		"methods":                               buildConnectionMethodList(methods),
		"async_methods":                         buildConnectionAsyncMethodList(asyncMethods),
	}

	content, err := renderTemplate("templates/connection.h.mustache", data)
	if err != nil {
		panic(err)
	}

	return filepath.Clean(headerPath), content
}

func generateConnectionCc(_ *api.Service, ann *serviceAnnotations, methods, asyncMethods []*api.Method, _ *config.Library, _ *api.API) (string, string) {
	ccPath := ann.ConnectionCcPath()

	hasGrpc := ann.HasGrpcTransport

	var localIncludes []string
	localIncludes = append(localIncludes,
		ann.ConnectionHeaderPath(),
		ann.OptionsHeaderPath(),
	)
	if hasGrpc {
		localIncludes = append(localIncludes, ann.ConnectionImplHeaderPath())
	}
	localIncludes = append(localIncludes, ann.OptionDefaultsHeaderPath())
	if hasGrpc {
		localIncludes = append(localIncludes, ann.StubFactoryHeaderPath())
	}
	localIncludes = append(localIncludes,
		ann.TracingConnectionHeaderPath(),
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

	isLocationDependent := ann.IsLocationDependent()
	hasNonLocationOverload := ann.HasNonLocationOverload()

	data := map[string]any{
		"copyright_year":                ann.CopyrightYear,
		"proto_file_name":               ann.ProtoFileName,
		"product_namespace":             ann.Namespace(),
		"product_internal_namespace":    ann.InternalNamespace(),
		"connection_class_name":         ann.ConnectionClassName(),
		"service_name":                  ann.ServiceName,
		"stub_class_name":               ann.StubClassName(),
		"tracing_connection_class_name": ann.TracingConnectionClassName(),
		"has_grpc":                      hasGrpc,
		"is_location_dependent":         isLocationDependent,
		"has_non_location_overload":     hasNonLocationOverload,
		"local_includes":                localIncludes,
		"system_includes":               []string{"memory", "utility"},
		"methods":                       buildConnectionMethodList(methods),
		"async_methods":                 buildConnectionAsyncMethodList(asyncMethods),
	}

	content, err := renderTemplate("templates/connection.cc.mustache", data)
	if err != nil {
		panic(err)
	}

	return filepath.Clean(ccPath), content
}

