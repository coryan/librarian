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

package config

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/yaml"
)

func TestCppConfig_Unmarshal(t *testing.T) {
	yamlInput := `
product_path: generator/integration_tests/golden/v1
forwarding_product_path: generator/integration_tests/golden
service_endpoint_env_var: GOLDEN_KITCHEN_SINK_ENDPOINT
emulator_endpoint_env_var: GOLDEN_KITCHEN_SINK_EMULATOR_HOST
generate_rest_transport: true
generate_grpc_transport: true
endpoint_location_style: LOCATION_OPTIONALLY_DEPENDENT
backwards_compatibility_namespace_alias: true
omitted_rpcs:
  - Omitted1
  - GoldenKitchenSink.Omitted2
gen_async_rpcs:
  - GetDatabase
  - DropDatabase
omitted_services:
  - DeprecatedService
retryable_status_codes:
  - GoldenKitchenSink.kInternal
  - kUnavailable
idempotency_overrides:
  - rpc_name: GoldenThingAdmin.DropDatabase
    idempotency: IDEMPOTENT
  - rpc_name: GoldenKitchenSink.ListLogs
    idempotency: NON_IDEMPOTENT
generate_round_robin_decorator: true
omit_client: false
omit_connection: false
omit_stub_factory: false
omit_streaming_updater: true
omit_repo_metadata: true
experimental: true
preserve_proto_field_names_in_json: true
initial_copyright_year: "2022"
proto_file_source: GOOGLEAPIS
service_name_mapping:
  OriginalService: MappedService
service_name_to_comment:
  OriginalService: "Overridden comment"
additional_proto_files:
  - generator/integration_tests/backup.proto
service_config: generator/integration_tests/test.yaml
`

	got, err := yaml.Unmarshal[CppLibrary]([]byte(yamlInput))
	if err != nil {
		t.Fatalf("yaml.Unmarshal failed: %v", err)
	}

	grpcTransport := true
	want := CppLibrary{
		CppDefault: CppDefault{
			ProductPath:                     "generator/integration_tests/golden/v1",
			ForwardingProductPath:           "generator/integration_tests/golden",
			GenerateRestTransport:           true,
			GenerateGrpcTransport:           &grpcTransport,
			EndpointLocationStyle:           "LOCATION_OPTIONALLY_DEPENDENT",
			BackwardsCompatibilityNamespace: true,
			RetryableStatusCodes: []string{
				"GoldenKitchenSink.kInternal",
				"kUnavailable",
			},
			GenerateRoundRobinDecorator:   true,
			OmitRepoMetadata:              true,
			Experimental:                  true,
			PreserveProtoFieldNamesInJson: true,
			InitialCopyrightYear:          "2022",
		},
		ServiceEndpointEnvVar:  "GOLDEN_KITCHEN_SINK_ENDPOINT",
		EmulatorEndpointEnvVar: "GOLDEN_KITCHEN_SINK_EMULATOR_HOST",
		OmittedRPCs: []string{
			"Omitted1",
			"GoldenKitchenSink.Omitted2",
		},
		GenAsyncRPCs: []string{
			"GetDatabase",
			"DropDatabase",
		},
		OmittedServices: []string{
			"DeprecatedService",
		},
		IdempotencyOverrides: []IdempotencyRule{
			{RPCName: "GoldenThingAdmin.DropDatabase", Idempotency: "IDEMPOTENT"},
			{RPCName: "GoldenKitchenSink.ListLogs", Idempotency: "NON_IDEMPOTENT"},
		},
		OmitStreamingUpdater: true,
		ServiceNameMapping: map[string]string{
			"OriginalService": "MappedService",
		},
		ServiceNameToComment: map[string]string{
			"OriginalService": "Overridden comment",
		},
		AdditionalProtoFiles: []string{
			"generator/integration_tests/backup.proto",
		},
		ServiceConfig:   "generator/integration_tests/test.yaml",
		ProtoFileSource: "GOOGLEAPIS",
	}

	if diff := cmp.Diff(&want, got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestGetServiceConfig(t *testing.T) {
	var nilLib *CppLibrary
	if got := nilLib.GetServiceConfig(); got != "" {
		t.Errorf("nilLib.GetServiceConfig() = %q, want empty", got)
	}

	emptyLib := &CppLibrary{}
	if got := emptyLib.GetServiceConfig(); got != "" {
		t.Errorf("emptyLib.GetServiceConfig() = %q, want empty", got)
	}

	lib1 := &CppLibrary{ServiceConfig: "path/to/service.yaml"}
	if got := lib1.GetServiceConfig(); got != "path/to/service.yaml" {
		t.Errorf("lib1.GetServiceConfig() = %q, want %q", got, "path/to/service.yaml")
	}

	lib2 := &CppLibrary{OverrideServiceConfigYamlName: "path/to/override.yaml"}
	if got := lib2.GetServiceConfig(); got != "path/to/override.yaml" {
		t.Errorf("lib2.GetServiceConfig() = %q, want %q", got, "path/to/override.yaml")
	}

	// ServiceConfig takes precedence if both set
	lib3 := &CppLibrary{
		ServiceConfig:                 "path/to/service.yaml",
		OverrideServiceConfigYamlName: "path/to/override.yaml",
	}
	if got := lib3.GetServiceConfig(); got != "path/to/service.yaml" {
		t.Errorf("lib3.GetServiceConfig() = %q, want %q", got, "path/to/service.yaml")
	}
}

func TestHasGrpcTransport(t *testing.T) {
	// Nil receiver returns true (default)
	var nilDef *CppDefault
	if !nilDef.HasGrpcTransport() {
		t.Errorf("nil CppDefault.HasGrpcTransport() got false, want true")
	}

	// Nil GenerateGrpcTransport returns true
	def := &CppDefault{}
	if !def.HasGrpcTransport() {
		t.Errorf("CppDefault with nil GenerateGrpcTransport got false, want true")
	}

	// Explicit true returns true
	tTrue := true
	def.GenerateGrpcTransport = &tTrue
	if !def.HasGrpcTransport() {
		t.Errorf("CppDefault with explicit true got false, want true")
	}

	// Explicit false returns false
	tFalse := false
	def.GenerateGrpcTransport = &tFalse
	if def.HasGrpcTransport() {
		t.Errorf("CppDefault with explicit false got true, want false")
	}
}

func TestConfig_UnmarshalCppInLibrary(t *testing.T) {
	yamlInput := `
language: cpp
default:
  cpp:
    product_path: google/cloud/default
libraries:
  - name: test_lib
    copyright_year: "2026"
    apis:
      - path: test.proto
    cpp:
      service_endpoint_env_var: TEST_ENDPOINT
`
	got, err := yaml.Unmarshal[Config]([]byte(yamlInput))
	if err != nil {
		t.Fatalf("yaml.Unmarshal failed: %v", err)
	}

	if got.Language != LanguageCpp {
		t.Errorf("got Language %q, want %q", got.Language, LanguageCpp)
	}
	if got.Default == nil || got.Default.Cpp == nil || got.Default.Cpp.ProductPath != "google/cloud/default" {
		t.Errorf("got default Cpp %v, want product_path google/cloud/default", got.Default)
	}
	if len(got.Libraries) != 1 || got.Libraries[0].Cpp == nil || got.Libraries[0].Cpp.ServiceEndpointEnvVar != "TEST_ENDPOINT" {
		t.Errorf("got library Cpp %v, want service_endpoint_env_var TEST_ENDPOINT", got.Libraries)
	}
}
