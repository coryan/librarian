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

// MessageAnnotations decorates api.Message with Python-specific metadata.
type MessageAnnotations struct {
	Model                         *ModelAnnotations
	Message                       *api.Message
	Name                          string
	DocLines                      []string
	FirstDocLine                  string
	RemainingDocLines             []string
	Fields                        []*FieldAnnotations
	OneOfs                        []*OneOfAnnotations
	NestedMessages                []*MessageAnnotations
	NestedEnums                   []*EnumAnnotations
	Indent                        string
	HasOneOfs                     bool
	HasOneOfsMultiple             bool
	HasOneOfsSingle               bool
	HasNextPageToken              bool
	HasFields                     bool
	HasNestedEnums                bool
	HasNestedMessages             bool
	HasNestedMessagesOnly         bool
	HasNoNestedClasses            bool
	HasNoNestedClassesWithContent bool
	HasRemainingDocLines          bool
}

func (c *codec) annotateMessage(message *api.Message, model *ModelAnnotations) error {
	depth := 0
	for p := message.Parent; p != nil; p = p.Parent {
		depth++
	}
	indent := strings.Repeat("    ", depth)
	docLines := formatRSTDocLines(message.Documentation, 72, 4)
	ann := &MessageAnnotations{
		Model:    model,
		Message:  message,
		Name:     pascalCase(message.Name),
		DocLines: docLines,
		Indent:   indent,
	}
	if len(docLines) > 0 {
		ann.FirstDocLine = docLines[0]
		ann.RemainingDocLines = docLines[1:]
		ann.HasRemainingDocLines = len(ann.RemainingDocLines) > 0
	}

	for _, field := range message.Fields {
		if err := c.annotateField(field, ann); err != nil {
			return err
		}
		if fAnn, ok := field.Codec.(*FieldAnnotations); ok {
			ann.Fields = append(ann.Fields, fAnn)
			if field.Name == "next_page_token" {
				ann.HasNextPageToken = true
			}
			if fAnn.OneOfDoc != "" {
				ann.HasOneOfs = true
			}
		}
	}

	for _, oneOf := range message.OneOfs {
		if err := c.annotateOneOf(oneOf, ann); err != nil {
			return err
		}
		if oAnn, ok := oneOf.Codec.(*OneOfAnnotations); ok {
			ann.OneOfs = append(ann.OneOfs, oAnn)
			if len(oneOf.Fields) > 1 {
				ann.HasOneOfsMultiple = true
			}
		}
	}

	ann.HasFields = len(ann.Fields) > 0

	// Annotate nested enums
	for _, nestedEnum := range message.Enums {
		if err := c.annotateEnum(nestedEnum, model); err != nil {
			return err
		}
		if eAnn, ok := nestedEnum.Codec.(*EnumAnnotations); ok {
			ann.NestedEnums = append(ann.NestedEnums, eAnn)
		}
	}

	// Annotate nested messages (excluding maps)
	for _, nestedMsg := range message.Messages {
		if nestedMsg.IsMap {
			continue
		}
		if err := c.annotateMessage(nestedMsg, model); err != nil {
			return err
		}
		if mAnn, ok := nestedMsg.Codec.(*MessageAnnotations); ok {
			ann.NestedMessages = append(ann.NestedMessages, mAnn)
		}
	}

	ann.HasNestedEnums = len(ann.NestedEnums) > 0
	ann.HasNestedMessages = len(ann.NestedMessages) > 0
	ann.HasNestedMessagesOnly = ann.HasNestedMessages && !ann.HasNestedEnums
	ann.HasOneOfsSingle = ann.HasOneOfs && !ann.HasOneOfsMultiple
	ann.HasNoNestedClasses = !ann.HasNestedEnums && !ann.HasNestedMessages
	ann.HasNoNestedClassesWithContent = ann.HasNoNestedClasses && (ann.HasFields || ann.HasNextPageToken)

	message.Codec = ann
	return nil
}
