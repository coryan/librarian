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

// EnumValueAnnotations decorates api.EnumValue with Python-specific metadata.
type EnumValueAnnotations struct {
	Enum     *EnumAnnotations
	Name     string
	Number   int32
	DocLines []string
}

func (c *codec) annotateEnumValue(ev *api.EnumValue, enum *EnumAnnotations) error {
	docLines := formatRSTDocLines(ev.Documentation, 72, 12)
	ann := &EnumValueAnnotations{
		Enum:     enum,
		Name:     ev.Name,
		Number:   ev.Number,
		DocLines: docLines,
	}
	ev.Codec = ann
	return nil
}
