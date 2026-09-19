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
	for _, test := range []struct {
		name string
		msg  *api.Message
		want *MessageAnnotations
	}{
		{
			name: "basic message with fields",
			msg: func() *api.Message {
				m := api.NewTestMessage("Secret").
					WithFields(
						api.NewTestField("name").WithType(api.TypezString),
						api.NewTestField("labels").WithMap(),
					)
				m.Documentation = "A secret object."
				return m
			}(),
			want: &MessageAnnotations{
				Name:     "Secret",
				DocLines: []string{"A secret object."},
				Fields: []*FieldAnnotations{
					{Name: "name"},
					{Name: "labels", IsMap: true},
				},
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			model := api.NewTestAPI([]*api.Message{test.msg}, nil, nil)
			codec := newTestCodec(t, model, nil)
			if err := codec.annotateModel(); err != nil {
				t.Fatalf("annotateModel() failed: %v", err)
			}
			ann, ok := test.msg.Codec.(*MessageAnnotations)
			if !ok {
				t.Fatalf("expected MessageAnnotations, got %T", test.msg.Codec)
			}
			if diff := cmp.Diff(test.want, ann,
				cmpopts.IgnoreFields(MessageAnnotations{}, "Model"),
				cmpopts.IgnoreFields(FieldAnnotations{}, "Message", "TypeName"),
			); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
