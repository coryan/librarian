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

// CppDefault contains C++-specific default configuration shared by all C++ libraries.
type CppDefault struct {
	// ProductPath is the relative path where generated source and headers are placed.
	ProductPath string `yaml:"product_path,omitempty"`

	// ForwardingProductPath is the relative path for top-level forwarding headers.
	ForwardingProductPath string `yaml:"forwarding_product_path,omitempty"`

	// GenerateRestTransport indicates whether REST transport code should be generated.
	GenerateRestTransport bool `yaml:"generate_rest_transport,omitempty"`

	// GenerateGrpcTransport indicates whether gRPC transport code should be generated.
	// When nil, defaults to true.
	GenerateGrpcTransport *bool `yaml:"generate_grpc_transport,omitempty"`

	// EndpointLocationStyle specifies the endpoint location resolution style.
	EndpointLocationStyle string `yaml:"endpoint_location_style,omitempty"`

	// BackwardsCompatibilityNamespace enables backwards compatibility namespace aliases.
	BackwardsCompatibilityNamespace bool `yaml:"backwards_compatibility_namespace_alias,omitempty"`

	// RetryableStatusCodes lists the gRPC status codes considered retryable.
	RetryableStatusCodes []string `yaml:"retryable_status_codes,omitempty"`

	// GenerateRoundRobinDecorator indicates whether to generate the round robin decorator.
	GenerateRoundRobinDecorator bool `yaml:"generate_round_robin_decorator,omitempty"`

	// OmitRepoMetadata disables generating .repo-metadata.json for the library.
	OmitRepoMetadata bool `yaml:"omit_repo_metadata,omitempty"`

	// Experimental indicates whether the service is experimental, adding ExperimentalTag
	// parameters to client constructors and connection factory functions.
	Experimental bool `yaml:"experimental,omitempty"`

	// PreserveProtoFieldNamesInJson indicates whether REST services expect JSON field names
	// in snake_case (proto field names) rather than camelCase.
	PreserveProtoFieldNamesInJson bool `yaml:"preserve_proto_field_names_in_json,omitempty"`

	// InitialCopyrightYear specifies the initial copyright year to use in generated files.
	InitialCopyrightYear string `yaml:"initial_copyright_year,omitempty"`
}

// HasGrpcTransport returns whether gRPC transport generation is enabled.
// Defaults to true when GenerateGrpcTransport is nil.
func (c *CppDefault) HasGrpcTransport() bool {
	if c == nil || c.GenerateGrpcTransport == nil {
		return true
	}
	return *c.GenerateGrpcTransport
}

// CppLibrary contains C++-specific library configuration.
// It inherits from CppDefault, allowing library-specific overrides of global settings.
type CppLibrary struct {
	CppDefault `yaml:",inline"`

	// ServiceEndpointEnvVar is the name of the environment variable used to override the service endpoint.
	ServiceEndpointEnvVar string `yaml:"service_endpoint_env_var,omitempty"`

	// EmulatorEndpointEnvVar is the name of the environment variable used to configure an emulator endpoint.
	EmulatorEndpointEnvVar string `yaml:"emulator_endpoint_env_var,omitempty"`

	// OmittedRPCs is a list of RPC names to omit from code generation.
	OmittedRPCs []string `yaml:"omitted_rpcs,omitempty"`

	// GenAsyncRPCs is a list of RPC names for which asynchronous client methods should be generated.
	GenAsyncRPCs []string `yaml:"gen_async_rpcs,omitempty"`

	// OmittedServices is a list of service names to omit from code generation.
	OmittedServices []string `yaml:"omitted_services,omitempty"`

	// IdempotencyOverrides defines custom idempotency settings for specific RPCs.
	IdempotencyOverrides []IdempotencyRule `yaml:"idempotency_overrides,omitempty"`

	// OmitClient disables generating the high-level Client class.
	OmitClient bool `yaml:"omit_client,omitempty"`

	// OmitConnection disables generating the Connection interface and implementation.
	OmitConnection bool `yaml:"omit_connection,omitempty"`

	// OmitStubFactory disables generating the StubFactory.
	OmitStubFactory bool `yaml:"omit_stub_factory,omitempty"`

	// OmitStreamingUpdater disables the generated resumption function for client-streaming RPCs.
	OmitStreamingUpdater bool `yaml:"omit_streaming_updater,omitempty"`

	// ServiceNameMapping maps upstream service names to custom library class names.
	ServiceNameMapping map[string]string `yaml:"service_name_mapping,omitempty"`

	// ServiceNameToComment maps upstream service names to custom replacement comments.
	ServiceNameToComment map[string]string `yaml:"service_name_to_comment,omitempty"`

	// AdditionalProtoFiles lists extra proto files to include in generation.
	AdditionalProtoFiles []string `yaml:"additional_proto_files,omitempty"`

	// ServiceConfig is the path to the service configuration YAML file.
	ServiceConfig string `yaml:"service_config,omitempty"`

	// OverrideServiceConfigYamlName is an alias for ServiceConfig matching generator_config.proto.
	OverrideServiceConfigYamlName string `yaml:"override_service_config_yaml_name,omitempty"`

	// ProtoFileSource specifies the source type of proto files (e.g. GOOGLEAPIS, DISCOVERY_DOCUMENT).
	ProtoFileSource string `yaml:"proto_file_source,omitempty"`
}

// GetServiceConfig returns the service configuration file path, checking ServiceConfig first
// and falling back to OverrideServiceConfigYamlName.
func (c *CppLibrary) GetServiceConfig() string {
	if c == nil {
		return ""
	}
	if c.ServiceConfig != "" {
		return c.ServiceConfig
	}
	return c.OverrideServiceConfigYamlName
}

// IdempotencyRule defines an idempotency override for an RPC.
type IdempotencyRule struct {
	// RPCName is the name of the RPC (e.g. "GoldenThingAdmin.DropDatabase").
	RPCName string `yaml:"rpc_name"`

	// Idempotency is the idempotency level (e.g. "IDEMPOTENT", "NON_IDEMPOTENT").
	Idempotency string `yaml:"idempotency"`
}
