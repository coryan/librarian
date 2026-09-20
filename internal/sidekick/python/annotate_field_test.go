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

func TestAnnotateField(t *testing.T) {
	for _, test := range []struct {
		name  string
		field *api.Field
		want  *FieldAnnotations
	}{
		{
			name: "primitive field",
			field: func() *api.Field {
				f := api.NewTestField("secret_name").
					WithType(api.TypezString).
					WithNumber(1)
				f.Documentation = "The name of the secret."
				return f
			}(),
			want: &FieldAnnotations{
				Name:           "secret_name",
				Number:         1,
				ProtoType:      "STRING",
				TypeAnnotation: "str",
				SphinxType:     "str",
				DocLines:       []string{"The name of the secret."},
			},
		},
		{
			name: "python keyword field escaped",
			field: api.NewTestField("from").
				WithType(api.TypezString).
				WithNumber(2),
			want: &FieldAnnotations{
				Name:           "from_",
				Number:         2,
				ProtoType:      "STRING",
				TypeAnnotation: "str",
				SphinxType:     "str",
			},
		},
		{
			name: "repeated field",
			field: api.NewTestField("aliases").
				WithType(api.TypezString).
				WithNumber(3).
				WithRepeated(),
			want: &FieldAnnotations{
				Name:           "aliases",
				Number:         3,
				ProtoType:      "STRING",
				TypeAnnotation: "MutableSequence[str]",
				SphinxType:     "MutableSequence[str]",
				IsRepeated:     true,
			},
		},
		{
			name: "map field",
			field: api.NewTestField("labels").
				WithNumber(4).
				WithMap(),
			want: &FieldAnnotations{
				Name:           "labels",
				Number:         4,
				IsMap:          true,
				KeyProtoType:   "STRING",
				ValProtoType:   "STRING",
				KeyType:        "str",
				ValType:        "str",
				TypeAnnotation: "MutableMapping[str, str]",
				SphinxType:     "MutableMapping[str, str]",
			},
		},
		{
			name: "message field in same file",
			field: api.NewTestField("payload").
				WithType(api.TypezMessage).
				WithTypezID(".test.Payload").
				WithNumber(5),
			want: &FieldAnnotations{
				Name:           "payload",
				Number:         5,
				ProtoType:      "MESSAGE",
				TypeAnnotation: "'Payload'",
				SphinxType:     "test.types.Payload",
				FieldArg:       "message",
				TypeRef:        "'Payload'",
			},
		},
		{
			name: "enum field in same file",
			field: api.NewTestField("status").
				WithType(api.TypezEnum).
				WithTypezID(".test.Status").
				WithNumber(6),
			want: &FieldAnnotations{
				Name:           "status",
				Number:         6,
				ProtoType:      "ENUM",
				TypeAnnotation: "'Status'",
				SphinxType:     "test.types.Status",
				FieldArg:       "enum",
				TypeRef:        "'Status'",
			},
		},
		{
			name: "cross-file message field in same package",
			field: api.NewTestField("other").
				WithType(api.TypezMessage).
				WithTypezID(".test.OtherMessage").
				WithNumber(7),
			want: &FieldAnnotations{
				Name:           "other",
				Number:         7,
				ProtoType:      "MESSAGE",
				TypeAnnotation: "t_other.OtherMessage",
				SphinxType:     "test.types.OtherMessage",
				FieldArg:       "message",
				TypeRef:        "t_other.OtherMessage",
			},
		},
		{
			name: "external protobuf message field",
			field: api.NewTestField("expire_time").
				WithType(api.TypezMessage).
				WithTypezID(".google.protobuf.Timestamp").
				WithNumber(8),
			want: &FieldAnnotations{
				Name:           "expire_time",
				Number:         8,
				ProtoType:      "MESSAGE",
				TypeAnnotation: "timestamp_pb2.Timestamp",
				SphinxType:     "google.protobuf.timestamp_pb2.Timestamp",
				FieldArg:       "message",
				TypeRef:        "timestamp_pb2.Timestamp",
			},
		},
		{
			name: "optional primitive field with synthetic oneof",
			field: api.NewTestField("description").
				WithType(api.TypezString).
				WithNumber(9).
				WithOptional(),
			want: &FieldAnnotations{
				Name:           "description",
				Number:         9,
				ProtoType:      "STRING",
				TypeAnnotation: "str",
				SphinxType:     "str",
				IsOptional:     true,
				OneOfDoc:       "_description",
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			payloadMsg := api.NewTestMessage("Payload")
			otherMsg := api.NewTestMessage("OtherMessage")
			statusEnum := api.NewTestEnum("Status")

			msg := api.NewTestMessage("Secret").WithFields(test.field)

			model := api.NewTestAPI([]*api.Message{msg, payloadMsg, otherMsg}, []*api.Enum{statusEnum}, nil).
				WithDefinitionLocation(".test.Secret", "secret.proto", 1).
				WithDefinitionLocation(".test.Payload", "secret.proto", 10).
				WithDefinitionLocation(".test.Status", "secret.proto", 20).
				WithDefinitionLocation(".test.OtherMessage", "other.proto", 1)

			codec := newTestCodec(t, model, nil)
			if err := codec.annotateModel(); err != nil {
				t.Fatalf("annotateModel() failed: %v", err)
			}
			ann, ok := test.field.Codec.(*FieldAnnotations)
			if !ok {
				t.Fatalf("expected FieldAnnotations, got %T", test.field.Codec)
			}
			if diff := cmp.Diff(test.want, ann,
				cmpopts.IgnoreFields(FieldAnnotations{}, "Message", "Field"),
			); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
