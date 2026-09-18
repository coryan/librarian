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

// Package cpp implements librarian orchestration for C++ client libraries.
package cpp

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/sidekick/api"
	sidekickcpp "github.com/googleapis/librarian/internal/sidekick/cpp"
	"github.com/googleapis/librarian/internal/sidekick/parser"
	"github.com/googleapis/librarian/internal/sources"
)

// Generate generates a C++ client library.
func Generate(ctx context.Context, cfg *config.Config, library *config.Library, src *sources.Sources) error {
	if len(library.APIs) == 0 {
		return fmt.Errorf("library %q has no APIs defined", library.Name)
	}

	var pc *config.Protoc
	if cfg != nil && cfg.Tools != nil {
		pc = cfg.Tools.Protoc
	}

	for _, apiCfg := range library.APIs {
		modelConfig, err := libraryToModelConfig(library, apiCfg, src, pc)
		if err != nil {
			return err
		}
		model, err := parser.CreateModel(modelConfig)
		if err != nil {
			return err
		}
		outdir := library.Output
		if outdir == "" && library.Cpp != nil && library.Cpp.ProductPath != "" {
			outdir = library.Cpp.ProductPath
		}
		if outdir == "" {
			outdir = "."
		}
		if err := sidekickcpp.Generate(ctx, model, outdir, library); err != nil {
			return err
		}
	}
	return nil
}

// Format formats all generated C++ files in the library output directories.
func Format(ctx context.Context, library *config.Library) error {
	dirs := map[string]struct{}{}
	if library.Output != "" {
		dirs[library.Output] = struct{}{}
	}
	if library.Cpp != nil {
		if library.Cpp.ProductPath != "" {
			dirs[library.Cpp.ProductPath] = struct{}{}
		}
		if library.Cpp.ForwardingProductPath != "" {
			dirs[library.Cpp.ForwardingProductPath] = struct{}{}
		}
	}
	for dir := range dirs {
		if err := sidekickcpp.Format(ctx, dir); err != nil {
			return err
		}
	}
	return nil
}

// libraryToModelConfig constructs a ModelConfig for parser.CreateModel.
func libraryToModelConfig(library *config.Library, apiCfg *config.API, srcs *sources.Sources, pc *config.Protoc) (*parser.ModelConfig, error) {
	specFormat := config.SpecProtobuf
	if library.SpecificationFormat != "" {
		specFormat = library.SpecificationFormat
	}

	specSource := apiCfg.Path
	var includeList []string
	if strings.HasSuffix(specSource, ".proto") {
		protoFile := filepath.Base(specSource)
		specSource = filepath.Dir(specSource)
		includeList = append(includeList, protoFile)
		if library.Cpp != nil {
			for _, f := range library.Cpp.AdditionalProtoFiles {
				rel := f
				if specSource != "" && strings.HasPrefix(f, specSource+"/") {
					rel = strings.TrimPrefix(f, specSource+"/")
				}
				includeList = append(includeList, rel)
			}
		}
	}

	roots := library.Roots
	if len(roots) == 0 && srcs != nil && srcs.Showcase != "" {
		roots = []string{"showcase", "googleapis"}
	}

	var src *sources.SourceConfig
	if srcs != nil {
		src = sources.NewSourceConfig(srcs, roots)
		if len(includeList) > 0 {
			src.IncludeList = includeList
		}
	}

	var svcConfigFile string
	var skippedIDs []string
	if library.Cpp != nil {
		svcConfigFile = library.Cpp.GetServiceConfig()
		skippedIDs = library.Cpp.OmittedServices
	}

	modelCfg := &parser.ModelConfig{
		Language:            config.LanguageCpp,
		SpecificationFormat: specFormat,
		SpecificationSource: specSource,
		Source:              src,
		Protoc:              pc,
		ServiceConfig:       svcConfigFile,
		Override: api.ModelOverride{
			Title:      library.TitleOverride,
			SkippedIDs: skippedIDs,
		},
	}
	return modelCfg, nil
}
