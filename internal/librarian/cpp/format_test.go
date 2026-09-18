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
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/googleapis/librarian/internal/config"
)

func TestFormat_NoFiles(t *testing.T) {
	outdir := t.TempDir()
	lib := &config.Library{
		Output: outdir,
	}
	ctx := context.Background()
	if err := Format(ctx, nil, lib); err != nil {
		t.Fatalf("Format() error = %v", err)
	}
}

func TestFormat_CustomClangFormatPath(t *testing.T) {
	outdir := t.TempDir()
	// create a dummy source file
	_ = os.WriteFile(filepath.Join(outdir, "foo.cc"), []byte("int main() {}\n"), 0o644)
	cfg := &config.Config{
		Tools: &config.Tools{
			ClangFormat: &config.ClangFormat{
				Path: "nonexistent-clang-format-path",
			},
		},
	}
	lib := &config.Library{
		Output: outdir,
	}
	ctx := context.Background()
	// nonexistent binary should be skipped gracefully when not found on LookPath
	if err := Format(ctx, cfg, lib); err != nil {
		t.Fatalf("expected Format() with nonexistent path to return nil on LookPath failure, got %v", err)
	}
}
