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
	"strings"

	"github.com/googleapis/librarian/internal/config"
)

// Add executes C++-specific mutations of the given [config.Library] entry
// to be added to librarian.yaml via `librarian add`.
func Add(lib *config.Library) *config.Library {
	if len(lib.APIs) > 0 {
		apiPath := lib.APIs[0].Path
		productPath := apiPath
		if strings.HasSuffix(productPath, ".proto") {
			productPath = filepath.Dir(productPath)
		}
		if lib.Output == "" {
			lib.Output = productPath
		}
		if lib.Cpp == nil {
			lib.Cpp = &config.CppLibrary{
				CppDefault: config.CppDefault{
					ProductPath:          productPath,
					InitialCopyrightYear: lib.CopyrightYear,
					RetryableStatusCodes: []string{"kUnavailable"},
				},
			}
		}
	}
	return lib
}
