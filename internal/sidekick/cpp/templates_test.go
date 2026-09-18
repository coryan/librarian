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
	"io/fs"
	"strings"
	"testing"

	"github.com/cbroglie/mustache"
)

func TestTemplates_AllParse(t *testing.T) {
	expectedTemplates := []string{
		"templates/client.h.mustache",
		"templates/client.cc.mustache",
		"templates/connection.h.mustache",
		"templates/connection.cc.mustache",
		"templates/connection_idempotency_policy.h.mustache",
		"templates/connection_idempotency_policy.cc.mustache",
		"templates/options.h.mustache",
		"templates/internal/stub.h.mustache",
		"templates/internal/stub.cc.mustache",
		"templates/internal/connection_impl.h.mustache",
		"templates/internal/connection_impl.cc.mustache",
		"templates/internal/stub_factory.h.mustache",
		"templates/internal/stub_factory.cc.mustache",
		"templates/internal/option_defaults.h.mustache",
		"templates/internal/option_defaults.cc.mustache",
		"templates/internal/retry_traits.h.mustache",
		"templates/internal/sources.cc.mustache",
		"templates/internal/auth_decorator.h.mustache",
		"templates/internal/auth_decorator.cc.mustache",
		"templates/internal/logging_decorator.h.mustache",
		"templates/internal/logging_decorator.cc.mustache",
		"templates/internal/metadata_decorator.h.mustache",
		"templates/internal/metadata_decorator.cc.mustache",
		"templates/internal/tracing_connection.h.mustache",
		"templates/internal/tracing_connection.cc.mustache",
		"templates/internal/tracing_stub.h.mustache",
		"templates/internal/tracing_stub.cc.mustache",
		"templates/internal/round_robin_decorator.h.mustache",
		"templates/internal/round_robin_decorator.cc.mustache",
		"templates/mocks/mock_connection.h.mustache",
		"templates/forwarding/client.h.mustache",
		"templates/forwarding/connection.h.mustache",
		"templates/forwarding/connection_idempotency_policy.h.mustache",
		"templates/forwarding/options.h.mustache",
		"templates/forwarding/mock_connection.h.mustache",
	}

	for _, tmplPath := range expectedTemplates {
		content, err := fs.ReadFile(templates, tmplPath)
		if err != nil {
			t.Errorf("failed to read embedded template %s: %v", tmplPath, err)
			continue
		}
		tmpl, err := mustache.ParseString(string(content))
		if err != nil {
			t.Errorf("failed to parse template %s: %v", tmplPath, err)
			continue
		}
		// Test rendering with sample data
		data := map[string]string{
			"namespace":            "golden_v1",
			"namespace_internal":   "golden_v1_internal",
			"namespace_mocks":      "golden_v1_mocks",
			"forwarding_namespace": "golden",
			"client_class_name":    "GoldenKitchenSinkClient",
			"service_name":         "GoldenKitchenSink",
			"service_name_upper":   "GOLDEN_KITCHEN_SINK",
			"default_endpoint":     "goldenkitchensink.googleapis.com",
		}
		rendered, err := tmpl.Render(data)
		if err != nil {
			t.Errorf("failed to render template %s: %v", tmplPath, err)
			continue
		}
		if len(strings.TrimSpace(rendered)) == 0 {
			t.Errorf("rendered output for %s was empty", tmplPath)
		}
	}
}
