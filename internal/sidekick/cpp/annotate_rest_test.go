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

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/sidekick/api"
)

func TestHTTPVerb(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"GET", "Get"},
		{"get", "Get"},
		{"POST", "Post"},
		{"put", "Put"},
		{"PATCH", "Patch"},
		{"delete", "Delete"},
		{"CUSTOM", "Custom"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := httpVerb(tt.input)
			if got != tt.want {
				t.Errorf("httpVerb(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestIsRestMethod(t *testing.T) {
	req := api.NewTestMessage("Req")
	resp := api.NewTestMessage("Resp")

	mNoBindings := api.NewTestMethod("NoBindings").WithInput(req).WithOutput(resp)
	mNoBindings.PathInfo = nil
	if isRestMethod(mNoBindings) {
		t.Errorf("expected isRestMethod=false for method with nil PathInfo")
	}

	mEmptyBindings := api.NewTestMethod("EmptyBindings").WithInput(req).WithOutput(resp)
	mEmptyBindings.PathInfo = &api.PathInfo{}
	if isRestMethod(mEmptyBindings) {
		t.Errorf("expected isRestMethod=false for method with empty Bindings")
	}

	mWithBindings := api.NewTestMethod("WithBindings").WithInput(req).WithOutput(resp)
	mWithBindings.PathInfo = &api.PathInfo{
		Bindings: []*api.PathBinding{
			{
				Verb: "GET",
				PathTemplate: &api.PathTemplate{
					Segments: []api.PathSegment{
						{Literal: "v1"},
						{Literal: "items"},
					},
				},
			},
		},
	}
	if !isRestMethod(mWithBindings) {
		t.Errorf("expected isRestMethod=true for method with PathInfo bindings")
	}

	mStreaming := api.NewTestMethod("Streaming").WithInput(req).WithOutput(resp)
	mStreaming.ServerSideStreaming = true
	mStreaming.PathInfo = mWithBindings.PathInfo
	if isRestMethod(mStreaming) {
		t.Errorf("expected isRestMethod=false for streaming method")
	}
}

func TestFormatRequestResource(t *testing.T) {
	req := api.NewTestMessage("Req")
	resp := api.NewTestMessage("Resp")

	m := api.NewTestMethod("Test").WithInput(req).WithOutput(resp)
	if got := formatRequestResource(m); got != "request" {
		t.Errorf("formatRequestResource(nil PathInfo) = %q, want 'request'", got)
	}

	m.PathInfo = &api.PathInfo{BodyFieldPath: "*"}
	if got := formatRequestResource(m); got != "request" {
		t.Errorf("formatRequestResource(body=*) = %q, want 'request'", got)
	}

	m.PathInfo = &api.PathInfo{BodyFieldPath: "item"}
	if got := formatRequestResource(m); got != "request.item()" {
		t.Errorf("formatRequestResource(body=item) = %q, want 'request.item()'", got)
	}
}

func TestFormatRestPath(t *testing.T) {
	req := api.NewTestMessage("Req")
	resp := api.NewTestMessage("Resp")
	m := api.NewTestMethod("GetItem").WithInput(req).WithOutput(resp)
	m.PathInfo = &api.PathInfo{
		Bindings: []*api.PathBinding{
			{
				Verb: "GET",
				PathTemplate: &api.PathTemplate{
					Segments: []api.PathSegment{
						{Literal: "v1"},
						{Literal: "projects"},
						{Variable: &api.PathVariable{FieldPath: []string{"project"}}},
						{Literal: "items"},
					},
				},
			},
		},
	}

	syncPath := formatRestPath(m, false)
	wantSync := `absl::StrCat("/", rest_internal::DetermineApiVersion("v1", options), "/", "projects", "/", request.project(), "/", "items")`
	if syncPath != wantSync {
		t.Errorf("formatRestPath(sync) =\n  %s\nwant:\n  %s", syncPath, wantSync)
	}

	asyncPath := formatRestPath(m, true)
	wantAsync := `absl::StrCat("/", rest_internal::DetermineApiVersion("v1", *options), "/", "projects", "/", request.project(), "/", "items")`
	if asyncPath != wantAsync {
		t.Errorf("formatRestPath(async) =\n  %s\nwant:\n  %s", asyncPath, wantAsync)
	}

	// With custom verb
	m.PathInfo.Bindings[0].PathTemplate.Verb = "customVerb"
	verbPath := formatRestPath(m, false)
	wantVerb := `absl::StrCat("/", rest_internal::DetermineApiVersion("v1", options), "/", "projects", "/", request.project(), "/", "items", ":customVerb")`
	if verbPath != wantVerb {
		t.Errorf("formatRestPath(with verb) =\n  %s\nwant:\n  %s", verbPath, wantVerb)
	}
}

func TestFormatHTTPQueryParameters(t *testing.T) {
	reqMsg := api.NewTestMessage("GetItemRequest")
	fParent := &api.Field{Name: "parent", Typez: api.TypezString}
	fPageSize := &api.Field{Name: "page_size", Typez: api.TypezInt32}
	fFilter := &api.Field{Name: "filter", Typez: api.TypezString}
	fFlag := &api.Field{Name: "active", Typez: api.TypezBool}
	fWrapper := &api.Field{Name: "note", Typez: api.TypezMessage, TypezID: ".google.protobuf.StringValue"}
	fRepeated := &api.Field{Name: "tags", Typez: api.TypezString, Repeated: true}
	reqMsg.Fields = []*api.Field{fParent, fPageSize, fFilter, fFlag, fWrapper, fRepeated}

	respMsg := api.NewTestMessage("GetItemResponse")
	method := api.NewTestMethod("GetItem").WithInput(reqMsg).WithOutput(respMsg)
	method.PathInfo = &api.PathInfo{
		Bindings: []*api.PathBinding{
			{
				Verb: "GET",
				PathTemplate: &api.PathTemplate{
					Segments: []api.PathSegment{
						{Literal: "v1"},
						{Variable: &api.PathVariable{FieldPath: []string{"parent"}}},
						{Literal: "items"},
					},
				},
			},
		},
	}

	model := api.NewTestAPI([]*api.Message{reqMsg, respMsg}, nil, nil)
	got := formatHTTPQueryParameters(method, model)

	// "parent" is in the path template, so it must not be in query parameters.
	if strings.Contains(got, `"parent"`) {
		t.Errorf("query_params should not contain path variable 'parent', got: %s", got)
	}
	// "tags" is repeated, so it should be excluded.
	if strings.Contains(got, `"tags"`) {
		t.Errorf("query_params should not contain repeated field 'tags', got: %s", got)
	}
	// "page_size" (int32)
	if !strings.Contains(got, `query_params.push_back({"page_size", std::to_string(request.page_size())});`) {
		t.Errorf("missing page_size param in: %s", got)
	}
	// "filter" (string)
	if !strings.Contains(got, `query_params.push_back({"filter", request.filter()});`) {
		t.Errorf("missing filter param in: %s", got)
	}
	// "active" (bool)
	if !strings.Contains(got, `query_params.push_back({"active", (request.active() ? "1" : "0")});`) {
		t.Errorf("missing active param in: %s", got)
	}
	// "note" (StringValue WKT)
	if !strings.Contains(got, `query_params.push_back({"note", (request.has_note() ? request.note().value() : "")});`) {
		t.Errorf("missing note param in: %s", got)
	}
	// TrimEmptyQueryParameters trailer
	if !strings.Contains(got, `query_params = rest_internal::TrimEmptyQueryParameters(std::move(query_params));`) {
		t.Errorf("missing TrimEmptyQueryParameters in: %s", got)
	}
}

func TestFormatHTTPQueryParameters_ExcludesBodyFieldPath(t *testing.T) {
	reqMsg := api.NewTestMessage("UpdateItemRequest")
	fParent := &api.Field{Name: "parent", Typez: api.TypezString}
	fResource := &api.Field{Name: "resource", Typez: api.TypezMessage, TypezID: ".test.v1.Resource"}
	fUpdateMask := &api.Field{Name: "update_mask", Typez: api.TypezString}
	reqMsg.Fields = []*api.Field{fParent, fResource, fUpdateMask}

	respMsg := api.NewTestMessage("Resource")
	method := api.NewTestMethod("UpdateItem").WithInput(reqMsg).WithOutput(respMsg)
	method.PathInfo = &api.PathInfo{
		BodyFieldPath: "resource",
		Bindings: []*api.PathBinding{
			{
				Verb: "PATCH",
				PathTemplate: &api.PathTemplate{
					Segments: []api.PathSegment{
						{Literal: "v1"},
						{Variable: &api.PathVariable{FieldPath: []string{"parent"}}},
					},
				},
			},
		},
	}

	model := api.NewTestAPI([]*api.Message{reqMsg, respMsg}, nil, nil)
	got := formatHTTPQueryParameters(method, model)

	if strings.Contains(got, `"parent"`) {
		t.Errorf("query_params should not contain path variable 'parent', got: %s", got)
	}
	if strings.Contains(got, `"resource"`) {
		t.Errorf("query_params should not contain body field 'resource', got: %s", got)
	}
	if !strings.Contains(got, `query_params.push_back({"update_mask", request.update_mask()});`) {
		t.Errorf("missing update_mask query param, got: %s", got)
	}
}

func TestAnnotateService_PreserveProtoFieldNamesInJson(t *testing.T) {
	svc := api.NewTestService("TestService")
	model := api.NewTestAPI(nil, nil, []*api.Service{svc})

	annDefault := annotateService(svc, nil, model)
	if annDefault.PreserveProtoFieldNamesInJson {
		t.Errorf("got %v, want false", annDefault.PreserveProtoFieldNamesInJson)
	}

	libTrue := &config.Library{
		Cpp: &config.CppLibrary{
			CppDefault: config.CppDefault{
				PreserveProtoFieldNamesInJson: true,
			},
		},
	}
	annTrue := annotateService(svc, libTrue, model)
	if !annTrue.PreserveProtoFieldNamesInJson {
		t.Errorf("got %v, want true", annTrue.PreserveProtoFieldNamesInJson)
	}
}

func TestConnectionGenerator_EndpointLocationStyle(t *testing.T) {
	svc := api.NewTestService("TestService").WithPackage("test.v1")
	model := api.NewTestAPI(nil, nil, []*api.Service{svc})
	if err := api.CrossReference(model); err != nil {
		t.Fatal(err)
	}

	lib := &config.Library{
		Cpp: &config.CppLibrary{
			CppDefault: config.CppDefault{
				ProductPath:           "test/v1",
				EndpointLocationStyle: "LOCATION_DEPENDENT_COMPAT",
			},
		},
	}
	ann := annotateService(svc, lib, model)

	_, headerContent := generateConnectionHeader(svc, ann, nil, nil, lib, model)
	gotInclude := extractBlock(t, headerContent, "#include <string>", "#include <string>")
	if diff := cmp.Diff("#include <string>", gotInclude); diff != "" {
		t.Errorf("include mismatch (-want +got):\n%s", diff)
	}
	gotFactory := extractBlock(t, headerContent, "std::shared_ptr<TestServiceConnection> MakeTestServiceConnection(\n    std::string const& location", ");")
	wantFactory := "std::shared_ptr<TestServiceConnection> MakeTestServiceConnection(\n    std::string const& location, Options options = {});"
	if diff := cmp.Diff(wantFactory, gotFactory); diff != "" {
		t.Errorf("factory declaration mismatch (-want +got):\n%s", diff)
	}
	if !strings.Contains(headerContent, "@deprecated Please use the `location` overload instead.") {
		t.Errorf("expected deprecated doc comment in header for LOCATION_DEPENDENT_COMPAT, got: %s", headerContent)
	}

	_, ccContent := generateConnectionCc(svc, ann, nil, nil, lib, model)
	gotDef := extractBlock(t, ccContent, "MakeTestServiceConnection(\n    std::string const& location, Options options) {", "\n}")
	if !strings.Contains(gotDef, "TestServiceDefaultOptions(\n      location, std::move(options))") {
		t.Errorf("expected DefaultOptions with location argument in cc, got: %s", gotDef)
	}
	gotCompatOverload := extractBlock(t, ccContent, "MakeTestServiceConnection(\n    Options options) {", "\n}")
	wantCompatOverload := "MakeTestServiceConnection(\n    Options options) {\n  return MakeTestServiceConnection(std::string{}, std::move(options));\n}"
	if diff := cmp.Diff(wantCompatOverload, gotCompatOverload); diff != "" {
		t.Errorf("compatibility overload mismatch (-want +got):\n%s", diff)
	}
}

func TestRestConnectionGenerator_EndpointLocationStyleDocs(t *testing.T) {
	svc := api.NewTestService("TestService").WithPackage("test.v1")
	model := api.NewTestAPI(nil, nil, []*api.Service{svc})
	if err := api.CrossReference(model); err != nil {
		t.Fatal(err)
	}

	// LOCATION_DEPENDENT_COMPAT has deprecated comment
	libCompat := &config.Library{
		Cpp: &config.CppLibrary{
			CppDefault: config.CppDefault{
				ProductPath:           "test/v1",
				EndpointLocationStyle: "LOCATION_DEPENDENT_COMPAT",
			},
		},
	}
	annCompat := annotateService(svc, libCompat, model)
	_, headerCompat := generateRestConnectionHeader(annCompat)
	if !strings.Contains(headerCompat, "@deprecated Please use the `location` overload instead.") {
		t.Errorf("expected deprecated doc comment in REST header for LOCATION_DEPENDENT_COMPAT, got: %s", headerCompat)
	}

	// LOCATION_OPTIONALLY_DEPENDENT has global endpoint comment
	libOpt := &config.Library{
		Cpp: &config.CppLibrary{
			CppDefault: config.CppDefault{
				ProductPath:           "test/v1",
				EndpointLocationStyle: "LOCATION_OPTIONALLY_DEPENDENT",
			},
		},
	}
	annOpt := annotateService(svc, libOpt, model)
	_, headerOpt := generateRestConnectionHeader(annOpt)
	if !strings.Contains(headerOpt, "creating a connection to the global service endpoint.") {
		t.Errorf("expected global endpoint doc comment in REST header for LOCATION_OPTIONALLY_DEPENDENT, got: %s", headerOpt)
	}
}
