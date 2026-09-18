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

func buildRestConnectionImplMethodList(ann *serviceAnnotations, methods []*api.Method) []map[string]any {
	var list []map[string]any
	for _, m := range getRestMethods(methods) {
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
		}

		if isPaginated(m) {
			entry["is_paginated"] = true
		} else if isLongrunning(m) {
			entry["is_longrunning"] = true
			if mann.IsGRPCLongrunningOperation() {
				entry["is_grpc_lro"] = true
				if isResponseTypeEmpty(m) {
					entry["is_response_type_empty"] = true
				}
				extractor := "&google::cloud::internal::ExtractLongRunningResultResponse<" + mann.LongrunningDeducedResponseType() + ">"
				if isLongrunningMetadataTypeUsedAsResponse(m) {
					extractor = "&google::cloud::internal::ExtractLongRunningResultMetadata<" + mann.LongrunningDeducedResponseType() + ">"
				}
				entry["lro_extractor"] = extractor
			} else if mann.IsHTTPLongrunningOperation() {
				entry["is_http_lro"] = true
				entry["longrunning_response_type"] = ann.LongrunningResponseType()
				entry["longrunning_get_operation_request_type"] = ann.LongrunningGetOperationRequestType()
				entry["longrunning_cancel_operation_request_type"] = ann.LongrunningCancelOperationRequestType()
				entry["longrunning_set_operation_fields"] = mann.LongrunningSetOperationFields()
				entry["longrunning_await_set_operation_fields"] = mann.LongrunningAwaitSetOperationFields()
			}
		} else {
			entry["is_plain_unary"] = true
		}
		list = append(list, entry)
	}
	return list
}

func buildRestConnectionImplAsyncMethodList(asyncMethods []*api.Method) []map[string]any {
	var list []map[string]any
	for _, m := range getRestAsyncMethods(asyncMethods) {
		mann := m.Codec.(*methodAnnotations)
		entry := map[string]any{
			"method_name":   mann.MethodName(),
			"request_type":  mann.RequestType(),
			"response_type": mann.ResponseType(),
			"return_type":   mann.ReturnType(),
		}
		list = append(list, entry)
	}
	return list
}

func generateRestConnectionImplHeader(_ *api.Service, ann *serviceAnnotations, methods, asyncMethods []*api.Method, _ *config.Library, _ *api.API) (string, string) {
	headerPath := ann.ConnectionImplRestHeaderPath()
	guard := formatHeaderIncludeGuard(headerPath)

	restMethods := getRestMethods(methods)
	var localIncludes []string
	localIncludes = append(localIncludes,
		ann.ConnectionHeaderPath(),
		ann.IdempotencyPolicyHeaderPath(),
		ann.OptionsHeaderPath(),
		ann.StubRestHeaderPath(),
		ann.RetryTraitsHeaderPath(),
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
		protoIncludes = append(protoIncludes, ann.LongrunningOperationIncludeHeader())
	}

	data := map[string]any{
		"header_include_guard":            guard,
		"copyright_year":                  ann.CopyrightYear,
		"proto_file_name":                 ann.ProtoFileName,
		"product_namespace":               ann.Namespace(),
		"product_internal_namespace":      ann.InternalNamespace(),
		"connection_class_name":           ann.ConnectionClassName(),
		"connection_impl_rest_class_name": ann.ConnectionImplRestClassName(),
		"stub_rest_class_name":            ann.StubRestClassName(),
		"service_name":                    ann.ServiceName,
		"retry_policy_name":               ann.RetryPolicyName(),

		"idempotency_class_name": ann.IdempotencyClassName(),
		"local_includes":         localIncludes,
		"proto_includes":         protoIncludes,
		"methods":                buildRestConnectionImplMethodList(ann, methods),
		"async_methods":          buildRestConnectionImplAsyncMethodList(asyncMethods),
		"has_lro":                hasLongrunningMethod(restMethods),
	}

	content, err := renderTemplate("templates/internal/rest_connection_impl.h.mustache", data)
	if err != nil {
		panic(err)
	}

	return filepath.Clean(headerPath), content
}

func generateRestConnectionImplCc(_ *api.Service, ann *serviceAnnotations, methods, asyncMethods []*api.Method, _ *config.Library, _ *api.API) (string, string) {
	ccPath := ann.ConnectionImplRestCcPath()

	restMethods := getRestMethods(methods)
	var localIncludes []string
	localIncludes = append(localIncludes,
		ann.ConnectionImplRestHeaderPath(),
		ann.StubFactoryRestHeaderPath(),
		"google/cloud/common_options.h",
		"google/cloud/credentials.h",
	)
	if ann.HasGRPCLongrunningOperation() {
		localIncludes = append(localIncludes, "google/cloud/internal/async_rest_long_running_operation.h")
	} else if ann.HasHTTPLongrunningOperation() {
		localIncludes = append(localIncludes, "google/cloud/internal/async_rest_long_running_operation_custom.h")
		localIncludes = append(localIncludes, "google/cloud/internal/rest_lro_helpers.h")
	}
	if len(asyncMethods) > 0 {
		localIncludes = append(localIncludes, "google/cloud/internal/async_rest_retry_loop.h")
	}
	if ann.HasLongrunningMethod() {
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
		"copyright_year":                  ann.CopyrightYear,
		"proto_file_name":                 ann.ProtoFileName,
		"product_namespace":               ann.Namespace(),
		"product_internal_namespace":      ann.InternalNamespace(),
		"connection_class_name":           ann.ConnectionClassName(),
		"connection_impl_rest_class_name": ann.ConnectionImplRestClassName(),
		"stub_rest_class_name":            ann.StubRestClassName(),
		"service_name":                    ann.ServiceName,
		"retry_policy_name":               ann.RetryPolicyName(),

		"idempotency_class_name": ann.IdempotencyClassName(),
		"local_includes":         localIncludes,
		"methods":                buildRestConnectionImplMethodList(ann, methods),
		"async_methods":          buildRestConnectionImplAsyncMethodList(asyncMethods),
	}

	content, err := renderTemplate("templates/internal/rest_connection_impl.cc.mustache", data)
	if err != nil {
		panic(err)
	}

	return filepath.Clean(ccPath), content
}
