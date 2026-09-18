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

func buildRestLoggingDecoratorMethodList(methods []*api.Method) []map[string]any {
	var list []map[string]any
	for _, m := range getRestMethods(methods) {
		mann := m.Codec.(*methodAnnotations)
		entry := map[string]any{
			"method_name":   mann.MethodName(),
			"request_type":  mann.RequestType(),
			"response_type": mann.ResponseType(),
			"return_type":   mann.ReturnType(),
		}
		if mann.IsLongrunning() {
			entry["is_longrunning"] = true
		}
		list = append(list, entry)
	}
	return list
}

func buildRestLoggingDecoratorAsyncMethodList(asyncMethods []*api.Method) []map[string]any {
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

func generateRestLoggingDecoratorHeader(_ *api.Service, ann *serviceAnnotations, methods, asyncMethods []*api.Method, _ *config.Library, _ *api.API) (string, string) {
	headerPath := ann.LoggingRestHeaderPath()
	guard := ann.LoggingRestHeaderIncludeGuard()

	localIncludes := []string{
		ann.StubRestHeaderPath(),
		"google/cloud/future.h",
		"google/cloud/internal/rest_context.h",
		"google/cloud/tracing_options.h",
		"google/cloud/version.h",
	}
	slices.Sort(localIncludes)

	var protoIncludes []string
	protoIncludes = append(protoIncludes, ann.ProtoHeaderPath())
	if hasLongrunningMethod(methods) {
		protoIncludes = append(protoIncludes, ann.LongrunningOperationIncludeHeader())
	}
	slices.Sort(protoIncludes)

	data := map[string]any{
		"header_include_guard":                      guard,
		"copyright_year":                            ann.CopyrightYear,
		"proto_file_name":                           ann.ProtoFileName,
		"product_internal_namespace":                ann.InternalNamespace(),
		"logging_rest_class_name":                   ann.LoggingRestClassName(),
		"stub_rest_class_name":                      ann.StubRestClassName(),
		"local_includes":                            localIncludes,
		"proto_includes":                            protoIncludes,
		"methods":                                   buildRestLoggingDecoratorMethodList(methods),
		"async_methods":                             buildRestLoggingDecoratorAsyncMethodList(asyncMethods),
		"has_lro":                                   hasLongrunningMethod(methods),
		"longrunning_response_type":                 ann.LongrunningResponseType(),
		"longrunning_get_operation_request_type":    ann.LongrunningGetOperationRequestType(),
		"longrunning_cancel_operation_request_type": ann.LongrunningCancelOperationRequestType(),
	}

	content, err := renderTemplate("templates/internal/rest_logging_decorator.h.mustache", data)
	if err != nil {
		panic(err)
	}

	return filepath.Clean(headerPath), content
}

func generateRestLoggingDecoratorCc(_ *api.Service, ann *serviceAnnotations, methods, asyncMethods []*api.Method, _ *config.Library, _ *api.API) (string, string) {
	ccPath := ann.LoggingRestCcPath()

	localIncludes := []string{
		ann.LoggingRestHeaderPath(),
		"google/cloud/internal/log_wrapper.h",
		"google/cloud/status_or.h",
	}
	slices.Sort(localIncludes)

	data := map[string]any{
		"copyright_year":                            ann.CopyrightYear,
		"proto_file_name":                           ann.ProtoFileName,
		"product_internal_namespace":                ann.InternalNamespace(),
		"logging_rest_class_name":                   ann.LoggingRestClassName(),
		"stub_rest_class_name":                      ann.StubRestClassName(),
		"local_includes":                            localIncludes,
		"methods":                                   buildRestLoggingDecoratorMethodList(methods),
		"async_methods":                             buildRestLoggingDecoratorAsyncMethodList(asyncMethods),
		"has_lro":                                   hasLongrunningMethod(methods),
		"longrunning_response_type":                 ann.LongrunningResponseType(),
		"longrunning_get_operation_request_type":    ann.LongrunningGetOperationRequestType(),
		"longrunning_cancel_operation_request_type": ann.LongrunningCancelOperationRequestType(),
	}

	content, err := renderTemplate("templates/internal/rest_logging_decorator.cc.mustache", data)
	if err != nil {
		panic(err)
	}

	return filepath.Clean(ccPath), content
}
