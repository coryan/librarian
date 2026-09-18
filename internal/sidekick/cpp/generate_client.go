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
	"fmt"
	"path/filepath"
	"slices"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/sidekick/api"
)

func buildClientMethodList(ann *serviceAnnotations, methods []*api.Method, model *api.API) []map[string]any {
	var list []map[string]any
	hasIAM, setMethod := hasIamPolicyExtension(methods)

	for _, m := range methods {
		if isStreamingWrite(m) {
			continue
		}
		mann := m.Codec.(*methodAnnotations)
		isDep := m.Deprecated
		depMacro := ""
		if isDep {
			depMacro = "  GOOGLE_CLOUD_CPP_DEPRECATED(\"This RPC is deprecated.\")\n"
		}

		entry := map[string]any{
			"method_name":          mann.MethodName(),
			"request_type":         mann.RequestType(),
			"response_type":        mann.ResponseType(),
			"return_type":          mann.ReturnType(),
			"method_dep_macro":     depMacro,
			"has_request_overload": !isBidiStreaming(m),
		}

		if isBidiStreaming(m) {
			entry["is_bidi_streaming"] = true
			entry["bidir_comment"] = formatMethodComments(m, "", model, ann.IsDiscoveryDocumentProto)
			list = append(list, entry)
			continue
		}

		if isLongrunning(m) {
			entry["is_longrunning"] = true
			entry["longrunning_operation_type"] = mann.LongrunningOperationType()
			if isResponseTypeEmpty(m) {
				entry["is_response_type_empty"] = true
			} else {
				entry["longrunning_deduced_response_type"] = mann.LongrunningDeducedResponseType()
			}
			entry["start_comment"] = formatStartMethodComments(m.Name, mann.LongrunningOperationType(), isDep)
			entry["await_comment"] = formatAwaitMethodComments(m.Name, mann.LongrunningOperationType(), isDep)
		} else if isPaginated(m) {
			entry["is_paginated"] = true
			entry["range_output_type"] = mann.RangeOutputType()
		} else if isStreamingRead(m) {
			entry["is_streaming_read"] = true
		} else {
			entry["is_plain_unary"] = true
		}

		entry["req_comment"] = formatMethodCommentsProtobufRequest(m, model, ann.IsDiscoveryDocumentProto)

		var sigList []map[string]any
		for _, s := range mann.Signatures {
			sigComment := formatMethodCommentsMethodSignature(m, s.MethodSignature, model, ann.IsDiscoveryDocumentProto)
			sigEntry := map[string]any{
				"method_name":                       mann.MethodName(),
				"request_type":                      mann.RequestType(),
				"response_type":                     mann.ResponseType(),
				"return_type":                       mann.ReturnType(),
				"longrunning_operation_type":        mann.LongrunningOperationType(),
				"longrunning_deduced_response_type": mann.LongrunningDeducedResponseType(),
				"range_output_type":                 mann.RangeOutputType(),
				"method_dep_macro":                  depMacro,
				"sig_comment":                       sigComment,
				"current_method_signature":          s.SignatureString(),
				"current_request_setters":           s.RequestSetters(),
				"is_longrunning":                    isLongrunning(m),
				"is_response_type_empty":            isResponseTypeEmpty(m),
				"is_paginated":                      isPaginated(m),
				"is_streaming_read":                 isStreamingRead(m),
				"is_plain_unary":                    !isLongrunning(m) && !isPaginated(m) && !isStreamingRead(m),
				"has_iam_updater":                   hasIAM && setMethod == m,
				"service_name":                      ann.ServiceName,
			}
			if isLongrunning(m) {
				sigEntry["start_comment"] = formatStartMethodComments(m.Name, mann.LongrunningOperationType(), isDep)
			}
			sigList = append(sigList, sigEntry)
		}
		entry["signatures"] = sigList

		list = append(list, entry)
	}
	return list
}

func buildClientAsyncMethodList(ann *serviceAnnotations, asyncMethods []*api.Method, model *api.API) []map[string]any {
	var list []map[string]any
	for _, m := range asyncMethods {
		if isStreamingRead(m) || isStreamingWrite(m) {
			continue
		}
		mann := m.Codec.(*methodAnnotations)
		isDep := m.Deprecated
		depMacro := ""
		if isDep {
			depMacro = "  GOOGLE_CLOUD_CPP_DEPRECATED(\"This RPC is deprecated.\")\n"
		}
		entry := map[string]any{
			"method_name":      mann.MethodName(),
			"request_type":     mann.RequestType(),
			"return_type":      mann.ReturnType(),
			"method_dep_macro": depMacro,
			"req_comment":      formatMethodCommentsProtobufRequest(m, model, ann.IsDiscoveryDocumentProto),
		}

		var sigList []map[string]any
		for _, s := range mann.Signatures {
			sigComment := formatMethodCommentsMethodSignature(m, s.MethodSignature, model, ann.IsDiscoveryDocumentProto)
			sigList = append(sigList, map[string]any{
				"method_name":              mann.MethodName(),
				"request_type":             mann.RequestType(),
				"return_type":              mann.ReturnType(),
				"method_dep_macro":         depMacro,
				"sig_comment":              sigComment,
				"current_method_signature": s.SignatureString(),
				"current_request_setters":  s.RequestSetters(),
			})
		}
		entry["signatures"] = sigList

		list = append(list, entry)
	}
	return list
}

func generateClientHeader(svc *api.Service, ann *serviceAnnotations, methods, asyncMethods []*api.Method, _ *config.Library, model *api.API) (string, string) {
	headerPath := ann.ClientHeaderPath()
	guard := formatHeaderIncludeGuard(headerPath)

	hasIAM, _ := hasIamPolicyExtension(methods)

	connectionHeader := ann.ConnectionHeaderPath()
	if !ann.HasGrpcTransport && ann.HasRestTransport {
		connectionHeader = ann.ConnectionRestHeaderPath()
	}

	var localIncludes []string
	localIncludes = append(localIncludes,
		connectionHeader,
		"google/cloud/future.h",
		"google/cloud/options.h",
		"google/cloud/polling_policy.h",
		"google/cloud/status_or.h",
		"google/cloud/version.h",
	)
	if hasLongrunningMethod(methods) {
		localIncludes = append(localIncludes, "google/cloud/no_await_tag.h")
	}
	if hasIAM {
		localIncludes = append(localIncludes, "google/cloud/internal/make_status.h")
	}
	slices.Sort(localIncludes)
	if hasIAM {
		localIncludes = append(localIncludes, "google/cloud/iam_updater.h")
	}

	var protoIncludes []string
	protoIncludes = append(protoIncludes, methodSignatureWellKnownProtobufTypeIncludes(methods)...)
	if ann.HasGRPCLongrunningOperation() {
		protoIncludes = append(protoIncludes, "google/longrunning/operations.grpc.pb.h")
	}
	slices.Sort(protoIncludes)

	var sysIncludes []string
	if hasMessageWithMapField(methods, model) {
		sysIncludes = append(sysIncludes, "map")
	}
	sysIncludes = append(sysIncludes, "memory", "string")
	slices.Sort(sysIncludes)

	data := map[string]any{
		"header_include_guard":  guard,
		"copyright_year":        ann.CopyrightYear,
		"proto_file_name":       ann.ProtoFileName,
		"product_namespace":     ann.Namespace(),
		"connection_class_name": ann.ConnectionClassName(),
		"client_class_name":     ann.ClientClassName(),
		"service_name":          ann.ServiceName,
		"is_deprecated":         svc.Deprecated,
		"class_comment_block":   ann.ClassCommentBlock(),
		"local_includes":        localIncludes,
		"proto_includes":        protoIncludes,
		"system_includes":       sysIncludes,
		"methods":               buildClientMethodList(ann, methods, model),
		"async_methods":         buildClientAsyncMethodList(ann, asyncMethods, model),
	}

	content, err := renderTemplate("templates/client.h.mustache", data)
	if err != nil {
		panic(err)
	}

	return filepath.Clean(headerPath), content
}

func generateClientCc(svc *api.Service, ann *serviceAnnotations, methods, asyncMethods []*api.Method, lib *config.Library, model *api.API) (string, string) {
	ccPath := ann.ClientCcPath()

	hasDepWarnings := methodSignatureUsesDeprecatedField(svc, methods, lib)
	hasIAM, _ := hasIamPolicyExtension(methods)

	var includes []string
	if hasDepWarnings {
		includes = append(includes, `#include "google/cloud/internal/disable_deprecation_warnings.inc"`)
	}
	includes = append(includes, fmt.Sprintf(`#include "%s"`, ann.ClientHeaderPath()))
	includes = append(includes, `#include <memory>`)
	if hasIAM {
		includes = append(includes, fmt.Sprintf(`#include "%s"`, ann.OptionsHeaderPath()))
		includes = append(includes, `#include <thread>`)
	}
	includes = append(includes, `#include <utility>`)

	data := map[string]any{
		"copyright_year":        ann.CopyrightYear,
		"proto_file_name":       ann.ProtoFileName,
		"product_namespace":     ann.Namespace(),
		"connection_class_name": ann.ConnectionClassName(),
		"client_class_name":     ann.ClientClassName(),
		"service_name":          ann.ServiceName,
		"includes":              includes,
		"methods":               buildClientMethodList(ann, methods, model),
		"async_methods":         buildClientAsyncMethodList(ann, asyncMethods, model),
	}

	content, err := renderTemplate("templates/client.cc.mustache", data)
	if err != nil {
		panic(err)
	}

	return filepath.Clean(ccPath), content
}
