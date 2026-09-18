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
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
)

func TestConvertConfig_BasicService(t *testing.T) {
	input := `
# Secret Manager
service {
  service_proto_path: "google/cloud/secretmanager/v1/service.proto"
  product_path: "google/cloud/secretmanager/v1"
  forwarding_product_path: "google/cloud/secretmanager"
  initial_copyright_year: "2021"
  retryable_status_codes: ["kUnavailable"]
}
`
	cfg, err := ConvertConfig(input)
	if err != nil {
		t.Fatalf("ConvertConfig failed: %v", err)
	}

	if cfg.Language != config.LanguageCpp {
		t.Errorf("expected language %q, got %q", config.LanguageCpp, cfg.Language)
	}
	if len(cfg.Libraries) != 1 {
		t.Fatalf("expected 1 library, got %d", len(cfg.Libraries))
	}

	lib := cfg.Libraries[0]
	if lib.Name != "google-cloud-secretmanager-v1" {
		t.Errorf("expected name %q, got %q", "google-cloud-secretmanager-v1", lib.Name)
	}
	if lib.CopyrightYear != "2021" {
		t.Errorf("expected copyright year %q, got %q", "2021", lib.CopyrightYear)
	}
	if lib.Output != "google/cloud/secretmanager/v1" {
		t.Errorf("expected output %q, got %q", "google/cloud/secretmanager/v1", lib.Output)
	}
	if len(lib.APIs) != 1 || lib.APIs[0].Path != "google/cloud/secretmanager/v1/service.proto" {
		t.Errorf("unexpected APIs: %+v", lib.APIs)
	}
	if lib.Cpp == nil {
		t.Fatal("expected non-nil Cpp configuration")
	}
	if lib.Cpp.ForwardingProductPath != "google/cloud/secretmanager" {
		t.Errorf("expected forwarding_product_path %q, got %q", "google/cloud/secretmanager", lib.Cpp.ForwardingProductPath)
	}
	wantCodes := []string{"kUnavailable"}
	if diff := cmp.Diff(wantCodes, lib.Cpp.RetryableStatusCodes); diff != "" {
		t.Errorf("retryable status codes mismatch (-want +got):\n%s", diff)
	}
}

func TestConvertConfig_FullOptions(t *testing.T) {
	input := `
service {
  service_proto_path: "google/cloud/test/v1/test.proto"
  product_path: "google/cloud/test/v1"
  initial_copyright_year: "2025"
  service_endpoint_env_var: "TEST_ENDPOINT"
  emulator_endpoint_env_var: "TEST_EMULATOR"
  endpoint_location_style: LOCATION_DEPENDENT_COMPAT
  override_service_config_yaml_name: "google/cloud/test/v1/test.yaml"
  generate_rest_transport: true
  generate_grpc_transport: false
  generate_round_robin_decorator: true
  backwards_compatibility_namespace_alias: true
  omit_client: true
  omit_connection: true
  omit_stub_factory: true
  omit_streaming_updater: true
  omit_repo_metadata: true
  experimental: true
  preserve_proto_field_names_in_json: true
  additional_proto_files: ["google/protobuf/struct.proto"]
  omitted_rpcs: ["DeprecatedRpc"]
  gen_async_rpcs: ["AsyncRpc"]
  omitted_services: ["OldService"]
  retryable_status_codes: ["kUnavailable", "kDeadlineExceeded"]
  idempotency_overrides: [
    {
      rpc_name: "TestService.CustomCall"
      idempotency: IDEMPOTENT
    }
  ]
}
`
	cfg, err := ConvertConfig(input)
	if err != nil {
		t.Fatalf("ConvertConfig failed: %v", err)
	}
	if len(cfg.Libraries) != 1 {
		t.Fatalf("expected 1 library, got %d", len(cfg.Libraries))
	}
	cppLib := cfg.Libraries[0].Cpp
	if cppLib == nil {
		t.Fatal("expected non-nil Cpp configuration")
	}

	if cppLib.ServiceEndpointEnvVar != "TEST_ENDPOINT" {
		t.Errorf("unexpected ServiceEndpointEnvVar: %s", cppLib.ServiceEndpointEnvVar)
	}
	if cppLib.EmulatorEndpointEnvVar != "TEST_EMULATOR" {
		t.Errorf("unexpected EmulatorEndpointEnvVar: %s", cppLib.EmulatorEndpointEnvVar)
	}
	if cppLib.EndpointLocationStyle != "LOCATION_DEPENDENT_COMPAT" {
		t.Errorf("unexpected EndpointLocationStyle: %s", cppLib.EndpointLocationStyle)
	}
	if cppLib.OverrideServiceConfigYamlName != "google/cloud/test/v1/test.yaml" {
		t.Errorf("unexpected OverrideServiceConfigYamlName: %s", cppLib.OverrideServiceConfigYamlName)
	}
	if !cppLib.GenerateRestTransport {
		t.Errorf("expected GenerateRestTransport to be true")
	}
	if cppLib.GenerateGrpcTransport == nil || *cppLib.GenerateGrpcTransport {
		t.Errorf("expected GenerateGrpcTransport to be explicit false")
	}
	if !cppLib.GenerateRoundRobinDecorator {
		t.Errorf("expected GenerateRoundRobinDecorator to be true")
	}
	if !cppLib.BackwardsCompatibilityNamespace {
		t.Errorf("expected BackwardsCompatibilityNamespace to be true")
	}
	if !cppLib.OmitClient {
		t.Errorf("expected OmitClient to be true")
	}
	if !cppLib.OmitConnection {
		t.Errorf("expected OmitConnection to be true")
	}
	if !cppLib.OmitStubFactory {
		t.Errorf("expected OmitStubFactory to be true")
	}
	if !cppLib.OmitStreamingUpdater {
		t.Errorf("expected OmitStreamingUpdater to be true")
	}
	if !cppLib.OmitRepoMetadata {
		t.Errorf("expected OmitRepoMetadata to be true")
	}
	if !cppLib.Experimental {
		t.Errorf("expected Experimental to be true")
	}
	if !cppLib.PreserveProtoFieldNamesInJson {
		t.Errorf("expected PreserveProtoFieldNamesInJson to be true")
	}

	wantIdempotency := []config.IdempotencyRule{
		{RPCName: "TestService.CustomCall", Idempotency: "IDEMPOTENT"},
	}
	if diff := cmp.Diff(wantIdempotency, cppLib.IdempotencyOverrides); diff != "" {
		t.Errorf("idempotency overrides mismatch (-want +got):\n%s", diff)
	}
}

func TestConvertConfig_DiscoveryProducts(t *testing.T) {
	input := `
discovery_products {
  discovery_document_url: "file:///workspace/compute.json"
  operation_services: ["ZoneOperations"]
  rest_services {
    service_proto_path: "google/cloud/compute/addresses/v1/addresses.proto"
    product_path: "google/cloud/compute/addresses/v1"
    initial_copyright_year: "2023"
    retryable_status_codes: ["kUnavailable"]
    generate_rest_transport: true
    generate_grpc_transport: false
  }
}
`
	cfg, err := ConvertConfig(input)
	if err != nil {
		t.Fatalf("ConvertConfig failed: %v", err)
	}
	if len(cfg.Libraries) != 1 {
		t.Fatalf("expected 1 library from discovery products, got %d", len(cfg.Libraries))
	}
	lib := cfg.Libraries[0]
	if lib.Name != "google-cloud-compute-addresses-v1" {
		t.Errorf("expected library name google-cloud-compute-addresses-v1, got %s", lib.Name)
	}
	if !lib.Cpp.GenerateRestTransport {
		t.Errorf("expected GenerateRestTransport to be true")
	}
}

func TestConvertConfigFile_ProductionConfig(t *testing.T) {
	candidates := []string{
		"/usr/local/google/home/coryan/google-cloud-cpp/main/generator/generator_config.textproto",
		filepath.Join("..", "..", "sidekick", "cpp", "testdata", "generator_config.textproto"),
	}

	var foundPath string
	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			foundPath = p
			break
		}
	}

	if foundPath == "" {
		t.Skip("skipping production textproto test; file not found")
	}

	cfg, err := ConvertConfigFile(foundPath)
	if err != nil {
		t.Fatalf("ConvertConfigFile failed on %s: %v", foundPath, err)
	}

	if len(cfg.Libraries) < 50 {
		t.Errorf("expected at least 50 libraries in production config, got %d", len(cfg.Libraries))
	}

	// Verify secretmanager is among the parsed libraries
	var foundSM bool
	for _, lib := range cfg.Libraries {
		if lib.Output == "google/cloud/secretmanager/v1" {
			foundSM = true
			if lib.Cpp == nil || lib.Cpp.ForwardingProductPath != "google/cloud/secretmanager" {
				t.Errorf("secretmanager forwarding_product_path mismatch: %+v", lib.Cpp)
			}
			break
		}
	}
	if !foundSM {
		t.Errorf("expected secretmanager/v1 to be found among parsed libraries")
	}
}

func TestConvertConfig_Errors(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"unterminated_string", `service { service_proto_path: "foo }`},
		{"missing_brace", `service service_proto_path: "foo" }`},
		{"unclosed_brace", `service { service_proto_path: "foo"`},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ConvertConfig(tc.input)
			if err == nil {
				t.Errorf("expected error for %s, got nil", tc.name)
			}
		})
	}
}
