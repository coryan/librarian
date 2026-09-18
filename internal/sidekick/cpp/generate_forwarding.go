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

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/sidekick/api"
)

func generateForwardingClientHeader(_ *api.Service, serviceVars map[string]string, _ *config.Library) (string, string) {
	headerPath := serviceVars["forwarding_client_header_path"]
	guard := formatHeaderIncludeGuard(headerPath)
	data := map[string]any{
		"header_include_guard": guard,
		"proto_file_name":      serviceVars["proto_file_name"],
		"copyright_year":       serviceVars["copyright_year"],
		"product_namespace":    serviceVars["product_namespace"],
		"forwarding_namespace": serviceVars["forwarding_namespace"],
		"client_class_name":    serviceVars["client_class_name"],
		"local_includes": []string{
			serviceVars["forwarding_connection_header_path"],
			serviceVars["client_header_path"],
		},
	}
	content, err := renderTemplate("templates/forwarding/client.h.mustache", data)
	if err != nil {
		panic(err)
	}
	return filepath.Clean(headerPath), content
}

func generateForwardingConnectionHeader(_ *api.Service, serviceVars map[string]string, _ *config.Library) (string, string) {
	headerPath := serviceVars["forwarding_connection_header_path"]
	guard := formatHeaderIncludeGuard(headerPath)
	data := map[string]any{
		"header_include_guard":                  guard,
		"proto_file_name":                       serviceVars["proto_file_name"],
		"copyright_year":                        serviceVars["copyright_year"],
		"product_namespace":                     serviceVars["product_namespace"],
		"forwarding_namespace":                  serviceVars["forwarding_namespace"],
		"connection_class_name":                 serviceVars["connection_class_name"],
		"limited_error_count_retry_policy_name": serviceVars["limited_error_count_retry_policy_name"],
		"limited_time_retry_policy_name":        serviceVars["limited_time_retry_policy_name"],
		"retry_policy_name":                     serviceVars["retry_policy_name"],
		"local_includes": []string{
			serviceVars["forwarding_idempotency_policy_header_path"],
			serviceVars["connection_header_path"],
		},
	}
	content, err := renderTemplate("templates/forwarding/connection.h.mustache", data)
	if err != nil {
		panic(err)
	}
	return filepath.Clean(headerPath), content
}

func generateForwardingIdempotencyPolicyHeader(_ *api.Service, serviceVars map[string]string, _ *config.Library) (string, string) {
	headerPath := serviceVars["forwarding_idempotency_policy_header_path"]
	guard := formatHeaderIncludeGuard(headerPath)
	data := map[string]any{
		"header_include_guard":   guard,
		"proto_file_name":        serviceVars["proto_file_name"],
		"copyright_year":         serviceVars["copyright_year"],
		"product_namespace":      serviceVars["product_namespace"],
		"forwarding_namespace":   serviceVars["forwarding_namespace"],
		"idempotency_class_name": serviceVars["idempotency_class_name"],
		"local_includes": []string{
			serviceVars["idempotency_policy_header_path"],
		},
	}
	content, err := renderTemplate("templates/forwarding/connection_idempotency_policy.h.mustache", data)
	if err != nil {
		panic(err)
	}
	return filepath.Clean(headerPath), content
}

func generateForwardingMockConnectionHeader(_ *api.Service, serviceVars map[string]string, _ *config.Library) (string, string) {
	headerPath := serviceVars["forwarding_mock_connection_header_path"]
	guard := formatHeaderIncludeGuard(headerPath)
	data := map[string]any{
		"header_include_guard":       guard,
		"proto_file_name":            serviceVars["proto_file_name"],
		"copyright_year":             serviceVars["copyright_year"],
		"product_mocks_namespace":    serviceVars["product_mocks_namespace"],
		"forwarding_mocks_namespace": serviceVars["forwarding_mocks_namespace"],
		"mock_connection_class_name": serviceVars["mock_connection_class_name"],
		"local_includes": []string{
			serviceVars["forwarding_connection_header_path"],
			serviceVars["mock_connection_header_path"],
		},
	}
	content, err := renderTemplate("templates/forwarding/mock_connection.h.mustache", data)
	if err != nil {
		panic(err)
	}
	return filepath.Clean(headerPath), content
}

func generateForwardingOptionsHeader(_ *api.Service, serviceVars map[string]string, methods []*api.Method, _ *config.Library) (string, string) {
	headerPath := serviceVars["forwarding_options_header_path"]
	guard := formatHeaderIncludeGuard(headerPath)
	data := map[string]any{
		"header_include_guard": guard,
		"proto_file_name":      serviceVars["proto_file_name"],
		"copyright_year":       serviceVars["copyright_year"],
		"product_namespace":    serviceVars["product_namespace"],
		"forwarding_namespace": serviceVars["forwarding_namespace"],
		"service_name":         serviceVars["service_name"],
		"has_lro":              hasLongrunningMethod(methods),
		"local_includes": []string{
			serviceVars["forwarding_connection_header_path"],
			serviceVars["forwarding_idempotency_policy_header_path"],
			serviceVars["options_header_path"],
		},
	}
	content, err := renderTemplate("templates/forwarding/options.h.mustache", data)
	if err != nil {
		panic(err)
	}
	return filepath.Clean(headerPath), content
}
