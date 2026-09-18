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
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/sidekick/api"
)

// ServiceAnnotations contains C++-specific metadata for a gRPC service and its generated classes.
type ServiceAnnotations struct {
	// ServiceName is the CamelCase name of the service.
	ServiceName string

	// ProductPath is the relative path where generated source and headers are placed.
	ProductPath string

	// ForwardingProductPath is the relative path for top-level forwarding headers, if configured.
	ForwardingProductPath string

	// LibraryName is the library directory name extracted from product path.
	LibraryName string

	// ServiceSubdirectory is any version or sub-path element (e.g. "v1").
	ServiceSubdirectory string

	// C++ namespaces
	Namespace         string
	InternalNamespace string
	MocksNamespace    string

	// FilePathName is the snake_case base filename (e.g. "golden_kitchen_sink").
	FilePathName string

	// CopyrightYear is the copyright year for generated headers.
	CopyrightYear string

	// APIVersion is the proto API version (e.g. "v1").
	APIVersion string

	// OptionsGroupName is the Doxygen options group name.
	OptionsGroupName string

	// Class names
	ClientClassName                  string
	ConnectionClassName              string
	ConnectionImplClassName          string
	IdempotencyClassName             string
	MockConnectionClassName          string
	AuthClassName                    string
	LoggingClassName                 string
	MetadataClassName                string
	StubClassName                    string
	TracingConnectionClassName       string
	TracingStubClassName             string
	RoundRobinClassName              string
	RetryTraitsName                  string
	RetryPolicyName                  string
	LimitedErrorCountRetryPolicyName string
	LimitedTimeRetryPolicyName       string
	ConnectionOptionsName            string
	ConnectionOptionsTraitsName      string

	// Header and source file paths
	ClientHeaderPath            string
	ClientCcPath                string
	ClientSamplesCcPath         string
	ConnectionHeaderPath        string
	ConnectionCcPath            string
	ConnectionImplHeaderPath    string
	ConnectionImplCcPath        string
	StubHeaderPath              string
	StubCcPath                  string
	StubFactoryHeaderPath       string
	StubFactoryCcPath           string
	OptionsHeaderPath           string
	IdempotencyHeaderPath       string
	IdempotencyCcPath           string
	MockConnectionHeaderPath    string
	AuthHeaderPath              string
	AuthCcPath                  string
	LoggingHeaderPath           string
	LoggingCcPath               string
	MetadataHeaderPath          string
	MetadataCcPath              string
	TracingConnectionHeaderPath string
	TracingConnectionCcPath     string
	TracingStubHeaderPath       string
	TracingStubCcPath           string
	RoundRobinHeaderPath        string
	RoundRobinCcPath            string
	OptionDefaultsHeaderPath    string
	OptionDefaultsCcPath        string
	RetryTraitsHeaderPath       string
	SourcesCcPath               string

	// Header include guards
	ClientHeaderIncludeGuard            string
	ConnectionHeaderIncludeGuard        string
	ConnectionImplHeaderIncludeGuard    string
	StubHeaderIncludeGuard              string
	StubFactoryHeaderIncludeGuard       string
	OptionsHeaderIncludeGuard           string
	IdempotencyHeaderIncludeGuard       string
	MockConnectionHeaderIncludeGuard    string
	AuthHeaderIncludeGuard              string
	LoggingHeaderIncludeGuard           string
	MetadataHeaderIncludeGuard          string
	TracingConnectionHeaderIncludeGuard string
	TracingStubHeaderIncludeGuard       string
	RoundRobinHeaderIncludeGuard        string
	OptionDefaultsHeaderIncludeGuard    string
	RetryTraitsHeaderIncludeGuard       string

	// Decorator requirements
	HasRoundRobinDecorator bool
	HasTracingDecorator    bool
	HasLoggingDecorator    bool
	HasMetadataDecorator   bool
	HasAuthDecorator       bool

	// Methods
	Methods      []*api.Method
	AsyncMethods []*api.Method

	// Comments contains formatted Doxygen comments for the service class.
	Comments *CommentAnnotations

	// Options contains C++-specific options and defaults for the service.
	Options *OptionsAnnotations
}

// annotateService enriches an api.Service with C++-specific namespaces, class names, file paths, and decorators.
func annotateService(svc *api.Service, lib *config.Library, model *api.API) *ServiceAnnotations {
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

	prefix, libraryName, serviceSubdir := parseProductPath(productPath)
	_ = prefix
	ns := namespace(productPath, namespaceNormal)
	internalNs := namespace(productPath, namespaceInternal)
	mocksNs := namespace(productPath, namespaceMocks)

	filePathName := serviceNameToFilePath(serviceName)

	apiVersion := ""
	for _, m := range svc.Methods {
		if m.APIVersion != "" {
			apiVersion = m.APIVersion
			break
		}
	}

	methods, asyncMethods := getEffectiveMethods(svc, lib)

	ann := &ServiceAnnotations{
		ServiceName:                      serviceName,
		ProductPath:                      productPath,
		ForwardingProductPath:            forwardingProductPath,
		LibraryName:                      libraryName,
		ServiceSubdirectory:              serviceSubdir,
		Namespace:                        ns,
		InternalNamespace:                internalNs,
		MocksNamespace:                   mocksNs,
		FilePathName:                     filePathName,
		CopyrightYear:                    copyrightYear,
		APIVersion:                       apiVersion,
		OptionsGroupName:                 optionsGroup(productPath),
		ClientClassName:                  serviceName + "Client",
		ConnectionClassName:              serviceName + "Connection",
		ConnectionImplClassName:          serviceName + "ConnectionImpl",
		IdempotencyClassName:             serviceName + "ConnectionIdempotencyPolicy",
		MockConnectionClassName:          "Mock" + serviceName + "Connection",
		AuthClassName:                    serviceName + "Auth",
		LoggingClassName:                 serviceName + "Logging",
		MetadataClassName:                serviceName + "Metadata",
		StubClassName:                    serviceName + "Stub",
		TracingConnectionClassName:       serviceName + "TracingConnection",
		TracingStubClassName:             serviceName + "TracingStub",
		RoundRobinClassName:              serviceName + "RoundRobin",
		RetryTraitsName:                  serviceName + "RetryTraits",
		RetryPolicyName:                  serviceName + "RetryPolicy",
		LimitedErrorCountRetryPolicyName: serviceName + "LimitedErrorCountRetryPolicy",
		LimitedTimeRetryPolicyName:       serviceName + "LimitedTimeRetryPolicy",
		ConnectionOptionsName:            serviceName + "ConnectionOptions",
		ConnectionOptionsTraitsName:      serviceName + "ConnectionOptionsTraits",

		ClientHeaderPath:            productPath + filePathName + "_client.h",
		ClientCcPath:                productPath + filePathName + "_client.cc",
		ClientSamplesCcPath:         productPath + "samples/" + filePathName + "_client_samples.cc",
		ConnectionHeaderPath:        productPath + filePathName + "_connection.h",
		ConnectionCcPath:            productPath + filePathName + "_connection.cc",
		ConnectionImplHeaderPath:    productPath + "internal/" + filePathName + "_connection_impl.h",
		ConnectionImplCcPath:        productPath + "internal/" + filePathName + "_connection_impl.cc",
		StubHeaderPath:              productPath + "internal/" + filePathName + "_stub.h",
		StubCcPath:                  productPath + "internal/" + filePathName + "_stub.cc",
		StubFactoryHeaderPath:       productPath + "internal/" + filePathName + "_stub_factory.h",
		StubFactoryCcPath:           productPath + "internal/" + filePathName + "_stub_factory.cc",
		OptionsHeaderPath:           productPath + filePathName + "_options.h",
		IdempotencyHeaderPath:       productPath + filePathName + "_connection_idempotency_policy.h",
		IdempotencyCcPath:           productPath + filePathName + "_connection_idempotency_policy.cc",
		MockConnectionHeaderPath:    productPath + "mocks/mock_" + filePathName + "_connection.h",
		AuthHeaderPath:              productPath + "internal/" + filePathName + "_auth_decorator.h",
		AuthCcPath:                  productPath + "internal/" + filePathName + "_auth_decorator.cc",
		LoggingHeaderPath:           productPath + "internal/" + filePathName + "_logging_decorator.h",
		LoggingCcPath:               productPath + "internal/" + filePathName + "_logging_decorator.cc",
		MetadataHeaderPath:          productPath + "internal/" + filePathName + "_metadata_decorator.h",
		MetadataCcPath:              productPath + "internal/" + filePathName + "_metadata_decorator.cc",
		TracingConnectionHeaderPath: productPath + "internal/" + filePathName + "_tracing_connection.h",
		TracingConnectionCcPath:     productPath + "internal/" + filePathName + "_tracing_connection.cc",
		TracingStubHeaderPath:       productPath + "internal/" + filePathName + "_tracing_stub.h",
		TracingStubCcPath:           productPath + "internal/" + filePathName + "_tracing_stub.cc",
		RoundRobinHeaderPath:        productPath + "internal/" + filePathName + "_round_robin_decorator.h",
		RoundRobinCcPath:            productPath + "internal/" + filePathName + "_round_robin_decorator.cc",
		OptionDefaultsHeaderPath:    productPath + "internal/" + filePathName + "_option_defaults.h",
		OptionDefaultsCcPath:        productPath + "internal/" + filePathName + "_option_defaults.cc",
		RetryTraitsHeaderPath:       productPath + "internal/" + filePathName + "_retry_traits.h",
		SourcesCcPath:               productPath + "internal/" + filePathName + "_sources.cc",

		HasAuthDecorator:       true,
		HasLoggingDecorator:    true,
		HasMetadataDecorator:   true,
		HasTracingDecorator:    true,
		HasRoundRobinDecorator: lib != nil && lib.Cpp != nil && lib.Cpp.GenerateRoundRobinDecorator,

		Methods:      methods,
		AsyncMethods: asyncMethods,
	}

	ann.ClientHeaderIncludeGuard = formatHeaderIncludeGuard(ann.ClientHeaderPath)
	ann.ConnectionHeaderIncludeGuard = formatHeaderIncludeGuard(ann.ConnectionHeaderPath)
	ann.ConnectionImplHeaderIncludeGuard = formatHeaderIncludeGuard(ann.ConnectionImplHeaderPath)
	ann.StubHeaderIncludeGuard = formatHeaderIncludeGuard(ann.StubHeaderPath)
	ann.StubFactoryHeaderIncludeGuard = formatHeaderIncludeGuard(ann.StubFactoryHeaderPath)
	ann.OptionsHeaderIncludeGuard = formatHeaderIncludeGuard(ann.OptionsHeaderPath)
	ann.IdempotencyHeaderIncludeGuard = formatHeaderIncludeGuard(ann.IdempotencyHeaderPath)
	ann.MockConnectionHeaderIncludeGuard = formatHeaderIncludeGuard(ann.MockConnectionHeaderPath)
	ann.AuthHeaderIncludeGuard = formatHeaderIncludeGuard(ann.AuthHeaderPath)
	ann.LoggingHeaderIncludeGuard = formatHeaderIncludeGuard(ann.LoggingHeaderPath)
	ann.MetadataHeaderIncludeGuard = formatHeaderIncludeGuard(ann.MetadataHeaderPath)
	ann.TracingConnectionHeaderIncludeGuard = formatHeaderIncludeGuard(ann.TracingConnectionHeaderPath)
	ann.TracingStubHeaderIncludeGuard = formatHeaderIncludeGuard(ann.TracingStubHeaderPath)
	ann.RoundRobinHeaderIncludeGuard = formatHeaderIncludeGuard(ann.RoundRobinHeaderPath)
	ann.OptionDefaultsHeaderIncludeGuard = formatHeaderIncludeGuard(ann.OptionDefaultsHeaderPath)
	ann.RetryTraitsHeaderIncludeGuard = formatHeaderIncludeGuard(ann.RetryTraitsHeaderPath)

	ann.Comments = annotateComments(svc, serviceName, model)
	ann.Options = annotateOptions(svc, lib)

	svc.Codec = ann
	return ann
}
