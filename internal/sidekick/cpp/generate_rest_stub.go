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

func buildRestStubMethodList(svc *api.Service, methods []*api.Method, serviceVars map[string]string, lib *config.Library, model *api.API) []map[string]any {
	var list []map[string]any
	for _, m := range getRestMethods(methods) {
		mVars := buildMethodVars(svc, m, serviceVars, lib, model)
		entry := map[string]any{
			"method_name":                        mVars["method_name"],
			"request_type":                       mVars["request_type"],
			"response_type":                      mVars["response_type"],
			"return_type":                        mVars["return_type"],
			"method_http_verb":                   mVars["method_http_verb"],
			"method_http_query_parameters":       mVars["method_http_query_parameters"],
			"request_resource":                   mVars["request_resource"],
			"preserve_proto_field_names_in_json": mVars["preserve_proto_field_names_in_json"],
			"method_rest_path":                   mVars["method_rest_path"],
			"method_rest_path_async":             mVars["method_rest_path_async"],
		}
		if isLongrunning(m) {
			entry["is_longrunning"] = true
		} else if isResponseTypeEmpty(m) {
			entry["is_response_type_empty"] = true
		}
		list = append(list, entry)
	}
	return list
}

func buildRestStubAsyncMethodList(svc *api.Service, asyncMethods []*api.Method, serviceVars map[string]string, lib *config.Library, model *api.API) []map[string]any {
	var list []map[string]any
	for _, m := range getRestAsyncMethods(asyncMethods) {
		mVars := buildMethodVars(svc, m, serviceVars, lib, model)
		entry := map[string]any{
			"method_name":                        mVars["method_name"],
			"request_type":                       mVars["request_type"],
			"response_type":                      mVars["response_type"],
			"return_type":                        mVars["return_type"],
			"method_http_verb":                   mVars["method_http_verb"],
			"method_http_query_parameters":       mVars["method_http_query_parameters"],
			"request_resource":                   mVars["request_resource"],
			"preserve_proto_field_names_in_json": mVars["preserve_proto_field_names_in_json"],
			"method_rest_path_async":             mVars["method_rest_path_async"],
		}
		if isResponseTypeEmpty(m) {
			entry["is_response_type_empty"] = true
		}
		list = append(list, entry)
	}
	return list
}

func generateRestStubHeader(svc *api.Service, serviceVars map[string]string, methods, asyncMethods []*api.Method, lib *config.Library, model *api.API) (string, string) {
	headerPath := serviceVars["stub_rest_header_path"]
	guard := formatHeaderIncludeGuard(headerPath)

	localIncludes := []string{
		"google/cloud/completion_queue.h",
		"google/cloud/internal/rest_client.h",
		"google/cloud/internal/rest_context.h",
		"google/cloud/status_or.h",
		"google/cloud/version.h",
	}
	slices.Sort(localIncludes)

	var protoIncludes []string
	if addPaths, ok := serviceVars["additional_pb_header_paths"]; ok && addPaths != "" {
		var additionalProtoIncludes []string
		for add := range strings.SplitSeq(addPaths, ",") {
			if add != "" {
				additionalProtoIncludes = append(additionalProtoIncludes, add)
			}
		}
		slices.Sort(additionalProtoIncludes)
		protoIncludes = append(protoIncludes, additionalProtoIncludes...)
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
	mainProtoIncludes = append(mainProtoIncludes, serviceVars["proto_header_path"])
	if hasLongrunningMethod(methods) && !seenMixin["google/longrunning/operations.pb.h"] {
		mainProtoIncludes = append(mainProtoIncludes, "google/longrunning/operations.pb.h")
	}
	slices.Sort(mainProtoIncludes)
	protoIncludes = append(protoIncludes, mainProtoIncludes...)

	data := map[string]any{
		"header_include_guard":       guard,
		"copyright_year":             serviceVars["copyright_year"],
		"proto_file_name":            serviceVars["proto_file_name"],
		"product_internal_namespace": serviceVars["product_internal_namespace"],
		"stub_rest_class_name":       serviceVars["stub_rest_class_name"],
		"local_includes":             localIncludes,
		"proto_includes":             protoIncludes,
		"methods":                    buildRestStubMethodList(svc, methods, serviceVars, lib, model),
		"async_methods":              buildRestStubAsyncMethodList(svc, asyncMethods, serviceVars, lib, model),
		"has_lro":                    hasLongrunningMethod(methods),
	}

	content, err := renderTemplate("templates/internal/rest_stub.h.mustache", data)
	if err != nil {
		panic(err)
	}

	return filepath.Clean(headerPath), content
}

func generateRestStubCc(svc *api.Service, serviceVars map[string]string, methods, asyncMethods []*api.Method, lib *config.Library, model *api.API) (string, string) {
	ccPath := serviceVars["stub_rest_cc_path"]

	localIncludes := []string{
		serviceVars["stub_rest_header_path"],
		"google/cloud/common_options.h",
		"google/cloud/internal/absl_str_cat_quiet.h",
		"google/cloud/internal/rest_stub_helpers.h",
		"google/cloud/status_or.h",
	}
	slices.Sort(localIncludes)

	var protoIncludes []string
	protoIncludes = append(protoIncludes, serviceVars["proto_header_path"])
	if hasLongrunningMethod(methods) {
		protoIncludes = append(protoIncludes, "google/longrunning/operations.pb.h")
	}
	slices.Sort(protoIncludes)

	data := map[string]any{
		"copyright_year":                         serviceVars["copyright_year"],
		"proto_file_name":                        serviceVars["proto_file_name"],
		"product_internal_namespace":             serviceVars["product_internal_namespace"],
		"stub_rest_class_name":                   serviceVars["stub_rest_class_name"],
		"local_includes":                         localIncludes,
		"proto_includes":                         protoIncludes,
		"methods":                                buildRestStubMethodList(svc, methods, serviceVars, lib, model),
		"async_methods":                          buildRestStubAsyncMethodList(svc, asyncMethods, serviceVars, lib, model),
		"has_lro":                                hasLongrunningMethod(methods),
		"longrunning_get_operation_path_rest":    serviceVars["longrunning_get_operation_path_rest"],
		"longrunning_cancel_operation_path_rest": serviceVars["longrunning_cancel_operation_path_rest"],
		"preserve_proto_field_names_in_json":     serviceVars["preserve_proto_field_names_in_json"],
	}

	content, err := renderTemplate("templates/internal/rest_stub.cc.mustache", data)
	if err != nil {
		panic(err)
	}

	return filepath.Clean(ccPath), content
}
