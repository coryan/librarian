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

package python

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/serviceconfig"
	"github.com/googleapis/librarian/internal/sidekick/api"
)

// codec represents the configuration and context for the Python sidekick generator.
type codec struct {
	Model          *api.API
	Library        *config.Library
	GenerationYear string
	PackageName    string
	PackageVersion string
	DefaultVersion string
	CurrentVersion string
	GAPICNamespace string
	GAPICName      string
	OutDir         string
}

func newCodec(model *api.API, library *config.Library, outdir string) (*codec, error) {
	if model == nil {
		return nil, errors.New("python: model cannot be nil")
	}
	year := fmt.Sprintf("%04d", time.Now().Year())
	if library != nil && library.CopyrightYear != "" {
		year = library.CopyrightYear
	}
	pkgVersion := "0.0.0"
	if library != nil && library.Version != "" {
		pkgVersion = library.Version
	}
	pkgName := ""
	defaultVersion := ""
	if library != nil {
		pkgName = library.Name
		if library.Python != nil {
			defaultVersion = library.Python.DefaultVersion
		}
	}
	if pkgName == "" && model != nil {
		pkgName = model.Name
	}

	namespace := ""
	name := ""
	currentVersion := ""

	// Determine apiPath:
	// 1. Look for matching API in library.APIs (matching model.PackageName)
	// 2. Otherwise use library.APIs[0].Path if single API exists
	// 3. Fall back to model.PackageName
	apiPath := ""
	if library != nil && len(library.APIs) > 0 {
		for _, apiCfg := range library.APIs {
			if strings.ReplaceAll(apiCfg.Path, "/", ".") == model.PackageName || apiCfg.Path == model.PackageName {
				apiPath = apiCfg.Path
				break
			}
		}
		if apiPath == "" && len(library.APIs) == 1 {
			apiPath = library.APIs[0].Path
		}
	}
	if apiPath == "" && model != nil && model.PackageName != "" {
		apiPath = strings.ReplaceAll(model.PackageName, ".", "/")
	}

	if apiPath != "" {
		namespace = deriveGAPICNamespace(apiPath)
		name = deriveGAPICName(apiPath)
		currentVersion = serviceconfig.ExtractVersion(apiPath)
		if defaultVersion == "" {
			defaultVersion = currentVersion
		}
	}

	return &codec{
		Model:          model,
		Library:        library,
		GenerationYear: year,
		PackageName:    pkgName,
		PackageVersion: pkgVersion,
		DefaultVersion: defaultVersion,
		CurrentVersion: currentVersion,
		GAPICNamespace: namespace,
		GAPICName:      name,
		OutDir:         outdir,
	}, nil
}

func deriveGAPICNamespace(apiPath string) string {
	version := serviceconfig.ExtractVersion(apiPath)
	if version != "" {
		apiPath = strings.TrimSuffix(apiPath, "/"+version)
	}
	parts := strings.Split(apiPath, "/")
	if len(parts) < 2 {
		return apiPath
	}
	return parts[0] + "." + parts[1]
}

func deriveGAPICName(apiPath string) string {
	version := serviceconfig.ExtractVersion(apiPath)
	if version != "" {
		apiPath = strings.TrimSuffix(apiPath, "/"+version)
	}
	ns := deriveGAPICNamespace(apiPath)
	apiPath = strings.TrimPrefix(apiPath, strings.ReplaceAll(ns, ".", "/"))
	apiPath = strings.Trim(apiPath, "/")
	return strings.ReplaceAll(apiPath, "/", "_")
}

func (c *codec) packageDir() string {
	if c.GAPICNamespace == "" && c.GAPICName == "" {
		return ""
	}
	nsPath := strings.ReplaceAll(c.GAPICNamespace, ".", "/")
	pkgName := c.GAPICName
	version := c.CurrentVersion
	if version == "" {
		version = c.DefaultVersion
	}
	if version != "" {
		pkgName = fmt.Sprintf("%s_%s", c.GAPICName, version)
	}
	return filepath.Join(nsPath, pkgName)
}

func (c *codec) rootPackageDir() string {
	if c.GAPICNamespace == "" && c.GAPICName == "" {
		return ""
	}
	nsPath := strings.ReplaceAll(c.GAPICNamespace, ".", "/")
	return filepath.Join(nsPath, c.GAPICName)
}

func (c *codec) isDefaultVersion() bool {
	if c.DefaultVersion == "" || c.CurrentVersion == "" {
		return true
	}
	return c.CurrentVersion == c.DefaultVersion
}
