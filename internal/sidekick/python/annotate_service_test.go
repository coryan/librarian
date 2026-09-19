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

func TestAnnotateService(t *testing.T) {
	req := api.NewTestMessage("GetSecretRequest")
	resp := api.NewTestMessage("Secret")

	for _, test := range []struct {
		name string
		svc  *api.Service
		want *ServiceAnnotations
	}{
		{
			name: "basic service",
			svc: func() *api.Service {
				s := api.NewTestService("SecretManagerService").
					WithMethods(
						api.NewTestMethod("GetSecret").WithInput(req).WithOutput(resp),
					)
				s.Documentation = "Secret Manager Service API."
				return s
			}(),
			want: &ServiceAnnotations{
				Name:            "SecretManagerService",
				ProtoName:       "SecretManagerService",
				ServiceName:     "SecretManagerService",
				ClientName:      "SecretManagerServiceClient",
				AsyncClientName: "SecretManagerServiceAsyncClient",
				DocLines:        []string{"Secret Manager Service API."},
				Methods: []*MethodAnnotations{
					{
						Name:           "get_secret",
						ProtoName:      "GetSecret",
						InputTypeName:  "GetSecretRequest",
						OutputTypeName: "Secret",
					},
				},
				OwnMethods: []*MethodAnnotations{
					{
						Name:           "get_secret",
						ProtoName:      "GetSecret",
						InputTypeName:  "GetSecretRequest",
						OutputTypeName: "Secret",
					},
				},
			},
		},
		{
			name: "service ending with Client",
			svc:  api.NewTestService("EchoClient"),
			want: &ServiceAnnotations{
				Name:            "EchoClient",
				ProtoName:       "EchoClient",
				ServiceName:     "EchoClient",
				ClientName:      "EchoClient",
				AsyncClientName: "EchoAsyncClient",
			},
		},
		{
			name: "service with mixin methods preserving HasNext and HasNextOwn",
			svc: func() *api.Service {
				s := api.NewTestService("SecretManagerService")
				s.ID = "google.cloud.secretmanager.v1.SecretManagerService"
				opSvc := api.NewTestService("Operations").WithPackage("google.longrunning")
				opSvc.ID = "google.longrunning.Operations"
				opMethod := api.NewTestMethod("GetOperation").WithInput(req).WithOutput(resp)
				opMethod.SourceServiceID = opSvc.ID
				opSvc.WithMethods(opMethod)

				m1 := api.NewTestMethod("CreateSecret").WithInput(req).WithOutput(resp)
				m1.SourceServiceID = s.ID
				m2 := api.NewTestMethod("GetOperation").WithSourceMethod(opMethod)
				m2.SourceServiceID = opSvc.ID
				m3 := api.NewTestMethod("GetSecret").WithInput(req).WithOutput(resp)
				m3.SourceServiceID = s.ID

				s.WithMethods(m1, m2, m3)
				return s
			}(),
			want: &ServiceAnnotations{
				Name:            "SecretManagerService",
				ProtoName:       "SecretManagerService",
				ServiceName:     "SecretManagerService",
				ClientName:      "SecretManagerServiceClient",
				AsyncClientName: "SecretManagerServiceAsyncClient",
				Methods: []*MethodAnnotations{
					{
						Name:           "create_secret",
						ProtoName:      "CreateSecret",
						InputTypeName:  "GetSecretRequest",
						OutputTypeName: "Secret",
						HasNext:        true,
						HasNextOwn:     true,
					},
					{
						Name:           "get_operation",
						ProtoName:      "GetOperation",
						InputTypeName:  "GetSecretRequest",
						OutputTypeName: "Secret",
						IsMixin:        true,
						HasNext:        true,
						HasNextOwn:     false,
					},
					{
						Name:           "get_secret",
						ProtoName:      "GetSecret",
						InputTypeName:  "GetSecretRequest",
						OutputTypeName: "Secret",
						HasNext:        false,
						HasNextOwn:     false,
					},
				},
				OwnMethods: []*MethodAnnotations{
					{
						Name:           "create_secret",
						ProtoName:      "CreateSecret",
						InputTypeName:  "GetSecretRequest",
						OutputTypeName: "Secret",
						HasNext:        true,
						HasNextOwn:     true,
					},
					{
						Name:           "get_secret",
						ProtoName:      "GetSecret",
						InputTypeName:  "GetSecretRequest",
						OutputTypeName: "Secret",
						HasNext:        false,
						HasNextOwn:     false,
					},
				},
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			model := api.NewTestAPI([]*api.Message{req, resp}, nil, []*api.Service{test.svc})
			codec := newTestCodec(t, model, nil)
			if err := codec.annotateModel(); err != nil {
				t.Fatalf("annotateModel() failed: %v", err)
			}
			ann, ok := test.svc.Codec.(*ServiceAnnotations)
			if !ok {
				t.Fatalf("expected ServiceAnnotations, got %T", test.svc.Codec)
			}
			if diff := cmp.Diff(test.want, ann,
				cmpopts.IgnoreFields(ServiceAnnotations{}, "Model", "Service"),
				cmpopts.IgnoreFields(MethodAnnotations{}, "Service"),
			); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
