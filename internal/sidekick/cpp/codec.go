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

// Package cpp implements C++ client library code generation.
package cpp

import (
	"errors"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/sidekick/api"
)

// Codec coordinates model enrichment and template rendering for C++ client generation.
type Codec struct {
	Model   *api.API
	Library *config.Library
	OutDir  string
}

// newCodec creates a new C++ Codec for the given model, library, and output directory.
func newCodec(model *api.API, library *config.Library, outdir string) (*Codec, error) {
	if model == nil {
		return nil, errors.New("model cannot be nil")
	}
	return &Codec{
		Model:   model,
		Library: library,
		OutDir:  outdir,
	}, nil
}

// annotateModel enriches the API model with C++ specific metadata and types.
func (c *Codec) annotateModel() {
	for _, m := range c.Model.Messages {
		for _, f := range m.Fields {
			annotateField(f)
		}
	}
	for _, svc := range c.Model.Services {
		for _, m := range svc.Methods {
			annotateMethod(m, svc, c.Library, c.Model)
		}
		annotateService(svc, c.Library, c.Model)
	}
}
