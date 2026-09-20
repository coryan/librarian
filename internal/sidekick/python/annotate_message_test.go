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
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/googleapis/librarian/internal/sidekick/api"
)

func TestAnnotateMessage(t *testing.T) {
	msg := api.NewTestMessage("Secret").
		WithFields(
			api.NewTestField("name").WithType(api.TypezString).WithNumber(1),
			api.NewTestField("labels").WithNumber(2).WithMap(),
		)
	msg.Documentation = "A secret object."
	want := &MessageAnnotations{
		Name:                          "Secret",
		DocLines:                      []string{"A secret object."},
		FirstDocLine:                  "A secret object.",
		RemainingDocLines:             []string{},
		HasFields:                     true,
		HasNoNestedClasses:            true,
		HasNoNestedClassesWithContent: true,
		Fields: []*FieldAnnotations{
			{Name: "name", Number: 1, ProtoType: "STRING", TypeAnnotation: "str", SphinxType: "str"},
			{Name: "labels", Number: 2, IsMap: true, KeyProtoType: "STRING", ValProtoType: "STRING", KeyType: "str", ValType: "str", TypeAnnotation: "MutableMapping[str, str]", SphinxType: "MutableMapping[str, str]"},
		},
	}
	model := api.NewTestAPI([]*api.Message{msg}, nil, nil)
	codec := newTestCodec(t, model, nil)
	if err := codec.annotateModel(); err != nil {
		t.Fatalf("annotateModel() failed: %v", err)
	}
	ann, ok := msg.Codec.(*MessageAnnotations)
	if !ok {
		t.Fatalf("expected MessageAnnotations, got %T", msg.Codec)
	}
	if diff := cmp.Diff(want, ann,
		cmpopts.IgnoreFields(MessageAnnotations{}, "Model", "Message"),
		cmpopts.IgnoreFields(FieldAnnotations{}, "Message", "Field"),
	); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}
