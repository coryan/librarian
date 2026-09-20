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

// Package python provides a code generator for Python GAPIC client libraries.
package python

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"path"
	"path/filepath"
	"strings"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/sidekick/api"
	"github.com/googleapis/librarian/internal/sidekick/language"
)

var (
	// ErrEscapeOutputDir indicates that a generated file output path escapes the output directory.
	ErrEscapeOutputDir = errors.New("output path escapes output directory")
	// ErrDuplicateOutputPath indicates that multiple generated files target the exact same output path.
	ErrDuplicateOutputPath = errors.New("duplicate output path")
	// ErrInvalidOutputPath indicates that a generated file output path is empty, root, or absolute.
	ErrInvalidOutputPath = errors.New("invalid output path")
)

//go:embed all:templates
var templates embed.FS

// Generate generates Python GAPIC client code from the api.API model.
func Generate(ctx context.Context, model *api.API, outdir string, library *config.Library) error {
	c, err := newCodec(model, library, outdir)
	if err != nil {
		return err
	}
	if err := c.annotateModel(); err != nil {
		return err
	}
	provider := func(name string) (string, error) {
		contents, err := templates.ReadFile(name)
		if err == nil {
			return string(contents), nil
		}
		base := filepath.Base(name)
		if partial, err2 := templates.ReadFile(path.Join("templates", "partials", base)); err2 == nil {
			return string(partial), nil
		}
		return "", err
	}

	pkgDir := c.packageDir()
	modelFiles := []language.GeneratedFile{
		{
			TemplatePath: "templates/gapic_version.py.mustache",
			OutputPath:   filepath.Join(pkgDir, "gapic_version.py"),
		},
		{
			TemplatePath: "templates/py.typed.mustache",
			OutputPath:   filepath.Join(pkgDir, "py.typed"),
		},
		{
			TemplatePath: "templates/gapic_metadata.json.mustache",
			OutputPath:   filepath.Join(pkgDir, "gapic_metadata.json"),
		},
		{
			TemplatePath: "templates/_compat.py.mustache",
			OutputPath:   filepath.Join(pkgDir, "_compat.py"),
		},
	}

	if c.isDefaultVersion() {
		rootDir := c.rootPackageDir()
		if rootDir != "" && rootDir != pkgDir {
			modelFiles = append(modelFiles,
				language.GeneratedFile{
					TemplatePath: "templates/gapic_version.py.mustache",
					OutputPath:   filepath.Join(rootDir, "gapic_version.py"),
				},
				language.GeneratedFile{
					TemplatePath: "templates/py.typed.mustache",
					OutputPath:   filepath.Join(rootDir, "py.typed"),
				},
			)
		}
	}

	if model.HasServices() {
		modelFiles = append(modelFiles, language.GeneratedFile{
			TemplatePath: "templates/services/__init__.py.mustache",
			OutputPath:   filepath.Join(pkgDir, "services", "__init__.py"),
		})
	}

	modelAnn, _ := model.Codec.(*ModelAnnotations)
	if modelAnn != nil && len(modelAnn.TypesProtos) > 0 {
		modelFiles = append(modelFiles, language.GeneratedFile{
			TemplatePath: "templates/types/__init__.py.mustache",
			OutputPath:   filepath.Join(pkgDir, "types", "__init__.py"),
		})
	}

	type protoFile struct {
		proto *ProtoAnnotations
		file  language.GeneratedFile
	}
	var protoFiles []protoFile
	if modelAnn != nil {
		for _, p := range modelAnn.Protos {
			protoFiles = append(protoFiles, protoFile{
				proto: p,
				file: language.GeneratedFile{
					TemplatePath: "templates/types/proto.py.mustache",
					OutputPath:   filepath.Join(pkgDir, "types", p.ModuleName+".py"),
				},
			})
		}
	}

	type serviceFile struct {
		service *api.Service
		file    language.GeneratedFile
	}
	var serviceFiles []serviceFile

	for _, service := range model.Services {
		ann, ok := service.Codec.(*ServiceAnnotations)
		if !ok || ann == nil {
			continue
		}
		serviceDir := filepath.Join(pkgDir, "services", ann.DirectoryName)
		serviceFiles = append(serviceFiles,
			serviceFile{
				service: service,
				file: language.GeneratedFile{
					TemplatePath: "templates/services/service/__init__.py.mustache",
					OutputPath:   filepath.Join(serviceDir, "__init__.py"),
				},
			},
			serviceFile{
				service: service,
				file: language.GeneratedFile{
					TemplatePath: "templates/services/service/transports/README.rst.mustache",
					OutputPath:   filepath.Join(serviceDir, "transports", "README.rst"),
				},
			},
			serviceFile{
				service: service,
				file: language.GeneratedFile{
					TemplatePath: "templates/services/service/transports/base.py.mustache",
					OutputPath:   filepath.Join(serviceDir, "transports", "base.py"),
				},
			},
			serviceFile{
				service: service,
				file: language.GeneratedFile{
					TemplatePath: "templates/services/service/transports/grpc.py.mustache",
					OutputPath:   filepath.Join(serviceDir, "transports", "grpc.py"),
				},
			},
			serviceFile{
				service: service,
				file: language.GeneratedFile{
					TemplatePath: "templates/services/service/transports/grpc_asyncio.py.mustache",
					OutputPath:   filepath.Join(serviceDir, "transports", "grpc_asyncio.py"),
				},
			},
			serviceFile{
				service: service,
				file: language.GeneratedFile{
					TemplatePath: "templates/services/service/transports/__init__.py.mustache",
					OutputPath:   filepath.Join(serviceDir, "transports", "__init__.py"),
				},
			},
		)
	}

	allFiles := make([]language.GeneratedFile, 0, len(modelFiles)+len(serviceFiles)+len(protoFiles))
	allFiles = append(allFiles, modelFiles...)
	for _, sf := range serviceFiles {
		allFiles = append(allFiles, sf.file)
	}
	for _, pf := range protoFiles {
		allFiles = append(allFiles, pf.file)
	}

	if err := validateOutputContainment(outdir, allFiles); err != nil {
		return err
	}

	if err := language.GenerateFromModel(outdir, model, provider, modelFiles); err != nil {
		return err
	}

	for _, sf := range serviceFiles {
		if err := language.GenerateService(outdir, sf.service, provider, sf.file); err != nil {
			return err
		}
	}

	for _, pf := range protoFiles {
		if err := language.GenerateElement(outdir, pf.proto, provider, pf.file); err != nil {
			return err
		}
	}

	return nil
}

func validateOutputContainment(outdir string, files []language.GeneratedFile) error {
	absOutdir, err := filepath.Abs(outdir)
	if err != nil {
		return fmt.Errorf("resolving outdir %q: %w", outdir, err)
	}
	seen := make(map[string]bool, len(files))
	for _, gen := range files {
		if gen.OutputPath == "" || filepath.IsAbs(gen.OutputPath) {
			return fmt.Errorf("%w: %q", ErrInvalidOutputPath, gen.OutputPath)
		}
		cleanPath := filepath.Clean(gen.OutputPath)
		if cleanPath == "." || cleanPath == ".." || strings.HasPrefix(cleanPath, ".."+string(filepath.Separator)) {
			return fmt.Errorf("%w: %q", ErrEscapeOutputDir, gen.OutputPath)
		}
		target := filepath.Join(absOutdir, gen.OutputPath)
		rel, err := filepath.Rel(absOutdir, target)
		if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			return fmt.Errorf("%w: %q", ErrEscapeOutputDir, gen.OutputPath)
		}
		if seen[rel] {
			return fmt.Errorf("%w: %q", ErrDuplicateOutputPath, gen.OutputPath)
		}
		seen[rel] = true
	}
	return nil
}
