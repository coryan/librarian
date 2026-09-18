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
	"slices"
	"strings"
	"unicode"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/sidekick/api"
)

type namespaceType int

const (
	namespaceNormal namespaceType = iota
	namespaceInternal
	namespaceMocks
)

func formatProductPath(path string) string {
	path = strings.TrimPrefix(path, "/")
	if !strings.HasSuffix(path, "/") {
		path += "/"
	}
	return path
}

func parseProductPath(productPath string) (prefix, libraryName, serviceSubdirectory string) {
	parts := strings.Split(strings.Trim(productPath, "/"), "/")
	if len(parts) == 0 {
		return "", "", ""
	}
	if len(parts) > 2 && parts[0] == "google" && parts[1] == "cloud" {
		prefix = "google/cloud"
		libraryName = parts[2]
		serviceSubdirectory = strings.Join(parts[3:], "/")
		return
	}
	for i, p := range parts {
		if p == "golden" {
			prefix = strings.Join(parts[:i], "/")
			libraryName = "golden"
			serviceSubdirectory = strings.Join(parts[i+1:], "/")
			return
		}
	}
	libraryName = parts[len(parts)-1]
	prefix = strings.Join(parts[:len(parts)-1], "/")
	return
}

func libraryPath(productPath string) string {
	p, l, _ := parseProductPath(productPath)
	if p != "" {
		return p + "/" + l + "/"
	}
	return l + "/"
}

func optionsGroup(productPath string) string {
	lp := libraryPath(productPath)
	lp = strings.ReplaceAll(lp, "/", "-")
	return lp + "options"
}

func namespace(productPath string, nsType namespaceType) string {
	_, l, s := parseProductPath(productPath)
	ns := l
	if s != "" {
		ns += "_" + s
	}
	switch nsType {
	case namespaceInternal:
		ns += "_internal"
	case namespaceMocks:
		ns += "_mocks"
	}
	return ns
}

func formatHeaderIncludeGuard(headerPath string) string {
	g := "GOOGLE_CLOUD_CPP_" + headerPath
	g = strings.ReplaceAll(g, "/", "_")
	g = strings.ReplaceAll(g, ".", "_")
	return strings.ToUpper(g)
}

func camelCaseToSnakeCase(input string) string {
	var output strings.Builder
	for i := 0; i < len(input); i++ {
		c := input[i]
		lower := byte(unicode.ToLower(rune(c)))
		if c != '_' && i+2 < len(input) {
			if unicode.IsUpper(rune(input[i+1])) && unicode.IsLower(rune(input[i+2])) {
				output.WriteByte(lower)
				output.WriteByte('_')
				continue
			}
		}
		if c != '_' && i+1 < len(input) {
			if (unicode.IsLower(rune(c)) || unicode.IsDigit(rune(c))) && unicode.IsUpper(rune(input[i+1])) {
				output.WriteByte(lower)
				output.WriteByte('_')
				continue
			}
		}
		output.WriteByte(lower)
	}
	res := output.String()
	res = strings.ReplaceAll(res, "big_query", "bigquery")
	return res
}

func serviceNameToFilePath(serviceName string) string {
	components := strings.Split(serviceName, ".")
	last := components[len(components)-1]
	last = strings.TrimSuffix(last, "Service")
	components[len(components)-1] = last
	var formatted []string
	for _, c := range components {
		formatted = append(formatted, camelCaseToSnakeCase(c))
	}
	return strings.Join(formatted, "/")
}

func protoNameToCppName(protoName string) string {
	protoName = strings.TrimPrefix(protoName, ".")
	return strings.ReplaceAll(protoName, ".", "::")
}

func cppTypeToString(field *api.Field) string {
	if field == nil {
		return ""
	}
	switch field.Typez {
	case api.TypezInt32, api.TypezSint32, api.TypezSfixed32:
		return "std::int32_t"
	case api.TypezInt64, api.TypezSint64, api.TypezSfixed64:
		return "std::int64_t"
	case api.TypezUint32, api.TypezFixed32:
		return "std::uint32_t"
	case api.TypezUint64, api.TypezFixed64:
		return "std::uint64_t"
	case api.TypezDouble:
		return "double"
	case api.TypezFloat:
		return "float"
	case api.TypezBool:
		return "bool"
	case api.TypezString:
		return "std::string"
	case api.TypezBytes:
		return "std::string"
	case api.TypezEnum:
		return protoNameToCppName(field.TypezID)
	case api.TypezMessage:
		return protoNameToCppName(field.TypezID)
	default:
		return protoNameToCppName(field.TypezID)
	}
}

func buildServiceVars(svc *api.Service, lib *config.Library, model *api.API) map[string]string {
	vars := make(map[string]string)
	serviceName := svc.Name
	vars["service_name"] = serviceName

	productPath := ""
	var forwardingProductPath string
	if lib != nil && lib.Cpp != nil {
		productPath = formatProductPath(lib.Cpp.ProductPath)
		if lib.Cpp.ForwardingProductPath != "" {
			forwardingProductPath = formatProductPath(lib.Cpp.ForwardingProductPath)
		}
	}
	vars["product_path"] = productPath
	vars["forwarding_product_path"] = forwardingProductPath

	copyrightYear := "2026"
	if lib != nil && lib.CopyrightYear != "" {
		copyrightYear = lib.CopyrightYear
	}
	vars["copyright_year"] = copyrightYear

	filePathName := serviceNameToFilePath(serviceName)
	vars["file_path_name"] = filePathName

	vars["product_options_page"] = optionsGroup(productPath)
	apiVersion := ""
	for _, m := range svc.Methods {
		if m.APIVersion != "" {
			apiVersion = m.APIVersion
			break
		}
	}
	vars["api_version"] = apiVersion

	// Class names
	vars["client_class_name"] = serviceName + "Client"
	vars["connection_class_name"] = serviceName + "Connection"
	vars["connection_impl_class_name"] = serviceName + "ConnectionImpl"
	vars["idempotency_class_name"] = serviceName + "ConnectionIdempotencyPolicy"
	vars["mock_connection_class_name"] = "Mock" + serviceName + "Connection"
	vars["auth_class_name"] = serviceName + "Auth"
	vars["logging_class_name"] = serviceName + "Logging"
	vars["metadata_class_name"] = serviceName + "Metadata"
	vars["stub_class_name"] = serviceName + "Stub"
	vars["tracing_connection_class_name"] = serviceName + "TracingConnection"
	vars["tracing_stub_class_name"] = serviceName + "TracingStub"
	vars["retry_traits_name"] = serviceName + "RetryTraits"
	vars["retry_policy_name"] = serviceName + "RetryPolicy"
	vars["limited_error_count_retry_policy_name"] = serviceName + "LimitedErrorCountRetryPolicy"
	vars["limited_time_retry_policy_name"] = serviceName + "LimitedTimeRetryPolicy"
	vars["connection_options_name"] = serviceName + "ConnectionOptions"
	vars["connection_options_traits_name"] = serviceName + "ConnectionOptionsTraits"
	vars["round_robin_class_name"] = serviceName + "RoundRobin"
	vars["stub_factory_header_path"] = productPath + "internal/" + filePathName + "_stub_factory.h"
	vars["stub_factory_cc_path"] = productPath + "internal/" + filePathName + "_stub_factory.cc"

	// Paths
	vars["client_header_path"] = productPath + filePathName + "_client.h"
	vars["client_cc_path"] = productPath + filePathName + "_client.cc"
	vars["client_samples_cc_path"] = productPath + "samples/" + filePathName + "_client_samples.cc"
	vars["connection_header_path"] = productPath + filePathName + "_connection.h"
	vars["connection_cc_path"] = productPath + filePathName + "_connection.cc"
	vars["connection_impl_header_path"] = productPath + "internal/" + filePathName + "_connection_impl.h"
	vars["connection_impl_cc_path"] = productPath + "internal/" + filePathName + "_connection_impl.cc"
	vars["idempotency_policy_header_path"] = productPath + filePathName + "_connection_idempotency_policy.h"
	vars["idempotency_policy_cc_path"] = productPath + filePathName + "_connection_idempotency_policy.cc"
	vars["options_header_path"] = productPath + filePathName + "_options.h"
	vars["mock_connection_header_path"] = productPath + "mocks/mock_" + filePathName + "_connection.h"
	vars["auth_header_path"] = productPath + "internal/" + filePathName + "_auth_decorator.h"
	vars["auth_cc_path"] = productPath + "internal/" + filePathName + "_auth_decorator.cc"
	vars["logging_header_path"] = productPath + "internal/" + filePathName + "_logging_decorator.h"
	vars["logging_cc_path"] = productPath + "internal/" + filePathName + "_logging_decorator.cc"
	vars["metadata_header_path"] = productPath + "internal/" + filePathName + "_metadata_decorator.h"
	vars["metadata_cc_path"] = productPath + "internal/" + filePathName + "_metadata_decorator.cc"
	vars["stub_header_path"] = productPath + "internal/" + filePathName + "_stub.h"
	vars["stub_cc_path"] = productPath + "internal/" + filePathName + "_stub.cc"
	vars["tracing_connection_header_path"] = productPath + "internal/" + filePathName + "_tracing_connection.h"
	vars["tracing_connection_cc_path"] = productPath + "internal/" + filePathName + "_tracing_connection.cc"
	vars["tracing_stub_header_path"] = productPath + "internal/" + filePathName + "_tracing_stub.h"
	vars["tracing_stub_cc_path"] = productPath + "internal/" + filePathName + "_tracing_stub.cc"
	vars["option_defaults_header_path"] = productPath + "internal/" + filePathName + "_option_defaults.h"
	vars["option_defaults_cc_path"] = productPath + "internal/" + filePathName + "_option_defaults.cc"
	vars["retry_traits_header_path"] = productPath + "internal/" + filePathName + "_retry_traits.h"
	vars["round_robin_header_path"] = productPath + "internal/" + filePathName + "_round_robin_decorator.h"
	vars["round_robin_cc_path"] = productPath + "internal/" + filePathName + "_round_robin_decorator.cc"
	vars["sources_cc_path"] = productPath + "internal/" + filePathName + "_sources.cc"

	// Forwarding paths
	if forwardingProductPath != "" {
		vars["forwarding_client_header_path"] = forwardingProductPath + filePathName + "_client.h"
		vars["forwarding_connection_header_path"] = forwardingProductPath + filePathName + "_connection.h"
		vars["forwarding_idempotency_policy_header_path"] = forwardingProductPath + filePathName + "_connection_idempotency_policy.h"
		vars["forwarding_mock_connection_header_path"] = forwardingProductPath + "mocks/mock_" + filePathName + "_connection.h"
		vars["forwarding_options_header_path"] = forwardingProductPath + filePathName + "_options.h"
	}

	// Namespaces
	vars["product_namespace"] = namespace(productPath, namespaceNormal)
	vars["product_internal_namespace"] = namespace(productPath, namespaceInternal)
	vars["product_mocks_namespace"] = namespace(productPath, namespaceMocks)
	if forwardingProductPath != "" {
		vars["forwarding_namespace"] = namespace(forwardingProductPath, namespaceNormal)
		vars["forwarding_mocks_namespace"] = namespace(forwardingProductPath, namespaceMocks)
	}

	// Proto info
	protoFile := ""
	if loc, ok := model.DefinitionLocation(svc.ID[1:]); ok {
		protoFile = loc.Filename
	} else if lib != nil && len(lib.APIs) > 0 {
		protoFile = lib.APIs[0].Path
	}
	vars["proto_file_name"] = protoFile
	vars["proto_grpc_header_path"] = strings.TrimSuffix(protoFile, ".proto") + ".grpc.pb.h"
	vars["proto_header_path"] = strings.TrimSuffix(protoFile, ".proto") + ".pb.h"
	vars["grpc_service"] = svc.ID[1:]
	vars["grpc_stub_fqn"] = protoNameToCppName(svc.ID[1:])

	// Endpoints
	vars["service_endpoint"] = svc.DefaultHost
	endpointEnvVar := ""
	emulatorEnvVar := ""
	locationStyle := ""
	var addPbFiles []string
	var retryCodes []string
	if lib != nil && lib.Cpp != nil {
		endpointEnvVar = lib.Cpp.ServiceEndpointEnvVar
		emulatorEnvVar = lib.Cpp.EmulatorEndpointEnvVar
		locationStyle = lib.Cpp.EndpointLocationStyle
		if lib.Cpp.GenerateRoundRobinDecorator {
			vars["generate_round_robin_decorator"] = "true"
		}
		addPbFiles = lib.Cpp.AdditionalProtoFiles
		retryCodes = lib.Cpp.RetryableStatusCodes
	}
	if endpointEnvVar == "" {
		endpointEnvVar = "GOOGLE_CLOUD_CPP_" + strings.ToUpper(camelCaseToSnakeCase(svc.Name)) + "_ENDPOINT"
	}
	vars["service_endpoint_env_var"] = endpointEnvVar
	prefix := strings.TrimSuffix(endpointEnvVar, "_ENDPOINT")
	vars["service_authority_env_var"] = prefix + "_AUTHORITY"
	vars["emulator_endpoint_env_var"] = emulatorEnvVar
	vars["endpoint_location_style"] = locationStyle
	var addPb []string
	for _, add := range addPbFiles {
		addPb = append(addPb, strings.TrimSuffix(add, ".proto")+".pb.h")
	}
	vars["additional_pb_header_paths"] = strings.Join(addPb, ",")

	// Retry traits
	setRetryVars(vars, retryCodes)

	// Class comments
	vars["class_comment_block"] = formatClassComments(svc, serviceName, model)

	return vars
}

func setRetryVars(vars map[string]string, codes []string) {
	if len(codes) == 0 {
		return
	}
	sortedCodes := slices.Clone(codes)
	slices.Sort(sortedCodes)

	serviceName := vars["service_name"]
	var expr strings.Builder
	expr.WriteString("status.code() != StatusCode::kOk")

	var transientComments []string
	for _, code := range sortedCodes {
		parts := strings.Split(code, ".")
		if len(parts) == 1 {
			expr.WriteString(" && status.code() != StatusCode::" + parts[0])
			transientComments = append(transientComments, parts[0])
		} else if parts[0] == serviceName {
			expr.WriteString(" && status.code() != StatusCode::" + parts[1])
			transientComments = append(transientComments, parts[1])
		}
	}
	vars["retry_status_code_expression"] = expr.String()

	if len(transientComments) > 0 {
		var b strings.Builder
		b.WriteString("\n * In this class the following status codes are treated as transient errors:")
		for _, c := range transientComments {
			fmt.Fprintf(&b, "\n * - [`%s`](@ref google::cloud::StatusCode)", c)
		}
		vars["transient_errors_comment"] = b.String()
	}
}
