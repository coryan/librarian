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

func generateRetryTraitsHeader(_ *api.Service, ann *serviceAnnotations, _ *config.Library) (string, string, bool) {
	expr := ann.RetryStatusCodeExpression()
	if expr == "" {
		return "", "", false
	}
	headerPath := ann.RetryTraitsHeaderPath()
	guard := ann.RetryTraitsHeaderIncludeGuard()

	data := map[string]any{
		"header_include_guard":         guard,
		"copyright_year":               ann.CopyrightYear,
		"proto_file_name":              ann.ProtoFileName,
		"product_internal_namespace":   ann.InternalNamespace(),
		"retry_traits_name":            ann.RetryTraitsName(),
		"retry_status_code_expression": expr,
	}

	content, err := renderTemplate("templates/internal/retry_traits.h.mustache", data)
	if err != nil {
		panic(err)
	}

	return filepath.Clean(headerPath), content, true
}
