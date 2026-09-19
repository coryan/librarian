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

func TestAnnotateMethod(t *testing.T) {
	req := api.NewTestMessage("GetSecretRequest")
	resp := api.NewTestMessage("Secret")

	for _, test := range []struct {
		name   string
		method *api.Method
		want   *MethodAnnotations
	}{
		{
			name: "unary method",
			method: func() *api.Method {
				m := api.NewTestMethod("GetSecret").
					WithInput(req).
					WithOutput(resp)
				m.Documentation = "Gets a secret."
				return m
			}(),
			want: &MethodAnnotations{
				Name:           "get_secret",
				ProtoName:      "GetSecret",
				DocLines:       []string{"Gets a secret."},
				InputTypeName:  "GetSecretRequest",
				OutputTypeName: "Secret",
			},
		},
		{
			name: "streaming method",
			method: api.NewTestMethod("StreamSecrets").
				WithInput(req).
				WithOutput(resp).
				WithServerSideStreaming(),
			want: &MethodAnnotations{
				Name:           "stream_secrets",
				ProtoName:      "StreamSecrets",
				InputTypeName:  "GetSecretRequest",
				OutputTypeName: "Secret",
				IsStreaming:    true,
			},
		},
		{
			name: "mixin method",
			method: func() *api.Method {
				opSvc := api.NewTestService("Operations").WithPackage("google.longrunning")
				opMethod := api.NewTestMethod("GetOperation").WithInput(req).WithOutput(resp)
				opSvc.WithMethods(opMethod)
				m := api.NewTestMethod("GetOperation").WithSourceMethod(opMethod)
				return m
			}(),
			want: &MethodAnnotations{
				Name:           "get_operation",
				ProtoName:      "GetOperation",
				InputTypeName:  "GetSecretRequest",
				OutputTypeName: "Secret",
				IsMixin:        true,
			},
		},
		{
			name: "lro method",
			method: api.NewTestMethod("CreateSecret").
				WithInput(req).
				WithOutput(resp).
				WithOperationInfo(&api.OperationInfo{ResponseTypeID: "Secret", MetadataTypeID: "CreateSecretMetadata"}),
			want: &MethodAnnotations{
				Name:           "create_secret",
				ProtoName:      "CreateSecret",
				InputTypeName:  "GetSecretRequest",
				OutputTypeName: "Secret",
				IsLRO:          true,
			},
		},
		{
			name: "paged method",
			method: api.NewTestMethod("ListSecrets").
				WithInput(req).
				WithOutput(resp).
				WithPagination(api.NewTestField("page_token").WithType(api.TypezString)),
			want: &MethodAnnotations{
				Name:           "list_secrets",
				ProtoName:      "ListSecrets",
				InputTypeName:  "GetSecretRequest",
				OutputTypeName: "Secret",
				IsPaged:        true,
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			svc := api.NewTestService("SecretManagerService").WithMethods(test.method)
			model := api.NewTestAPI([]*api.Message{req, resp}, nil, []*api.Service{svc})
			codec := newTestCodec(t, model, nil)
			if err := codec.annotateModel(); err != nil {
				t.Fatalf("annotateModel() failed: %v", err)
			}
			ann, ok := test.method.Codec.(*MethodAnnotations)
			if !ok {
				t.Fatalf("expected MethodAnnotations, got %T", test.method.Codec)
			}
			if diff := cmp.Diff(test.want, ann, cmpopts.IgnoreFields(MethodAnnotations{}, "Service")); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
