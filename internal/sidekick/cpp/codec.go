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
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/sidekick/api"
)

// codec represents the configuration and state for a C++ sidekick Codec.
type codec struct {
	// Model is the universal API IR model.
	Model *api.API

	// Library is the librarian library configuration.
	Library *config.Library

	// Cpp is the C++-specific library configuration.
	Cpp *config.CppLibrary

	// Output is the output directory path.
	Output string
}

func newCodec(model *api.API, outdir string, library *config.Library) *codec {
	var cppCfg *config.CppLibrary
	if library != nil && library.Cpp != nil {
		cppCfg = library.Cpp
	} else {
		cppCfg = &config.CppLibrary{}
	}
	return &codec{
		Model:   model,
		Library: library,
		Cpp:     cppCfg,
		Output:  outdir,
	}
}
