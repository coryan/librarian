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
	"strings"
	"testing"
)

func TestRenderTemplate_NotFound(t *testing.T) {
	_, err := renderTemplate("templates/nonexistent.mustache", nil)
	if err == nil {
		t.Errorf("expected error for nonexistent template, got nil")
	}
}

func TestRenderTemplate_ExistingTemplate(t *testing.T) {
	data := map[string]any{
		"copyright_year": "2026",
	}
	s, err := renderTemplate("templates/common/copyright_header.mustache", data)
	if err != nil {
		t.Fatalf("renderTemplate failed: %v", err)
	}
	if !strings.Contains(s, "Copyright 2026 Google LLC") {
		t.Errorf("expected rendered content to contain 'Copyright 2026 Google LLC', got %q", s)
	}
}
