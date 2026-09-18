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
	"embed"
	"fmt"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/sidekick/api"
)

//go:embed all:templates
var templates embed.FS

// Generate orchestrates C++ client library code generation for the provided API model.
func Generate(ctx context.Context, model *api.API, outdir string, library *config.Library) error {
	codec, err := newCodec(model, library, outdir)
	if err != nil {
		return fmt.Errorf("creating C++ codec: %w", err)
	}
	if err := codec.annotateModel(); err != nil {
		return fmt.Errorf("annotating C++ model: %w", err)
	}
	return nil
}
