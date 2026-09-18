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

func generateForwardingClientHeader(_ *api.Service, ann *serviceAnnotations, _ *config.Library) (string, string) {
	headerPath := ann.ForwardingClientHeaderPath()
	guard := formatHeaderIncludeGuard(headerPath)
	data := map[string]any{
		"header_include_guard": guard,
		"proto_file_name":      ann.ProtoFileName,
		"copyright_year":       ann.CopyrightYear,
		"product_namespace":    ann.Namespace(),
		"forwarding_namespace": ann.ForwardingNamespace(),
		"client_class_name":    ann.ClientClassName(),
		"local_includes": []string{
			ann.ForwardingConnectionHeaderPath(),
			ann.ClientHeaderPath(),
		},
	}
	content, err := renderTemplate("templates/forwarding/client.h.mustache", data)
	if err != nil {
		panic(err)
	}
	return filepath.Clean(headerPath), content
}

func generateForwardingConnectionHeader(_ *api.Service, ann *serviceAnnotations, _ *config.Library) (string, string) {
	headerPath := ann.ForwardingConnectionHeaderPath()
	guard := formatHeaderIncludeGuard(headerPath)
	data := map[string]any{
		"header_include_guard":                  guard,
		"proto_file_name":                       ann.ProtoFileName,
		"copyright_year":                        ann.CopyrightYear,
		"product_namespace":                     ann.Namespace(),
		"forwarding_namespace":                  ann.ForwardingNamespace(),
		"connection_class_name":                 ann.ConnectionClassName(),
		"limited_error_count_retry_policy_name": ann.LimitedErrorCountRetryPolicyName(),
		"limited_time_retry_policy_name":        ann.LimitedTimeRetryPolicyName(),
		"retry_policy_name":                     ann.RetryPolicyName(),
		"local_includes": []string{
			ann.ForwardingIdempotencyPolicyHeaderPath(),
			ann.ConnectionHeaderPath(),
		},
	}
	content, err := renderTemplate("templates/forwarding/connection.h.mustache", data)
	if err != nil {
		panic(err)
	}
	return filepath.Clean(headerPath), content
}

func generateForwardingIdempotencyPolicyHeader(_ *api.Service, ann *serviceAnnotations, _ *config.Library) (string, string) {
	headerPath := ann.ForwardingIdempotencyPolicyHeaderPath()
	guard := formatHeaderIncludeGuard(headerPath)
	data := map[string]any{
		"header_include_guard":   guard,
		"proto_file_name":        ann.ProtoFileName,
		"copyright_year":         ann.CopyrightYear,
		"product_namespace":      ann.Namespace(),
		"forwarding_namespace":   ann.ForwardingNamespace(),
		"idempotency_class_name": ann.IdempotencyClassName(),
		"local_includes": []string{
			ann.IdempotencyHeaderPath(),
		},
	}
	content, err := renderTemplate("templates/forwarding/connection_idempotency_policy.h.mustache", data)
	if err != nil {
		panic(err)
	}
	return filepath.Clean(headerPath), content
}

func generateForwardingMockConnectionHeader(_ *api.Service, ann *serviceAnnotations, _ *config.Library) (string, string) {
	headerPath := ann.ForwardingMockConnectionHeaderPath()
	guard := formatHeaderIncludeGuard(headerPath)
	data := map[string]any{
		"header_include_guard":       guard,
		"proto_file_name":            ann.ProtoFileName,
		"copyright_year":             ann.CopyrightYear,
		"product_mocks_namespace":    ann.MocksNamespace(),
		"forwarding_mocks_namespace": ann.ForwardingMocksNamespace(),
		"mock_connection_class_name": ann.MockConnectionClassName(),
		"local_includes": []string{
			ann.ForwardingConnectionHeaderPath(),
			ann.MockConnectionHeaderPath(),
		},
	}
	content, err := renderTemplate("templates/forwarding/mock_connection.h.mustache", data)
	if err != nil {
		panic(err)
	}
	return filepath.Clean(headerPath), content
}

func generateForwardingOptionsHeader(_ *api.Service, ann *serviceAnnotations, methods []*api.Method, _ *config.Library) (string, string) {
	headerPath := ann.ForwardingOptionsHeaderPath()
	guard := formatHeaderIncludeGuard(headerPath)
	data := map[string]any{
		"header_include_guard": guard,
		"proto_file_name":      ann.ProtoFileName,
		"copyright_year":       ann.CopyrightYear,
		"product_namespace":    ann.Namespace(),
		"forwarding_namespace": ann.ForwardingNamespace(),
		"service_name":         ann.ServiceName,
		"has_lro":              hasLongrunningMethod(methods),
		"local_includes": []string{
			ann.ForwardingConnectionHeaderPath(),
			ann.ForwardingIdempotencyPolicyHeaderPath(),
			ann.OptionsHeaderPath(),
		},
	}
	content, err := renderTemplate("templates/forwarding/options.h.mustache", data)
	if err != nil {
		panic(err)
	}
	return filepath.Clean(headerPath), content
}
