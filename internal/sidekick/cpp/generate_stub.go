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

type mixinStubInfo struct {
	stubName string
	stubFQN  string
	header   string
}

func getMixinStubs(svc *api.Service, methods []*api.Method) []mixinStubInfo {
	wellKnown := []struct {
		id       string
		stubName string
		stubFQN  string
		header   string
	}{
		{
			id:       ".google.longrunning.Operations",
			stubName: "operations_stub",
			stubFQN:  "google::longrunning::Operations",
			header:   "google/longrunning/operations.grpc.pb.h",
		},
		{
			id:       ".google.iam.v1.IAMPolicy",
			stubName: "iampolicy_stub",
			stubFQN:  "google::iam::v1::IAMPolicy",
			header:   "google/iam/v1/iam_policy.grpc.pb.h",
		},
		{
			id:       ".google.cloud.location.Locations",
			stubName: "locations_stub",
			stubFQN:  "google::cloud::location::Locations",
			header:   "google/cloud/location/locations.grpc.pb.h",
		},
	}

	seen := make(map[string]bool)
	for _, m := range methods {
		if m.SourceService != nil && m.SourceService.ID != svc.ID {
			seen[m.SourceService.ID] = true
		}
	}

	var result []mixinStubInfo
	for _, wk := range wellKnown {
		if seen[wk.id] {
			result = append(result, mixinStubInfo{
				stubName: wk.stubName,
				stubFQN:  wk.stubFQN,
				header:   wk.header,
			})
		}
	}
	return result
}

func buildStubMethodList(svc *api.Service, methods []*api.Method, serviceVars map[string]string, lib *config.Library, model *api.API) []map[string]any {
	var list []map[string]any
	for _, m := range methods {
		mVars := buildMethodVars(svc, m, serviceVars, lib, model)
		entry := map[string]any{
			"method_name":   mVars["method_name"],
			"request_type":  mVars["request_type"],
			"response_type": mVars["response_type"],
			"return_type":   mVars["return_type"],
			"grpc_stub":     mVars["grpc_stub"],
		}
		if isStreamingWrite(m) {
			entry["is_streaming_write"] = true
		} else if isBidiStreaming(m) {
			entry["is_bidi_streaming"] = true
		} else if isLongrunning(m) {
			entry["is_longrunning"] = true
		} else if isStreamingRead(m) {
			entry["is_streaming_read"] = true
		} else if isResponseTypeEmpty(m) {
			entry["is_response_type_empty"] = true
			entry["is_plain_unary"] = true
		} else {
			entry["is_other_unary"] = true
			entry["is_plain_unary"] = true
		}
		list = append(list, entry)
	}
	return list
}

func buildStubAsyncMethodList(svc *api.Service, asyncMethods []*api.Method, serviceVars map[string]string, lib *config.Library, model *api.API) []map[string]any {
	var list []map[string]any
	for _, m := range asyncMethods {
		if isBidiStreaming(m) || isLongrunning(m) {
			continue
		}
		mVars := buildMethodVars(svc, m, serviceVars, lib, model)
		entry := map[string]any{
			"method_name":   mVars["method_name"],
			"request_type":  mVars["request_type"],
			"response_type": mVars["response_type"],
			"return_type":   mVars["return_type"],
			"grpc_stub":     mVars["grpc_stub"],
		}
		if isStreamingRead(m) {
			entry["is_streaming_read"] = true
		} else if isStreamingWrite(m) {
			entry["is_streaming_write"] = true
		} else if isResponseTypeEmpty(m) {
			entry["is_response_type_empty"] = true
			entry["is_unary"] = true
		} else {
			entry["is_other_unary"] = true
			entry["is_unary"] = true
		}
		list = append(list, entry)
	}
	return list
}

func generateStubHeader(svc *api.Service, serviceVars map[string]string, methods []*api.Method, asyncMethods []*api.Method, lib *config.Library, model *api.API) (string, string) {
	headerPath := serviceVars["stub_header_path"]
	guard := formatHeaderIncludeGuard(headerPath)

	hasAsync := len(asyncMethods) > 0 || hasLongrunningMethod(methods)
	needsCompletionQueue := hasAsync || hasBidiStreamingMethod(methods)

	var localIncludes []string
	if hasBidiStreamingMethod(methods) {
		localIncludes = append(localIncludes, "google/cloud/async_streaming_read_write_rpc.h")
	}
	if needsCompletionQueue {
		localIncludes = append(localIncludes, "google/cloud/completion_queue.h")
	}
	if hasAsync {
		localIncludes = append(localIncludes, "google/cloud/future.h")
	}
	if hasAsynchronousStreamingReadMethod(asyncMethods) {
		localIncludes = append(localIncludes, "google/cloud/internal/async_streaming_read_rpc.h")
	}
	if hasAsynchronousStreamingWriteMethod(asyncMethods) {
		localIncludes = append(localIncludes, "google/cloud/internal/async_streaming_write_rpc.h")
	}
	if hasStreamingReadMethod(methods) {
		localIncludes = append(localIncludes, "google/cloud/internal/streaming_read_rpc.h")
	}
	if hasStreamingWriteMethod(methods) {
		localIncludes = append(localIncludes, "google/cloud/internal/streaming_write_rpc.h")
	}
	localIncludes = append(localIncludes,
		"google/cloud/options.h",
		"google/cloud/status_or.h",
		"google/cloud/version.h",
	)
	slices.Sort(localIncludes)

	var protoIncludes []string
	if serviceVars["additional_pb_header_paths"] != "" {
		var additionalPb []string
		for h := range strings.SplitSeq(serviceVars["additional_pb_header_paths"], ",") {
			if h != "" {
				additionalPb = append(additionalPb, h)
			}
		}
		slices.Sort(additionalPb)
		protoIncludes = append(protoIncludes, additionalPb...)
	}
	allMixins := getMixinStubs(svc, methods)
	var mixinHeaders []string
	for _, mixin := range allMixins {
		if mixin.header != "" {
			mixinHeaders = append(mixinHeaders, mixin.header)
		}
	}
	slices.Sort(mixinHeaders)
	protoIncludes = append(protoIncludes, mixinHeaders...)

	var mainPb []string
	if serviceVars["proto_grpc_header_path"] != "" {
		mainPb = append(mainPb, serviceVars["proto_grpc_header_path"])
	}
	includeLroHeader := hasLongrunningMethod(methods) && !slices.Contains(mixinHeaders, "google/longrunning/operations.grpc.pb.h")
	if includeLroHeader {
		mainPb = append(mainPb, "google/longrunning/operations.grpc.pb.h")
	}
	slices.Sort(mainPb)
	protoIncludes = append(protoIncludes, mainPb...)

	hasLro := hasLongrunningMethod(methods)
	var filteredMixins []map[string]string
	for _, mixin := range allMixins {
		if hasLro && mixin.stubName == "operations_stub" {
			continue
		}
		filteredMixins = append(filteredMixins, map[string]string{
			"stub_name": mixin.stubName,
			"stub_fqn":  mixin.stubFQN,
			"header":    mixin.header,
		})
	}

	data := map[string]any{
		"header_include_guard":       guard,
		"copyright_year":             serviceVars["copyright_year"],
		"proto_file_name":            serviceVars["proto_file_name"],
		"product_internal_namespace": serviceVars["product_internal_namespace"],
		"stub_class_name":            serviceVars["stub_class_name"],
		"grpc_stub_fqn":              serviceVars["grpc_stub_fqn"],
		"local_includes":             localIncludes,
		"proto_includes":             protoIncludes,
		"methods":                    buildStubMethodList(svc, methods, serviceVars, lib, model),
		"async_methods":              buildStubAsyncMethodList(svc, asyncMethods, serviceVars, lib, model),
		"has_lro":                    hasLro,
		"has_mixins":                 len(filteredMixins) > 0,
		"mixins":                     filteredMixins,
	}

	content, err := renderTemplate("templates/internal/stub.h.mustache", data)
	if err != nil {
		panic(err)
	}

	return filepath.Clean(headerPath), content
}

func generateStubCc(svc *api.Service, serviceVars map[string]string, methods []*api.Method, asyncMethods []*api.Method, lib *config.Library, model *api.API) (string, string) {
	ccPath := serviceVars["stub_cc_path"]

	var ccLocalIncludes = []string{
		serviceVars["stub_header_path"],
		"google/cloud/grpc_error_delegate.h",
		"google/cloud/status_or.h",
	}
	if hasBidiStreamingMethod(methods) {
		ccLocalIncludes = append(ccLocalIncludes, "google/cloud/internal/async_read_write_stream_impl.h")
	}
	if hasAsynchronousStreamingReadMethod(asyncMethods) {
		ccLocalIncludes = append(ccLocalIncludes, "google/cloud/internal/async_streaming_read_rpc_impl.h")
	}
	if hasAsynchronousStreamingWriteMethod(asyncMethods) {
		ccLocalIncludes = append(ccLocalIncludes, "google/cloud/internal/async_streaming_write_rpc_impl.h")
	}
	if hasStreamingWriteMethod(methods) {
		ccLocalIncludes = append(ccLocalIncludes, "google/cloud/internal/streaming_write_rpc_impl.h")
	}
	slices.Sort(ccLocalIncludes)

	var pbIncludes []string
	if serviceVars["proto_grpc_header_path"] != "" {
		pbIncludes = append(pbIncludes, serviceVars["proto_grpc_header_path"])
	}
	if hasLongrunningMethod(methods) {
		pbIncludes = append(pbIncludes, "google/longrunning/operations.grpc.pb.h")
	}

	data := map[string]any{
		"copyright_year":             serviceVars["copyright_year"],
		"proto_file_name":            serviceVars["proto_file_name"],
		"product_internal_namespace": serviceVars["product_internal_namespace"],
		"stub_class_name":            serviceVars["stub_class_name"],
		"local_includes":             ccLocalIncludes,
		"proto_includes":             pbIncludes,
		"methods":                    buildStubMethodList(svc, methods, serviceVars, lib, model),
		"async_methods":              buildStubAsyncMethodList(svc, asyncMethods, serviceVars, lib, model),
		"has_lro":                    hasLongrunningMethod(methods),
	}

	content, err := renderTemplate("templates/internal/stub.cc.mustache", data)
	if err != nil {
		panic(err)
	}

	return filepath.Clean(ccPath), content
}
