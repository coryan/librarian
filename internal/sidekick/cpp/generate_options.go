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

func generateOptionsHeader(_ *api.Service, ann *serviceAnnotations, methods []*api.Method, _ *config.Library) (string, string) {
	headerPath := ann.OptionsHeaderPath()
	guard := ann.OptionsHeaderIncludeGuard()

	localIncludes := []string{
		ann.ConnectionHeaderPath(),
		ann.IdempotencyHeaderPath(),
		"google/cloud/backoff_policy.h",
		"google/cloud/options.h",
		"google/cloud/version.h",
	}
	slices.Sort(localIncludes)

	data := map[string]any{
		"header_include_guard":   guard,
		"copyright_year":         ann.CopyrightYear,
		"proto_file_name":        ann.ProtoFileName,
		"product_namespace":      ann.Namespace(),
		"product_options_page":   ann.OptionsGroupName(),
		"retry_policy_name":      ann.RetryPolicyName(),
		"service_name":           ann.ServiceName,
		"idempotency_class_name": ann.IdempotencyClassName(),
		"has_lro":                hasLongrunningMethod(methods),
		"local_includes":         localIncludes,
	}

	content, err := renderTemplate("templates/options.h.mustache", data)
	if err != nil {
		panic(err)
	}

	return filepath.Clean(headerPath), content
}
