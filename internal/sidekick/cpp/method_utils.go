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
	"slices"
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

func hasExplicitRoutingMethod(methods []*api.Method) bool {
	for _, m := range methods {
		if len(m.Routing) > 0 {
			return true
		}
	}
	return false
}

