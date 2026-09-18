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

// Package cpp provides functionality for generating C++ client libraries.
package cpp

import (
	"context"
	"path/filepath"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/serviceconfig"
	"github.com/googleapis/librarian/internal/sidekick/api"
	sidekickcpp "github.com/googleapis/librarian/internal/sidekick/cpp"
	"github.com/googleapis/librarian/internal/sidekick/parser"
	"github.com/googleapis/librarian/internal/sources"
)

// Generate generates a C++ client library.
func Generate(ctx context.Context, cfg *config.Config, library *config.Library, src *sources.Sources) error {
	var model *api.API
	if len(library.APIs) > 0 && src != nil && src.Googleapis != "" {
		var pc *config.Protoc
		if cfg != nil && cfg.Tools != nil {
			pc = cfg.Tools.Protoc
		}
		modelConfig := libraryToModelConfig(library, library.APIs[0], src, pc)
		var err error
		model, err = parser.CreateModel(modelConfig)
		if err != nil {
			return err
		}
	} else {
		model = api.NewTestAPI(nil, nil, nil)
	}

	if src != nil && src.Googleapis != "" && library.Cpp != nil && library.Cpp.SourceRoot == "" {
		library.Cpp.SourceRoot = src.Googleapis
	}

	return sidekickcpp.Generate(ctx, model, library.Output, library)
}

// DefaultOutput derives the default output directory from the API path.
func DefaultOutput(api, defaultOut string) string {
	if defaultOut != "" {
		return filepath.Join(defaultOut, api)
	}
	return api
}

// Add executes C++-specific mutations of the given [config.Library]
// entry to be added to librarian.yaml via `librarian add`.
func Add(lib *config.Library, cfg *config.Config) *config.Library {
	if lib.Cpp == nil {
		lib.Cpp = &config.CppLibrary{}
	}
	if lib.Cpp.ProductPath == "" {
		lib.Cpp.ProductPath = lib.Output
	}
	return lib
}

func libraryToModelConfig(library *config.Library, apiCfg *config.API, src *sources.Sources, pc *config.Protoc) *parser.ModelConfig {
	sourceConfig := sources.NewSourceConfig(src, library.Roots)
	root := src.Googleapis
	apiPath := apiCfg.Path
	if filepath.Ext(apiPath) != "" {
		apiPath = filepath.Dir(apiPath)
	}
	svcConfig, err := serviceconfig.Find(root, apiPath, config.LanguageCpp)
	if err != nil {
		svcConfig = &serviceconfig.API{}
	}
	if library.Cpp != nil && library.Cpp.OverrideServiceConfigYAMLName != "" {
		svcConfig.ServiceConfig = library.Cpp.OverrideServiceConfigYAMLName
	}
	specFormat := config.SpecProtobuf
	if library.SpecificationFormat != "" {
		specFormat = library.SpecificationFormat
	}

	return &parser.ModelConfig{
		Language:            config.LanguageCpp,
		SpecificationFormat: specFormat,
		ServiceConfig:       svcConfig.ServiceConfig,
		SpecificationSource: apiPath,
		Source:              sourceConfig,
		Protoc:              pc,
	}
}
