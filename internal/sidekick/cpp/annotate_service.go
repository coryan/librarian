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
	"strings"

	"github.com/googleapis/librarian/internal/sidekick/api"
	"github.com/googleapis/librarian/internal/sidekick/language"
)

type serviceAnnotations struct {
	Name                   string
	BaseFileName           string
	ProductPath            string
	ForwardingPath         string
	HasGrpc                bool
	HasRest                bool
	HasRoundRobin          bool
	HasRetryTraits         bool
	RetryableStatusCodes   []string
	EndpointLocationStyle  string
	OmitClient             bool
	OmitConnection         bool
	OmitStubFactory        bool
	CopyrightYear          string
	Service                *api.Service
	Methods                []*methodAnnotations
	StubMethods            []*methodAnnotations
	AsyncMethods           []*methodAnnotations
	AsyncStubMethods       []*methodAnnotations
	ProtoGrpcHeaderPath    string
	ServiceEndpointEnvVar  string
	EmulatorEndpointEnvVar string
}

func (ann *serviceAnnotations) isDiscovery() bool {
	return false
}

func (ann *serviceAnnotations) hasIamGetAndSet() bool {
	if ann.Service == nil {
		return false
	}
	hasGet := slices.ContainsFunc(ann.Service.Methods, isIamGetMethod)
	hasSet := slices.ContainsFunc(ann.Service.Methods, isIamSetMethod)
	return hasGet && hasSet
}

func (ann *serviceAnnotations) HasIamUpdater() bool {
	return ann.hasIamGetAndSet()
}

func (ann *serviceAnnotations) HasGrpcLRO() bool {
	return ann.HasGrpc && ann.HasLRO()
}

func (ann *serviceAnnotations) StreamingReadMethods() []*methodAnnotations {
	var res []*methodAnnotations
	for _, m := range ann.Methods {
		if m.IsStreamingRead() {
			res = append(res, m)
		}
	}
	return res
}

func (ann *serviceAnnotations) HasStreamingReadMethod() bool {
	return len(ann.StreamingReadMethods()) > 0
}

func (ann *serviceAnnotations) NonStreamingMethods() []*methodAnnotations {
	var res []*methodAnnotations
	for _, m := range ann.Methods {
		if !m.IsStreaming() {
			res = append(res, m)
		}
	}
	return res
}

func (ann *serviceAnnotations) HasPaginatedMethod() bool {
	return slices.ContainsFunc(ann.Methods, func(m *methodAnnotations) bool {
		return m.IsPaginated
	})
}

func (ann *serviceAnnotations) HasBidirStreamingMethod() bool {
	return slices.ContainsFunc(ann.Methods, func(m *methodAnnotations) bool {
		return m.IsBidiStreaming()
	})
}

func (ann *serviceAnnotations) HasAsyncMethod() bool {
	return len(ann.AsyncMethods) > 0
}

func (ann *serviceAnnotations) HasStreamingMethod() bool {
	return slices.ContainsFunc(ann.StubMethods, func(m *methodAnnotations) bool {
		return m.IsStreaming()
	})
}

func (ann *serviceAnnotations) HasStreamingWriteMethod() bool {
	return slices.ContainsFunc(ann.StubMethods, func(m *methodAnnotations) bool {
		return m.IsStreamingWrite()
	})
}

func (ann *serviceAnnotations) HasAsynchronousStreamingReadMethod() bool {
	return slices.ContainsFunc(ann.AsyncStubMethods, func(m *methodAnnotations) bool {
		return m.IsStreamingRead()
	})
}

func (ann *serviceAnnotations) HasAsynchronousStreamingWriteMethod() bool {
	return slices.ContainsFunc(ann.AsyncStubMethods, func(m *methodAnnotations) bool {
		return m.IsStreamingWrite()
	})
}

func (ann *serviceAnnotations) HasExplicitRoutingMethod() bool {
	return slices.ContainsFunc(ann.StubMethods, func(m *methodAnnotations) bool {
		return m.HasExplicitRouting()
	})
}

func (ann *serviceAnnotations) HasTracedStreamRange() bool {
	return ann.HasPaginatedMethod() || ann.HasStreamingReadMethod()
}

func (ann *serviceAnnotations) HasRequestId() bool {
	if ann.Service == nil {
		return false
	}
	return slices.ContainsFunc(ann.Service.Methods, func(m *api.Method) bool {
		return m.HasAutoPopulatedFields()
	})
}

func (ann *serviceAnnotations) HasDurationInclude() bool {
	for _, m := range ann.Methods {
		for _, s := range m.Signatures {
			for _, p := range s.Params {
				if p.Field != nil && (p.Field.TypezID == "google.protobuf.Duration" || p.Field.TypezID == ".google.protobuf.Duration") {
					return true
				}
			}
		}
	}
	for _, m := range ann.AsyncMethods {
		for _, s := range m.Signatures {
			for _, p := range s.Params {
				if p.Field != nil && (p.Field.TypezID == "google.protobuf.Duration" || p.Field.TypezID == ".google.protobuf.Duration") {
					return true
				}
			}
		}
	}
	return false
}

// HasEndpointLocation returns true if the service uses location-dependent endpoints.
func (ann *serviceAnnotations) HasEndpointLocation() bool {
	return ann.EndpointLocationStyle != ""
}

// HasLRO returns true if the service contains any methods returning long-running operations.
func (ann *serviceAnnotations) HasLRO() bool {
	if ann.Service == nil {
		return false
	}
	return slices.ContainsFunc(ann.Service.Methods, func(m *api.Method) bool {
		return m.OperationInfo != nil
	})
}

func (ann *serviceAnnotations) MethodSignatureUsesDeprecatedField() bool {
	for _, m := range ann.Methods {
		for _, s := range m.Signatures {
			for _, p := range s.Params {
				if p.Field != nil && p.Field.Deprecated {
					return true
				}
			}
		}
	}
	return false
}

func (ann *serviceAnnotations) ServiceAuthorityEnvVar() string {
	return strings.TrimSuffix(ann.ServiceEndpointEnvVar, "_ENDPOINT") + "_AUTHORITY"
}

func (ann *serviceAnnotations) DefaultEndpoint() string {
	if ann.Service != nil && ann.Service.DefaultHost != "" {
		return ann.Service.DefaultHost
	}
	return ""
}

func (ann *serviceAnnotations) ServiceGrpcFqn() string {
	if ann.Service == nil || ann.Service.Package == "" {
		return ann.Name
	}
	return ann.Service.Package + "." + ann.Name
}

func (ann *serviceAnnotations) StreamingUpdaterFunctionName() string {
	return ann.Name + "StreamingReadStreamingUpdater"
}

func (ann *serviceAnnotations) ApiVersion() string {
	for _, m := range ann.Methods {
		if m.Method != nil && m.Method.APIVersion != "" {
			return m.Method.APIVersion
		}
	}
	return ""
}

func (ann *serviceAnnotations) HasLocationsMixin() bool {
	return slices.ContainsFunc(ann.StubMethods, func(m *methodAnnotations) bool {
		return m.Method != nil && m.Method.SourceService != nil && m.Method.Service != nil &&
			m.Method.SourceService.Name != m.Method.Service.Name && m.Method.SourceService.Name == "Locations"
	})
}

func (ann *serviceAnnotations) HasIamMixin() bool {
	return slices.ContainsFunc(ann.StubMethods, func(m *methodAnnotations) bool {
		return m.Method != nil && m.Method.SourceService != nil && m.Method.Service != nil &&
			m.Method.SourceService.Name != m.Method.Service.Name && m.Method.SourceService.Name == "IAMPolicy"
	})
}

func (ann *serviceAnnotations) HasOperationsMixin() bool {
	if ann.HasLRO() {
		return false
	}
	return slices.ContainsFunc(ann.StubMethods, func(m *methodAnnotations) bool {
		return m.Method != nil && m.Method.SourceService != nil && m.Method.Service != nil &&
			m.Method.SourceService.Name != m.Method.Service.Name && m.Method.SourceService.Name == "Operations"
	})
}

func (ann *serviceAnnotations) StubFactoryMakeDefaultStub() string {
	var b strings.Builder
	cppStubType := protoNameToCppName(ann.ServiceGrpcFqn())
	fmt.Fprintf(&b, "  auto service_grpc_stub = %s::NewStub(channel);\n", cppStubType)
	if ann.HasOperationsMixin() {
		b.WriteString("  auto service_operations_stub = google::longrunning::Operations::NewStub(channel);\n")
	}
	if ann.HasIamMixin() {
		b.WriteString("  auto service_iampolicy_stub = google::iam::v1::IAMPolicy::NewStub(channel);\n")
	}
	if ann.HasLocationsMixin() {
		b.WriteString("  auto service_locations_stub = google::cloud::location::Locations::NewStub(channel);\n")
	}

	if !ann.HasGrpcLRO() {
		moves := "std::move(service_grpc_stub)"
		if ann.HasOperationsMixin() {
			moves += ", std::move(service_operations_stub)"
		}
		if ann.HasIamMixin() {
			moves += ", std::move(service_iampolicy_stub)"
		}
		if ann.HasLocationsMixin() {
			moves += ", std::move(service_locations_stub)"
		}
		fmt.Fprintf(&b, "  std::shared_ptr<%s> stub =\n    std::make_shared<%s>(%s);\n",
			ann.StubClassName(), ann.DefaultStubClassName(), moves)
	} else {
		moves := "std::move(service_grpc_stub)"
		if ann.HasLocationsMixin() {
			moves += ", std::move(service_locations_stub)"
		}
		fmt.Fprintf(&b, "  std::shared_ptr<%s> stub =\n    std::make_shared<%s>(\n      %s,\n      google::longrunning::Operations::NewStub(channel));\n",
			ann.StubClassName(), ann.DefaultStubClassName(), moves)
	}
	return b.String()
}

func (ann *serviceAnnotations) ClientHeader() string {
	return ann.BaseFileName + "_client.h"
}

func (ann *serviceAnnotations) ClientSource() string {
	return ann.BaseFileName + "_client.cc"
}

func (ann *serviceAnnotations) ConnectionHeader() string {
	return ann.BaseFileName + "_connection.h"
}

func (ann *serviceAnnotations) ConnectionSource() string {
	return ann.BaseFileName + "_connection.cc"
}

func (ann *serviceAnnotations) IdempotencyPolicyHeader() string {
	return ann.BaseFileName + "_connection_idempotency_policy.h"
}

func (ann *serviceAnnotations) IdempotencyPolicySource() string {
	return ann.BaseFileName + "_connection_idempotency_policy.cc"
}

func (ann *serviceAnnotations) OptionsHeader() string {
	return ann.BaseFileName + "_options.h"
}

func (ann *serviceAnnotations) RestConnectionHeader() string {
	return ann.BaseFileName + "_rest_connection.h"
}

func (ann *serviceAnnotations) RestConnectionSource() string {
	return ann.BaseFileName + "_rest_connection.cc"
}

func (ann *serviceAnnotations) MockConnectionHeader() string {
	return filepath.Join("mocks", "mock_"+ann.BaseFileName+"_connection.h")
}

func (ann *serviceAnnotations) OptionDefaultsHeader() string {
	return filepath.Join("internal", ann.BaseFileName+"_option_defaults.h")
}

func (ann *serviceAnnotations) OptionDefaultsSource() string {
	return filepath.Join("internal", ann.BaseFileName+"_option_defaults.cc")
}

func (ann *serviceAnnotations) TracingConnectionHeader() string {
	return filepath.Join("internal", ann.BaseFileName+"_tracing_connection.h")
}

func (ann *serviceAnnotations) TracingConnectionSource() string {
	return filepath.Join("internal", ann.BaseFileName+"_tracing_connection.cc")
}

func (ann *serviceAnnotations) RetryTraitsHeader() string {
	return filepath.Join("internal", ann.BaseFileName+"_retry_traits.h")
}

func (ann *serviceAnnotations) SourcesSource() string {
	return filepath.Join("internal", ann.BaseFileName+"_sources.cc")
}

func (ann *serviceAnnotations) ConnectionImplHeader() string {
	return filepath.Join("internal", ann.BaseFileName+"_connection_impl.h")
}

func (ann *serviceAnnotations) ConnectionImplSource() string {
	return filepath.Join("internal", ann.BaseFileName+"_connection_impl.cc")
}

func (ann *serviceAnnotations) StubFactoryHeader() string {
	return filepath.Join("internal", ann.BaseFileName+"_stub_factory.h")
}

func (ann *serviceAnnotations) StubFactorySource() string {
	return filepath.Join("internal", ann.BaseFileName+"_stub_factory.cc")
}

func (ann *serviceAnnotations) AuthDecoratorHeader() string {
	return filepath.Join("internal", ann.BaseFileName+"_auth_decorator.h")
}

func (ann *serviceAnnotations) AuthDecoratorSource() string {
	return filepath.Join("internal", ann.BaseFileName+"_auth_decorator.cc")
}

func (ann *serviceAnnotations) LoggingDecoratorHeader() string {
	return filepath.Join("internal", ann.BaseFileName+"_logging_decorator.h")
}

func (ann *serviceAnnotations) LoggingDecoratorSource() string {
	return filepath.Join("internal", ann.BaseFileName+"_logging_decorator.cc")
}

func (ann *serviceAnnotations) MetadataDecoratorHeader() string {
	return filepath.Join("internal", ann.BaseFileName+"_metadata_decorator.h")
}

func (ann *serviceAnnotations) MetadataDecoratorSource() string {
	return filepath.Join("internal", ann.BaseFileName+"_metadata_decorator.cc")
}

func (ann *serviceAnnotations) StubHeader() string {
	return filepath.Join("internal", ann.BaseFileName+"_stub.h")
}

func (ann *serviceAnnotations) StubSource() string {
	return filepath.Join("internal", ann.BaseFileName+"_stub.cc")
}

func (ann *serviceAnnotations) TracingStubHeader() string {
	return filepath.Join("internal", ann.BaseFileName+"_tracing_stub.h")
}

func (ann *serviceAnnotations) TracingStubSource() string {
	return filepath.Join("internal", ann.BaseFileName+"_tracing_stub.cc")
}

func (ann *serviceAnnotations) RoundRobinDecoratorHeader() string {
	return filepath.Join("internal", ann.BaseFileName+"_round_robin_decorator.h")
}

func (ann *serviceAnnotations) RoundRobinDecoratorSource() string {
	return filepath.Join("internal", ann.BaseFileName+"_round_robin_decorator.cc")
}

func (ann *serviceAnnotations) RestConnectionImplHeader() string {
	return filepath.Join("internal", ann.BaseFileName+"_rest_connection_impl.h")
}

func (ann *serviceAnnotations) RestConnectionImplSource() string {
	return filepath.Join("internal", ann.BaseFileName+"_rest_connection_impl.cc")
}

func (ann *serviceAnnotations) RestLoggingDecoratorHeader() string {
	return filepath.Join("internal", ann.BaseFileName+"_rest_logging_decorator.h")
}

func (ann *serviceAnnotations) RestLoggingDecoratorSource() string {
	return filepath.Join("internal", ann.BaseFileName+"_rest_logging_decorator.cc")
}

func (ann *serviceAnnotations) RestMetadataDecoratorHeader() string {
	return filepath.Join("internal", ann.BaseFileName+"_rest_metadata_decorator.h")
}

func (ann *serviceAnnotations) RestMetadataDecoratorSource() string {
	return filepath.Join("internal", ann.BaseFileName+"_rest_metadata_decorator.cc")
}

func (ann *serviceAnnotations) RestStubHeader() string {
	return filepath.Join("internal", ann.BaseFileName+"_rest_stub.h")
}

func (ann *serviceAnnotations) RestStubSource() string {
	return filepath.Join("internal", ann.BaseFileName+"_rest_stub.cc")
}

func (ann *serviceAnnotations) RestStubFactoryHeader() string {
	return filepath.Join("internal", ann.BaseFileName+"_rest_stub_factory.h")
}

func (ann *serviceAnnotations) RestStubFactorySource() string {
	return filepath.Join("internal", ann.BaseFileName+"_rest_stub_factory.cc")
}

func (ann *serviceAnnotations) SourceCcIncludes() []string {
	var includes []string
	if !ann.OmitClient {
		includes = append(includes, filepath.ToSlash(filepath.Join(ann.ProductPath, ann.ClientSource())))
	}
	if !ann.OmitConnection {
		includes = append(includes,
			filepath.ToSlash(filepath.Join(ann.ProductPath, ann.ConnectionSource())),
			filepath.ToSlash(filepath.Join(ann.ProductPath, ann.IdempotencyPolicySource())),
		)
	}
	if ann.HasRest {
		includes = append(includes, filepath.ToSlash(filepath.Join(ann.ProductPath, ann.RestConnectionSource())))
	}
	if !ann.OmitConnection {
		includes = append(includes,
			filepath.ToSlash(filepath.Join(ann.ProductPath, ann.OptionDefaultsSource())),
			filepath.ToSlash(filepath.Join(ann.ProductPath, ann.TracingConnectionSource())),
		)
	}
	if ann.HasGrpc {
		if !ann.OmitConnection {
			includes = append(includes, filepath.ToSlash(filepath.Join(ann.ProductPath, ann.ConnectionImplSource())))
		}
		if !ann.OmitStubFactory {
			includes = append(includes, filepath.ToSlash(filepath.Join(ann.ProductPath, ann.StubFactorySource())))
		}
		includes = append(includes,
			filepath.ToSlash(filepath.Join(ann.ProductPath, ann.AuthDecoratorSource())),
			filepath.ToSlash(filepath.Join(ann.ProductPath, ann.LoggingDecoratorSource())),
			filepath.ToSlash(filepath.Join(ann.ProductPath, ann.MetadataDecoratorSource())),
			filepath.ToSlash(filepath.Join(ann.ProductPath, ann.StubSource())),
			filepath.ToSlash(filepath.Join(ann.ProductPath, ann.TracingStubSource())),
		)
		if ann.HasRoundRobin {
			includes = append(includes, filepath.ToSlash(filepath.Join(ann.ProductPath, ann.RoundRobinDecoratorSource())))
		}
	}
	if ann.HasRest {
		includes = append(includes,
			filepath.ToSlash(filepath.Join(ann.ProductPath, ann.RestConnectionImplSource())),
			filepath.ToSlash(filepath.Join(ann.ProductPath, ann.RestLoggingDecoratorSource())),
			filepath.ToSlash(filepath.Join(ann.ProductPath, ann.RestMetadataDecoratorSource())),
			filepath.ToSlash(filepath.Join(ann.ProductPath, ann.RestStubSource())),
			filepath.ToSlash(filepath.Join(ann.ProductPath, ann.RestStubFactorySource())),
		)
	}
	slices.Sort(includes)
	return includes
}

func (ann *serviceAnnotations) generatedFiles(forwardingRelDir string) []language.GeneratedFile {
	var files []language.GeneratedFile

	// Public client & connection
	if !ann.OmitClient {
		files = append(files,
			language.GeneratedFile{TemplatePath: "templates/client.h.mustache", OutputPath: ann.ClientHeader()},
			language.GeneratedFile{TemplatePath: "templates/client.cc.mustache", OutputPath: ann.ClientSource()},
		)
	}
	if !ann.OmitConnection {
		files = append(files,
			language.GeneratedFile{TemplatePath: "templates/connection.h.mustache", OutputPath: ann.ConnectionHeader()},
			language.GeneratedFile{TemplatePath: "templates/connection.cc.mustache", OutputPath: ann.ConnectionSource()},
			language.GeneratedFile{TemplatePath: "templates/connection_idempotency_policy.h.mustache", OutputPath: ann.IdempotencyPolicyHeader()},
			language.GeneratedFile{TemplatePath: "templates/connection_idempotency_policy.cc.mustache", OutputPath: ann.IdempotencyPolicySource()},
			language.GeneratedFile{TemplatePath: "templates/options.h.mustache", OutputPath: ann.OptionsHeader()},
		)
	}
	if ann.HasRest {
		files = append(files,
			language.GeneratedFile{TemplatePath: "templates/rest_connection.h.mustache", OutputPath: ann.RestConnectionHeader()},
			language.GeneratedFile{TemplatePath: "templates/rest_connection.cc.mustache", OutputPath: ann.RestConnectionSource()},
		)
	}

	// Mock headers
	if !ann.OmitConnection {
		files = append(files,
			language.GeneratedFile{TemplatePath: "templates/mocks/mock_connection.h.mustache", OutputPath: ann.MockConnectionHeader()},
		)
	}

	// Internal common & tracing
	if !ann.OmitConnection {
		files = append(files,
			language.GeneratedFile{TemplatePath: "templates/internal/option_defaults.h.mustache", OutputPath: ann.OptionDefaultsHeader()},
			language.GeneratedFile{TemplatePath: "templates/internal/option_defaults.cc.mustache", OutputPath: ann.OptionDefaultsSource()},
			language.GeneratedFile{TemplatePath: "templates/internal/tracing_connection.h.mustache", OutputPath: ann.TracingConnectionHeader()},
			language.GeneratedFile{TemplatePath: "templates/internal/tracing_connection.cc.mustache", OutputPath: ann.TracingConnectionSource()},
		)
		if ann.HasRetryTraits {
			files = append(files,
				language.GeneratedFile{TemplatePath: "templates/internal/retry_traits.h.mustache", OutputPath: ann.RetryTraitsHeader()},
			)
		}
	}
	if !ann.OmitClient {
		files = append(files,
			language.GeneratedFile{TemplatePath: "templates/internal/sources.cc.mustache", OutputPath: ann.SourcesSource()},
		)
	}

	// Internal gRPC
	if ann.HasGrpc {
		if !ann.OmitConnection {
			files = append(files,
				language.GeneratedFile{TemplatePath: "templates/internal/connection_impl.h.mustache", OutputPath: ann.ConnectionImplHeader()},
				language.GeneratedFile{TemplatePath: "templates/internal/connection_impl.cc.mustache", OutputPath: ann.ConnectionImplSource()},
			)
		}
		if !ann.OmitStubFactory {
			files = append(files,
				language.GeneratedFile{TemplatePath: "templates/internal/stub_factory.h.mustache", OutputPath: ann.StubFactoryHeader()},
				language.GeneratedFile{TemplatePath: "templates/internal/stub_factory.cc.mustache", OutputPath: ann.StubFactorySource()},
			)
		}
		files = append(files,
			language.GeneratedFile{TemplatePath: "templates/internal/auth_decorator.h.mustache", OutputPath: ann.AuthDecoratorHeader()},
			language.GeneratedFile{TemplatePath: "templates/internal/auth_decorator.cc.mustache", OutputPath: ann.AuthDecoratorSource()},
			language.GeneratedFile{TemplatePath: "templates/internal/logging_decorator.h.mustache", OutputPath: ann.LoggingDecoratorHeader()},
			language.GeneratedFile{TemplatePath: "templates/internal/logging_decorator.cc.mustache", OutputPath: ann.LoggingDecoratorSource()},
			language.GeneratedFile{TemplatePath: "templates/internal/metadata_decorator.h.mustache", OutputPath: ann.MetadataDecoratorHeader()},
			language.GeneratedFile{TemplatePath: "templates/internal/metadata_decorator.cc.mustache", OutputPath: ann.MetadataDecoratorSource()},
			language.GeneratedFile{TemplatePath: "templates/internal/stub.h.mustache", OutputPath: ann.StubHeader()},
			language.GeneratedFile{TemplatePath: "templates/internal/stub.cc.mustache", OutputPath: ann.StubSource()},
			language.GeneratedFile{TemplatePath: "templates/internal/tracing_stub.h.mustache", OutputPath: ann.TracingStubHeader()},
			language.GeneratedFile{TemplatePath: "templates/internal/tracing_stub.cc.mustache", OutputPath: ann.TracingStubSource()},
		)
		if ann.HasRoundRobin {
			files = append(files,
				language.GeneratedFile{TemplatePath: "templates/internal/round_robin_decorator.h.mustache", OutputPath: ann.RoundRobinDecoratorHeader()},
				language.GeneratedFile{TemplatePath: "templates/internal/round_robin_decorator.cc.mustache", OutputPath: ann.RoundRobinDecoratorSource()},
			)
		}
	}

	// Internal REST
	if ann.HasRest {
		files = append(files,
			language.GeneratedFile{TemplatePath: "templates/internal/rest_connection_impl.h.mustache", OutputPath: ann.RestConnectionImplHeader()},
			language.GeneratedFile{TemplatePath: "templates/internal/rest_connection_impl.cc.mustache", OutputPath: ann.RestConnectionImplSource()},
			language.GeneratedFile{TemplatePath: "templates/internal/rest_logging_decorator.h.mustache", OutputPath: ann.RestLoggingDecoratorHeader()},
			language.GeneratedFile{TemplatePath: "templates/internal/rest_logging_decorator.cc.mustache", OutputPath: ann.RestLoggingDecoratorSource()},
			language.GeneratedFile{TemplatePath: "templates/internal/rest_metadata_decorator.h.mustache", OutputPath: ann.RestMetadataDecoratorHeader()},
			language.GeneratedFile{TemplatePath: "templates/internal/rest_metadata_decorator.cc.mustache", OutputPath: ann.RestMetadataDecoratorSource()},
			language.GeneratedFile{TemplatePath: "templates/internal/rest_stub.h.mustache", OutputPath: ann.RestStubHeader()},
			language.GeneratedFile{TemplatePath: "templates/internal/rest_stub.cc.mustache", OutputPath: ann.RestStubSource()},
			language.GeneratedFile{TemplatePath: "templates/internal/rest_stub_factory.h.mustache", OutputPath: ann.RestStubFactoryHeader()},
			language.GeneratedFile{TemplatePath: "templates/internal/rest_stub_factory.cc.mustache", OutputPath: ann.RestStubFactorySource()},
		)
	}

	// Top-level forwarding headers
	if ann.ForwardingPath != "" && forwardingRelDir != "" {
		if !ann.OmitClient {
			files = append(files,
				language.GeneratedFile{TemplatePath: "templates/forwarding/client.h.mustache", OutputPath: filepath.Join(forwardingRelDir, ann.ClientHeader())},
			)
		}
		if !ann.OmitConnection {
			files = append(files,
				language.GeneratedFile{TemplatePath: "templates/forwarding/connection.h.mustache", OutputPath: filepath.Join(forwardingRelDir, ann.ConnectionHeader())},
				language.GeneratedFile{TemplatePath: "templates/forwarding/connection_idempotency_policy.h.mustache", OutputPath: filepath.Join(forwardingRelDir, ann.IdempotencyPolicyHeader())},
				language.GeneratedFile{TemplatePath: "templates/forwarding/options.h.mustache", OutputPath: filepath.Join(forwardingRelDir, ann.OptionsHeader())},
				language.GeneratedFile{TemplatePath: "templates/forwarding/mock_connection.h.mustache", OutputPath: filepath.Join(forwardingRelDir, ann.MockConnectionHeader())},
			)
		}
	}

	return files
}

func (c *codec) annotateService(service *api.Service) *serviceAnnotations {
	codes := c.retryableStatusCodesForService(service.Name)
	var endpointLocationStyle string
	if c.Cpp != nil {
		endpointLocationStyle = c.Cpp.EndpointLocationStyle
	}
	protoGrpcPath := ""
	if c.Library != nil && len(c.Library.APIs) > 0 && c.Library.APIs[0].Path != "" {
		protoGrpcPath = strings.TrimSuffix(c.Library.APIs[0].Path, ".proto") + ".grpc.pb.h"
	}

	serviceEndpointEnvVar := "GOOGLE_CLOUD_CPP_" + strings.ToUpper(camelCaseToSnakeCase(service.Name)) + "_ENDPOINT"
	emulatorEndpointEnvVar := ""
	if c.Cpp != nil {
		if c.Cpp.ServiceEndpointEnvVar != "" {
			serviceEndpointEnvVar = c.Cpp.ServiceEndpointEnvVar
		}
		if c.Cpp.EmulatorEndpointEnvVar != "" {
			emulatorEndpointEnvVar = c.Cpp.EmulatorEndpointEnvVar
		}
	}

	ann := &serviceAnnotations{
		Name:                   service.Name,
		BaseFileName:           serviceNameToFileName(service.Name),
		ProductPath:            c.Cpp.ProductPath,
		ForwardingPath:         c.Cpp.ForwardingProductPath,
		HasGrpc:                c.hasGrpc(),
		HasRest:                c.hasRest(),
		HasRoundRobin:          c.hasRoundRobin(),
		HasRetryTraits:         len(codes) > 0,
		RetryableStatusCodes:   codes,
		EndpointLocationStyle:  endpointLocationStyle,
		OmitClient:             c.Cpp.OmitClient,
		OmitConnection:         c.Cpp.OmitConnection,
		OmitStubFactory:        c.Cpp.OmitStubFactory,
		CopyrightYear:          c.copyrightYear(),
		Service:                service,
		ProtoGrpcHeaderPath:    protoGrpcPath,
		ServiceEndpointEnvVar:  serviceEndpointEnvVar,
		EmulatorEndpointEnvVar: emulatorEndpointEnvVar,
	}

	var methods []*methodAnnotations
	var stubMethods []*methodAnnotations
	var asyncMethods []*methodAnnotations
	var asyncStubMethods []*methodAnnotations

	for _, m := range service.Methods {
		if c.isOmittedMethod(service.Name, m) {
			if c.isGenAsyncRpc(service.Name, m.Name) {
				mAnn := c.annotateMethod(service, m, ann)
				stubAsync := *mAnn
				asyncStubMethods = append(asyncStubMethods, &stubAsync)
				if !mAnn.IsStreaming() && !mAnn.IsLRO {
					asyncMethods = append(asyncMethods, &stubAsync)
				}
			}
			continue
		}

		mAnn := c.annotateMethod(service, m, ann)
		stubMethods = append(stubMethods, mAnn)
		if !mAnn.IsStreamingWrite() {
			methods = append(methods, mAnn)
		}
		if c.isGenAsyncRpc(service.Name, m.Name) {
			stubAsync := *mAnn
			asyncStubMethods = append(asyncStubMethods, &stubAsync)
			if !mAnn.IsStreaming() && !mAnn.IsLRO {
				asyncMethods = append(asyncMethods, &stubAsync)
			}
		}
	}
	ann.Methods = methods
	ann.StubMethods = stubMethods
	ann.AsyncMethods = asyncMethods
	ann.AsyncStubMethods = asyncStubMethods

	service.Codec = ann
	return ann
}

func (c *codec) isOmittedMethod(serviceName string, m *api.Method) bool {
	if m.IsLroPoller {
		return true
	}
	if m.SourceServiceID != "" && m.Service != nil && m.SourceServiceID != m.Service.ID {
		if m.PathInfo == nil {
			return true
		}
	}
	if c.Cpp == nil {
		return false
	}
	methodName := m.Name
	qualifiedName := serviceName + "." + methodName
	return slices.ContainsFunc(c.Cpp.OmittedRPCs, func(o string) bool {
		return o == methodName || o == qualifiedName
	})
}

func (c *codec) isGenAsyncRpc(serviceName, methodName string) bool {
	if c.Cpp == nil {
		return false
	}
	qualifiedName := serviceName + "." + methodName
	return slices.ContainsFunc(c.Cpp.GenAsyncRPCs, func(o string) bool {
		return o == methodName || o == qualifiedName
	})
}

func (c *codec) annotateModel() error {
	for _, s := range c.Model.Services {
		if c.isOmittedService(s) {
			continue
		}
		c.annotateService(s)
	}
	return nil
}
