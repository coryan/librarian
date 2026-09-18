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
	"github.com/googleapis/librarian/internal/sidekick/api"
)

func TestGenerate_EmitsCMakeLists(t *testing.T) {
	outdir := t.TempDir()
	model := api.NewTestAPI(nil, nil, nil)
	lib := &config.Library{
		Name:   "test-library",
		Output: outdir,
		Cpp:    &config.CppLibrary{},
	}

	ctx := context.Background()
	if err := Generate(ctx, model, outdir, lib); err != nil {
		t.Fatalf("Generate() failed: %v", err)
	}

	cmakeFile := filepath.Join(outdir, "CMakeLists.txt")
	info, err := os.Stat(cmakeFile)
	if err != nil {
		t.Fatalf("expected CMakeLists.txt to exist: %v", err)
	}
	if info.Size() != 0 {
		t.Errorf("expected empty CMakeLists.txt, got %d bytes", info.Size())
	}
}
