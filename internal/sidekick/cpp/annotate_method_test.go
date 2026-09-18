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

package cpp

import (
	"testing"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/sidekick/api"
)

func TestAnnotateMethod_Unary(t *testing.T) {
	req := api.NewTestMessage("GetRequest").WithPackage("test.v1")
	resp := api.NewTestMessage("GetResponse").WithPackage("test.v1")
	method := api.NewTestMethod("Get").WithInput(req).WithOutput(resp)
	svc := api.NewTestService("TestService").WithPackage("test.v1").WithMethods(method)
	model := api.NewTestAPI([]*api.Message{req, resp}, nil, []*api.Service{svc})
	if err := api.CrossReference(model); err != nil {
		t.Fatal(err)
	}

	ann := annotateMethod(method, svc, nil, model)
	if ann.MethodName != "Get" {
		t.Errorf("got MethodName %q, want 'Get'", ann.MethodName)
	}
	if ann.MethodNameSnake != "get" {
		t.Errorf("got MethodNameSnake %q, want 'get'", ann.MethodNameSnake)
	}
	if ann.ReturnType != "StatusOr<test::v1::GetResponse>" {
		t.Errorf("got ReturnType %q, want 'StatusOr<test::v1::GetResponse>'", ann.ReturnType)
	}
	if ann.RequestType != "test::v1::GetRequest" {
		t.Errorf("got RequestType %q, want 'test::v1::GetRequest'", ann.RequestType)
	}
	if !ann.IsNonStreaming {
		t.Errorf("expected IsNonStreaming=true")
	}
	if method.Codec != ann {
		t.Errorf("method.Codec not set to annotations")
	}
	if ann.Comments == nil || ann.Comments.MethodComment == "" {
		t.Errorf("expected ann.Comments.MethodComment to be populated")
	}
}

func TestAnnotateMethod_EmptyReturn(t *testing.T) {
	req := api.NewTestMessage("DeleteRequest").WithPackage("test.v1")
	emptyResp := api.NewTestMessage("Empty").WithPackage("google.protobuf")
	method := api.NewTestMethod("Delete").WithInput(req).WithOutput(emptyResp)
	method.ReturnsEmpty = true
	svc := api.NewTestService("TestService").WithPackage("test.v1").WithMethods(method)
	model := api.NewTestAPI([]*api.Message{req, emptyResp}, nil, []*api.Service{svc})
	if err := api.CrossReference(model); err != nil {
		t.Fatal(err)
	}

	ann := annotateMethod(method, svc, nil, model)
	if ann.ReturnType != "Status" {
		t.Errorf("got ReturnType %q, want 'Status'", ann.ReturnType)
	}
}

func TestAnnotateMethod_Longrunning(t *testing.T) {
	req := api.NewTestMessage("CreateRequest").WithPackage("test.v1")
	opResp := api.NewTestMessage("Operation").WithPackage("google.longrunning")
	method := api.NewTestMethod("Create").WithInput(req).WithOutput(opResp)
	method.WithOperationInfo(&api.OperationInfo{
		MetadataTypeID: ".test.v1.CreateMetadata",
		ResponseTypeID: ".google.protobuf.Empty",
	})
	svc := api.NewTestService("TestService").WithPackage("test.v1").WithMethods(method)
	model := api.NewTestAPI([]*api.Message{req, opResp}, nil, []*api.Service{svc})
	if err := api.CrossReference(model); err != nil {
		t.Fatal(err)
	}

	ann := annotateMethod(method, svc, nil, model)
	if !ann.IsLongrunning {
		t.Errorf("expected IsLongrunning=true")
	}
	if ann.LongrunningMetadataType != "test::v1::CreateMetadata" {
		t.Errorf("got LongrunningMetadataType %q, want test::v1::CreateMetadata", ann.LongrunningMetadataType)
	}
	// When ResponseTypeID is Empty, deduced response type falls back to metadata
	if ann.LongrunningDeducedResponseType != "test::v1::CreateMetadata" {
		t.Errorf("got LongrunningDeducedResponseType %q, want test::v1::CreateMetadata", ann.LongrunningDeducedResponseType)
	}
}

func TestAnnotateMethod_Pagination(t *testing.T) {
	req := api.NewTestMessage("ListRequest").WithPackage("test.v1").WithFields(
		api.NewTestField("page_size").WithType(api.TypezInt32),
		api.NewTestField("page_token").WithType(api.TypezString),
	)
	itemMsg := api.NewTestMessage("Item").WithPackage("test.v1")
	resp := api.NewTestMessage("ListResponse").WithPackage("test.v1").WithFields(
		api.NewTestField("items").WithType(api.TypezMessage).WithRepeated(),
		api.NewTestField("next_page_token").WithType(api.TypezString),
	)
	resp.Fields[0].TypezID = itemMsg.ID
	resp.Pagination = &api.PaginationInfo{
		PageableItem:  resp.Fields[0],
		NextPageToken: resp.Fields[1],
	}

	method := api.NewTestMethod("List").WithInput(req).WithOutput(resp)
	method.WithPagination(req.Fields[1])

	svc := api.NewTestService("TestService").WithPackage("test.v1").WithMethods(method)
	model := api.NewTestAPI([]*api.Message{req, resp, itemMsg}, nil, []*api.Service{svc})
	if err := api.CrossReference(model); err != nil {
		t.Fatal(err)
	}

	ann := annotateMethod(method, svc, nil, model)
	if !ann.IsPaginated {
		t.Errorf("expected IsPaginated=true")
	}
	if ann.RangeOutputFieldName != "items" {
		t.Errorf("got RangeOutputFieldName %q, want 'items'", ann.RangeOutputFieldName)
	}
	if ann.RangeOutputType != "test::v1::Item" {
		t.Errorf("got RangeOutputType %q, want 'test::v1::Item'", ann.RangeOutputType)
	}
}

func TestAnnotateMethod_Streaming(t *testing.T) {
	req := api.NewTestMessage("StreamRequest").WithPackage("test.v1")
	resp := api.NewTestMessage("StreamResponse").WithPackage("test.v1")

	bidiMethod := api.NewTestMethod("Bidi").WithInput(req).WithOutput(resp).WithBidiStreaming()
	serverMethod := api.NewTestMethod("Server").WithInput(req).WithOutput(resp).WithServerSideStreaming()
	clientMethod := api.NewTestMethod("Client").WithInput(req).WithOutput(resp).WithClientSideStreaming()

	svc := api.NewTestService("StreamService").WithPackage("test.v1").WithMethods(bidiMethod, serverMethod, clientMethod)
	model := api.NewTestAPI([]*api.Message{req, resp}, nil, []*api.Service{svc})
	if err := api.CrossReference(model); err != nil {
		t.Fatal(err)
	}

	bidiAnn := annotateMethod(bidiMethod, svc, nil, model)
	if !bidiAnn.IsBidiStreaming {
		t.Errorf("expected IsBidiStreaming=true for Bidi")
	}

	serverAnn := annotateMethod(serverMethod, svc, nil, model)
	if !serverAnn.IsStreamingRead {
		t.Errorf("expected IsStreamingRead=true for Server")
	}

	clientAnn := annotateMethod(clientMethod, svc, nil, model)
	if !clientAnn.IsStreamingWrite {
		t.Errorf("expected IsStreamingWrite=true for Client")
	}
}

func TestAnnotateMethod_Signatures(t *testing.T) {
	parentField := api.NewTestField("parent").WithType(api.TypezString)
	pageSizeField := api.NewTestField("page_size").WithType(api.TypezInt32)
	req := api.NewTestMessage("ListRequest").WithPackage("test.v1").WithFields(parentField, pageSizeField)
	resp := api.NewTestMessage("ListResponse").WithPackage("test.v1")

	method := api.NewTestMethod("List").WithInput(req).WithOutput(resp)
	method.Signatures = []*api.MethodSignature{
		{Names: []string{"parent"}},
		{Names: []string{"parent", "page_size"}},
	}

	svc := api.NewTestService("TestService").WithPackage("test.v1").WithMethods(method)
	model := api.NewTestAPI([]*api.Message{req, resp}, nil, []*api.Service{svc})
	if err := api.CrossReference(model); err != nil {
		t.Fatal(err)
	}

	ann := annotateMethod(method, svc, nil, model)
	if len(ann.Signatures) != 2 {
		t.Fatalf("got %d signatures, want 2", len(ann.Signatures))
	}
	sig0 := ann.Signatures[0]
	if len(sig0.Params) != 1 || sig0.Params[0] != "std::string const& parent" {
		t.Errorf("sig0 params = %v, want ['std::string const& parent']", sig0.Params)
	}
	sig1 := ann.Signatures[1]
	if len(sig1.Params) != 2 || sig1.Params[0] != "std::string const& parent" || sig1.Params[1] != "std::int32_t page_size" {
		t.Errorf("sig1 params = %v, want ['std::string const& parent', 'std::int32_t page_size']", sig1.Params)
	}
}

func TestAnnotateMethod_RequestID(t *testing.T) {
	reqIDField := api.NewTestField("request_id").WithType(api.TypezString)
	req := api.NewTestMessage("Req").WithPackage("test.v1").WithFields(reqIDField)
	resp := api.NewTestMessage("Resp").WithPackage("test.v1")

	method := api.NewTestMethod("DoSomething").WithInput(req).WithOutput(resp)
	method.AutoPopulated = []*api.Field{reqIDField}

	svc := api.NewTestService("TestService").WithPackage("test.v1").WithMethods(method)
	model := api.NewTestAPI([]*api.Message{req, resp}, nil, []*api.Service{svc})
	if err := api.CrossReference(model); err != nil {
		t.Fatal(err)
	}

	ann := annotateMethod(method, svc, nil, model)
	if !ann.HasRequestID {
		t.Errorf("expected HasRequestID=true")
	}
	if ann.RequestIDFieldName != "request_id" {
		t.Errorf("got RequestIDFieldName %q, want 'request_id'", ann.RequestIDFieldName)
	}
}

func TestAnnotateMethod_IdempotencyOverride(t *testing.T) {
	req := api.NewTestMessage("Req").WithPackage("test.v1")
	resp := api.NewTestMessage("Resp").WithPackage("test.v1")
	method := api.NewTestMethod("Mutate").WithInput(req).WithOutput(resp)
	svc := api.NewTestService("TestService").WithPackage("test.v1").WithMethods(method)
	model := api.NewTestAPI([]*api.Message{req, resp}, nil, []*api.Service{svc})
	if err := api.CrossReference(model); err != nil {
		t.Fatal(err)
	}

	lib := &config.Library{
		Cpp: &config.CppLibrary{
			CppDefault: config.CppDefault{
				ProductPath: "generator/integration_tests/golden/v1",
			},
			IdempotencyOverrides: []config.IdempotencyRule{
				{
					RPCName:     "TestService.Mutate",
					Idempotency: "IDEMPOTENT",
				},
			},
		},
	}

	ann := annotateMethod(method, svc, lib, model)
	if ann.Idempotency != "kIdempotent" {
		t.Errorf("got Idempotency %q, want 'kIdempotent'", ann.Idempotency)
	}
}
