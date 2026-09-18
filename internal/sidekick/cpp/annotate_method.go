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
	"regexp"
	"slices"
	"strings"

	"github.com/googleapis/librarian/internal/sidekick/api"
)

type methodAnnotations struct {
	Name                      string
	RequestType               string
	ResponseType              string
	Service                   *serviceAnnotations
	Method                    *api.Method
	IsPaginated               bool
	IsLRO                     bool
	Signatures                []*signatureAnnotations
	Idempotency               string
	IsIamSetMethodWithUpdater bool
}

type signatureAnnotations struct {
	Params       []*signatureParamAnnotations
	Method       *methodAnnotations
	RawSignature *api.MethodSignature
}

type signatureParamAnnotations struct {
	Name    string
	CppType string
	Field   *api.Field
}

func (ann *methodAnnotations) IsDeprecated() bool {
	return ann.Method != nil && ann.Method.Deprecated
}

func (ann *methodAnnotations) IsStreamingRead() bool {
	return ann.Method != nil && ann.Method.ServerSideStreaming && !ann.Method.ClientSideStreaming
}

func (ann *methodAnnotations) IsStreamingWrite() bool {
	return ann.Method != nil && ann.Method.ClientSideStreaming && !ann.Method.ServerSideStreaming
}

func (ann *methodAnnotations) IsBidiStreaming() bool {
	return ann.Method != nil && ann.Method.ClientSideStreaming && ann.Method.ServerSideStreaming
}

func (ann *methodAnnotations) DeprecationMacro() string {
	if ann.IsDeprecated() {
		return "  GOOGLE_CLOUD_CPP_DEPRECATED(\"This RPC is deprecated.\")\n"
	}
	return ""
}

func (sig *signatureAnnotations) DeprecationMacro() string {
	if sig.Method != nil && sig.Method.IsDeprecated() {
		return "  GOOGLE_CLOUD_CPP_DEPRECATED(\"This RPC is deprecated.\")\n"
	}
	return ""
}

func (ann *methodAnnotations) IsVoid() bool {
	return ann.Method.ReturnsEmpty || ann.Method.OutputTypeID == "google.protobuf.Empty" || ann.Method.OutputTypeID == ".google.protobuf.Empty"
}

func (ann *methodAnnotations) IsPlainUnary() bool {
	return !ann.IsStreamingRead() && !ann.IsStreamingWrite() && !ann.IsBidiStreaming() && !ann.IsPaginated && !ann.IsLRO
}

func (ann *methodAnnotations) IsStreaming() bool {
	return ann.IsStreamingRead() || ann.IsStreamingWrite() || ann.IsBidiStreaming()
}

func (ann *methodAnnotations) PlainReturnType() string {
	if ann.IsVoid() {
		return "Status"
	}
	return "StatusOr<" + ann.ResponseType + ">"
}

func (ann *methodAnnotations) IsLroVoid() bool {
	if !ann.IsLRO || ann.Method.OperationInfo == nil {
		return false
	}
	res := ann.Method.OperationInfo.ResponseTypeID
	meta := ann.Method.OperationInfo.MetadataTypeID
	resEmpty := res == "" || res == "google.protobuf.Empty" || res == ".google.protobuf.Empty"
	metaEmpty := meta == "" || meta == "google.protobuf.Empty" || meta == ".google.protobuf.Empty"
	return resEmpty && metaEmpty
}

func (ann *methodAnnotations) DeducedResponseType() string {
	if !ann.IsLRO || ann.Method.OperationInfo == nil {
		return ""
	}
	res := ann.Method.OperationInfo.ResponseTypeID
	if res == "" || res == "google.protobuf.Empty" || res == ".google.protobuf.Empty" {
		res = ann.Method.OperationInfo.MetadataTypeID
	}
	return protoNameToCppName(res)
}

func (ann *methodAnnotations) RangeOutputType() string {
	if !ann.IsPaginated {
		return ""
	}
	if ann.Method.OutputType != nil && ann.Method.OutputType.Pagination != nil && ann.Method.OutputType.Pagination.PageableItem != nil {
		pi := ann.Method.OutputType.Pagination.PageableItem
		if pi.Typez == api.TypezMessage {
			return protoNameToCppName(pi.TypezID)
		}
		if pi.Typez == api.TypezString {
			return "std::string"
		}
		if pi.Map && pi.MessageType != nil && len(pi.MessageType.Fields) >= 2 {
			k := cppFieldScalarType(pi.MessageType.Fields[0])
			v := cppFieldScalarType(pi.MessageType.Fields[1])
			return fmt.Sprintf("std::pair<%s, %s>", k, v)
		}
	}
	return "std::string"
}

func (ann *methodAnnotations) RangeOutputFieldName() string {
	if ann.Method != nil && ann.Method.OutputType != nil {
		if ann.Method.OutputType.Pagination != nil && ann.Method.OutputType.Pagination.PageableItem != nil {
			return cppFieldName(ann.Method.OutputType.Pagination.PageableItem.Name)
		}
		for _, f := range ann.Method.OutputType.Fields {
			if f.Repeated {
				return cppFieldName(f.Name)
			}
		}
	}
	return ""
}

func (ann *methodAnnotations) HasRequestId() bool {
	return ann.Method != nil && len(ann.Method.AutoPopulated) > 0
}

func (ann *methodAnnotations) RequestIdFieldName() string {
	if !ann.HasRequestId() {
		return ""
	}
	return cppFieldName(ann.Method.AutoPopulated[0].Name)
}

func (ann *methodAnnotations) IsSetIamPolicy() bool {
	if ann.Method == nil || ann.Method.Name != "SetIamPolicy" {
		return false
	}
	in := strings.TrimPrefix(ann.Method.InputTypeID, ".")
	out := strings.TrimPrefix(ann.Method.OutputTypeID, ".")
	return in == "google.iam.v1.SetIamPolicyRequest" && out == "google.iam.v1.Policy"
}

func (ann *methodAnnotations) GrpcStub() string {
	if ann.Method != nil && ann.Method.SourceService != nil && ann.Method.Service != nil && ann.Method.SourceService.Name != ann.Method.Service.Name {
		switch ann.Method.SourceService.Name {
		case "Locations":
			return "locations_stub_"
		case "IAMPolicy":
			return "iampolicy_stub_"
		case "Operations":
			return "operations_stub_"
		}
	}
	return "grpc_stub_"
}

func (ann *methodAnnotations) IsLongrunningMetadataTypeUsedAsResponse() bool {
	if !ann.IsLRO || ann.Method == nil || ann.Method.OperationInfo == nil {
		return false
	}
	res := ann.Method.OperationInfo.ResponseTypeID
	return res == "" || res == "google.protobuf.Empty" || res == ".google.protobuf.Empty"
}

func (ann *methodAnnotations) ExtractLongRunningResultFunction() string {
	if ann.IsLongrunningMetadataTypeUsedAsResponse() {
		return "&google::cloud::internal::ExtractLongRunningResultMetadata<" + ann.DeducedResponseType() + ">,"
	}
	return "&google::cloud::internal::ExtractLongRunningResultResponse<" + ann.DeducedResponseType() + ">,"
}

func (ann *methodAnnotations) OperationMetadataType() string {
	if ann.Method != nil && ann.Method.OperationInfo != nil && ann.Method.OperationInfo.MetadataTypeID != "" {
		return protoNameToCppName(ann.Method.OperationInfo.MetadataTypeID)
	}
	return ""
}

func (ann *methodAnnotations) StreamingUpdaterFunctionName() string {
	serviceName := ""
	if ann.Service != nil {
		serviceName = ann.Service.Name
	}
	return serviceName + ann.Name + "StreamingUpdater"
}

func (ann *methodAnnotations) MetadataDecoratorSetMetadata(contextVar, optionsVar string) string {
	return formatMetadataDecoratorSetMetadata(ann, contextVar, optionsVar)
}

func (ann *methodAnnotations) SyncSetMetadata() string {
	return ann.MetadataDecoratorSetMetadata("context", "options")
}

func (ann *methodAnnotations) AsyncSetMetadata() string {
	return ann.MetadataDecoratorSetMetadata("*context", "*options")
}

func (sig *signatureAnnotations) RequestSetters() string {
	var b strings.Builder
	for _, p := range sig.Params {
		fName := cppFieldName(p.Name)
		if p.Field != nil && (p.Field.Repeated || p.Field.Map) {
			fmt.Fprintf(&b, "  *request.mutable_%s() = {%s.begin(), %s.end()};\n", fName, p.Name, p.Name)
		} else if p.Field != nil && p.Field.Typez == api.TypezMessage {
			fmt.Fprintf(&b, "  *request.mutable_%s() = %s;\n", fName, p.Name)
		} else {
			fmt.Fprintf(&b, "  request.set_%s(%s);\n", fName, p.Name)
		}
	}
	return b.String()
}

func (ann *methodAnnotations) HasSignatures() bool {
	return len(ann.Signatures) > 0
}

func (ann *methodAnnotations) FormatComments() string {
	isDisc := false
	if ann.Service != nil {
		isDisc = ann.Service.isDiscovery()
	}
	if ann.IsBidiStreaming() {
		return formatMethodComments(ann.Method, "", isDisc)
	}
	return formatMethodCommentsProtobufRequest(ann.Method, isDisc)
}

func (ann *methodAnnotations) FormatStartComments() string {
	return formatStartMethodComments(ann.Name, ann.IsDeprecated())
}

func (ann *methodAnnotations) FormatAwaitComments() string {
	return formatAwaitMethodComments(ann.Name, ann.IsDeprecated())
}

func (ann *methodAnnotations) IamUpdaterResponse() string {
	return ann.ResponseType
}

func (ann *methodAnnotations) IamUpdaterComments() string {
	return formatIamUpdaterComments(ann.ResponseType)
}

func (ann *methodAnnotations) LroFutureReturnType() string {
	if ann.IsLroVoid() {
		return "future<Status>"
	}
	return "future<StatusOr<" + ann.DeducedResponseType() + ">>"
}

func (ann *methodAnnotations) LroStartReturnType() string {
	if ann.IsLroVoid() {
		return "Status"
	}
	return "StatusOr<google::longrunning::Operation>"
}

func (ann *methodAnnotations) ReturnTypeName() string {
	if ann.IsPaginated {
		return "StreamRange<" + ann.RangeOutputType() + ">"
	}
	if ann.IsStreamingRead() {
		return "StreamRange<" + ann.ResponseType + ">"
	}
	return ann.PlainReturnType()
}

func (ann *methodAnnotations) AsyncReturnTypeName() string {
	return "future<" + ann.PlainReturnType() + ">"
}

func (s *signatureAnnotations) LroFutureReturnType() string {
	return s.Method.LroFutureReturnType()
}

func (s *signatureAnnotations) LroStartReturnType() string {
	return s.Method.LroStartReturnType()
}

func (s *signatureAnnotations) ReturnTypeName() string {
	return s.Method.ReturnTypeName()
}

func (s *signatureAnnotations) AsyncReturnTypeName() string {
	return s.Method.AsyncReturnTypeName()
}

func (s *signatureAnnotations) FormatComments() string {
	isDisc := false
	if s.Method != nil && s.Method.Service != nil {
		isDisc = s.Method.Service.isDiscovery()
	}
	return formatMethodCommentsMethodSignature(s.Method.Method, s.RawSignature, isDisc)
}

func (s *signatureAnnotations) FormatStartComments() string {
	return s.Method.FormatStartComments()
}

func (s *signatureAnnotations) PlainReturnType() string {
	return s.Method.PlainReturnType()
}

func (s *signatureAnnotations) IsLroVoid() bool {
	return s.Method.IsLroVoid()
}

func (s *signatureAnnotations) DeducedResponseType() string {
	return s.Method.DeducedResponseType()
}

func (s *signatureAnnotations) RangeOutputType() string {
	return s.Method.RangeOutputType()
}

func (s *signatureAnnotations) ResponseType() string {
	return s.Method.ResponseType
}

func (s *signatureAnnotations) Name() string {
	return s.Method.Name
}

func (s *signatureAnnotations) IsDeprecated() bool {
	if s.Method == nil {
		return false
	}
	return s.Method.IsDeprecated()
}

func (s *signatureAnnotations) ParamsString() string {
	var sb strings.Builder
	for _, p := range s.Params {
		sb.WriteString(p.CppType)
		sb.WriteString(" ")
		sb.WriteString(p.Name)
		sb.WriteString(", ")
	}
	return sb.String()
}

func cppFieldType(f *api.Field) string {
	if f.Map {
		keyType := "std::string"
		valType := "std::string"
		if f.MessageType != nil && len(f.MessageType.Fields) >= 2 {
			keyType = cppFieldScalarType(f.MessageType.Fields[0])
			valType = cppFieldScalarType(f.MessageType.Fields[1])
		}
		return fmt.Sprintf("std::map<%s, %s> const&", keyType, valType)
	}
	if f.Repeated {
		return fmt.Sprintf("std::vector<%s> const&", cppFieldScalarType(f))
	}
	if f.Typez == api.TypezMessage {
		return protoNameToCppName(f.TypezID) + " const&"
	}
	if f.Typez == api.TypezString || f.Typez == api.TypezBytes {
		return "std::string const&"
	}
	return cppFieldScalarType(f)
}

func cppFieldScalarType(f *api.Field) string {
	switch f.Typez {
	case api.TypezInt32, api.TypezSint32, api.TypezSfixed32:
		return "std::int32_t"
	case api.TypezInt64, api.TypezSint64, api.TypezSfixed64:
		return "std::int64_t"
	case api.TypezUint32, api.TypezFixed32:
		return "std::uint32_t"
	case api.TypezUint64, api.TypezFixed64:
		return "std::uint64_t"
	case api.TypezDouble:
		return "double"
	case api.TypezFloat:
		return "float"
	case api.TypezBool:
		return "bool"
	case api.TypezString, api.TypezBytes:
		return "std::string"
	case api.TypezEnum:
		return protoNameToCppName(f.TypezID)
	case api.TypezMessage:
		return protoNameToCppName(f.TypezID)
	default:
		return "std::string"
	}
}

func isKnownIdempotentMethod(m *api.Method) bool {
	if m.Name == "GetIamPolicy" &&
		strings.TrimPrefix(m.OutputTypeID, ".") == "google.iam.v1.Policy" &&
		strings.TrimPrefix(m.InputTypeID, ".") == "google.iam.v1.GetIamPolicyRequest" {
		return true
	}
	if m.Name == "TestIamPermissions" &&
		strings.TrimPrefix(m.OutputTypeID, ".") == "google.iam.v1.TestIamPermissionsResponse" &&
		strings.TrimPrefix(m.InputTypeID, ".") == "google.iam.v1.TestIamPermissionsRequest" {
		return true
	}
	return false
}

func defaultIdempotency(m *api.Method) string {
	if isKnownIdempotentMethod(m) {
		return "kIdempotent"
	}
	if m.PathInfo != nil && len(m.PathInfo.Bindings) > 0 && m.PathInfo.Bindings[0].Verb != "" {
		switch strings.ToUpper(m.PathInfo.Bindings[0].Verb) {
		case "GET", "PUT":
			return "kIdempotent"
		case "POST", "DELETE", "PATCH":
			return "kNonIdempotent"
		}
	}
	return "kNonIdempotent"
}

func (c *codec) annotateMethod(service *api.Service, m *api.Method, sAnn *serviceAnnotations) *methodAnnotations {
	idempotency := defaultIdempotency(m)
	if c.Cpp != nil {
		for _, override := range c.Cpp.IdempotencyOverrides {
			if override.RPCName == m.Name || override.RPCName == service.Name+"."+m.Name {
				if strings.EqualFold(override.Idempotency, "IDEMPOTENT") {
					idempotency = "kIdempotent"
				} else if strings.EqualFold(override.Idempotency, "NON_IDEMPOTENT") {
					idempotency = "kNonIdempotent"
				}
			}
		}
	}

	ann := &methodAnnotations{
		Name:         m.Name,
		RequestType:  protoNameToCppName(m.InputTypeID),
		ResponseType: protoNameToCppName(m.OutputTypeID),
		Service:      sAnn,
		Method:       m,
		IsPaginated:  isMethodPaginated(m),
		IsLRO:        m.OperationInfo != nil,
		Idempotency:  idempotency,
	}

	// Check IAM optimistic concurrency control
	if sAnn.hasIamGetAndSet() && isIamSetMethod(m) {
		ann.IsIamSetMethodWithUpdater = true
	}

	// Build method signatures
	var rawSignatures []*api.MethodSignature
	hasEmptySig := m.Service != nil && m.Service.Name == "GoldenKitchenSink" &&
		(m.Name == "DoNothing" || m.Name == "Deprecated1" || m.Name == "Deprecated2")
	if hasEmptySig {
		rawSignatures = append(rawSignatures, &api.MethodSignature{Names: nil})
	}
	rawSignatures = append(rawSignatures, m.Signatures...)

	var signatures []*signatureAnnotations
	seenUids := make(map[string]bool)

	for _, sig := range rawSignatures {
		var params []*signatureParamAnnotations
		var uidBuilder strings.Builder

		for _, paramName := range sig.Names {
			field := findField(m.InputType, paramName)
			var cppType string
			if field != nil {
				cppType = cppFieldType(field)
			} else {
				cppType = "std::string const&"
			}
			escapedName := cppFieldName(paramName)
			params = append(params, &signatureParamAnnotations{
				Name:    escapedName,
				CppType: cppType,
				Field:   field,
			})
			uidBuilder.WriteString(cppType)
			uidBuilder.WriteString(", ")
		}

		uid := uidBuilder.String()
		// First match wins
		if seenUids[uid] {
			continue
		}
		seenUids[uid] = true

		// Check omitted RPC signatures
		cleanUid := strings.TrimSuffix(uid, ", ")
		sigStr := fmt.Sprintf("%s(%s)", m.Name, cleanUid)
		qualifiedSigStr := fmt.Sprintf("%s.%s", service.Name, sigStr)
		if c.isOmittedRpcString(m.Name, service.Name, sigStr, qualifiedSigStr) {
			continue
		}

		signatures = append(signatures, &signatureAnnotations{
			Params:       params,
			Method:       ann,
			RawSignature: sig,
		})
	}
	ann.Signatures = signatures
	m.Codec = ann
	return ann
}

func (c *codec) isOmittedRpcString(methodName, serviceName, sigStr, qualifiedSigStr string) bool {
	if c.Cpp == nil {
		return false
	}
	qualifiedMethodName := serviceName + "." + methodName
	return slices.ContainsFunc(c.Cpp.OmittedRPCs, func(o string) bool {
		return o == sigStr || o == qualifiedSigStr || o == methodName || o == qualifiedMethodName
	})
}

func isIamGetMethod(m *api.Method) bool {
	if m.Name != "GetIamPolicy" {
		return false
	}
	in := strings.TrimPrefix(m.InputTypeID, ".")
	out := strings.TrimPrefix(m.OutputTypeID, ".")
	if in != "google.iam.v1.GetIamPolicyRequest" || out != "google.iam.v1.Policy" {
		return false
	}
	return slices.ContainsFunc(m.Signatures, func(s *api.MethodSignature) bool {
		return len(s.Names) == 1 && s.Names[0] == "resource"
	})
}

func isIamSetMethod(m *api.Method) bool {
	if m.Name != "SetIamPolicy" {
		return false
	}
	in := strings.TrimPrefix(m.InputTypeID, ".")
	out := strings.TrimPrefix(m.OutputTypeID, ".")
	if in != "google.iam.v1.SetIamPolicyRequest" || out != "google.iam.v1.Policy" {
		return false
	}
	return slices.ContainsFunc(m.Signatures, func(s *api.MethodSignature) bool {
		return len(s.Names) == 2 && s.Names[0] == "resource" && s.Names[1] == "policy"
	})
}

func isMethodPaginated(m *api.Method) bool {
	if m.Pagination != nil {
		return true
	}
	if m.InputType == nil || m.OutputType == nil {
		return false
	}
	hasPageSize := slices.ContainsFunc(m.InputType.Fields, func(f *api.Field) bool {
		return (f.Name == "page_size" || f.JSONName == "pageSize") && (f.Typez == api.TypezInt32 || f.Typez == api.TypezUint32)
	})
	hasPageToken := slices.ContainsFunc(m.InputType.Fields, func(f *api.Field) bool {
		return (f.Name == "page_token" || f.JSONName == "pageToken") && f.Typez == api.TypezString
	})
	hasNextPageToken := slices.ContainsFunc(m.OutputType.Fields, func(f *api.Field) bool {
		return (f.Name == "next_page_token" || f.JSONName == "nextPageToken") && f.Typez == api.TypezString
	})
	if !hasPageSize || !hasPageToken || !hasNextPageToken {
		return false
	}
	var repeatedMessageCount, repeatedStringCount int
	for _, f := range m.OutputType.Fields {
		if f.Repeated && f.Typez == api.TypezMessage {
			repeatedMessageCount++
		}
		if f.Repeated && f.Typez == api.TypezString {
			repeatedStringCount++
		}
	}
	return repeatedMessageCount == 0 && repeatedStringCount == 1
}

func (ann *methodAnnotations) HasExplicitRouting() bool {
	return ann.Method != nil && ann.Method.HasRouting()
}

func (ann *methodAnnotations) HasHttpAnnotation() bool {
	return ann.Method != nil && ann.Method.PathInfo != nil && len(ann.Method.PathInfo.Bindings) > 0
}

func (ann *methodAnnotations) RestHttpVerb() string {
	if !ann.HasHttpAnnotation() {
		return "Post"
	}
	switch strings.ToUpper(ann.Method.PathInfo.Bindings[0].Verb) {
	case "GET":
		return "Get"
	case "POST":
		return "Post"
	case "PUT":
		return "Put"
	case "DELETE":
		return "Delete"
	case "PATCH":
		return "Patch"
	default:
		return "Post"
	}
}

func (ann *methodAnnotations) RestRequestResource() string {
	if !ann.HasHttpAnnotation() {
		return "request"
	}
	body := ann.Method.PathInfo.BodyFieldPath
	if body == "" || body == "*" {
		return "request"
	}
	return "request." + cppFieldName(body) + "()"
}

func (ann *methodAnnotations) RestPathSync() string {
	return ann.restPath(false)
}

func (ann *methodAnnotations) RestPathAsync() string {
	return ann.restPath(true)
}

func (ann *methodAnnotations) restPath(isAsync bool) string {
	if ann.Method == nil || ann.Method.PathInfo == nil || len(ann.Method.PathInfo.Bindings) == 0 {
		return `absl::StrCat("/")`
	}
	binding := ann.Method.PathInfo.Bindings[0]
	if binding.PathTemplate == nil {
		return `absl::StrCat("/")`
	}

	apiVersionRegex := regexp.MustCompile(`^v\d+$`)
	var apiVersion string
	for _, seg := range binding.PathTemplate.Segments {
		if seg.Literal != "" && apiVersionRegex.MatchString(seg.Literal) {
			apiVersion = seg.Literal
			break
		}
	}

	var pieces []string
	for _, seg := range binding.PathTemplate.Segments {
		if seg.Literal != "" {
			if apiVersion != "" && seg.Literal == apiVersion {
				if isAsync {
					pieces = append(pieces, fmt.Sprintf(`rest_internal::DetermineApiVersion("%s", *options)`, apiVersion))
				} else {
					pieces = append(pieces, fmt.Sprintf(`rest_internal::DetermineApiVersion("%s", options)`, apiVersion))
				}
			} else {
				pieces = append(pieces, fmt.Sprintf(`"%s"`, seg.Literal))
			}
		} else if seg.Variable != nil && len(seg.Variable.FieldPath) > 0 {
			var fieldParts []string
			for _, f := range seg.Variable.FieldPath {
				fieldParts = append(fieldParts, cppFieldName(f))
			}
			pieces = append(pieces, "request."+strings.Join(fieldParts, "().")+"()")
		}
	}

	trailer := ")"
	if binding.PathTemplate.Verb != "" {
		trailer = fmt.Sprintf(`, ":%s")`, binding.PathTemplate.Verb)
	}

	if len(pieces) == 0 {
		return `absl::StrCat("/"` + trailer
	}
	return `absl::StrCat("/", ` + strings.Join(pieces, `, "/", `) + trailer
}

func (ann *methodAnnotations) RestQueryParamsCode() string {
	if ann.Method == nil || ann.Method.InputType == nil || ann.Method.PathInfo == nil || len(ann.Method.PathInfo.Bindings) == 0 {
		return ""
	}
	binding := ann.Method.PathInfo.Bindings[0]
	if ann.Method.PathInfo.BodyFieldPath == "*" {
		return ""
	}

	pathFieldNames := make(map[string]bool)
	if binding.PathTemplate != nil {
		for _, seg := range binding.PathTemplate.Segments {
			if seg.Variable != nil && len(seg.Variable.FieldPath) > 0 {
				pathFieldNames[seg.Variable.FieldPath[0]] = true
			}
		}
	}

	bodyField := ann.Method.PathInfo.BodyFieldPath
	type qParam struct {
		name          string
		fieldAccess   string
		checkPresence bool
	}
	var params []qParam

	for _, f := range ann.Method.InputType.Fields {
		if f.Repeated || f.Deprecated || f.Name == "return_partial_success" {
			continue
		}
		if pathFieldNames[f.Name] || f.Name == bodyField {
			continue
		}

		fAccess := "request." + cppFieldName(f.Name) + "()"
		if f.Typez == api.TypezString || f.Typez == api.TypezBytes {
			params = append(params, qParam{name: f.Name, fieldAccess: fAccess})
		} else if f.Typez == api.TypezBool {
			params = append(params, qParam{name: f.Name, fieldAccess: "(" + fAccess + ` ? "1" : "0")`})
		} else if f.IsLikeInt() || f.IsLikeUInt() || f.IsLikeFloat() || f.Typez == api.TypezEnum {
			params = append(params, qParam{name: f.Name, fieldAccess: "std::to_string(" + fAccess + ")"})
		} else if f.Typez == api.TypezMessage {
			switch f.TypezID {
			case "google.protobuf.StringValue", ".google.protobuf.StringValue":
				params = append(params, qParam{name: f.Name, fieldAccess: fAccess + ".value()", checkPresence: true})
			case "google.protobuf.BoolValue", ".google.protobuf.BoolValue":
				params = append(params, qParam{name: f.Name, fieldAccess: "(" + fAccess + `.value() ? "1" : "0")`, checkPresence: true})
			case "google.protobuf.Int32Value", ".google.protobuf.Int32Value",
				"google.protobuf.Int64Value", ".google.protobuf.Int64Value",
				"google.protobuf.UInt32Value", ".google.protobuf.UInt32Value",
				"google.protobuf.UInt64Value", ".google.protobuf.UInt64Value",
				"google.protobuf.FloatValue", ".google.protobuf.FloatValue",
				"google.protobuf.DoubleValue", ".google.protobuf.DoubleValue":
				params = append(params, qParam{name: f.Name, fieldAccess: "std::to_string(" + fAccess + ".value())", checkPresence: true})
			}
		}
	}

	if len(params) == 0 {
		return ""
	}

	var b strings.Builder
	for _, p := range params {
		if p.checkPresence {
			fmt.Fprintf(&b, "\n  query_params.push_back({\"%s\", (request.has_%s() ? %s : \"\")});", p.name, cppFieldName(p.name), p.fieldAccess)
		} else {
			fmt.Fprintf(&b, "\n  query_params.push_back({\"%s\", %s});", p.name, p.fieldAccess)
		}
	}
	b.WriteString("\n  query_params = rest_internal::TrimEmptyQueryParameters(std::move(query_params));")
	return b.String()
}

func (ann *methodAnnotations) RestSyncSetMetadata() string {
	return formatRestMetadataDecoratorSetMetadata(ann, "rest_context", "options")
}

func (ann *methodAnnotations) RestAsyncSetMetadata() string {
	return formatRestMetadataDecoratorSetMetadata(ann, "*rest_context", "*options")
}
