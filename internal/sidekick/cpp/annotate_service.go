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
	"slices"
	"strings"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/sidekick/api"
)

// serviceAnnotations contains C++-specific metadata for a gRPC and/or REST service.
// Annotations are private to the package to encapsulate implementation details and enforce
// accessor method usage for derived properties.
type serviceAnnotations struct {
	Service               *api.Service
	ServiceName           string
	ProductPath           string
	ForwardingProductPath string
	CopyrightYear         string
	APIVersion            string
	ProtoFileName         string
	GrpcService           string
	HasGrpcTransport      bool
	HasRestTransport      bool

	// Decorator flags
	HasRoundRobinDecorator bool
	HasTracingDecorator    bool
	HasLoggingDecorator    bool
	HasMetadataDecorator   bool
	HasAuthDecorator       bool

	// REST configuration
	PreserveProtoFieldNamesInJson bool
	EndpointLocationStyle         string
	IsDiscoveryDocumentProto      bool

	// Endpoints and environment variables
	ServiceEndpoint        string
	ServiceEndpointEnvVar  string
	ServiceAuthorityEnvVar string
	EmulatorEndpointEnvVar string

	// Additional headers and status codes
	AdditionalPbHeaderPaths []string
	RetryableStatusCodes    []string

	// Methods
	Methods      []*api.Method
	AsyncMethods []*api.Method

	// Associated annotations
	Comments *commentAnnotations
	Options  *optionsAnnotations
}

func (s *serviceAnnotations) FilePathName() string {
	return serviceNameToFilePath(s.ServiceName)
}

func (s *serviceAnnotations) LibraryName() string {
	_, l, _ := parseProductPath(s.ProductPath)
	return l
}

func (s *serviceAnnotations) ServiceSubdirectory() string {
	_, _, sub := parseProductPath(s.ProductPath)
	return sub
}

func (s *serviceAnnotations) Namespace() string {
	return namespace(s.ProductPath, namespaceNormal)
}

func (s *serviceAnnotations) InternalNamespace() string {
	return namespace(s.ProductPath, namespaceInternal)
}

func (s *serviceAnnotations) MocksNamespace() string {
	return namespace(s.ProductPath, namespaceMocks)
}

func (s *serviceAnnotations) ForwardingNamespace() string {
	if s.ForwardingProductPath == "" {
		return ""
	}
	return namespace(s.ForwardingProductPath, namespaceNormal)
}

func (s *serviceAnnotations) ForwardingMocksNamespace() string {
	if s.ForwardingProductPath == "" {
		return ""
	}
	return namespace(s.ForwardingProductPath, namespaceMocks)
}

func (s *serviceAnnotations) OptionsGroupName() string {
	return optionsGroup(s.ProductPath)
}

func (s *serviceAnnotations) IsLocationDependent() bool {
	return s.EndpointLocationStyle == "LOCATION_DEPENDENT" ||
		s.EndpointLocationStyle == "LOCATION_DEPENDENT_COMPAT" ||
		s.EndpointLocationStyle == "LOCATION_OPTIONALLY_DEPENDENT"
}

func (s *serviceAnnotations) HasNonLocationOverload() bool {
	return s.EndpointLocationStyle == "LOCATION_DEPENDENT_COMPAT" ||
		s.EndpointLocationStyle == "LOCATION_OPTIONALLY_DEPENDENT"
}

// Class names

func (s *serviceAnnotations) ClientClassName() string     { return s.ServiceName + "Client" }
func (s *serviceAnnotations) ConnectionClassName() string { return s.ServiceName + "Connection" }
func (s *serviceAnnotations) ConnectionImplClassName() string {
	return s.ServiceName + "ConnectionImpl"
}
func (s *serviceAnnotations) IdempotencyClassName() string {
	return s.ServiceName + "ConnectionIdempotencyPolicy"
}
func (s *serviceAnnotations) MockConnectionClassName() string {
	return "Mock" + s.ServiceName + "Connection"
}
func (s *serviceAnnotations) AuthClassName() string     { return s.ServiceName + "Auth" }
func (s *serviceAnnotations) LoggingClassName() string  { return s.ServiceName + "Logging" }
func (s *serviceAnnotations) MetadataClassName() string { return s.ServiceName + "Metadata" }
func (s *serviceAnnotations) StubClassName() string     { return s.ServiceName + "Stub" }
func (s *serviceAnnotations) TracingConnectionClassName() string {
	return s.ServiceName + "TracingConnection"
}
func (s *serviceAnnotations) TracingStubClassName() string { return s.ServiceName + "TracingStub" }
func (s *serviceAnnotations) RoundRobinClassName() string  { return s.ServiceName + "RoundRobin" }
func (s *serviceAnnotations) RetryTraitsName() string      { return s.ServiceName + "RetryTraits" }
func (s *serviceAnnotations) RetryPolicyName() string      { return s.ServiceName + "RetryPolicy" }
func (s *serviceAnnotations) LimitedErrorCountRetryPolicyName() string {
	return s.ServiceName + "LimitedErrorCountRetryPolicy"
}
func (s *serviceAnnotations) LimitedTimeRetryPolicyName() string {
	return s.ServiceName + "LimitedTimeRetryPolicy"
}
func (s *serviceAnnotations) ConnectionOptionsName() string {
	return s.ServiceName + "ConnectionOptions"
}
func (s *serviceAnnotations) ConnectionOptionsTraitsName() string {
	return s.ServiceName + "ConnectionOptionsTraits"
}
func (s *serviceAnnotations) StubRestClassName() string { return s.ServiceName + "RestStub" }
func (s *serviceAnnotations) ConnectionImplRestClassName() string {
	return s.ServiceName + "RestConnectionImpl"
}
func (s *serviceAnnotations) LoggingRestClassName() string  { return s.ServiceName + "RestLogging" }
func (s *serviceAnnotations) MetadataRestClassName() string { return s.ServiceName + "RestMetadata" }

// Paths.
func (s *serviceAnnotations) ClientHeaderPath() string {
	return s.ProductPath + s.FilePathName() + "_client.h"
}
func (s *serviceAnnotations) ClientCcPath() string {
	return s.ProductPath + s.FilePathName() + "_client.cc"
}
func (s *serviceAnnotations) ClientSamplesCcPath() string {
	return s.ProductPath + "samples/" + s.FilePathName() + "_client_samples.cc"
}
func (s *serviceAnnotations) ConnectionHeaderPath() string {
	return s.ProductPath + s.FilePathName() + "_connection.h"
}
func (s *serviceAnnotations) ConnectionCcPath() string {
	return s.ProductPath + s.FilePathName() + "_connection.cc"
}
func (s *serviceAnnotations) ConnectionImplHeaderPath() string {
	return s.ProductPath + "internal/" + s.FilePathName() + "_connection_impl.h"
}
func (s *serviceAnnotations) ConnectionImplCcPath() string {
	return s.ProductPath + "internal/" + s.FilePathName() + "_connection_impl.cc"
}
func (s *serviceAnnotations) StubHeaderPath() string {
	return s.ProductPath + "internal/" + s.FilePathName() + "_stub.h"
}
func (s *serviceAnnotations) StubCcPath() string {
	return s.ProductPath + "internal/" + s.FilePathName() + "_stub.cc"
}
func (s *serviceAnnotations) StubFactoryHeaderPath() string {
	return s.ProductPath + "internal/" + s.FilePathName() + "_stub_factory.h"
}
func (s *serviceAnnotations) StubFactoryCcPath() string {
	return s.ProductPath + "internal/" + s.FilePathName() + "_stub_factory.cc"
}
func (s *serviceAnnotations) OptionsHeaderPath() string {
	return s.ProductPath + s.FilePathName() + "_options.h"
}
func (s *serviceAnnotations) IdempotencyHeaderPath() string {
	return s.ProductPath + s.FilePathName() + "_connection_idempotency_policy.h"
}
func (s *serviceAnnotations) IdempotencyPolicyHeaderPath() string { return s.IdempotencyHeaderPath() }
func (s *serviceAnnotations) IdempotencyCcPath() string {
	return s.ProductPath + s.FilePathName() + "_connection_idempotency_policy.cc"
}

func (s *serviceAnnotations) MockConnectionHeaderPath() string {
	return s.ProductPath + "mocks/mock_" + s.FilePathName() + "_connection.h"
}
func (s *serviceAnnotations) AuthHeaderPath() string {
	return s.ProductPath + "internal/" + s.FilePathName() + "_auth_decorator.h"
}
func (s *serviceAnnotations) AuthCcPath() string {
	return s.ProductPath + "internal/" + s.FilePathName() + "_auth_decorator.cc"
}
func (s *serviceAnnotations) LoggingHeaderPath() string {
	return s.ProductPath + "internal/" + s.FilePathName() + "_logging_decorator.h"
}
func (s *serviceAnnotations) LoggingCcPath() string {
	return s.ProductPath + "internal/" + s.FilePathName() + "_logging_decorator.cc"
}
func (s *serviceAnnotations) MetadataHeaderPath() string {
	return s.ProductPath + "internal/" + s.FilePathName() + "_metadata_decorator.h"
}
func (s *serviceAnnotations) MetadataCcPath() string {
	return s.ProductPath + "internal/" + s.FilePathName() + "_metadata_decorator.cc"
}
func (s *serviceAnnotations) TracingConnectionHeaderPath() string {
	return s.ProductPath + "internal/" + s.FilePathName() + "_tracing_connection.h"
}
func (s *serviceAnnotations) TracingConnectionCcPath() string {
	return s.ProductPath + "internal/" + s.FilePathName() + "_tracing_connection.cc"
}
func (s *serviceAnnotations) TracingStubHeaderPath() string {
	return s.ProductPath + "internal/" + s.FilePathName() + "_tracing_stub.h"
}
func (s *serviceAnnotations) TracingStubCcPath() string {
	return s.ProductPath + "internal/" + s.FilePathName() + "_tracing_stub.cc"
}
func (s *serviceAnnotations) RoundRobinHeaderPath() string {
	return s.ProductPath + "internal/" + s.FilePathName() + "_round_robin_decorator.h"
}
func (s *serviceAnnotations) RoundRobinCcPath() string {
	return s.ProductPath + "internal/" + s.FilePathName() + "_round_robin_decorator.cc"
}
func (s *serviceAnnotations) OptionDefaultsHeaderPath() string {
	return s.ProductPath + "internal/" + s.FilePathName() + "_option_defaults.h"
}
func (s *serviceAnnotations) OptionDefaultsCcPath() string {
	return s.ProductPath + "internal/" + s.FilePathName() + "_option_defaults.cc"
}
func (s *serviceAnnotations) RetryTraitsHeaderPath() string {
	return s.ProductPath + "internal/" + s.FilePathName() + "_retry_traits.h"
}
func (s *serviceAnnotations) SourcesCcPath() string {
	return s.ProductPath + "internal/" + s.FilePathName() + "_sources.cc"
}

// REST paths.
func (s *serviceAnnotations) ConnectionRestHeaderPath() string {
	return s.ProductPath + s.FilePathName() + "_rest_connection.h"
}
func (s *serviceAnnotations) ConnectionRestCcPath() string {
	return s.ProductPath + s.FilePathName() + "_rest_connection.cc"
}
func (s *serviceAnnotations) ConnectionImplRestHeaderPath() string {
	return s.ProductPath + "internal/" + s.FilePathName() + "_rest_connection_impl.h"
}
func (s *serviceAnnotations) ConnectionImplRestCcPath() string {
	return s.ProductPath + "internal/" + s.FilePathName() + "_rest_connection_impl.cc"
}
func (s *serviceAnnotations) StubRestHeaderPath() string {
	return s.ProductPath + "internal/" + s.FilePathName() + "_rest_stub.h"
}
func (s *serviceAnnotations) StubRestCcPath() string {
	return s.ProductPath + "internal/" + s.FilePathName() + "_rest_stub.cc"
}
func (s *serviceAnnotations) StubFactoryRestHeaderPath() string {
	return s.ProductPath + "internal/" + s.FilePathName() + "_rest_stub_factory.h"
}
func (s *serviceAnnotations) StubFactoryRestCcPath() string {
	return s.ProductPath + "internal/" + s.FilePathName() + "_rest_stub_factory.cc"
}
func (s *serviceAnnotations) LoggingRestHeaderPath() string {
	return s.ProductPath + "internal/" + s.FilePathName() + "_rest_logging_decorator.h"
}
func (s *serviceAnnotations) LoggingRestCcPath() string {
	return s.ProductPath + "internal/" + s.FilePathName() + "_rest_logging_decorator.cc"
}
func (s *serviceAnnotations) MetadataRestHeaderPath() string {
	return s.ProductPath + "internal/" + s.FilePathName() + "_rest_metadata_decorator.h"
}
func (s *serviceAnnotations) MetadataRestCcPath() string {
	return s.ProductPath + "internal/" + s.FilePathName() + "_rest_metadata_decorator.cc"
}

// Forwarding paths.
func (s *serviceAnnotations) ForwardingClientHeaderPath() string {
	if s.ForwardingProductPath == "" {
		return ""
	}
	return s.ForwardingProductPath + s.FilePathName() + "_client.h"
}
func (s *serviceAnnotations) ForwardingConnectionHeaderPath() string {
	if s.ForwardingProductPath == "" {
		return ""
	}
	return s.ForwardingProductPath + s.FilePathName() + "_connection.h"
}
func (s *serviceAnnotations) ForwardingIdempotencyPolicyHeaderPath() string {
	if s.ForwardingProductPath == "" {
		return ""
	}
	return s.ForwardingProductPath + s.FilePathName() + "_connection_idempotency_policy.h"
}
func (s *serviceAnnotations) ForwardingMockConnectionHeaderPath() string {
	if s.ForwardingProductPath == "" {
		return ""
	}
	return s.ForwardingProductPath + "mocks/mock_" + s.FilePathName() + "_connection.h"
}
func (s *serviceAnnotations) ForwardingOptionsHeaderPath() string {
	if s.ForwardingProductPath == "" {
		return ""
	}
	return s.ForwardingProductPath + s.FilePathName() + "_options.h"
}

// Include guards.
func (s *serviceAnnotations) ClientHeaderIncludeGuard() string {
	return formatHeaderIncludeGuard(s.ClientHeaderPath())
}
func (s *serviceAnnotations) ConnectionHeaderIncludeGuard() string {
	return formatHeaderIncludeGuard(s.ConnectionHeaderPath())
}
func (s *serviceAnnotations) ConnectionImplHeaderIncludeGuard() string {
	return formatHeaderIncludeGuard(s.ConnectionImplHeaderPath())
}
func (s *serviceAnnotations) StubHeaderIncludeGuard() string {
	return formatHeaderIncludeGuard(s.StubHeaderPath())
}
func (s *serviceAnnotations) StubFactoryHeaderIncludeGuard() string {
	return formatHeaderIncludeGuard(s.StubFactoryHeaderPath())
}
func (s *serviceAnnotations) OptionsHeaderIncludeGuard() string {
	return formatHeaderIncludeGuard(s.OptionsHeaderPath())
}
func (s *serviceAnnotations) IdempotencyHeaderIncludeGuard() string {
	return formatHeaderIncludeGuard(s.IdempotencyHeaderPath())
}
func (s *serviceAnnotations) MockConnectionHeaderIncludeGuard() string {
	return formatHeaderIncludeGuard(s.MockConnectionHeaderPath())
}
func (s *serviceAnnotations) AuthHeaderIncludeGuard() string {
	return formatHeaderIncludeGuard(s.AuthHeaderPath())
}
func (s *serviceAnnotations) LoggingHeaderIncludeGuard() string {
	return formatHeaderIncludeGuard(s.LoggingHeaderPath())
}
func (s *serviceAnnotations) MetadataHeaderIncludeGuard() string {
	return formatHeaderIncludeGuard(s.MetadataHeaderPath())
}
func (s *serviceAnnotations) TracingConnectionHeaderIncludeGuard() string {
	return formatHeaderIncludeGuard(s.TracingConnectionHeaderPath())
}
func (s *serviceAnnotations) TracingStubHeaderIncludeGuard() string {
	return formatHeaderIncludeGuard(s.TracingStubHeaderPath())
}
func (s *serviceAnnotations) RoundRobinHeaderIncludeGuard() string {
	return formatHeaderIncludeGuard(s.RoundRobinHeaderPath())
}
func (s *serviceAnnotations) OptionDefaultsHeaderIncludeGuard() string {
	return formatHeaderIncludeGuard(s.OptionDefaultsHeaderPath())
}
func (s *serviceAnnotations) RetryTraitsHeaderIncludeGuard() string {
	return formatHeaderIncludeGuard(s.RetryTraitsHeaderPath())
}
func (s *serviceAnnotations) ConnectionRestHeaderIncludeGuard() string {
	return formatHeaderIncludeGuard(s.ConnectionRestHeaderPath())
}
func (s *serviceAnnotations) ConnectionImplRestHeaderIncludeGuard() string {
	return formatHeaderIncludeGuard(s.ConnectionImplRestHeaderPath())
}
func (s *serviceAnnotations) StubRestHeaderIncludeGuard() string {
	return formatHeaderIncludeGuard(s.StubRestHeaderPath())
}
func (s *serviceAnnotations) StubFactoryRestHeaderIncludeGuard() string {
	return formatHeaderIncludeGuard(s.StubFactoryRestHeaderPath())
}
func (s *serviceAnnotations) LoggingRestHeaderIncludeGuard() string {
	return formatHeaderIncludeGuard(s.LoggingRestHeaderPath())
}
func (s *serviceAnnotations) MetadataRestHeaderIncludeGuard() string {
	return formatHeaderIncludeGuard(s.MetadataRestHeaderPath())
}

// Proto & gRPC.
func (s *serviceAnnotations) ProtoGrpcHeaderPath() string {
	return strings.TrimSuffix(s.ProtoFileName, ".proto") + ".grpc.pb.h"
}
func (s *serviceAnnotations) ProtoHeaderPath() string {
	return strings.TrimSuffix(s.ProtoFileName, ".proto") + ".pb.h"
}
func (s *serviceAnnotations) GrpcStubFQN() string {
	return protoNameToCppName(s.GrpcService)
}

// Retry traits.
func (s *serviceAnnotations) RetryStatusCodeExpression() string {
	return retryStatusCodeExpression(s.ServiceName, s.RetryableStatusCodes)
}
func (s *serviceAnnotations) TransientErrorsComment() string {
	return transientErrorsComment(s.ServiceName, s.RetryableStatusCodes)
}

func (s *serviceAnnotations) ClassCommentBlock() string {
	if s.Comments != nil {
		return s.Comments.ClassComment
	}
	return ""
}

func (s *serviceAnnotations) HasGRPCLongrunningOperation() bool {
	return slices.ContainsFunc(s.Methods, func(m *api.Method) bool {
		return m.OperationInfo != nil
	})
}

func (s *serviceAnnotations) HasHTTPLongrunningOperation() bool {
	return slices.ContainsFunc(s.Methods, func(m *api.Method) bool {
		return m.OperationService != ""
	})
}

func (s *serviceAnnotations) HasLongrunningMethod() bool {
	return s.HasGRPCLongrunningOperation() || s.HasHTTPLongrunningOperation()
}

func (s *serviceAnnotations) OperationService() string {
	for _, m := range s.Methods {
		if m.OperationService != "" {
			return m.OperationService
		}
	}
	return ""
}

func (s *serviceAnnotations) LongrunningOperationIncludeHeader() string {
	if s.HasHTTPLongrunningOperation() {
		return getComputeOperationInfo(s.OperationService()).HeaderInclude
	}
	return "google/longrunning/operations.pb.h"
}

func (s *serviceAnnotations) LongrunningResponseType() string {
	if s.HasHTTPLongrunningOperation() {
		for _, m := range s.Methods {
			if m.OperationService != "" && m.OutputTypeID != "" {
				return protoNameToCppName(m.OutputTypeID)
			}
		}
	}
	return "google::longrunning::Operation"
}

func (s *serviceAnnotations) LongrunningGetOperationRequestType() string {
	if s.HasHTTPLongrunningOperation() {
		return getComputeOperationInfo(s.OperationService()).GetRequestType
	}
	return "google::longrunning::GetOperationRequest"
}

func (s *serviceAnnotations) LongrunningCancelOperationRequestType() string {
	if s.HasHTTPLongrunningOperation() {
		return getComputeOperationInfo(s.OperationService()).CancelRequestType
	}
	return "google::longrunning::CancelOperationRequest"
}

func (s *serviceAnnotations) LongrunningSetOperationFields() string {
	if s.HasHTTPLongrunningOperation() {
		return getComputeOperationInfo(s.OperationService()).SetOperationFields
	}
	return ""
}

func (s *serviceAnnotations) LongrunningAwaitSetOperationFields() string {
	if s.HasHTTPLongrunningOperation() {
		return getComputeOperationInfo(s.OperationService()).AwaitSetOperationFields
	}
	return ""
}

func (s *serviceAnnotations) LongrunningGetOperationPathRest() string {
	if s.HasHTTPLongrunningOperation() {
		return getComputeOperationInfo(s.OperationService()).GetOperationPath
	}
	return `absl::StrCat("/", rest_internal::DetermineApiVersion("v1", *options) ,"/", request.name())`
}

func (s *serviceAnnotations) LongrunningCancelOperationPathRest() string {
	if s.HasHTTPLongrunningOperation() {
		return getComputeOperationInfo(s.OperationService()).CancelOperationPath
	}
	return `absl::StrCat("/", rest_internal::DetermineApiVersion("v1", *options) ,"/", request.name(), ":cancel")`
}

// annotateService enriches an api.Service with C++-specific namespaces, class names, file paths, and decorators.
func annotateService(svc *api.Service, lib *config.Library, model *api.API) *serviceAnnotations {
	if svc == nil {
		return nil
	}

	serviceName := svc.Name
	productPath := ""
	forwardingProductPath := ""
	if lib != nil && lib.Cpp != nil {
		productPath = formatProductPath(lib.Cpp.ProductPath)
		if lib.Cpp.ForwardingProductPath != "" {
			forwardingProductPath = formatProductPath(lib.Cpp.ForwardingProductPath)
		}
	}

	copyrightYear := "2026"
	if lib != nil && lib.CopyrightYear != "" {
		copyrightYear = lib.CopyrightYear
	}

	apiVersion := ""
	for _, m := range svc.Methods {
		if m.APIVersion != "" {
			apiVersion = m.APIVersion
			break
		}
	}

	protoFile := ""
	if model != nil {
		if loc, ok := model.DefinitionLocation(svc.ID[1:]); ok {
			protoFile = loc.Filename
		}
	}
	if protoFile == "" && lib != nil && len(lib.APIs) > 0 {
		protoFile = lib.APIs[0].Path
	}

	endpointEnvVar := ""
	emulatorEnvVar := ""
	locationStyle := ""
	var addPbFiles []string
	var retryCodes []string
	hasRoundRobin := false
	preserveProtoFieldNames := false
	if lib != nil && lib.Cpp != nil {
		endpointEnvVar = lib.Cpp.ServiceEndpointEnvVar
		emulatorEnvVar = lib.Cpp.EmulatorEndpointEnvVar
		locationStyle = lib.Cpp.EndpointLocationStyle
		hasRoundRobin = lib.Cpp.GenerateRoundRobinDecorator
		addPbFiles = lib.Cpp.AdditionalProtoFiles
		retryCodes = lib.Cpp.RetryableStatusCodes
		preserveProtoFieldNames = lib.Cpp.PreserveProtoFieldNamesInJson
	}
	if endpointEnvVar == "" {
		endpointEnvVar = "GOOGLE_CLOUD_CPP_" + strings.ToUpper(camelCaseToSnakeCase(svc.Name)) + "_ENDPOINT"
	}
	prefix := strings.TrimSuffix(endpointEnvVar, "_ENDPOINT")
	authorityEnvVar := prefix + "_AUTHORITY"

	var addPb []string
	for _, add := range addPbFiles {
		addPb = append(addPb, strings.TrimSuffix(add, ".proto")+".pb.h")
	}

	methods, asyncMethods := getEffectiveMethods(svc, lib)
	hasGrpc := lib == nil || lib.Cpp == nil || lib.Cpp.HasGrpcTransport()
	hasRest := lib != nil && lib.Cpp != nil && lib.Cpp.HasRestTransport()

	ann := &serviceAnnotations{
		Service:                       svc,
		ServiceName:                   serviceName,
		ProductPath:                   productPath,
		ForwardingProductPath:         forwardingProductPath,
		CopyrightYear:                 copyrightYear,
		APIVersion:                    apiVersion,
		ProtoFileName:                 protoFile,
		GrpcService:                   svc.ID[1:],
		HasGrpcTransport:              hasGrpc,
		HasRestTransport:              hasRest,
		HasRoundRobinDecorator:        hasRoundRobin,
		HasTracingDecorator:           true,
		HasLoggingDecorator:           true,
		HasMetadataDecorator:          true,
		HasAuthDecorator:              true,
		PreserveProtoFieldNamesInJson: preserveProtoFieldNames,
		EndpointLocationStyle:         locationStyle,
		IsDiscoveryDocumentProto:      (lib != nil && ((lib.Cpp != nil && lib.Cpp.IsDiscoveryDocumentProto) || lib.SpecificationFormat == "discovery")) || slices.ContainsFunc(svc.Methods, func(m *api.Method) bool { return m.OperationService != "" }),

		ServiceEndpoint:         svc.DefaultHost,
		ServiceEndpointEnvVar:   endpointEnvVar,
		ServiceAuthorityEnvVar:  authorityEnvVar,
		EmulatorEndpointEnvVar:  emulatorEnvVar,
		AdditionalPbHeaderPaths: addPb,
		RetryableStatusCodes:    retryCodes,
		Methods:                 methods,
		AsyncMethods:            asyncMethods,
	}

	ann.Comments = annotateComments(svc, serviceName, model, ann.IsDiscoveryDocumentProto)
	ann.Options = annotateOptions(svc, lib)

	svc.Codec = ann
	return ann
}
