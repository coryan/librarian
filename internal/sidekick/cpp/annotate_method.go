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
	"strings"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/sidekick/api"
)

// methodAnnotations contains C++-specific metadata for an RPC method.
// Annotations are private to the package to encapsulate implementation details and enforce
// accessor method usage for derived properties.
type methodAnnotations struct {
	Method          *api.Method
	Service         *api.Service
	Model           *api.API
	Idempotency     string
	Signatures      []*signatureAnnotations
	Comments        *commentAnnotations
	HTTPVerb        string
	RequestResource string
	RestPath        string
	RestPathAsync   string
	HTTPQueryParams string
}

func (m *methodAnnotations) MethodName() string {
	if m == nil || m.Method == nil {
		return ""
	}
	return m.Method.Name
}

func (m *methodAnnotations) MethodNameSnake() string {
	return camelCaseToSnakeCase(m.MethodName())
}

func (m *methodAnnotations) RequestType() string {
	if m == nil || m.Method == nil {
		return ""
	}
	return protoNameToCppName(m.Method.InputTypeID)
}

func (m *methodAnnotations) ResponseType() string {
	if m == nil || m.Method == nil {
		return ""
	}
	return protoNameToCppName(m.Method.OutputTypeID)
}

func (m *methodAnnotations) ResponseMessageType() string {
	if m == nil || m.Method == nil {
		return ""
	}
	return strings.TrimPrefix(m.Method.OutputTypeID, ".")
}

func (m *methodAnnotations) ReturnType() string {
	if m == nil || m.Method == nil {
		return ""
	}
	if isResponseTypeEmpty(m.Method) {
		return "Status"
	}
	return "StatusOr<" + m.ResponseType() + ">"
}

func (m *methodAnnotations) IsNonStreaming() bool {
	return m != nil && m.Method != nil && isNonStreaming(m.Method)
}

func (m *methodAnnotations) IsStreamingRead() bool {
	return m != nil && m.Method != nil && isStreamingRead(m.Method)
}

func (m *methodAnnotations) IsStreamingWrite() bool {
	return m != nil && m.Method != nil && isStreamingWrite(m.Method)
}

func (m *methodAnnotations) IsBidiStreaming() bool {
	return m != nil && m.Method != nil && isBidiStreaming(m.Method)
}

func (m *methodAnnotations) IsPaginated() bool {
	return m != nil && m.Method != nil && isPaginated(m.Method)
}

func (m *methodAnnotations) IsLongrunning() bool {
	return m != nil && m.Method != nil && isLongrunning(m.Method)
}

func (m *methodAnnotations) IsDeprecated() bool {
	return m != nil && m.Method != nil && isDeprecated(m.Method)
}

func (m *methodAnnotations) HasRequestID() bool {
	return m != nil && m.Method != nil && hasRequestID(m.Method)
}

func (m *methodAnnotations) RequestIDFieldName() string {
	if m != nil && m.Method != nil && len(m.Method.AutoPopulated) > 0 {
		return m.Method.AutoPopulated[0].Name
	}
	return ""
}

func (m *methodAnnotations) LongrunningOperationType() string {
	if m == nil || m.Method == nil {
		return ""
	}
	return protoNameToCppName(m.Method.OutputTypeID)
}

func (m *methodAnnotations) LongrunningMetadataType() string {
	if m != nil && m.Method != nil && m.Method.OperationInfo != nil {
		return protoNameToCppName(m.Method.OperationInfo.MetadataTypeID)
	}
	return ""
}

func (m *methodAnnotations) LongrunningResponseType() string {
	if m != nil && m.Method != nil && m.Method.OperationInfo != nil {
		return protoNameToCppName(m.Method.OperationInfo.ResponseTypeID)
	}
	return ""
}

func (m *methodAnnotations) LongrunningDeducedResponseType() string {
	if m == nil || m.Method == nil {
		return ""
	}
	if m.Method.OperationService != "" && m.Method.OutputTypeID != "" {
		return protoNameToCppName(m.Method.OutputTypeID)
	}
	if m.Method.OperationInfo == nil {
		return ""
	}
	deduced := m.Method.OperationInfo.ResponseTypeID
	if deduced == ".google.protobuf.Empty" || deduced == "google.protobuf.Empty" {
		deduced = m.Method.OperationInfo.MetadataTypeID
	}
	return protoNameToCppName(deduced)
}

func (m *methodAnnotations) LongrunningDeducedResponseMessageType() string {
	if m == nil || m.Method == nil {
		return ""
	}
	if m.Method.OperationService != "" && m.Method.OutputTypeID != "" {
		return strings.TrimPrefix(m.Method.OutputTypeID, ".")
	}
	if m.Method.OperationInfo == nil {
		return ""
	}
	deduced := m.Method.OperationInfo.ResponseTypeID
	if deduced == ".google.protobuf.Empty" || deduced == "google.protobuf.Empty" {
		deduced = m.Method.OperationInfo.MetadataTypeID
	}
	return strings.TrimPrefix(deduced, ".")
}

func (m *methodAnnotations) IsGRPCLongrunningOperation() bool {
	return m != nil && m.Method != nil && isLongrunning(m.Method) && m.Method.OperationService == ""
}

func (m *methodAnnotations) IsHTTPLongrunningOperation() bool {
	return m != nil && m.Method != nil && isLongrunning(m.Method) && m.Method.OperationService != ""
}

func (m *methodAnnotations) LongrunningSetOperationFields() string {
	if m != nil && m.Method != nil && m.Method.OperationService != "" {
		return getComputeOperationInfo(m.Method.OperationService).SetOperationFields
	}
	return ""
}

func (m *methodAnnotations) LongrunningAwaitSetOperationFields() string {
	if m != nil && m.Method != nil && m.Method.OperationService != "" {
		return getComputeOperationInfo(m.Method.OperationService).AwaitSetOperationFields
	}
	return ""
}

func (m *methodAnnotations) pageableItem() *api.Field {
	if m == nil || m.Method == nil {
		return nil
	}
	if m.Method.OutputType != nil && m.Method.OutputType.Pagination != nil {
		return m.Method.OutputType.Pagination.PageableItem
	}
	if m.Model != nil {
		if respMsg := m.Model.Message(m.Method.OutputTypeID); respMsg != nil && respMsg.Pagination != nil {
			return respMsg.Pagination.PageableItem
		}
	}
	return nil
}

func (m *methodAnnotations) RangeOutputFieldName() string {
	if item := m.pageableItem(); item != nil {
		return item.Name
	}
	return ""
}

func (m *methodAnnotations) RangeOutputType() string {
	if item := m.pageableItem(); item != nil {
		if item.Map && m.Model != nil {
			if entry := m.Model.Message(item.TypezID); entry != nil {
				var keyType, valType string
				for _, f := range entry.Fields {
					switch f.Name {
					case "key":
						if f.Typez == api.TypezString {
							keyType = "std::string"
						} else {
							keyType = protoNameToCppName(f.TypezID)
						}
					case "value":
						if f.Typez == api.TypezString {
							valType = "std::string"
						} else {
							valType = protoNameToCppName(f.TypezID)
						}
					}
				}
				if keyType != "" && valType != "" {
					return fmt.Sprintf("std::pair<%s, %s>", keyType, valType)
				}
			}
		}
		switch item.Typez {
		case api.TypezMessage:
			return protoNameToCppName(item.TypezID)
		case api.TypezString:
			return "std::string"
		}
	}
	return ""
}

func (m *methodAnnotations) GrpcStub() string {
	if m != nil && m.Method != nil && m.Service != nil && m.Method.SourceService != nil && m.Method.SourceService.ID != m.Service.ID {
		return strings.ToLower(m.Method.SourceService.Name) + "_stub_"
	}
	return "grpc_stub_"
}

func (m *methodAnnotations) MethodRequestParams() string {
	if m == nil || m.Method == nil {
		return ""
	}
	if len(m.Method.Routing) == 0 && m.Method.PathInfo != nil && len(m.Method.PathInfo.Bindings) > 0 {
		template := m.Method.PathInfo.Bindings[0].PathTemplate
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
				return strings.Join(params, `, "&", `)
			}
		}
	}
	return ""
}

func (m *methodAnnotations) IsRestMethod() bool {
	return m != nil && m.Method != nil && isRestMethod(m.Method)
}

// signatureAnnotations describes a single method signature overload.
// Annotations are private to the package to encapsulate implementation details and enforce
// accessor method usage for derived properties.
type signatureAnnotations struct {
	MethodSignature *api.MethodSignature
	Params          []string
	ParamNames      []string
	ParamTypes      []string
	Setters         string
	Index           int
}

func (s *signatureAnnotations) SignatureString() string {
	if s == nil || len(s.Params) == 0 {
		return ""
	}
	return strings.Join(s.Params, ", ") + ", "
}

func (s *signatureAnnotations) RequestSetters() string {
	if s == nil {
		return ""
	}
	return s.Setters
}

// annotateMethod enriches an api.Method with C++-specific metadata and types.
func annotateMethod(m *api.Method, svc *api.Service, lib *config.Library, model *api.API) *methodAnnotations {
	if m == nil {
		return nil
	}

	ann := &methodAnnotations{
		Method:      m,
		Service:     svc,
		Model:       model,
		Idempotency: defaultIdempotency(m, svc.Name, lib),
	}

	if isRestMethod(m) {
		if len(m.PathInfo.Bindings) > 0 {
			ann.HTTPVerb = httpVerb(m.PathInfo.Bindings[0].Verb)
			ann.RequestResource = formatRequestResource(m)
			ann.RestPath = formatRestPath(m, false)
			ann.RestPathAsync = formatRestPath(m, true)
			ann.HTTPQueryParams = formatHTTPQueryParameters(m, model)
		}
	}

	// Method signatures
	omitted := make(map[string]bool)
	if lib != nil && lib.Cpp != nil {
		for _, o := range lib.Cpp.OmittedRPCs {
			omitted[o] = true
		}
	}

	seenUIDs := make(map[string]bool)
	validIndex := 0
	for _, sig := range m.Signatures {
		var sigPieces []string
		var uidPieces []string
		var paramNames []string
		var paramTypes []string
		var setters strings.Builder

		for _, f := range sig.Fields {
			paramName := f.Name
			paramNames = append(paramNames, paramName)

			var cppType string
			if f.Map {
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
			} else if f.Typez == api.TypezString || f.Typez == api.TypezBytes {
				cppType = "std::string const&"
				fmt.Fprintf(&setters, "  request.set_%s(%s);\n", paramName, paramName)
			} else if f.Typez == api.TypezMessage {
				cppType = protoNameToCppName(f.TypezID) + " const&"
				fmt.Fprintf(&setters, "  *request.mutable_%s() = %s;\n", paramName, paramName)
			} else {
				cppType = cppTypeToString(f)
				fmt.Fprintf(&setters, "  request.set_%s(%s);\n", paramName, paramName)
			}
			paramTypes = append(paramTypes, cppType)
			sigPieces = append(sigPieces, fmt.Sprintf("%s %s", cppType, paramName))
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

		ann.Signatures = append(ann.Signatures, &signatureAnnotations{
			MethodSignature: sig,
			Params:          sigPieces,
			ParamNames:      paramNames,
			ParamTypes:      paramTypes,
			Setters:         setters.String(),
			Index:           validIndex,
		})
		validIndex++
	}

	isDiscovery := (lib != nil && ((lib.Cpp != nil && lib.Cpp.IsDiscoveryDocumentProto) || lib.SpecificationFormat == "discovery")) || m.OperationService != ""
	ann.Comments = annotateMethodComments(m, model, isDiscovery)

	m.Codec = ann
	return ann
}
