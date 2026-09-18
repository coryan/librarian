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
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFormat_FormatsCppFiles(t *testing.T) {
	tempDir := t.TempDir()

	unformattedH := `
namespace google {
namespace cloud {
class   Foo   {
 public:
  void   bar ( ) ;
};
}
}
`
	unformattedCC := `
#include "foo.h"
namespace google {
namespace cloud {
void Foo::bar(  ) {
}
}
}
`
	hPath := filepath.Join(tempDir, "foo.h")
	ccPath := filepath.Join(tempDir, "foo.cc")
	if err := os.WriteFile(hPath, []byte(unformattedH), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(ccPath, []byte(unformattedCC), 0644); err != nil {
		t.Fatal(err)
	}

	if err := Format(t.Context(), tempDir); err != nil {
		t.Fatalf("Format failed: %v", err)
	}

	formattedH, err := os.ReadFile(hPath)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(formattedH), "class   Foo") || strings.Contains(string(formattedH), "bar ( )") {
		t.Errorf("expected clang-format to normalize spaces, got:\n%s", string(formattedH))
	}

	formattedCC, err := os.ReadFile(ccPath)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(formattedCC), "bar(  )") {
		t.Errorf("expected clang-format to normalize spaces, got:\n%s", string(formattedCC))
	}
}

func TestFormat_EmptyDir(t *testing.T) {
	tempDir := t.TempDir()
	if err := Format(t.Context(), tempDir); err != nil {
		t.Fatalf("expected nil for empty dir, got %v", err)
	}
}

func TestFormat_NonexistentDir(t *testing.T) {
	nonExistent := filepath.Join(t.TempDir(), "does_not_exist")
	if err := Format(t.Context(), nonExistent); err != nil {
		t.Fatalf("expected nil for nonexistent dir, got %v", err)
	}
}

func TestFormatFiles_Empty(t *testing.T) {
	if err := FormatFiles(t.Context()); err != nil {
		t.Fatalf("expected nil for empty files list, got %v", err)
	}
}
