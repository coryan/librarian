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
	"maps"
	"slices"
	"strconv"
	"strings"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/sidekick/api"
)

func isNonStreaming(m *api.Method) bool {
	return !m.ClientSideStreaming && !m.ServerSideStreaming
}

func isStreamingRead(m *api.Method) bool {
	return m.ServerSideStreaming && !m.ClientSideStreaming
}

func isStreamingWrite(m *api.Method) bool {
	return m.ClientSideStreaming && !m.ServerSideStreaming
}

func isBidiStreaming(m *api.Method) bool {
	return m.ClientSideStreaming && m.ServerSideStreaming
}

func isStreaming(m *api.Method) bool {
	return m.ClientSideStreaming || m.ServerSideStreaming
}

func isPaginated(m *api.Method) bool {
	return m.Pagination != nil
}

func isLongrunning(m *api.Method) bool {
	return m.IsLRO || m.OperationInfo != nil
}

func hasRequestID(m *api.Method) bool {
	return len(m.AutoPopulated) > 0
}

func isDeprecated(m *api.Method) bool {
	return m.Deprecated
}

func isResponseTypeEmpty(m *api.Method) bool {
	return m.ReturnsEmpty
}

func hasLongrunningMethod(methods []*api.Method) bool {
	return slices.ContainsFunc(methods, isLongrunning)
}

func hasPaginatedMethod(methods []*api.Method) bool {
	return slices.ContainsFunc(methods, isPaginated)
}

func hasStreamingReadMethod(methods []*api.Method) bool {
	return slices.ContainsFunc(methods, isStreamingRead)
}

func hasStreamingWriteMethod(methods []*api.Method) bool {
	return slices.ContainsFunc(methods, isStreamingWrite)
}

func hasBidiStreamingMethod(methods []*api.Method) bool {
	return slices.ContainsFunc(methods, isBidiStreaming)
}

func hasAsyncMethod(asyncMethods []*api.Method) bool {
	return len(asyncMethods) > 0
}

func hasAsynchronousStreamingReadMethod(asyncMethods []*api.Method) bool {
	return slices.ContainsFunc(asyncMethods, isStreamingRead)
}

func hasAsynchronousStreamingWriteMethod(asyncMethods []*api.Method) bool {
	return slices.ContainsFunc(asyncMethods, isStreamingWrite)
}

func hasMessageWithMapField(methods []*api.Method, model *api.API) bool {
	for _, m := range methods {
		req := model.Message(m.InputTypeID)
		if req != nil {
			for _, f := range req.Fields {
				if f.Map {
					return true
				}
			}
		}
		resp := model.Message(m.OutputTypeID)
		if resp != nil {
			for _, f := range resp.Fields {
				if f.Map {
					return true
				}
			}
		}
	}
	return false
}

func hasIamPolicyExtension(methods []*api.Method) (bool, *api.Method) {
	var hasGet bool
	var setMethod *api.Method
	for _, m := range methods {
		if m.OutputTypeID == ".google.iam.v1.Policy" {
			if m.InputTypeID == ".google.iam.v1.GetIamPolicyRequest" {
				for _, sig := range m.Signatures {
					if len(sig.Names) == 1 && sig.Names[0] == "resource" {
						hasGet = true
					}
				}
			}
			if m.InputTypeID == ".google.iam.v1.SetIamPolicyRequest" {
				for _, sig := range m.Signatures {
					if len(sig.Names) == 2 && sig.Names[0] == "resource" && sig.Names[1] == "policy" {
						setMethod = m
					}
				}
			}
		}
	}
	return hasGet && setMethod != nil, setMethod
}

func getEffectiveMethods(svc *api.Service, lib *config.Library) (methods []*api.Method, asyncMethods []*api.Method) {
	omitted := make(map[string]bool)
	if lib != nil && lib.Cpp != nil {
		for _, o := range lib.Cpp.OmittedRPCs {
			omitted[o] = true
		}
	}
	genAsync := make(map[string]bool)
	if lib != nil && lib.Cpp != nil {
		for _, a := range lib.Cpp.GenAsyncRPCs {
			genAsync[a] = true
		}
	}

	for _, m := range svc.Methods {
		if m.IsLroPoller {
			continue
		}
		methodName := m.Name
		qualifiedName := svc.Name + "." + methodName

		isOmitted := omitted[methodName] || omitted[qualifiedName]
		if !isOmitted {
			methods = append(methods, m)
		}
		if genAsync[methodName] || genAsync[qualifiedName] {
			asyncMethods = append(asyncMethods, m)
		}
	}
	return methods, asyncMethods
}

func formatFieldAccessor(fieldPath []string) string {
	if len(fieldPath) == 0 {
		return ""
	}
	var b strings.Builder
	for i, f := range fieldPath {
		if i > 0 {
			b.WriteString("().")
		}
		b.WriteString(f)
	}
	return b.String()
}

func defaultIdempotency(m *api.Method, svcName string, lib *config.Library) string {
	if lib != nil && lib.Cpp != nil {
		for _, override := range lib.Cpp.IdempotencyOverrides {
			if override.RPCName == svcName+"."+m.Name || override.RPCName == m.Name {
				if override.Idempotency == "IDEMPOTENT" {
					return "kIdempotent"
				}
				if override.Idempotency == "NON_IDEMPOTENT" {
					return "kNonIdempotent"
				}
			}
		}
	}
	// Known idempotent methods
	if (m.Name == "GetIamPolicy" && m.OutputTypeID == ".google.iam.v1.Policy" && m.InputTypeID == ".google.iam.v1.GetIamPolicyRequest") ||
		(m.Name == "TestIamPermissions" && m.OutputTypeID == ".google.iam.v1.TestIamPermissionsResponse" && m.InputTypeID == ".google.iam.v1.TestIamPermissionsRequest") {
		return "kIdempotent"
	}
	if m.PathInfo != nil && len(m.PathInfo.Bindings) > 0 {
		verb := m.PathInfo.Bindings[0].Verb
		switch verb {
		case "GET", "PUT":
			return "kIdempotent"
		}
	}
	return "kNonIdempotent"
}

func buildMethodVars(svc *api.Service, m *api.Method, serviceVars map[string]string, lib *config.Library, model *api.API) map[string]string {
	vars := make(map[string]string)
	maps.Copy(vars, serviceVars)

	methodName := m.Name
	vars["method_name"] = methodName
	vars["method_name_snake"] = camelCaseToSnakeCase(methodName)
	vars["request_type"] = protoNameToCppName(m.InputTypeID)
	vars["response_type"] = protoNameToCppName(m.OutputTypeID)
	vars["response_message_type"] = strings.TrimPrefix(m.OutputTypeID, ".")

	if isResponseTypeEmpty(m) {
		vars["return_type"] = "Status"
	} else {
		vars["return_type"] = "StatusOr<" + vars["response_type"] + ">"
	}

	if m.SourceService != nil && m.SourceService.ID != svc.ID {
		vars["grpc_stub"] = strings.ToLower(m.SourceService.Name) + "_stub_"
	} else {
		vars["grpc_stub"] = "grpc_stub_"
	}

	vars["idempotency"] = defaultIdempotency(m, svc.Name, lib)

	if len(m.AutoPopulated) > 0 {
		vars["request_id_field_name"] = m.AutoPopulated[0].Name
	}

	if isLongrunning(m) {
		vars["longrunning_operation_type"] = protoNameToCppName(m.OutputTypeID)
		if m.OperationInfo != nil {
			vars["longrunning_metadata_type"] = protoNameToCppName(m.OperationInfo.MetadataTypeID)
			vars["longrunning_response_type"] = protoNameToCppName(m.OperationInfo.ResponseTypeID)

			deduced := m.OperationInfo.ResponseTypeID
			if deduced == ".google.protobuf.Empty" || deduced == "google.protobuf.Empty" {
				deduced = m.OperationInfo.MetadataTypeID
			}
			vars["longrunning_deduced_response_type"] = protoNameToCppName(deduced)
			vars["longrunning_deduced_response_message_type"] = strings.TrimPrefix(deduced, ".")
		}
	}

	if isPaginated(m) {
		respMsg := model.Message(m.OutputTypeID)
		if respMsg != nil && respMsg.Pagination != nil && respMsg.Pagination.PageableItem != nil {
			item := respMsg.Pagination.PageableItem
			vars["range_output_field_name"] = item.Name
			switch item.Typez {
			case api.TypezMessage:
				vars["range_output_type"] = protoNameToCppName(item.TypezID)
			case api.TypezString:
				vars["range_output_type"] = "std::string"
			}
		}
	}

	// Signatures
	omitted := make(map[string]bool)
	if lib != nil && lib.Cpp != nil {
		for _, o := range lib.Cpp.OmittedRPCs {
			omitted[o] = true
		}
	}

	seenUIDs := make(map[string]bool)
	validSigIndex := 0
	for _, sig := range m.Signatures {
		var sigPieces []string
		var uidPieces []string
		var setters strings.Builder
		for _, f := range sig.Fields {
			paramName := f.Name
			var cppType string
			if f.Map {
				// Map type
				keyType := "std::string"
				valType := "std::string"
				if f.MessageType != nil && len(f.MessageType.Fields) >= 2 {
					keyType = cppTypeToString(f.MessageType.Fields[0])
					valType = cppTypeToString(f.MessageType.Fields[1])
				}
				cppType = fmt.Sprintf("std::map<%s, %s> const&", keyType, valType)
				fmt.Fprintf(&setters, "  *request.mutable_%s() = {%s.begin(), %s.end()};\n", paramName, paramName, paramName)
			} else if f.Repeated {
				elemType := cppTypeToString(f)
				cppType = fmt.Sprintf("std::vector<%s> const&", elemType)
				fmt.Fprintf(&setters, "  *request.mutable_%s() = {%s.begin(), %s.end()};\n", paramName, paramName, paramName)
			} else if f.Typez == api.TypezMessage {
				cppType = cppTypeToString(f) + " const&"
				fmt.Fprintf(&setters, "  *request.mutable_%s() = %s;\n", paramName, paramName)
			} else if f.Typez == api.TypezString || f.Typez == api.TypezBytes {
				cppType = cppTypeToString(f) + " const&"
				fmt.Fprintf(&setters, "  request.set_%s(%s);\n", paramName, paramName)
			} else {
				cppType = cppTypeToString(f)
				fmt.Fprintf(&setters, "  request.set_%s(%s);\n", paramName, paramName)
			}
			sigPieces = append(sigPieces, fmt.Sprintf("%s %s", cppType, paramName))
			uidPieces = append(uidPieces, cppType)
		}
		uid := strings.Join(uidPieces, ", ") + ", "
		if seenUIDs[uid] {
			continue
		}
		seenUIDs[uid] = true

		sigName := fmt.Sprintf("%s(%s)", methodName, strings.Join(uidPieces, ", "))
		qualifiedSigName := fmt.Sprintf("%s.%s", svc.Name, sigName)
		if omitted[sigName] || omitted[qualifiedSigName] {
			continue
		}

		key := "method_signature" + strconv.Itoa(validSigIndex)
		key2 := "method_request_setters" + strconv.Itoa(validSigIndex)
		sigStr := ""
		if len(sigPieces) > 0 {
			sigStr = strings.Join(sigPieces, ", ") + ", "
		}
		vars[key] = sigStr
		vars[key2] = setters.String()
		validSigIndex++
	}
	vars["method_signature_count"] = strconv.Itoa(validSigIndex)

	// Routing / Request params
	if len(m.Routing) == 0 && m.PathInfo != nil && len(m.PathInfo.Bindings) > 0 {
		template := m.PathInfo.Bindings[0].PathTemplate
		if template != nil {
			var params []string
			for _, seg := range template.Segments {
				if seg.Variable != nil && len(seg.Variable.FieldPath) > 0 {
					fieldName := strings.Join(seg.Variable.FieldPath, ".")
					accessor := formatFieldAccessor(seg.Variable.FieldPath)
					params = append(params, fmt.Sprintf(`"%s=", internal::UrlEncode(request.%s())`, fieldName, accessor))
				}
			}
			if len(params) > 0 {
				vars["method_request_params"] = strings.Join(params, `, "&", `)
			}
		}
	}

	return vars
}

func methodSignatureWellKnownProtobufTypeIncludes(methods []*api.Method) []string {
	var includes []string
	seen := make(map[string]bool)
	for _, m := range methods {
		for _, sig := range m.Signatures {
			for _, f := range sig.Fields {
				if f.TypezID == ".google.protobuf.Duration" {
					inc := "google/protobuf/duration.pb.h"
					if !seen[inc] {
						seen[inc] = true
						includes = append(includes, inc)
					}
				}
			}
		}
	}
	slices.Sort(includes)
	return includes
}
