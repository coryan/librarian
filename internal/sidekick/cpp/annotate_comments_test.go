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
	"strings"
	"testing"

	"github.com/googleapis/librarian/internal/sidekick/api"
)

func TestAnnotateComments_ClassComments(t *testing.T) {
	svc := api.NewTestService("TestService").WithPackage("test.v1")
	svc.Documentation = "TestService provides testing utilities."
	model := api.NewTestAPI(nil, nil, []*api.Service{svc})

	ann := annotateComments(svc, "TestService", model, false)

	if !strings.Contains(ann.ClassComment, "/// TestService provides testing utilities.") {
		t.Errorf("class comment missing documentation line")
	}
	if !strings.Contains(ann.ClassComment, "@par Equality") {
		t.Errorf("class comment missing Equality section")
	}
	if !strings.Contains(ann.ClassComment, "@par Thread Safety") {
		t.Errorf("class comment missing Thread Safety section")
	}
}

func TestAnnotateComments_MethodProtobufRequest(t *testing.T) {
	req := api.NewTestMessage("GetRequest").WithPackage("test.v1")
	resp := api.NewTestMessage("GetResponse").WithPackage("test.v1")
	method := api.NewTestMethod("Get").WithInput(req).WithOutput(resp)
	method.Documentation = "Retrieves the specified resource."
	svc := api.NewTestService("TestService").WithPackage("test.v1").WithMethods(method)
	model := api.NewTestAPI([]*api.Message{req, resp}, nil, []*api.Service{svc})
	if err := api.CrossReference(model); err != nil {
		t.Fatal(err)
	}

	ann := annotateMethodComments(method, model, false)
	if !strings.Contains(ann.MethodComment, "@param request Unary RPCs") {
		t.Errorf("expected @param request documentation")
	}
	if !strings.Contains(ann.MethodComment, "[test.v1.GetRequest]") {
		t.Errorf("expected link to request type [test.v1.GetRequest]")
	}
	if !strings.Contains(ann.MethodComment, "[Protobuf mapping rules]") {
		t.Errorf("expected [Protobuf mapping rules] trailer")
	}
}

func TestAnnotateComments_MethodSignature(t *testing.T) {
	nameField := api.NewTestField("name").WithType(api.TypezString)
	nameField.Documentation = "The resource name."
	req := api.NewTestMessage("GetRequest").WithPackage("test.v1").WithFields(nameField)
	resp := api.NewTestMessage("GetResponse").WithPackage("test.v1")
	method := api.NewTestMethod("Get").WithInput(req).WithOutput(resp)
	method.Documentation = "Retrieves the specified resource."
	sig := &api.MethodSignature{
		Names: []string{"name"},
	}
	method.Signatures = []*api.MethodSignature{sig}

	svc := api.NewTestService("TestService").WithPackage("test.v1").WithMethods(method)
	model := api.NewTestAPI([]*api.Message{req, resp}, nil, []*api.Service{svc})
	if err := api.CrossReference(model); err != nil {
		t.Fatal(err)
	}

	ann := annotateMethodComments(method, model, false)
	comments, ok := ann.SignatureComments[0]
	if !ok {
		t.Fatalf("expected SignatureComments[0]")
	}
	if !strings.Contains(comments, "@param name  The resource name.") {
		t.Errorf("expected '@param name  The resource name.', got:\n%s", comments)
	}
}

func TestAnnotateComments_DiscoveryLinks(t *testing.T) {
	req := api.NewTestMessage("GetRequest").WithPackage("test.v1")
	resp := api.NewTestMessage("GetResponse").WithPackage("test.v1")
	method := api.NewTestMethod("Get").WithInput(req).WithOutput(resp)
	svc := api.NewTestService("TestService").WithPackage("test.v1").WithMethods(method)
	model := api.NewTestAPI([]*api.Message{req, resp}, nil, []*api.Service{svc})
	model.AddDefinitionLocation("test.v1.GetRequest", api.SourceLocation{Filename: "test/v1/test.proto", Line: 10})
	model.AddDefinitionLocation("test.v1.GetResponse", api.SourceLocation{Filename: "test/v1/test.proto", Line: 20})
	if err := api.CrossReference(model); err != nil {
		t.Fatal(err)
	}

	ann := annotateMethodComments(method, model, true)
	if !strings.Contains(ann.MethodComment, "@cloud_cpp_reference_link") {
		t.Errorf("expected @cloud_cpp_reference_link for discovery, got:\n%s", ann.MethodComment)
	}
}

func TestAnnotateComments_ResolveReferences(t *testing.T) {
	model := api.NewTestAPI(nil, nil, nil)
	model.AddDefinitionLocation("test.v1.MyType", api.SourceLocation{Filename: "test/v1/types.proto", Line: 42})

	comment := "Returns a [MyType][test.v1.MyType] message."
	refs := resolveCommentReferences(comment, model)
	loc, ok := refs["test.v1.MyType"]
	if !ok {
		t.Fatalf("expected reference for test.v1.MyType")
	}
	if loc.Filename != "test/v1/types.proto" || loc.Line != 42 {
		t.Errorf("got location %+v, want filename test/v1/types.proto, line 42", loc)
	}
}
