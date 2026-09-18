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
	"testing"
)

func TestFormatHeaderIncludeGuard(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{
			input: "google/cloud/golden/v1/golden_kitchen_sink_client.h",
			want:  "GOOGLE_CLOUD_CPP_GOOGLE_CLOUD_GOLDEN_V1_GOLDEN_KITCHEN_SINK_CLIENT_H",
		},
		{
			input: "/google/cloud/golden/v1/golden_kitchen_sink_client.h",
			want:  "GOOGLE_CLOUD_CPP_GOOGLE_CLOUD_GOLDEN_V1_GOLDEN_KITCHEN_SINK_CLIENT_H",
		},
		{
			input: "generator/integration_tests/golden/v1/golden_kitchen_sink_client.h",
			want:  "GOOGLE_CLOUD_CPP_GENERATOR_INTEGRATION_TESTS_GOLDEN_V1_GOLDEN_KITCHEN_SINK_CLIENT_H",
		},
		{
			input: "generator/integration_tests/golden/golden_kitchen_sink_client.h",
			want:  "GOOGLE_CLOUD_CPP_GENERATOR_INTEGRATION_TESTS_GOLDEN_GOLDEN_KITCHEN_SINK_CLIENT_H",
		},
		{
			input: "generator/integration_tests/golden/v1/mocks/mock_golden_kitchen_sink_connection.h",
			want:  "GOOGLE_CLOUD_CPP_GENERATOR_INTEGRATION_TESTS_GOLDEN_V1_MOCKS_MOCK_GOLDEN_KITCHEN_SINK_CONNECTION_H",
		},
		{
			input: "generator/integration_tests/golden/mocks/mock_golden_kitchen_sink_connection.h",
			want:  "GOOGLE_CLOUD_CPP_GENERATOR_INTEGRATION_TESTS_GOLDEN_MOCKS_MOCK_GOLDEN_KITCHEN_SINK_CONNECTION_H",
		},
		{
			input: "generator/integration_tests/golden/v1/internal/golden_kitchen_sink_stub.h",
			want:  "GOOGLE_CLOUD_CPP_GENERATOR_INTEGRATION_TESTS_GOLDEN_V1_INTERNAL_GOLDEN_KITCHEN_SINK_STUB_H",
		},
		{
			input: `google\cloud\secretmanager\v1\secret_manager_client.h`,
			want:  "GOOGLE_CLOUD_CPP_GOOGLE_CLOUD_SECRETMANAGER_V1_SECRET_MANAGER_CLIENT_H",
		},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			got := formatHeaderIncludeGuard(tc.input)
			if got != tc.want {
				t.Errorf("formatHeaderIncludeGuard(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

func TestServiceAnnotations_IncludeGuardsAndPaths(t *testing.T) {
	ann := &serviceAnnotations{
		Name:           "GoldenKitchenSink",
		BaseFileName:   "golden_kitchen_sink",
		ProductPath:    "generator/integration_tests/golden/v1",
		ForwardingPath: "generator/integration_tests/golden",
	}

	tests := []struct {
		name      string
		gotPath   string
		wantPath  string
		gotGuard  string
		wantGuard string
	}{
		{
			name:      "Client",
			gotPath:   ann.ClientHeaderPath(),
			wantPath:  "generator/integration_tests/golden/v1/golden_kitchen_sink_client.h",
			gotGuard:  ann.ClientHeaderGuard(),
			wantGuard: "GOOGLE_CLOUD_CPP_GENERATOR_INTEGRATION_TESTS_GOLDEN_V1_GOLDEN_KITCHEN_SINK_CLIENT_H",
		},
		{
			name:      "Connection",
			gotPath:   ann.ConnectionHeaderPath(),
			wantPath:  "generator/integration_tests/golden/v1/golden_kitchen_sink_connection.h",
			gotGuard:  ann.ConnectionHeaderGuard(),
			wantGuard: "GOOGLE_CLOUD_CPP_GENERATOR_INTEGRATION_TESTS_GOLDEN_V1_GOLDEN_KITCHEN_SINK_CONNECTION_H",
		},
		{
			name:      "IdempotencyPolicy",
			gotPath:   ann.IdempotencyPolicyHeaderPath(),
			wantPath:  "generator/integration_tests/golden/v1/golden_kitchen_sink_connection_idempotency_policy.h",
			gotGuard:  ann.IdempotencyPolicyHeaderGuard(),
			wantGuard: "GOOGLE_CLOUD_CPP_GENERATOR_INTEGRATION_TESTS_GOLDEN_V1_GOLDEN_KITCHEN_SINK_CONNECTION_IDEMPOTENCY_POLICY_H",
		},
		{
			name:      "Options",
			gotPath:   ann.OptionsHeaderPath(),
			wantPath:  "generator/integration_tests/golden/v1/golden_kitchen_sink_options.h",
			gotGuard:  ann.OptionsHeaderGuard(),
			wantGuard: "GOOGLE_CLOUD_CPP_GENERATOR_INTEGRATION_TESTS_GOLDEN_V1_GOLDEN_KITCHEN_SINK_OPTIONS_H",
		},
		{
			name:      "RestConnection",
			gotPath:   ann.RestConnectionHeaderPath(),
			wantPath:  "generator/integration_tests/golden/v1/golden_kitchen_sink_rest_connection.h",
			gotGuard:  ann.RestConnectionHeaderGuard(),
			wantGuard: "GOOGLE_CLOUD_CPP_GENERATOR_INTEGRATION_TESTS_GOLDEN_V1_GOLDEN_KITCHEN_SINK_REST_CONNECTION_H",
		},
		{
			name:      "MockConnection",
			gotPath:   ann.MockConnectionHeaderPath(),
			wantPath:  "generator/integration_tests/golden/v1/mocks/mock_golden_kitchen_sink_connection.h",
			gotGuard:  ann.MockConnectionHeaderGuard(),
			wantGuard: "GOOGLE_CLOUD_CPP_GENERATOR_INTEGRATION_TESTS_GOLDEN_V1_MOCKS_MOCK_GOLDEN_KITCHEN_SINK_CONNECTION_H",
		},
		{
			name:      "OptionDefaults",
			gotPath:   ann.OptionDefaultsHeaderPath(),
			wantPath:  "generator/integration_tests/golden/v1/internal/golden_kitchen_sink_option_defaults.h",
			gotGuard:  ann.OptionDefaultsHeaderGuard(),
			wantGuard: "GOOGLE_CLOUD_CPP_GENERATOR_INTEGRATION_TESTS_GOLDEN_V1_INTERNAL_GOLDEN_KITCHEN_SINK_OPTION_DEFAULTS_H",
		},
		{
			name:      "TracingConnection",
			gotPath:   ann.TracingConnectionHeaderPath(),
			wantPath:  "generator/integration_tests/golden/v1/internal/golden_kitchen_sink_tracing_connection.h",
			gotGuard:  ann.TracingConnectionHeaderGuard(),
			wantGuard: "GOOGLE_CLOUD_CPP_GENERATOR_INTEGRATION_TESTS_GOLDEN_V1_INTERNAL_GOLDEN_KITCHEN_SINK_TRACING_CONNECTION_H",
		},
		{
			name:      "RetryTraits",
			gotPath:   ann.RetryTraitsHeaderPath(),
			wantPath:  "generator/integration_tests/golden/v1/internal/golden_kitchen_sink_retry_traits.h",
			gotGuard:  ann.RetryTraitsHeaderGuard(),
			wantGuard: "GOOGLE_CLOUD_CPP_GENERATOR_INTEGRATION_TESTS_GOLDEN_V1_INTERNAL_GOLDEN_KITCHEN_SINK_RETRY_TRAITS_H",
		},
		{
			name:      "ConnectionImpl",
			gotPath:   ann.ConnectionImplHeaderPath(),
			wantPath:  "generator/integration_tests/golden/v1/internal/golden_kitchen_sink_connection_impl.h",
			gotGuard:  ann.ConnectionImplHeaderGuard(),
			wantGuard: "GOOGLE_CLOUD_CPP_GENERATOR_INTEGRATION_TESTS_GOLDEN_V1_INTERNAL_GOLDEN_KITCHEN_SINK_CONNECTION_IMPL_H",
		},
		{
			name:      "StubFactory",
			gotPath:   ann.StubFactoryHeaderPath(),
			wantPath:  "generator/integration_tests/golden/v1/internal/golden_kitchen_sink_stub_factory.h",
			gotGuard:  ann.StubFactoryHeaderGuard(),
			wantGuard: "GOOGLE_CLOUD_CPP_GENERATOR_INTEGRATION_TESTS_GOLDEN_V1_INTERNAL_GOLDEN_KITCHEN_SINK_STUB_FACTORY_H",
		},
		{
			name:      "AuthDecorator",
			gotPath:   ann.AuthDecoratorHeaderPath(),
			wantPath:  "generator/integration_tests/golden/v1/internal/golden_kitchen_sink_auth_decorator.h",
			gotGuard:  ann.AuthDecoratorHeaderGuard(),
			wantGuard: "GOOGLE_CLOUD_CPP_GENERATOR_INTEGRATION_TESTS_GOLDEN_V1_INTERNAL_GOLDEN_KITCHEN_SINK_AUTH_DECORATOR_H",
		},
		{
			name:      "LoggingDecorator",
			gotPath:   ann.LoggingDecoratorHeaderPath(),
			wantPath:  "generator/integration_tests/golden/v1/internal/golden_kitchen_sink_logging_decorator.h",
			gotGuard:  ann.LoggingDecoratorHeaderGuard(),
			wantGuard: "GOOGLE_CLOUD_CPP_GENERATOR_INTEGRATION_TESTS_GOLDEN_V1_INTERNAL_GOLDEN_KITCHEN_SINK_LOGGING_DECORATOR_H",
		},
		{
			name:      "MetadataDecorator",
			gotPath:   ann.MetadataDecoratorHeaderPath(),
			wantPath:  "generator/integration_tests/golden/v1/internal/golden_kitchen_sink_metadata_decorator.h",
			gotGuard:  ann.MetadataDecoratorHeaderGuard(),
			wantGuard: "GOOGLE_CLOUD_CPP_GENERATOR_INTEGRATION_TESTS_GOLDEN_V1_INTERNAL_GOLDEN_KITCHEN_SINK_METADATA_DECORATOR_H",
		},
		{
			name:      "Stub",
			gotPath:   ann.StubHeaderPath(),
			wantPath:  "generator/integration_tests/golden/v1/internal/golden_kitchen_sink_stub.h",
			gotGuard:  ann.StubHeaderGuard(),
			wantGuard: "GOOGLE_CLOUD_CPP_GENERATOR_INTEGRATION_TESTS_GOLDEN_V1_INTERNAL_GOLDEN_KITCHEN_SINK_STUB_H",
		},
		{
			name:      "TracingStub",
			gotPath:   ann.TracingStubHeaderPath(),
			wantPath:  "generator/integration_tests/golden/v1/internal/golden_kitchen_sink_tracing_stub.h",
			gotGuard:  ann.TracingStubHeaderGuard(),
			wantGuard: "GOOGLE_CLOUD_CPP_GENERATOR_INTEGRATION_TESTS_GOLDEN_V1_INTERNAL_GOLDEN_KITCHEN_SINK_TRACING_STUB_H",
		},
		{
			name:      "RoundRobinDecorator",
			gotPath:   ann.RoundRobinDecoratorHeaderPath(),
			wantPath:  "generator/integration_tests/golden/v1/internal/golden_kitchen_sink_round_robin_decorator.h",
			gotGuard:  ann.RoundRobinDecoratorHeaderGuard(),
			wantGuard: "GOOGLE_CLOUD_CPP_GENERATOR_INTEGRATION_TESTS_GOLDEN_V1_INTERNAL_GOLDEN_KITCHEN_SINK_ROUND_ROBIN_DECORATOR_H",
		},
		{
			name:      "RestConnectionImpl",
			gotPath:   ann.RestConnectionImplHeaderPath(),
			wantPath:  "generator/integration_tests/golden/v1/internal/golden_kitchen_sink_rest_connection_impl.h",
			gotGuard:  ann.RestConnectionImplHeaderGuard(),
			wantGuard: "GOOGLE_CLOUD_CPP_GENERATOR_INTEGRATION_TESTS_GOLDEN_V1_INTERNAL_GOLDEN_KITCHEN_SINK_REST_CONNECTION_IMPL_H",
		},
		{
			name:      "RestLoggingDecorator",
			gotPath:   ann.RestLoggingDecoratorHeaderPath(),
			wantPath:  "generator/integration_tests/golden/v1/internal/golden_kitchen_sink_rest_logging_decorator.h",
			gotGuard:  ann.RestLoggingDecoratorHeaderGuard(),
			wantGuard: "GOOGLE_CLOUD_CPP_GENERATOR_INTEGRATION_TESTS_GOLDEN_V1_INTERNAL_GOLDEN_KITCHEN_SINK_REST_LOGGING_DECORATOR_H",
		},
		{
			name:      "RestMetadataDecorator",
			gotPath:   ann.RestMetadataDecoratorHeaderPath(),
			wantPath:  "generator/integration_tests/golden/v1/internal/golden_kitchen_sink_rest_metadata_decorator.h",
			gotGuard:  ann.RestMetadataDecoratorHeaderGuard(),
			wantGuard: "GOOGLE_CLOUD_CPP_GENERATOR_INTEGRATION_TESTS_GOLDEN_V1_INTERNAL_GOLDEN_KITCHEN_SINK_REST_METADATA_DECORATOR_H",
		},
		{
			name:      "RestStub",
			gotPath:   ann.RestStubHeaderPath(),
			wantPath:  "generator/integration_tests/golden/v1/internal/golden_kitchen_sink_rest_stub.h",
			gotGuard:  ann.RestStubHeaderGuard(),
			wantGuard: "GOOGLE_CLOUD_CPP_GENERATOR_INTEGRATION_TESTS_GOLDEN_V1_INTERNAL_GOLDEN_KITCHEN_SINK_REST_STUB_H",
		},
		{
			name:      "RestStubFactory",
			gotPath:   ann.RestStubFactoryHeaderPath(),
			wantPath:  "generator/integration_tests/golden/v1/internal/golden_kitchen_sink_rest_stub_factory.h",
			gotGuard:  ann.RestStubFactoryHeaderGuard(),
			wantGuard: "GOOGLE_CLOUD_CPP_GENERATOR_INTEGRATION_TESTS_GOLDEN_V1_INTERNAL_GOLDEN_KITCHEN_SINK_REST_STUB_FACTORY_H",
		},
		{
			name:      "ForwardingClient",
			gotPath:   ann.ForwardingClientHeaderPath(),
			wantPath:  "generator/integration_tests/golden/golden_kitchen_sink_client.h",
			gotGuard:  ann.ForwardingClientHeaderGuard(),
			wantGuard: "GOOGLE_CLOUD_CPP_GENERATOR_INTEGRATION_TESTS_GOLDEN_GOLDEN_KITCHEN_SINK_CLIENT_H",
		},
		{
			name:      "ForwardingConnection",
			gotPath:   ann.ForwardingConnectionHeaderPath(),
			wantPath:  "generator/integration_tests/golden/golden_kitchen_sink_connection.h",
			gotGuard:  ann.ForwardingConnectionHeaderGuard(),
			wantGuard: "GOOGLE_CLOUD_CPP_GENERATOR_INTEGRATION_TESTS_GOLDEN_GOLDEN_KITCHEN_SINK_CONNECTION_H",
		},
		{
			name:      "ForwardingIdempotencyPolicy",
			gotPath:   ann.ForwardingIdempotencyPolicyHeaderPath(),
			wantPath:  "generator/integration_tests/golden/golden_kitchen_sink_connection_idempotency_policy.h",
			gotGuard:  ann.ForwardingIdempotencyPolicyHeaderGuard(),
			wantGuard: "GOOGLE_CLOUD_CPP_GENERATOR_INTEGRATION_TESTS_GOLDEN_GOLDEN_KITCHEN_SINK_CONNECTION_IDEMPOTENCY_POLICY_H",
		},
		{
			name:      "ForwardingOptions",
			gotPath:   ann.ForwardingOptionsHeaderPath(),
			wantPath:  "generator/integration_tests/golden/golden_kitchen_sink_options.h",
			gotGuard:  ann.ForwardingOptionsHeaderGuard(),
			wantGuard: "GOOGLE_CLOUD_CPP_GENERATOR_INTEGRATION_TESTS_GOLDEN_GOLDEN_KITCHEN_SINK_OPTIONS_H",
		},
		{
			name:      "ForwardingMockConnection",
			gotPath:   ann.ForwardingMockConnectionHeaderPath(),
			wantPath:  "generator/integration_tests/golden/mocks/mock_golden_kitchen_sink_connection.h",
			gotGuard:  ann.ForwardingMockConnectionHeaderGuard(),
			wantGuard: "GOOGLE_CLOUD_CPP_GENERATOR_INTEGRATION_TESTS_GOLDEN_MOCKS_MOCK_GOLDEN_KITCHEN_SINK_CONNECTION_H",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.gotPath != tc.wantPath {
				t.Errorf("%s path = %q, want %q", tc.name, tc.gotPath, tc.wantPath)
			}
			if tc.gotGuard != tc.wantGuard {
				t.Errorf("%s guard = %q, want %q", tc.name, tc.gotGuard, tc.wantGuard)
			}
		})
	}
}
