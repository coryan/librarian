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
	"embed"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/cbroglie/mustache"
)

//go:embed all:templates
var templates embed.FS

type mustacheProvider struct {
	impl    func(string) (string, error)
	dirname string
}

func (p *mustacheProvider) Get(name string) (string, error) {
	if suffix, ok := strings.CutPrefix(name, "/"); ok {
		return p.impl(suffix + ".mustache")
	}
	if strings.HasPrefix(name, "templates/") {
		return p.impl(name + ".mustache")
	}
	for _, dir := range []string{"common/", "internal/", "forwarding/", "mocks/"} {
		if strings.HasPrefix(name, dir) {
			return p.impl(filepath.Join("templates", name) + ".mustache")
		}
	}
	return p.impl(filepath.Join(p.dirname, name) + ".mustache")
}

// renderTemplate renders a template identified by tmplPath using data.
func renderTemplate(tmplPath string, data any) (string, error) {
	content, err := templates.ReadFile(tmplPath)
	if err != nil {
		return "", fmt.Errorf("reading template %s: %w", tmplPath, err)
	}
	provider := &mustacheProvider{
		impl: func(name string) (string, error) {
			b, err := templates.ReadFile(name)
			return string(b), err
		},
		dirname: filepath.Dir(tmplPath),
	}
	rendered, err := mustache.RenderPartials(string(content), provider, data)
	if err != nil {
		return "", fmt.Errorf("rendering template %s: %w", tmplPath, err)
	}
	return rendered, nil
}
