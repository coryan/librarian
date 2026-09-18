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
	"github.com/googleapis/librarian/internal/yaml"
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
	cfg, err := ConvertTextprotoToConfig([]byte(input))
	if err != nil {
		t.Fatalf("ConvertTextprotoToConfig failed: %v", err)
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
	if lib.Output != "google/cloud/secretmanager/v1" {
		t.Errorf("expected output %q, got %q", "google/cloud/secretmanager/v1", lib.Output)
	}
	if len(lib.APIs) != 1 || lib.APIs[0].Path != "google/cloud/secretmanager/v1/service.proto" {
		t.Errorf("unexpected APIs: %+v", lib.APIs)
	}
	if lib.Cpp == nil {
		t.Fatal("expected non-nil Cpp configuration")
	}
	if lib.Cpp.ProductPath != "google/cloud/secretmanager/v1" {
		t.Errorf("expected product_path %q, got %q", "google/cloud/secretmanager/v1", lib.Cpp.ProductPath)
	}
	if lib.Cpp.ForwardingProductPath != "google/cloud/secretmanager" {
		t.Errorf("expected forwarding_product_path %q, got %q", "google/cloud/secretmanager", lib.Cpp.ForwardingProductPath)
	}
	if lib.Cpp.InitialCopyrightYear != "2021" {
		t.Errorf("expected initial_copyright_year %q, got %q", "2021", lib.Cpp.InitialCopyrightYear)
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
  forwarding_product_path: "google/cloud/test"
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
  omit_repo_metadata: true
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
	cfg, err := ConvertTextprotoToConfig([]byte(input))
	if err != nil {
		t.Fatalf("ConvertTextprotoToConfig failed: %v", err)
	}
	if len(cfg.Libraries) != 1 {
		t.Fatalf("expected 1 library, got %d", len(cfg.Libraries))
	}
	cppLib := cfg.Libraries[0].Cpp
	if cppLib == nil {
		t.Fatal("expected non-nil Cpp configuration")
	}

	if cppLib.ProductPath != "google/cloud/test/v1" {
		t.Errorf("unexpected ProductPath: %s", cppLib.ProductPath)
	}
	if cppLib.ForwardingProductPath != "google/cloud/test" {
		t.Errorf("unexpected ForwardingProductPath: %s", cppLib.ForwardingProductPath)
	}
	if cppLib.InitialCopyrightYear != "2025" {
		t.Errorf("unexpected InitialCopyrightYear: %s", cppLib.InitialCopyrightYear)
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
	if cppLib.OverrideServiceConfigYAMLName != "google/cloud/test/v1/test.yaml" {
		t.Errorf("unexpected OverrideServiceConfigYAMLName: %s", cppLib.OverrideServiceConfigYAMLName)
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
	if !cppLib.OmitRepoMetadata {
		t.Errorf("expected OmitRepoMetadata to be true")
	}

	wantProtos := []string{"google/protobuf/struct.proto"}
	if diff := cmp.Diff(wantProtos, cppLib.AdditionalProtoFiles); diff != "" {
		t.Errorf("additional proto files mismatch (-want +got):\n%s", diff)
	}
	wantOmittedRPCs := []string{"DeprecatedRpc"}
	if diff := cmp.Diff(wantOmittedRPCs, cppLib.OmittedRPCs); diff != "" {
		t.Errorf("omitted RPCs mismatch (-want +got):\n%s", diff)
	}
	wantAsyncRPCs := []string{"AsyncRpc"}
	if diff := cmp.Diff(wantAsyncRPCs, cppLib.GenAsyncRPCs); diff != "" {
		t.Errorf("gen async RPCs mismatch (-want +got):\n%s", diff)
	}
	wantOmittedServices := []string{"OldService"}
	if diff := cmp.Diff(wantOmittedServices, cppLib.OmittedServices); diff != "" {
		t.Errorf("omitted services mismatch (-want +got):\n%s", diff)
	}
	wantRetryCodes := []string{"kUnavailable", "kDeadlineExceeded"}
	if diff := cmp.Diff(wantRetryCodes, cppLib.RetryableStatusCodes); diff != "" {
		t.Errorf("retryable status codes mismatch (-want +got):\n%s", diff)
	}
	wantIdempotency := []config.IdempotencyRule{
		{RPCName: "TestService.CustomCall", Idempotency: "IDEMPOTENT"},
	}
	if diff := cmp.Diff(wantIdempotency, cppLib.IdempotencyOverrides); diff != "" {
		t.Errorf("idempotency overrides mismatch (-want +got):\n%s", diff)
	}
}

func TestConvertConfig_GoldenIntegrationTests(t *testing.T) {
	textprotoPath := filepath.Join("..", "..", "sidekick", "cpp", "testdata", "golden_config.textproto")
	if _, err := os.Stat(textprotoPath); err != nil {
		t.Skip("golden_config.textproto not found")
	}

	cfg, err := ConvertConfigFile(textprotoPath)
	if err != nil {
		t.Fatalf("ConvertConfigFile(%s) failed: %v", textprotoPath, err)
	}

	if cfg.Language != config.LanguageCpp {
		t.Fatalf("expected language %s, got %s", config.LanguageCpp, cfg.Language)
	}
	if len(cfg.Libraries) != 4 {
		t.Fatalf("expected 4 golden libraries, got %d", len(cfg.Libraries))
	}

	// Compare directly against golden_librarian.yaml
	yamlPath := filepath.Join("..", "..", "sidekick", "cpp", "testdata", "golden_librarian.yaml")
	yamlData, err := os.ReadFile(yamlPath)
	if err != nil {
		t.Fatalf("reading golden_librarian.yaml: %v", err)
	}
	want, err := yaml.Unmarshal[config.Config](yamlData)
	if err != nil {
		t.Fatalf("unmarshaling golden_librarian.yaml: %v", err)
	}

	if diff := cmp.Diff(want, cfg); diff != "" {
		t.Errorf("golden config mismatch (-want +got):\n%s", diff)
	}
}

func TestConvertConfig_ProductionConfig(t *testing.T) {
	foundPath := filepath.Join("..", "..", "sidekick", "cpp", "testdata", "generator_config.textproto")
	if _, err := os.Stat(foundPath); err != nil {
		t.Skip("skipping production textproto test; file not found")
	}

	cfg, err := ConvertConfigFile(foundPath)
	if err != nil {
		t.Fatalf("ConvertConfigFile failed on %s: %v", foundPath, err)
	}

	if len(cfg.Libraries) < 400 {
		t.Errorf("expected at least 400 libraries in production config, got %d", len(cfg.Libraries))
	}

	// Verify secretmanager is among the parsed libraries
	var foundSM bool
	for _, lib := range cfg.Libraries {
		if lib.Output == "google/cloud/secretmanager/v1" {
			foundSM = true
			if lib.Name != "google-cloud-secretmanager-v1" {
				t.Errorf("expected name google-cloud-secretmanager-v1, got %s", lib.Name)
			}
			if lib.Cpp == nil || lib.Cpp.ForwardingProductPath != "google/cloud/secretmanager" {
				t.Errorf("secretmanager forwarding_product_path mismatch: %+v", lib.Cpp)
			}
			if lib.Cpp.InitialCopyrightYear != "2021" {
				t.Errorf("expected initial copyright year 2021, got %s", lib.Cpp.InitialCopyrightYear)
			}
			break
		}
	}
	if !foundSM {
		t.Errorf("expected secretmanager/v1 to be found among parsed libraries")
	}

	// Verify a compute service is among parsed libraries
	var foundCompute bool
	for _, lib := range cfg.Libraries {
		if lib.Output == "google/cloud/compute/addresses/v1" {
			foundCompute = true
			if lib.Name != "google-cloud-compute-addresses-v1" {
				t.Errorf("expected name google-cloud-compute-addresses-v1, got %s", lib.Name)
			}
			if lib.Cpp == nil || !lib.Cpp.GenerateRestTransport {
				t.Errorf("compute addresses generate_rest_transport should be true: %+v", lib.Cpp)
			}
			if lib.Cpp.GenerateGrpcTransport == nil || *lib.Cpp.GenerateGrpcTransport {
				t.Errorf("compute addresses generate_grpc_transport should be false: %+v", lib.Cpp)
			}
			break
		}
	}
	if !foundCompute {
		t.Errorf("expected compute/addresses/v1 to be found among parsed libraries")
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
	cfg, err := ConvertTextprotoToConfig([]byte(input))
	if err != nil {
		t.Fatalf("ConvertTextprotoToConfig failed: %v", err)
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
	if lib.Cpp.GenerateGrpcTransport == nil || *lib.Cpp.GenerateGrpcTransport {
		t.Errorf("expected GenerateGrpcTransport to be false")
	}
}

func TestConvertConfig_Errors(t *testing.T) {
	for _, test := range []struct {
		name  string
		input string
	}{
		{"unterminated_string", `service { service_proto_path: "foo }`},
		{"missing_brace", `service service_proto_path: "foo" }`},
		{"unclosed_brace", `service { service_proto_path: "foo"`},
		{"unclosed_bracket", `service { omitted_rpcs: [ "foo"`},
		{"unclosed_idempotency_brace", `service { idempotency_overrides: [ { rpc_name: "foo"`},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, err := ConvertTextprotoToConfig([]byte(test.input))
			if err == nil {
				t.Errorf("expected error for %s, got nil", test.name)
			}
		})
	}
}
