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

// MethodAnnotations contains C++-specific metadata for an RPC method.
type MethodAnnotations struct {
	// MethodName is the CamelCase name of the RPC method.
	MethodName string

	// MethodNameSnake is the snake_case name of the RPC method.
	MethodNameSnake string

	// RequestType is the fully qualified C++ type name for the request message.
	RequestType string

	// ResponseType is the fully qualified C++ type name for the response message.
	ResponseType string

	// ResponseMessageType is the un-namespaced protobuf message name of the response.
	ResponseMessageType string

	// ReturnType is the C++ return type for synchronous unary operations (e.g. StatusOr<T> or Status).
	ReturnType string

	// Idempotency is the idempotency classification (e.g. "kIdempotent", "kNonIdempotent").
	Idempotency string

	// Streaming and RPC characteristics.
	IsNonStreaming   bool
	IsStreamingRead  bool
	IsStreamingWrite bool
	IsBidiStreaming  bool
	IsPaginated      bool
	IsLongrunning    bool
	IsDeprecated     bool

	// HasRequestID indicates if the method uses request ID auto-population.
	HasRequestID bool

	// RequestIDFieldName is the name of the request ID field, if any.
	RequestIDFieldName string

	// Long-running operation metadata types.
	LongrunningOperationType              string
	LongrunningMetadataType               string
	LongrunningResponseType               string
	LongrunningDeducedResponseType        string
	LongrunningDeducedResponseMessageType string

	// Pagination metadata.
	RangeOutputFieldName string
	RangeOutputType      string

	// Signatures contains the method signature overload descriptors.
	Signatures []*SignatureAnnotations

	// Comments contains formatted Doxygen comments for the method and its overloads.
	Comments *CommentAnnotations
}

// SignatureAnnotations describes a single method signature overload.
type SignatureAnnotations struct {
	// MethodSignature is the underlying API method signature model.
	MethodSignature *api.MethodSignature

	// Params is the list of C++ parameter declarations (e.g. "std::string const& parent").
	Params []string

	// ParamNames is the list of parameter variable names.
	ParamNames []string

	// ParamTypes is the list of parameter C++ types.
	ParamTypes []string

	// Setters contains C++ statements copying signature parameters into the request object.
	Setters string
}

// annotateMethod enriches an api.Method with C++-specific metadata and types.
func annotateMethod(m *api.Method, svc *api.Service, lib *config.Library, model *api.API) *MethodAnnotations {
	if m == nil {
		return nil
	}

	methodName := m.Name
	ann := &MethodAnnotations{
		MethodName:          methodName,
		MethodNameSnake:     camelCaseToSnakeCase(methodName),
		RequestType:         protoNameToCppName(m.InputTypeID),
		ResponseType:        protoNameToCppName(m.OutputTypeID),
		ResponseMessageType: strings.TrimPrefix(m.OutputTypeID, "."),
		IsNonStreaming:      isNonStreaming(m),
		IsStreamingRead:     isStreamingRead(m),
		IsStreamingWrite:    isStreamingWrite(m),
		IsBidiStreaming:     isBidiStreaming(m),
		IsPaginated:         isPaginated(m),
		IsLongrunning:       isLongrunning(m),
		IsDeprecated:        isDeprecated(m),
		Idempotency:         defaultIdempotency(m, svc.Name, lib),
	}

	if isResponseTypeEmpty(m) {
		ann.ReturnType = "Status"
	} else {
		ann.ReturnType = "StatusOr<" + ann.ResponseType + ">"
	}

	if hasRequestID(m) {
		ann.HasRequestID = true
		ann.RequestIDFieldName = m.AutoPopulated[0].Name
	}

	if isLongrunning(m) {
		ann.LongrunningOperationType = protoNameToCppName(m.OutputTypeID)
		if m.OperationInfo != nil {
			ann.LongrunningMetadataType = protoNameToCppName(m.OperationInfo.MetadataTypeID)
			ann.LongrunningResponseType = protoNameToCppName(m.OperationInfo.ResponseTypeID)

			deduced := m.OperationInfo.ResponseTypeID
			if deduced == ".google.protobuf.Empty" || deduced == "google.protobuf.Empty" {
				deduced = m.OperationInfo.MetadataTypeID
			}
			ann.LongrunningDeducedResponseType = protoNameToCppName(deduced)
			ann.LongrunningDeducedResponseMessageType = strings.TrimPrefix(deduced, ".")
		}
	}

	if isPaginated(m) && model != nil {
		respMsg := model.Message(m.OutputTypeID)
		if respMsg != nil && respMsg.Pagination != nil && respMsg.Pagination.PageableItem != nil {
			item := respMsg.Pagination.PageableItem
			ann.RangeOutputFieldName = item.Name
			switch item.Typez {
			case api.TypezMessage:
				ann.RangeOutputType = protoNameToCppName(item.TypezID)
			case api.TypezString:
				ann.RangeOutputType = "std::string"
			}
		}
	}

	// Method signatures
	seenUIDs := make(map[string]bool)
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

		uid := strings.Join(uidPieces, ", ")
		if seenUIDs[uid] {
			continue
		}
		seenUIDs[uid] = true

		ann.Signatures = append(ann.Signatures, &SignatureAnnotations{
			MethodSignature: sig,
			Params:          sigPieces,
			ParamNames:      paramNames,
			ParamTypes:      paramTypes,
			Setters:         setters.String(),
		})
	}

	ann.Comments = annotateMethodComments(m, model)

	m.Codec = ann
	return ann
}
