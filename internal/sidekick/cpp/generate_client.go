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
	"strconv"
	"strings"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/sidekick/api"
)

type signatureInfo struct {
	index int
	sig   *api.MethodSignature
}

func getValidSignatures(svc *api.Service, m *api.Method, lib *config.Library) []signatureInfo {
	omitted := make(map[string]bool)
	if lib != nil && lib.Cpp != nil {
		for _, o := range lib.Cpp.OmittedRPCs {
			omitted[o] = true
		}
	}

	var res []signatureInfo
	seenUIDs := make(map[string]bool)
	validIndex := 0
	for _, sig := range m.Signatures {
		var uidPieces []string
		for _, f := range sig.Fields {
			var cppType string
			if f.Map {
				keyType := "std::string"
				valType := "std::string"
				if f.MessageType != nil && len(f.MessageType.Fields) >= 2 {
					keyType = cppTypeToString(f.MessageType.Fields[0])
					valType = cppTypeToString(f.MessageType.Fields[1])
				}
				cppType = fmt.Sprintf("std::map<%s, %s> const&", keyType, valType)
			} else if f.Repeated {
				elemType := cppTypeToString(f)
				cppType = fmt.Sprintf("std::vector<%s> const&", elemType)
			} else if f.Typez == api.TypezMessage || f.Typez == api.TypezString || f.Typez == api.TypezBytes {
				cppType = cppTypeToString(f) + " const&"
			} else {
				cppType = cppTypeToString(f)
			}
			uidPieces = append(uidPieces, cppType)
		}
		uid := strings.Join(uidPieces, ", ") + ", "
		if seenUIDs[uid] {
			continue
		}
		seenUIDs[uid] = true

		sigName := fmt.Sprintf("%s(%s)", m.Name, strings.Join(uidPieces, ", "))
		qualifiedSigName := fmt.Sprintf("%s.%s", svc.Name, sigName)
		if omitted[sigName] || omitted[qualifiedSigName] {
			continue
		}

		res = append(res, signatureInfo{
			index: validIndex,
			sig:   sig,
		})
		validIndex++
	}
	return res
}

func methodSignatureUsesDeprecatedField(svc *api.Service, methods []*api.Method, lib *config.Library) bool {
	for _, m := range methods {
		sigs := getValidSignatures(svc, m, lib)
		for _, s := range sigs {
			for _, f := range s.sig.Fields {
				if f.Deprecated {
					return true
				}
			}
		}
	}
	return false
}

func buildClientMethodList(svc *api.Service, methods []*api.Method, serviceVars map[string]string, lib *config.Library, model *api.API) []map[string]any {
	var list []map[string]any
	hasIAM, setMethod := hasIamPolicyExtension(methods)

	for _, m := range methods {
		if isStreamingWrite(m) {
			continue
		}
		mVars := buildMethodVars(svc, m, serviceVars, lib, model)
		isDep := m.Deprecated
		depMacro := ""
		if isDep {
			depMacro = "  GOOGLE_CLOUD_CPP_DEPRECATED(\"This RPC is deprecated.\")\n"
		}

		entry := map[string]any{
			"method_name":          mVars["method_name"],
			"request_type":         mVars["request_type"],
			"response_type":        mVars["response_type"],
			"return_type":          mVars["return_type"],
			"method_dep_macro":     depMacro,
			"has_request_overload": !isBidiStreaming(m),
		}

		if isBidiStreaming(m) {
			entry["is_bidi_streaming"] = true
			entry["bidir_comment"] = formatMethodComments(m, "", model)
			list = append(list, entry)
			continue
		}

		if isLongrunning(m) {
			entry["is_longrunning"] = true
			entry["longrunning_operation_type"] = mVars["longrunning_operation_type"]
			if isResponseTypeEmpty(m) {
				entry["is_response_type_empty"] = true
			} else {
				entry["longrunning_deduced_response_type"] = mVars["longrunning_deduced_response_type"]
			}
			entry["start_comment"] = formatStartMethodComments(m.Name, mVars["longrunning_operation_type"], isDep)
			entry["await_comment"] = formatAwaitMethodComments(m.Name, mVars["longrunning_operation_type"], isDep)
		} else if isPaginated(m) {
			entry["is_paginated"] = true
			entry["range_output_type"] = mVars["range_output_type"]
		} else if isStreamingRead(m) {
			entry["is_streaming_read"] = true
		} else {
			entry["is_plain_unary"] = true
		}

		entry["req_comment"] = formatMethodCommentsProtobufRequest(m, model)

		sigs := getValidSignatures(svc, m, lib)
		var sigList []map[string]any
		for _, s := range sigs {
			idxStr := strconv.Itoa(s.index)
			sigComment := formatMethodCommentsMethodSignature(m, s.sig, model)
			sigEntry := map[string]any{
				"method_name":                       mVars["method_name"],
				"request_type":                      mVars["request_type"],
				"response_type":                     mVars["response_type"],
				"return_type":                       mVars["return_type"],
				"longrunning_operation_type":        mVars["longrunning_operation_type"],
				"longrunning_deduced_response_type": mVars["longrunning_deduced_response_type"],
				"range_output_type":                 mVars["range_output_type"],
				"method_dep_macro":                  depMacro,
				"sig_comment":                       sigComment,
				"current_method_signature":          mVars["method_signature"+idxStr],
				"current_request_setters":           mVars["method_request_setters"+idxStr],
				"is_longrunning":                    isLongrunning(m),
				"is_response_type_empty":            isResponseTypeEmpty(m),
				"is_paginated":                      isPaginated(m),
				"is_streaming_read":                 isStreamingRead(m),
				"is_plain_unary":                    !isLongrunning(m) && !isPaginated(m) && !isStreamingRead(m),
				"has_iam_updater":                   hasIAM && setMethod == m,
				"service_name":                      serviceVars["service_name"],
			}
			if isLongrunning(m) {
				sigEntry["start_comment"] = formatStartMethodComments(m.Name, mVars["longrunning_operation_type"], isDep)
			}
			sigList = append(sigList, sigEntry)
		}
		entry["signatures"] = sigList

		list = append(list, entry)
	}
	return list
}

func buildClientAsyncMethodList(svc *api.Service, asyncMethods []*api.Method, serviceVars map[string]string, lib *config.Library, model *api.API) []map[string]any {
	var list []map[string]any
	for _, m := range asyncMethods {
		if isStreamingRead(m) || isStreamingWrite(m) {
			continue
		}
		mVars := buildMethodVars(svc, m, serviceVars, lib, model)
		isDep := m.Deprecated
		depMacro := ""
		if isDep {
			depMacro = "  GOOGLE_CLOUD_CPP_DEPRECATED(\"This RPC is deprecated.\")\n"
		}
		entry := map[string]any{
			"method_name":      mVars["method_name"],
			"request_type":     mVars["request_type"],
			"return_type":      mVars["return_type"],
			"method_dep_macro": depMacro,
			"req_comment":      formatMethodCommentsProtobufRequest(m, model),
		}

		sigs := getValidSignatures(svc, m, lib)
		var sigList []map[string]any
		for _, s := range sigs {
			idxStr := strconv.Itoa(s.index)
			sigComment := formatMethodCommentsMethodSignature(m, s.sig, model)
			sigList = append(sigList, map[string]any{
				"method_name":              mVars["method_name"],
				"request_type":             mVars["request_type"],
				"return_type":              mVars["return_type"],
				"method_dep_macro":         depMacro,
				"sig_comment":              sigComment,
				"current_method_signature": mVars["method_signature"+idxStr],
				"current_request_setters":  mVars["method_request_setters"+idxStr],
			})
		}
		entry["signatures"] = sigList
		list = append(list, entry)
	}
	return list
}

func generateClientHeader(svc *api.Service, serviceVars map[string]string, methods, asyncMethods []*api.Method, lib *config.Library, model *api.API) (string, string) {
	headerPath := serviceVars["client_header_path"]
	guard := formatHeaderIncludeGuard(headerPath)

	hasIAM, _ := hasIamPolicyExtension(methods)

	hasGrpc := lib == nil || lib.Cpp == nil || lib.Cpp.HasGrpcTransport()
	connectionHeader := serviceVars["connection_header_path"]
	if !hasGrpc {
		connectionHeader = serviceVars["connection_rest_header_path"]
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
	if hasLongrunningMethod(methods) {
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
		"copyright_year":        serviceVars["copyright_year"],
		"proto_file_name":       serviceVars["proto_file_name"],
		"product_namespace":     serviceVars["product_namespace"],
		"connection_class_name": serviceVars["connection_class_name"],
		"client_class_name":     serviceVars["client_class_name"],
		"service_name":          serviceVars["service_name"],
		"is_deprecated":         svc.Deprecated,
		"class_comment_block":   serviceVars["class_comment_block"],
		"local_includes":        localIncludes,
		"proto_includes":        protoIncludes,
		"system_includes":       sysIncludes,
		"methods":               buildClientMethodList(svc, methods, serviceVars, lib, model),
		"async_methods":         buildClientAsyncMethodList(svc, asyncMethods, serviceVars, lib, model),
	}

	content, err := renderTemplate("templates/client.h.mustache", data)
	if err != nil {
		panic(err)
	}

	return filepath.Clean(headerPath), content
}

func generateClientCc(svc *api.Service, serviceVars map[string]string, methods, asyncMethods []*api.Method, lib *config.Library, model *api.API) (string, string) {
	ccPath := serviceVars["client_cc_path"]

	hasDepWarnings := methodSignatureUsesDeprecatedField(svc, methods, lib)
	hasIAM, _ := hasIamPolicyExtension(methods)

	var includes []string
	if hasDepWarnings {
		includes = append(includes, `#include "google/cloud/internal/disable_deprecation_warnings.inc"`)
	}
	includes = append(includes, fmt.Sprintf(`#include "%s"`, serviceVars["client_header_path"]))
	includes = append(includes, `#include <memory>`)
	if hasIAM {
		includes = append(includes, fmt.Sprintf(`#include "%s"`, serviceVars["options_header_path"]))
		includes = append(includes, `#include <thread>`)
	}
	includes = append(includes, `#include <utility>`)

	data := map[string]any{
		"copyright_year":        serviceVars["copyright_year"],
		"proto_file_name":       serviceVars["proto_file_name"],
		"product_namespace":     serviceVars["product_namespace"],
		"connection_class_name": serviceVars["connection_class_name"],
		"client_class_name":     serviceVars["client_class_name"],
		"service_name":          serviceVars["service_name"],
		"includes":              includes,
		"methods":               buildClientMethodList(svc, methods, serviceVars, lib, model),
		"async_methods":         buildClientAsyncMethodList(svc, asyncMethods, serviceVars, lib, model),
	}

	content, err := renderTemplate("templates/client.cc.mustache", data)
	if err != nil {
		panic(err)
	}

	return filepath.Clean(ccPath), content
}
