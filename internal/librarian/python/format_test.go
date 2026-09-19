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

package python

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/testhelper"
)

func TestFormat(t *testing.T) {
	testhelper.RequireCommand(t, "ruff")

	dir := t.TempDir()
	unformatted := `def hello( ):
    x = 1
    return x
`
	filePath := filepath.Join(dir, "example.py")
	if err := os.WriteFile(filePath, []byte(unformatted), 0o644); err != nil {
		t.Fatal(err)
	}

	library := &config.Library{
		Name:   "test-python-lib",
		Output: dir,
	}

	if err := Format(t.Context(), library); err != nil {
		t.Fatalf("Format() failed: %v", err)
	}

	formatted, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatal(err)
	}
	if len(formatted) == 0 {
		t.Errorf("formatted file is empty")
	}
}

func TestFormat_EmptyOutput(t *testing.T) {
	library := &config.Library{}
	if err := Format(t.Context(), library); err != nil {
		t.Fatalf("Format() on empty library failed: %v", err)
	}
}

func TestFormat_NilLibrary(t *testing.T) {
	if err := Format(t.Context(), nil); err != nil {
		t.Fatalf("Format() on nil library failed: %v", err)
	}
}
