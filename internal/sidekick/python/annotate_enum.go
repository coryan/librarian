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
	"github.com/googleapis/librarian/internal/sidekick/api"
)

// EnumAnnotations decorates api.Enum with Python-specific metadata.
type EnumAnnotations struct {
	Model    *ModelAnnotations
	Name     string
	DocLines []string
	Values   []*EnumValueAnnotations
}

func (c *codec) annotateEnum(enum *api.Enum, model *ModelAnnotations) error {
	docLines := formatDocLines(enum.Documentation)
	ann := &EnumAnnotations{
		Model:    model,
		Name:     pascalCase(enum.Name),
		DocLines: docLines,
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
