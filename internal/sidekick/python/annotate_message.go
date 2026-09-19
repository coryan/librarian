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

// MessageAnnotations decorates api.Message with Python-specific metadata.
type MessageAnnotations struct {
	Model    *ModelAnnotations
	Name     string
	DocLines []string
	Fields   []*FieldAnnotations
	OneOfs   []*OneOfAnnotations
}

func (c *codec) annotateMessage(message *api.Message, model *ModelAnnotations) error {
	docLines := formatDocLines(message.Documentation)
	ann := &MessageAnnotations{
		Model:    model,
		Name:     pascalCase(message.Name),
		DocLines: docLines,
	}

	for _, field := range message.Fields {
		if err := c.annotateField(field, ann); err != nil {
			return err
		}
		if fAnn, ok := field.Codec.(*FieldAnnotations); ok {
			ann.Fields = append(ann.Fields, fAnn)
		}
	}

	for _, oneOf := range message.OneOfs {
		if err := c.annotateOneOf(oneOf, ann); err != nil {
			return err
		}
		if oAnn, ok := oneOf.Codec.(*OneOfAnnotations); ok {
			ann.OneOfs = append(ann.OneOfs, oAnn)
		}
	}

	message.Codec = ann
	return nil
}
