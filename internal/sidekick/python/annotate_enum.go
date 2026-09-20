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
	"strings"

	"github.com/googleapis/librarian/internal/sidekick/api"
)

// EnumAnnotations decorates api.Enum with Python-specific metadata.
type EnumAnnotations struct {
	Model             *ModelAnnotations
	Enum              *api.Enum
	Name              string
	DocLines          []string
	FirstDocLine      string
	RemainingDocLines []string
	Values            []*EnumValueAnnotations
	Indent            string
}

func (c *codec) annotateEnum(enum *api.Enum, model *ModelAnnotations) error {
	depth := 0
	for p := enum.Parent; p != nil; p = p.Parent {
		depth++
	}
	indent := strings.Repeat("    ", depth)
	docLines := formatRSTDocLines(enum.Documentation, 72, 4)
	ann := &EnumAnnotations{
		Model:    model,
		Enum:     enum,
		Name:     pascalCase(enum.Name),
		DocLines: docLines,
		Indent:   indent,
	}
	if len(docLines) > 0 {
		ann.FirstDocLine = docLines[0]
		ann.RemainingDocLines = docLines[1:]
	}

	for _, ev := range enum.Values {
		if err := c.annotateEnumValue(ev, ann); err != nil {
			return err
		}
		if evAnn, ok := ev.Codec.(*EnumValueAnnotations); ok {
			ann.Values = append(ann.Values, evAnn)
		}
	}

	enum.Codec = ann
	return nil
}
