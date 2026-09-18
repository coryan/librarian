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
	"path/filepath"

	"github.com/googleapis/librarian/internal/sidekick/api"
	"github.com/googleapis/librarian/internal/sidekick/language"
)

type serviceAnnotations struct {
	Name            string
	BaseFileName    string
	ProductPath     string
	ForwardingPath  string
	HasGrpc         bool
	HasRest         bool
	HasRoundRobin   bool
	HasRetryTraits  bool
	OmitClient      bool
	OmitConnection  bool
	OmitStubFactory bool
	CopyrightYear   string
	Service         *api.Service
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
	ann := &serviceAnnotations{
		Name:            service.Name,
		BaseFileName:    serviceNameToFileName(service.Name),
		ProductPath:     c.Cpp.ProductPath,
		ForwardingPath:  c.Cpp.ForwardingProductPath,
		HasGrpc:         c.hasGrpc(),
		HasRest:         c.hasRest(),
		HasRoundRobin:   c.hasRoundRobin(),
		HasRetryTraits:  c.hasRetryTraits(),
		OmitClient:      c.Cpp.OmitClient,
		OmitConnection:  c.Cpp.OmitConnection,
		OmitStubFactory: c.Cpp.OmitStubFactory,
		CopyrightYear:   c.Cpp.InitialCopyrightYear,
		Service:         service,
	}
	service.Codec = ann
	return ann
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
