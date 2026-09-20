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

func TestAnnotateOneOf(t *testing.T) {
	oneOf := api.NewTestOneOf("payload").
		WithFields(
			api.NewTestField("data").WithType(api.TypezBytes),
			api.NewTestField("data_crc32c").WithType(api.TypezInt64),
		)
	want := &OneOfAnnotations{
		Name: "payload",
		Fields: []*FieldAnnotations{
			{
				Name:           "data",
				ProtoType:      "BYTES",
				TypeAnnotation: "bytes",
				SphinxType:     "bytes",
				OneOf:          "payload",
				OneOfDoc:       "payload",
			},
			{
				Name:           "data_crc32c",
				ProtoType:      "INT64",
				TypeAnnotation: "int",
				SphinxType:     "int",
				OneOf:          "payload",
				OneOfDoc:       "payload",
			},
		},
	}
	msg := api.NewTestMessage("SecretPayload").
		WithOneOfs(oneOf)
	model := api.NewTestAPI([]*api.Message{msg}, nil, nil)
	codec := newTestCodec(t, model, nil)
	if err := codec.annotateModel(); err != nil {
		t.Fatalf("annotateModel() failed: %v", err)
	}
	ann, ok := oneOf.Codec.(*OneOfAnnotations)
	if !ok {
		t.Fatalf("expected OneOfAnnotations, got %T", oneOf.Codec)
	}
	if diff := cmp.Diff(want, ann,
		cmpopts.IgnoreFields(OneOfAnnotations{}, "Message"),
		cmpopts.IgnoreFields(FieldAnnotations{}, "Message", "Field"),
	); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}
