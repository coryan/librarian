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

type mixinStub struct {
	stubName string
	stubFQN  string
	header   string
}

func getMixinStubs(svc *api.Service, methods []*api.Method) []mixinStub {
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

	var result []mixinStub
	for _, wk := range wellKnown {
		if seen[wk.id] {
			result = append(result, mixinStub{
				stubName: wk.stubName,
				stubFQN:  wk.stubFQN,
				header:   wk.header,
			})
			delete(seen, wk.id)
		}
	}

	for _, m := range methods {
		if m.SourceService != nil && m.SourceService.ID != svc.ID && seen[m.SourceService.ID] {
			sourceName := m.SourceService.Name
			stubName := strings.ToLower(sourceName) + "_stub"
			header := strings.TrimPrefix(strings.ReplaceAll(m.SourceService.ID, ".", "/"), "/") + ".grpc.pb.h"
			result = append(result, mixinStub{
				stubName: stubName,
				stubFQN:  protoNameToCppName(m.SourceService.ID[1:]),
				header:   header,
			})
			delete(seen, m.SourceService.ID)
		}
	}
	return result
}

func buildStubMethodList(methods []*api.Method) []map[string]any {
	var list []map[string]any
	for _, m := range methods {
		mann := m.Codec.(*methodAnnotations)
		entry := map[string]any{
			"method_name":   mann.MethodName(),
			"request_type":  mann.RequestType(),
			"response_type": mann.ResponseType(),
			"return_type":   mann.ReturnType(),
			"grpc_stub":     mann.GrpcStub(),
		}
		if mann.IsStreamingWrite() {
			entry["is_streaming_write"] = true
		} else if mann.IsBidiStreaming() {
			entry["is_bidi_streaming"] = true
		} else if mann.IsLongrunning() {
			entry["is_longrunning"] = true
		} else if mann.IsStreamingRead() {
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

func buildStubAsyncMethodList(asyncMethods []*api.Method) []map[string]any {
	var list []map[string]any
	for _, m := range asyncMethods {
		mann := m.Codec.(*methodAnnotations)
		if mann.IsBidiStreaming() || mann.IsLongrunning() {
			continue
		}
		entry := map[string]any{
			"method_name":   mann.MethodName(),
			"request_type":  mann.RequestType(),
			"response_type": mann.ResponseType(),
			"return_type":   mann.ReturnType(),
			"grpc_stub":     mann.GrpcStub(),
		}
		if mann.IsStreamingRead() {
			entry["is_streaming_read"] = true
		} else if mann.IsStreamingWrite() {
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

func generateStubHeader(svc *api.Service, ann *serviceAnnotations, methods, asyncMethods []*api.Method, _ *config.Library, _ *api.API) (string, string) {
	headerPath := ann.StubHeaderPath()
	guard := ann.StubHeaderIncludeGuard()

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
	if len(ann.AdditionalPbHeaderPaths) > 0 {
		protoIncludes = append(protoIncludes, ann.AdditionalPbHeaderPaths...)
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
	if h := ann.ProtoGrpcHeaderPath(); h != "" {
		mainPb = append(mainPb, h)
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
		"copyright_year":             ann.CopyrightYear,
		"proto_file_name":            ann.ProtoFileName,
		"product_internal_namespace": ann.InternalNamespace(),
		"stub_class_name":            ann.StubClassName(),
		"grpc_stub_fqn":              ann.GrpcStubFQN(),
		"local_includes":             localIncludes,
		"proto_includes":             protoIncludes,
		"methods":                    buildStubMethodList(methods),
		"async_methods":              buildStubAsyncMethodList(asyncMethods),
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

func generateStubCc(_ *api.Service, ann *serviceAnnotations, methods, asyncMethods []*api.Method, _ *config.Library, _ *api.API) (string, string) {
	ccPath := ann.StubCcPath()

	var ccLocalIncludes = []string{
		ann.StubHeaderPath(),
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
	if h := ann.ProtoGrpcHeaderPath(); h != "" {
		pbIncludes = append(pbIncludes, h)
	}
	if hasLongrunningMethod(methods) {
		pbIncludes = append(pbIncludes, "google/longrunning/operations.grpc.pb.h")
	}

	data := map[string]any{
		"copyright_year":             ann.CopyrightYear,
		"proto_file_name":            ann.ProtoFileName,
		"product_internal_namespace": ann.InternalNamespace(),
		"stub_class_name":            ann.StubClassName(),
		"local_includes":             ccLocalIncludes,
		"proto_includes":             pbIncludes,
		"methods":                    buildStubMethodList(methods),
		"async_methods":              buildStubAsyncMethodList(asyncMethods),
		"has_lro":                    hasLongrunningMethod(methods),
	}

	content, err := renderTemplate("templates/internal/stub.cc.mustache", data)
	if err != nil {
		panic(err)
	}

	return filepath.Clean(ccPath), content
}
