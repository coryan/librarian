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
	"embed"
	"os"
	"path/filepath"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/sidekick/api"
	"github.com/googleapis/librarian/internal/sidekick/language"
)

//go:embed all:templates
var templates embed.FS

// Generate generates a C++ client library from the given API model.
func Generate(ctx context.Context, model *api.API, outdir string, library *config.Library) error {
	c := newCodec(model, outdir, library)
	return c.generate(ctx)
}

func (c *codec) generate(_ context.Context) error {
	if len(c.Model.Services) == 0 {
		if err := os.MkdirAll(c.Output, 0o755); err != nil {
			return err
		}
		cmakePath := filepath.Join(c.Output, "CMakeLists.txt")
		return os.WriteFile(cmakePath, []byte{}, 0o644)
	}
	if err := c.annotateModel(); err != nil {
		return err
	}
	provider := func(name string) (string, error) {
		contents, err := templates.ReadFile(name)
		if err != nil {
			return "", err
		}
		return string(contents), nil
	}
	return c.generateServices(c.Output, provider)
}

func (c *codec) generateServices(outdir string, provider language.TemplateProvider) error {
	fwdRel := c.forwardingRelDir()
	for _, s := range c.Model.Services {
		if s.Codec == nil {
			continue
		}
		ann, ok := s.Codec.(*serviceAnnotations)
		if !ok {
			continue
		}
		for _, gen := range ann.generatedFiles(fwdRel) {
			if err := language.GenerateService(outdir, s, provider, gen); err != nil {
				return err
			}
		}
	}
	return nil
}
