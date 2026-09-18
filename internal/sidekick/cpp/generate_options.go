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
	"slices"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/sidekick/api"
)

func generateOptionsHeader(_ *api.Service, serviceVars map[string]string, methods []*api.Method, _ *config.Library) (string, string) {
	headerPath := serviceVars["options_header_path"]
	guard := formatHeaderIncludeGuard(headerPath)

	localIncludes := []string{
		serviceVars["connection_header_path"],
		serviceVars["idempotency_policy_header_path"],
		"google/cloud/backoff_policy.h",
		"google/cloud/options.h",
		"google/cloud/version.h",
	}
	slices.Sort(localIncludes)

	data := map[string]any{
		"header_include_guard":   guard,
		"copyright_year":         serviceVars["copyright_year"],
		"proto_file_name":        serviceVars["proto_file_name"],
		"product_namespace":      serviceVars["product_namespace"],
		"product_options_page":   serviceVars["product_options_page"],
		"retry_policy_name":      serviceVars["retry_policy_name"],
		"service_name":           serviceVars["service_name"],
		"idempotency_class_name": serviceVars["idempotency_class_name"],
		"has_lro":                hasLongrunningMethod(methods),
		"local_includes":         localIncludes,
	}

	content, err := renderTemplate("templates/options.h.mustache", data)
	if err != nil {
		panic(err)
	}

	return filepath.Clean(headerPath), content
}
