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
	"regexp"
	"slices"
	"strings"

	"github.com/googleapis/librarian/internal/sidekick/api"
	"github.com/googleapis/librarian/internal/sidekick/language"
)

type serviceAnnotations struct {
	Name                    string
	BaseFileName            string
	ProductPath             string
	ForwardingPath          string
	HasGrpc                 bool
	HasRest                 bool
	HasRoundRobin           bool
	HasRetryTraits          bool
	RetryableStatusCodes    []string
	EndpointLocationStyle   string
	OmitClient              bool
	OmitConnection          bool
	OmitStubFactory         bool
	CopyrightYear           string
	Service                 *api.Service
	Methods                 []*methodAnnotations
	StubMethods             []*methodAnnotations
	AsyncMethods            []*methodAnnotations
	AsyncStubMethods        []*methodAnnotations
	ProtoSourceFile         string
	ProtoGrpcHeaderPath     string
	ServiceEndpointEnvVar   string
	EmulatorEndpointEnvVar  string
	AdditionalPbHeaderPaths []string
}

func (ann *serviceAnnotations) SourcesCopyrightYear() string {
	if ann.CopyrightYear < "2024" {
		return "2024"
	}
	return ann.CopyrightYear
}

func (ann *serviceAnnotations) OptionsGroup() string {
	parts := strings.Split(strings.Trim(ann.ProductPath, "/"), "/")
	idx := -1
	for i, p := range parts {
		if p == "golden" {
			idx = i
			break
		}
	}
	if idx == -1 {
		if len(parts) > 2 && parts[0] == "google" && parts[1] == "cloud" {
			idx = 2
		} else if len(parts) > 0 {
			idx = len(parts) - 1
		}
	}
	if idx >= 0 {
		prefixAndLib := parts[:idx+1]
		return strings.Join(prefixAndLib, "-") + "-options"
	}
	return "options"
}

func (ann *serviceAnnotations) ProtoHeaderPath() string {
	if ann.ProtoSourceFile == "" {
		return ""
	}
	return strings.TrimSuffix(ann.ProtoSourceFile, ".proto") + ".pb.h"
}

func (ann *serviceAnnotations) PbIncludeByTransport() string {
	if ann.HasGrpc {
		return ann.ProtoGrpcHeaderPath
	}
	return ann.ProtoHeaderPath()
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
	return len(ann.AsyncStubMethods) > 0 || ann.HasLRO()
}

func (ann *serviceAnnotations) HasRestAsyncRetryLoop() bool {
	return len(ann.AsyncStubMethods) > 0
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

func (ann *serviceAnnotations) HasMessageWithMapField() bool {
	if ann.Service == nil {
		return false
	}
	for _, m := range ann.Service.Methods {
		if m.InputType != nil {
			for _, f := range m.InputType.Fields {
				if f.Map {
					return true
				}
			}
		}
		if m.OutputType != nil {
			for _, f := range m.OutputType.Fields {
				if f.Map {
					return true
				}
			}
		}
	}
	return false
}

func (ann *serviceAnnotations) FormatClassComments() string {
	doc := ""
	if ann.Service != nil {
		doc = ann.Service.Documentation
	}
	formatted := ""
	if doc == "" {
		formatted = "/// " + ann.Name + "Client"
	} else {
		lines := strings.Split(strings.TrimSuffix(doc, "\n"), "\n")
		var out []string
		for _, l := range lines {
			l = strings.TrimPrefix(l, " ")
			if l == "" {
				out = append(out, "///")
			} else {
				out = append(out, "/// "+l)
			}
		}
		formatted = strings.Join(out, "\n")
		formatted = strings.ReplaceAll(formatted, "[groups](#google.monitoring.v3.Group)", "[groups][google.monitoring.v3.Group]")
		if ann.Service != nil {
			formatted = strings.ReplaceAll(formatted, ann.Service.Name, ann.Name)
		}
	}

	re := regexp.MustCompile(`\[([a-zA-Z0-9_.]+)\]`)
	matches := re.FindAllStringSubmatch(doc, -1)
	refs := make(map[string]protoDefinitionLocation)
	for _, match := range matches {
		if len(match) > 1 {
			if loc, ok := findProtoLocation(match[1]); ok {
				refs[match[1]] = loc
			}
		}
	}
	var trailer string
	var refKeys []string
	for k := range refs {
		refKeys = append(refKeys, k)
	}
	slices.Sort(refKeys)
	for _, k := range refKeys {
		loc := refs[k]
		trailer += fmt.Sprintf("\n/// [%s]: @googleapis_reference_link{%s#L%d}", k, loc.Filename, loc.Line)
	}
	if trailer != "" {
		trailer += "\n///"
	}

	fixedComment := "\n///\n/// @par Equality\n" +
		"///\n" +
		"/// Instances of this class created via copy-construction or copy-assignment\n" +
		"/// always compare equal. Instances created with equal\n" +
		"/// `std::shared_ptr<*Connection>` objects compare equal. Objects that compare\n" +
		"/// equal share the same underlying resources.\n" +
		"///\n" +
		"/// @par Performance\n" +
		"///\n" +
		"/// Creating a new instance of this class is a relatively expensive operation,\n" +
		"/// new objects establish new connections to the service. In contrast,\n" +
		"/// copy-construction, move-construction, and the corresponding assignment\n" +
		"/// operations are relatively efficient as the copies share all underlying\n" +
		"/// resources.\n" +
		"///\n" +
		"/// @par Thread Safety\n" +
		"///\n" +
		"/// Concurrent access to different instances of this class, even if they compare\n" +
		"/// equal, is guaranteed to work. Two or more threads operating on the same\n" +
		"/// instance of this class is not guaranteed to work. Since copy-construction\n" +
		"/// and move-construction is a relatively efficient operation, consider using\n" +
		"/// such a copy when using this class from multiple threads.\n" +
		"///"

	result := "///\n" + formatted + fixedComment + trailer
	result = strings.ReplaceAll(result, "///  ", "/// ")
	return result
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

func (ann *serviceAnnotations) OptionDefaultsEndpointArg() string {
	switch ann.EndpointLocationStyle {
	case "LOCATION_DEPENDENT":
		return fmt.Sprintf("absl::StrCat(location, \"-\", %q)", ann.DefaultEndpoint())
	case "LOCATION_DEPENDENT_COMPAT":
		return fmt.Sprintf("absl::StrCat(location, location.empty() ? \"\" : \"-\", %q)", ann.DefaultEndpoint())
	case "LOCATION_OPTIONALLY_DEPENDENT":
		return fmt.Sprintf("// optional location tag for generating docs\n      absl::StrCat(location, location.empty() ? \"\" : \"-\", %q)", ann.DefaultEndpoint())
	default:
		return fmt.Sprintf("%q", ann.DefaultEndpoint())
	}
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

func (ann *serviceAnnotations) HasApiVersion() bool {
	return ann.ApiVersion() != ""
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
	return slices.ContainsFunc(ann.StubMethods, func(m *methodAnnotations) bool {
		return m.Method != nil && m.Method.SourceService != nil && m.Method.Service != nil &&
			m.Method.SourceService.Name != m.Method.Service.Name && m.Method.SourceService.Name == "Operations"
	})
}

func (ann *serviceAnnotations) StubFactoryMakeDefaultStub() string {
	var b strings.Builder
	cppStubType := protoNameToCppName(ann.ServiceGrpcFqn())
	fmt.Fprintf(&b, "  auto service_grpc_stub = %s::NewStub(channel);\n", cppStubType)
	if ann.HasOperationsMixin() && !ann.HasGrpcLRO() {
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

func (ann *serviceAnnotations) IdempotencyPolicyHeaderSystemIncludes() []string {
	var includes []string
	ext := ".grpc.pb.h"
	if !ann.HasGrpc {
		ext = ".pb.h"
	}
	if ann.PbIncludeByTransport() != "" {
		includes = append(includes, ann.PbIncludeByTransport())
	}
	if ann.HasLocationsMixin() {
		includes = append(includes, "google/cloud/location/locations"+ext)
	}
	if ann.HasIamMixin() {
		includes = append(includes, "google/iam/v1/iam_policy"+ext)
	}
	if ann.HasOperationsMixin() {
		includes = append(includes, "google/longrunning/operations"+ext)
	}
	includes = append(includes, "memory")
	return includes
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

func (ann *serviceAnnotations) ConnectionSourceLocalIncludes() []string {
	var includes []string
	if ann.OptionsHeaderPath() != "" {
		includes = append(includes, ann.OptionsHeaderPath())
	}
	if ann.HasGrpc && ann.ConnectionImplHeaderPath() != "" {
		includes = append(includes, ann.ConnectionImplHeaderPath())
	}
	if ann.OptionDefaultsHeaderPath() != "" {
		includes = append(includes, ann.OptionDefaultsHeaderPath())
	}
	if ann.HasGrpc && !ann.OmitStubFactory && ann.StubFactoryHeaderPath() != "" {
		includes = append(includes, ann.StubFactoryHeaderPath())
	}
	if ann.TracingConnectionHeaderPath() != "" {
		includes = append(includes, ann.TracingConnectionHeaderPath())
	}
	includes = append(includes,
		"google/cloud/background_threads.h",
		"google/cloud/common_options.h",
		"google/cloud/credentials.h",
		"google/cloud/grpc_options.h",
	)
	if ann.HasPaginatedMethod() {
		includes = append(includes, "google/cloud/internal/pagination_range.h")
	}
	includes = append(includes, "google/cloud/internal/unified_grpc_credentials.h")

	slices.Sort(includes)
	return append([]string{ann.ConnectionHeaderPath()}, includes...)
}

func (ann *serviceAnnotations) ConnectionHeaderLocalIncludes() []string {
	var includes []string
	if ann.HasRetryTraits && ann.RetryTraitsHeaderPath() != "" {
		includes = append(includes, ann.RetryTraitsHeaderPath())
	}
	if ann.IdempotencyPolicyHeaderPath() != "" {
		includes = append(includes, ann.IdempotencyPolicyHeaderPath())
	}
	if ann.HasLRO() {
		includes = append(includes, "google/cloud/no_await_tag.h")
	}
	includes = append(includes, "google/cloud/backoff_policy.h")
	if ann.HasLRO() || ann.HasAsyncMethod() {
		includes = append(includes, "google/cloud/future.h")
	}
	includes = append(includes, "google/cloud/internal/retry_policy_impl.h")
	includes = append(includes, "google/cloud/options.h")
	if ann.HasLRO() {
		includes = append(includes, "google/cloud/polling_policy.h")
	}
	includes = append(includes, "google/cloud/status_or.h")
	if ann.HasStreamingReadMethod() || ann.HasPaginatedMethod() {
		includes = append(includes, "google/cloud/stream_range.h")
	}
	if ann.HasBidirStreamingMethod() {
		includes = append(includes, "google/cloud/internal/async_read_write_stream_impl.h")
	}
	includes = append(includes, "google/cloud/version.h")

	slices.Sort(includes)
	return includes
}

func (ann *serviceAnnotations) ConnectionHeaderSystemIncludes() []string {
	var includes []string
	if len(ann.AdditionalPbHeaderPaths) > 0 {
		includes = append(includes, ann.AdditionalPbHeaderPaths...)
	}
	if ann.ProtoHeaderPath() != "" {
		includes = append(includes, ann.ProtoHeaderPath())
	}
	if ann.HasGrpcLRO() {
		includes = append(includes, "google/longrunning/operations.grpc.pb.h")
	}
	includes = append(includes, "memory")
	if ann.HasGrpc && ann.HasEndpointLocation() {
		includes = append(includes, "string")
	}
	slices.Sort(includes)
	return includes
}

func (ann *serviceAnnotations) TransientErrorsComment() string {
	if len(ann.RetryableStatusCodes) == 0 {
		return ""
	}
	var comment strings.Builder
	comment.WriteString("\n *\n * In this class the following status codes are treated as transient errors:")
	for _, code := range ann.RetryableStatusCodes {
		fmt.Fprintf(&comment, "\n * - [`%s`](@ref google::cloud::StatusCode)", code)
	}
	return comment.String()
}

func (ann *serviceAnnotations) ConnectionImplHeaderIncludes() []string {
	var includes []string
	includes = append(includes,
		ann.IdempotencyPolicyHeaderPath(),
		ann.OptionsHeaderPath(),
		ann.StubHeaderPath(),
		ann.ConnectionHeaderPath(),
	)
	if ann.HasRetryTraits {
		includes = append(includes, ann.RetryTraitsHeaderPath())
	}
	if ann.HasBidirStreamingMethod() {
		includes = append(includes, "google/cloud/async_streaming_read_write_rpc.h")
	}
	includes = append(includes,
		"google/cloud/background_threads.h",
		"google/cloud/backoff_policy.h",
	)
	if ann.HasGrpcLRO() {
		includes = append(includes, "google/cloud/future.h")
	}
	if ann.HasRequestId() {
		includes = append(includes, "google/cloud/internal/invocation_id_generator.h")
	}
	includes = append(includes, "google/cloud/options.h")
	if ann.HasGrpcLRO() {
		includes = append(includes, "google/cloud/polling_policy.h")
	}
	includes = append(includes, "google/cloud/status_or.h")
	if ann.HasStreamingReadMethod() || ann.HasPaginatedMethod() {
		includes = append(includes, "google/cloud/stream_range.h")
	}
	includes = append(includes, "google/cloud/version.h")
	slices.Sort(includes)
	return includes
}

func (ann *serviceAnnotations) NeedsCompletionQueue() bool {
	return ann.HasAsyncMethod() || ann.HasBidirStreamingMethod()
}

func (ann *serviceAnnotations) StubHeaderIncludes() []string {
	var includes []string
	if ann.HasBidirStreamingMethod() {
		includes = append(includes, "google/cloud/async_streaming_read_write_rpc.h")
	}
	if ann.NeedsCompletionQueue() {
		includes = append(includes, "google/cloud/completion_queue.h")
	}
	if ann.HasAsyncMethod() {
		includes = append(includes, "google/cloud/future.h")
	}
	if ann.HasAsynchronousStreamingReadMethod() {
		includes = append(includes, "google/cloud/internal/async_streaming_read_rpc.h")
	}
	if ann.HasAsynchronousStreamingWriteMethod() {
		includes = append(includes, "google/cloud/internal/async_streaming_write_rpc.h")
	}
	if ann.HasStreamingReadMethod() {
		includes = append(includes, "google/cloud/internal/streaming_read_rpc.h")
	}
	if ann.HasStreamingWriteMethod() {
		includes = append(includes, "google/cloud/internal/streaming_write_rpc.h")
	}
	includes = append(includes,
		"google/cloud/options.h",
		"google/cloud/status_or.h",
		"google/cloud/version.h",
	)
	slices.Sort(includes)
	return includes
}

func (ann *serviceAnnotations) StubSourceIncludes() []string {
	var includes []string
	includes = append(includes, ann.StubHeaderPath())
	if ann.HasBidirStreamingMethod() {
		includes = append(includes, "google/cloud/internal/async_read_write_stream_impl.h")
	}
	if ann.HasAsynchronousStreamingReadMethod() {
		includes = append(includes, "google/cloud/internal/async_streaming_read_rpc_impl.h")
	}
	if ann.HasAsynchronousStreamingWriteMethod() {
		includes = append(includes, "google/cloud/internal/async_streaming_write_rpc_impl.h")
	}
	if ann.HasStreamingWriteMethod() {
		includes = append(includes, "google/cloud/internal/streaming_write_rpc_impl.h")
	}
	includes = append(includes,
		"google/cloud/grpc_error_delegate.h",
		"google/cloud/status_or.h",
	)
	slices.Sort(includes)
	return includes
}

func (ann *serviceAnnotations) StubHeaderSystemIncludes() []string {
	var includes []string
	if len(ann.AdditionalPbHeaderPaths) > 0 {
		includes = append(includes, ann.AdditionalPbHeaderPaths...)
	}
	if ann.HasLocationsMixin() {
		includes = append(includes, "google/cloud/location/locations.grpc.pb.h")
	}
	if ann.HasIamMixin() {
		includes = append(includes, "google/iam/v1/iam_policy.grpc.pb.h")
	}
	if ann.HasOperationsMixin() {
		includes = append(includes, "google/longrunning/operations.grpc.pb.h")
	}
	if ann.ProtoGrpcHeaderPath != "" {
		includes = append(includes, ann.ProtoGrpcHeaderPath)
	}
	if !ann.HasOperationsMixin() && ann.HasGrpcLRO() {
		includes = append(includes, "google/longrunning/operations.grpc.pb.h")
	}
	includes = append(includes, "memory", "utility")
	return includes
}

func (ann *serviceAnnotations) HasOperationsMixinOrGrpcLRO() bool {
	return ann.HasOperationsMixin() || ann.HasGrpcLRO()
}

func (ann *serviceAnnotations) DefaultStubConstructor() string {
	var b strings.Builder
	stubFqn := protoNameToCppName(ann.ServiceGrpcFqn())
	if ann.HasGrpcLRO() {
		fmt.Fprintf(&b, "  Default%s(\n", ann.StubClassName())
		fmt.Fprintf(&b, "      std::unique_ptr<%s::StubInterface> grpc_stub,\n", stubFqn)
		if ann.HasLocationsMixin() {
			b.WriteString("      std::unique_ptr<google::cloud::location::Locations::StubInterface> locations_stub,\n")
		}
		if ann.HasIamMixin() {
			b.WriteString("      std::unique_ptr<google::iam::v1::IAMPolicy::StubInterface> iampolicy_stub,\n")
		}
		b.WriteString("      std::unique_ptr<google::longrunning::Operations::StubInterface> operations_stub)\n")
		b.WriteString("      : grpc_stub_(std::move(grpc_stub)),\n")
		if ann.HasLocationsMixin() {
			b.WriteString("        locations_stub_(std::move(locations_stub)),\n")
		}
		if ann.HasIamMixin() {
			b.WriteString("        iampolicy_stub_(std::move(iampolicy_stub)),\n")
		}
		b.WriteString("        operations_stub_(std::move(operations_stub)) {}")
	} else {
		hasMixins := ann.HasLocationsMixin() || ann.HasIamMixin() || ann.HasOperationsMixin()
		if !hasMixins {
			fmt.Fprintf(&b, "  explicit Default%s(\n", ann.StubClassName())
			fmt.Fprintf(&b, "      std::unique_ptr<%s::StubInterface> grpc_stub)\n", stubFqn)
			b.WriteString("      : grpc_stub_(std::move(grpc_stub)) {}")
		} else {
			fmt.Fprintf(&b, "  explicit Default%s(\n", ann.StubClassName())
			fmt.Fprintf(&b, "      std::unique_ptr<%s::StubInterface> grpc_stub,\n", stubFqn)
			if ann.HasOperationsMixin() {
				b.WriteString("      std::unique_ptr<google::longrunning::Operations::StubInterface> operations_stub\n")
			}
			if ann.HasIamMixin() {
				b.WriteString(",\n      std::unique_ptr<google::iam::v1::IAMPolicy::StubInterface> iampolicy_stub\n")
			}
			if ann.HasLocationsMixin() {
				b.WriteString(",\n      std::unique_ptr<google::cloud::location::Locations::StubInterface> locations_stub\n")
			}
			b.WriteString(")\n      : grpc_stub_(std::move(grpc_stub)),\n")
			var inits []string
			if ann.HasOperationsMixin() {
				inits = append(inits, "        operations_stub_(std::move(operations_stub))")
			}
			if ann.HasIamMixin() {
				inits = append(inits, "        iampolicy_stub_(std::move(iampolicy_stub))")
			}
			if ann.HasLocationsMixin() {
				inits = append(inits, "        locations_stub_(std::move(locations_stub))")
			}
			b.WriteString(strings.Join(inits, ",\n"))
			b.WriteString(" {}")
		}
	}
	return b.String()
}

func (ann *serviceAnnotations) DefaultStubPrivateMembers() []string {
	var members []string
	stubFqn := protoNameToCppName(ann.ServiceGrpcFqn())
	members = append(members, fmt.Sprintf("std::unique_ptr<%s::StubInterface> grpc_stub_;", stubFqn))
	if ann.HasGrpcLRO() {
		if ann.HasLocationsMixin() {
			members = append(members, "std::unique_ptr<google::cloud::location::Locations::StubInterface> locations_stub_;")
		}
		if ann.HasIamMixin() {
			members = append(members, "std::unique_ptr<google::iam::v1::IAMPolicy::StubInterface> iampolicy_stub_;")
		}
		members = append(members, "std::unique_ptr<google::longrunning::Operations::StubInterface> operations_stub_;")
	} else {
		if ann.HasOperationsMixin() {
			members = append(members, "std::unique_ptr<google::longrunning::Operations::StubInterface> operations_stub_;")
		}
		if ann.HasIamMixin() {
			members = append(members, "std::unique_ptr<google::iam::v1::IAMPolicy::StubInterface> iampolicy_stub_;")
		}
		if ann.HasLocationsMixin() {
			members = append(members, "std::unique_ptr<google::cloud::location::Locations::StubInterface> locations_stub_;")
		}
	}
	return members
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
	protoSourceFile := ""
	protoGrpcPath := ""
	if c.Library != nil && len(c.Library.APIs) > 0 && c.Library.APIs[0].Path != "" {
		protoSourceFile = c.Library.APIs[0].Path
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
	var additionalPbHeaders []string
	if c.Cpp != nil {
		for _, proto := range c.Cpp.AdditionalProtoFiles {
			additionalPbHeaders = append(additionalPbHeaders, strings.TrimSuffix(proto, ".proto")+".pb.h")
		}
	}

	ann := &serviceAnnotations{
		Name:                    service.Name,
		BaseFileName:            serviceNameToFileName(service.Name),
		ProductPath:             c.Cpp.ProductPath,
		ForwardingPath:          c.Cpp.ForwardingProductPath,
		HasGrpc:                 c.hasGrpc(),
		HasRest:                 c.hasRest(),
		HasRoundRobin:           c.hasRoundRobin(),
		HasRetryTraits:          len(codes) > 0,
		RetryableStatusCodes:    codes,
		EndpointLocationStyle:   endpointLocationStyle,
		OmitClient:              c.Cpp.OmitClient,
		OmitConnection:          c.Cpp.OmitConnection,
		OmitStubFactory:         c.Cpp.OmitStubFactory,
		CopyrightYear:           c.copyrightYear(),
		Service:                 service,
		ProtoSourceFile:         protoSourceFile,
		ProtoGrpcHeaderPath:     protoGrpcPath,
		ServiceEndpointEnvVar:   serviceEndpointEnvVar,
		EmulatorEndpointEnvVar:  emulatorEndpointEnvVar,
		AdditionalPbHeaderPaths: additionalPbHeaders,
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

func (ann *serviceAnnotations) RestMethods() []*methodAnnotations {
	var res []*methodAnnotations
	for _, m := range ann.StubMethods {
		if m.HasHttpAnnotation() && !m.IsStreaming() {
			res = append(res, m)
		}
	}
	return res
}

func (ann *serviceAnnotations) RestAsyncMethods() []*methodAnnotations {
	var res []*methodAnnotations
	for _, m := range ann.AsyncMethods {
		if m.HasHttpAnnotation() && !m.IsStreaming() && !m.IsLRO {
			res = append(res, m)
		}
	}
	return res
}

func (ann *serviceAnnotations) HasRestAsyncMethods() bool {
	return len(ann.RestAsyncMethods()) > 0
}

func (ann *serviceAnnotations) RestStubHeaderProtobufIncludes() []string {
	var includes []string
	if len(ann.AdditionalPbHeaderPaths) > 0 {
		includes = append(includes, ann.AdditionalPbHeaderPaths...)
	}
	if ann.HasLocationsMixin() {
		includes = append(includes, "google/cloud/location/locations.pb.h")
	}
	if ann.HasIamMixin() {
		includes = append(includes, "google/iam/v1/iam_policy.pb.h")
	}
	if ann.HasOperationsMixin() {
		includes = append(includes, "google/longrunning/operations.pb.h")
	}
	if ann.ProtoHeaderPath() != "" {
		includes = append(includes, ann.ProtoHeaderPath())
	}
	return includes
}

func (ann *serviceAnnotations) RestStubCcProtobufIncludes() []string {
	var includes []string
	if ann.ProtoHeaderPath() != "" {
		includes = append(includes, ann.ProtoHeaderPath())
	}
	if ann.HasGrpcLRO() {
		includes = append(includes, "google/longrunning/operations.pb.h")
	}
	return includes
}
