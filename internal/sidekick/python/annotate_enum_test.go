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

func TestAnnotateEnum(t *testing.T) {
	enum := api.NewTestEnum("SecretStatus").
		WithDocumentation("Status of secret.").
		WithValues(
			api.NewTestEnumValue("STATUS_UNSPECIFIED", 0),
			api.NewTestEnumValue("ENABLED", 1),
		)
	want := &EnumAnnotations{
		Name:              "SecretStatus",
		DocLines:          []string{"Status of secret."},
		FirstDocLine:      "Status of secret.",
		RemainingDocLines: []string{},
		Values: []*EnumValueAnnotations{
			{Name: "STATUS_UNSPECIFIED", Number: 0},
			{Name: "ENABLED", Number: 1},
		},
	}
	model := api.NewTestAPI(nil, []*api.Enum{enum}, nil)
	codec := newTestCodec(t, model, nil)
	if err := codec.annotateModel(); err != nil {
		t.Fatalf("annotateModel() failed: %v", err)
	}
	ann, ok := enum.Codec.(*EnumAnnotations)
	if !ok {
		t.Fatalf("expected EnumAnnotations, got %T", enum.Codec)
	}
	if diff := cmp.Diff(want, ann,
		cmpopts.IgnoreFields(EnumAnnotations{}, "Model", "Enum"),
		cmpopts.IgnoreFields(EnumValueAnnotations{}, "Enum"),
	); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}
