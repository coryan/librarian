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
	"fmt"
	"os"
	"path/filepath"
)

// ScaffoldVars holds variables for rendering C++ build and project scaffolding.
type ScaffoldVars struct {
	CopyrightYear       string
	Title               string
	Description         string
	DocumentationURI    string
	Library             string
	LibraryPrefix       string
	ProductNamespace    string
	ServiceSubdirectory string
	Directory           string
	Experimental        bool
}

type scaffoldFile struct {
	relPath  string
	tmplPath string
}

var scaffoldFiles = []scaffoldFile{
	{"CMakeLists.txt", "templates/scaffold/CMakeLists.txt.mustache"},
	{"BUILD.bazel", "templates/scaffold/BUILD.bazel.mustache"},
	{"README.md", "templates/scaffold/README.md.mustache"},
	{"quickstart/quickstart.cc", "templates/scaffold/quickstart/quickstart.cc.mustache"},
	{"quickstart/CMakeLists.txt", "templates/scaffold/quickstart/CMakeLists.txt.mustache"},
	{"quickstart/BUILD.bazel", "templates/scaffold/quickstart/BUILD.bazel.mustache"},
	{"quickstart/Makefile", "templates/scaffold/quickstart/Makefile.mustache"},
	{"quickstart/README.md", "templates/scaffold/quickstart/README.md.mustache"},
	{"quickstart/.bazelrc", "templates/scaffold/quickstart/bazelrc.mustache"},
}

func (v *ScaffoldVars) toMap() map[string]any {
	return map[string]any{
		"copyright_year":       v.CopyrightYear,
		"title":                v.Title,
		"description":          v.Description,
		"documentation_uri":    v.DocumentationURI,
		"library":              v.Library,
		"library_prefix":       v.LibraryPrefix,
		"product_namespace":    v.ProductNamespace,
		"service_subdirectory": v.ServiceSubdirectory,
		"directory":            v.Directory,
		"experimental":         v.Experimental,
	}
}

// Scaffold renders the scaffolding templates into outDir.
// If overwrite is false, existing files will not be modified.
func Scaffold(outDir string, vars *ScaffoldVars, overwrite bool) error {
	data := vars.toMap()
	for _, f := range scaffoldFiles {
		target := filepath.Join(outDir, f.relPath)
		if !overwrite {
			if _, err := os.Stat(target); err == nil {
				continue
			}
		}
		rendered, err := renderTemplate(f.tmplPath, data)
		if err != nil {
			return fmt.Errorf("rendering %s: %w", f.tmplPath, err)
		}
		if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			return fmt.Errorf("creating directory for %s: %w", target, err)
		}
		if err := os.WriteFile(target, []byte(rendered), 0644); err != nil {
			return fmt.Errorf("writing %s: %w", target, err)
		}
	}
	return nil
}
