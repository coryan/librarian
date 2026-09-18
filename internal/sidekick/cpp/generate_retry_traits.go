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

func generateRetryTraitsHeader(_ *api.Service, serviceVars map[string]string, _ *config.Library) (string, string, bool) {
	if serviceVars["retry_status_code_expression"] == "" {
		return "", "", false
	}
	headerPath := serviceVars["retry_traits_header_path"]
	guard := formatHeaderIncludeGuard(headerPath)

	data := map[string]any{
		"header_include_guard":         guard,
		"copyright_year":               serviceVars["copyright_year"],
		"proto_file_name":              serviceVars["proto_file_name"],
		"product_internal_namespace":   serviceVars["product_internal_namespace"],
		"retry_traits_name":            serviceVars["retry_traits_name"],
		"retry_status_code_expression": serviceVars["retry_status_code_expression"],
	}

	content, err := renderTemplate("templates/internal/retry_traits.h.mustache", data)
	if err != nil {
		panic(err)
	}

	return filepath.Clean(headerPath), content, true
}
