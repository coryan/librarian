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

// Package cpp provides C++ client library code generation.
package cpp

import (
	"context"
	"os"
	"path/filepath"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/sidekick/api"
)

// Generate generates a C++ client library from the given API model.
// For the initial scaffold, it emits an empty CMakeLists.txt file.
func Generate(ctx context.Context, model *api.API, outdir string, library *config.Library) error {
	c := newCodec(model, outdir, library)
	return c.generate(ctx)
}

func (c *codec) generate(_ context.Context) error {
	if err := os.MkdirAll(c.Output, 0o755); err != nil {
		return err
	}
	cmakePath := filepath.Join(c.Output, "CMakeLists.txt")
	return os.WriteFile(cmakePath, []byte{}, 0o644)
}
