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

func buildRestStubMethodList(ann *serviceAnnotations, methods []*api.Method) []map[string]any {
	var list []map[string]any
	preserveJson := "false"
	if ann.PreserveProtoFieldNamesInJson {
		preserveJson = "true"
	}
	for _, m := range getRestMethods(methods) {
		mann := m.Codec.(*methodAnnotations)
		entry := map[string]any{
			"method_name":                        mann.MethodName(),
			"request_type":                       mann.RequestType(),
			"response_type":                      mann.ResponseType(),
			"return_type":                        mann.ReturnType(),
			"method_http_verb":                   mann.HTTPVerb,
			"method_http_query_parameters":       mann.HTTPQueryParams,
			"request_resource":                   mann.RequestResource,
			"preserve_proto_field_names_in_json": preserveJson,
			"method_rest_path":                   mann.RestPath,
			"method_rest_path_async":             mann.RestPathAsync,
		}
		if mann.IsLongrunning() {
			entry["is_longrunning"] = true
		} else if isResponseTypeEmpty(m) {
			entry["is_response_type_empty"] = true
		}
		list = append(list, entry)
	}
	return list
}

func buildRestStubAsyncMethodList(ann *serviceAnnotations, asyncMethods []*api.Method) []map[string]any {
	var list []map[string]any
	preserveJson := "false"
	if ann.PreserveProtoFieldNamesInJson {
		preserveJson = "true"
	}
	for _, m := range getRestAsyncMethods(asyncMethods) {
		mann := m.Codec.(*methodAnnotations)
		entry := map[string]any{
			"method_name":                        mann.MethodName(),
			"request_type":                       mann.RequestType(),
			"response_type":                      mann.ResponseType(),
			"return_type":                        mann.ReturnType(),
			"method_http_verb":                   mann.HTTPVerb,
			"method_http_query_parameters":       mann.HTTPQueryParams,
			"request_resource":                   mann.RequestResource,
			"preserve_proto_field_names_in_json": preserveJson,
			"method_rest_path_async":             mann.RestPathAsync,
		}
		if isResponseTypeEmpty(m) {
			entry["is_response_type_empty"] = true
		}
		list = append(list, entry)
	}
	return list
}

func generateRestStubHeader(svc *api.Service, ann *serviceAnnotations, methods, asyncMethods []*api.Method, _ *config.Library, model *api.API) (string, string) {
	headerPath := ann.StubRestHeaderPath()
	guard := ann.StubRestHeaderIncludeGuard()

	localIncludes := []string{
		"google/cloud/completion_queue.h",
		"google/cloud/internal/rest_client.h",
		"google/cloud/internal/rest_context.h",
		"google/cloud/status_or.h",
		"google/cloud/version.h",
	}
	slices.Sort(localIncludes)

	var protoIncludes []string
	if len(ann.AdditionalPbHeaderPaths) > 0 {
		protoIncludes = append(protoIncludes, ann.AdditionalPbHeaderPaths...)
	}

	var mixinHeaders []string
	seenMixin := make(map[string]bool)
	for _, m := range methods {
		if m.SourceService != nil && m.SourceService.ID != svc.ID {
			h := ""
			if loc, ok := model.DefinitionLocation(m.SourceService.ID[1:]); ok {
				h = strings.TrimSuffix(loc.Filename, ".proto") + ".pb.h"
			} else if m.SourceService.ID == ".google.cloud.location.Locations" {
				h = "google/cloud/location/locations.pb.h"
			} else if m.SourceService.ID == ".google.iam.v1.IAMPolicy" {
				h = "google/iam/v1/iam_policy.pb.h"
			} else if m.SourceService.ID == ".google.longrunning.Operations" {
				h = "google/longrunning/operations.pb.h"
			}
			if h != "" && !seenMixin[h] {
				seenMixin[h] = true
				mixinHeaders = append(mixinHeaders, h)
			}
		}
	}
	slices.Sort(mixinHeaders)
	protoIncludes = append(protoIncludes, mixinHeaders...)

	var mainProtoIncludes []string
	mainProtoIncludes = append(mainProtoIncludes, ann.ProtoHeaderPath())
	if hasLongrunningMethod(methods) && !seenMixin["google/longrunning/operations.pb.h"] {
		mainProtoIncludes = append(mainProtoIncludes, "google/longrunning/operations.pb.h")
	}
	slices.Sort(mainProtoIncludes)
	protoIncludes = append(protoIncludes, mainProtoIncludes...)

	data := map[string]any{
		"header_include_guard":       guard,
		"copyright_year":             ann.CopyrightYear,
		"proto_file_name":            ann.ProtoFileName,
		"product_internal_namespace": ann.InternalNamespace(),
		"stub_rest_class_name":       ann.StubRestClassName(),
		"local_includes":             localIncludes,
		"proto_includes":             protoIncludes,
		"methods":                    buildRestStubMethodList(ann, methods),
		"async_methods":              buildRestStubAsyncMethodList(ann, asyncMethods),
		"has_lro":                    hasLongrunningMethod(methods),
	}

	content, err := renderTemplate("templates/internal/rest_stub.h.mustache", data)
	if err != nil {
		panic(err)
	}

	return filepath.Clean(headerPath), content
}

func generateRestStubCc(_ *api.Service, ann *serviceAnnotations, methods, asyncMethods []*api.Method, _ *config.Library, _ *api.API) (string, string) {
	ccPath := ann.StubRestCcPath()

	localIncludes := []string{
		ann.StubRestHeaderPath(),
		"google/cloud/common_options.h",
		"google/cloud/internal/absl_str_cat_quiet.h",
		"google/cloud/internal/rest_stub_helpers.h",
		"google/cloud/status_or.h",
	}
	slices.Sort(localIncludes)

	var protoIncludes []string
	protoIncludes = append(protoIncludes, ann.ProtoHeaderPath())
	if hasLongrunningMethod(methods) {
		protoIncludes = append(protoIncludes, "google/longrunning/operations.pb.h")
	}
	slices.Sort(protoIncludes)

	preserveJson := "false"
	if ann.PreserveProtoFieldNamesInJson {
		preserveJson = "true"
	}

	data := map[string]any{
		"copyright_year":                         ann.CopyrightYear,
		"proto_file_name":                        ann.ProtoFileName,
		"product_internal_namespace":             ann.InternalNamespace(),
		"stub_rest_class_name":                   ann.StubRestClassName(),
		"local_includes":                         localIncludes,
		"proto_includes":                         protoIncludes,
		"methods":                                buildRestStubMethodList(ann, methods),
		"async_methods":                          buildRestStubAsyncMethodList(ann, asyncMethods),
		"has_lro":                                hasLongrunningMethod(methods),
		"longrunning_get_operation_path_rest":    ann.LongrunningGetOperationPathRest(),
		"longrunning_cancel_operation_path_rest": ann.LongrunningCancelOperationPathRest(),
		"preserve_proto_field_names_in_json":     preserveJson,
	}

	content, err := renderTemplate("templates/internal/rest_stub.cc.mustache", data)
	if err != nil {
		panic(err)
	}

	return filepath.Clean(ccPath), content
}
