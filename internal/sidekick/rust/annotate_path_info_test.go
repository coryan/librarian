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

package rust

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	libconfig "github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/sidekick/api"
)

func serviceAnnotationsModel() *api.API {
	request := api.NewTestMessage("Request").WithPackage("test.v1")
	response := api.NewTestMessage("Response").
		WithPackage("test.v1").
		WithFields(
			api.NewTestField("field").
				WithType(api.TypezEnum).
				WithTypezID(".test.v1.UsedEnum"),
		)
	method := api.NewTestMethod("GetResource").
		WithInput(request).
		WithOutput(response).
		WithVerb("GET").
		WithPathTemplate((&api.PathTemplate{}).
			WithLiteral("v1").
			WithLiteral("resource"))
	emptyMethod := api.NewTestMethod("DeleteResource").
		WithInput(request).
		WithVerb("DELETE").
		WithPathTemplate((&api.PathTemplate{}).
			WithLiteral("v1").
			WithLiteral("resource"))
	emptyMethod.ReturnsEmpty = true
	emptyMethod.OutputTypeID = ".google.protobuf.Empty"

	noHttpMethod := api.NewTestMethod("DoAThing").
		WithInput(request).
		WithOutput(response)
	noHttpMethod.PathInfo = nil

	service := api.NewTestService("ResourceService").
		WithPackage("test.v1").
		WithMethods(method, emptyMethod, noHttpMethod)

	usedEnum := api.NewTestEnum("UsedEnum").WithPackage("test.v1")
	extraEnum := api.NewTestEnum("ExtraEnum").WithPackage("test.v1")

	model := api.NewTestAPI(
		[]*api.Message{request, response},
		[]*api.Enum{usedEnum, extraEnum},
		[]*api.Service{service})
	api.CrossReference(model)
	return model
}

func TestPathInfoAnnotations(t *testing.T) {
	binding := func(verb string) *api.PathBinding {
		return &api.PathBinding{
			Verb: verb,
			PathTemplate: (&api.PathTemplate{}).
				WithLiteral("v1").
				WithLiteral("resource"),
		}
	}

	for _, test := range []struct {
		name               string
		Bindings           []*api.PathBinding
		DefaultIdempotency string
	}{
		{"empty", []*api.PathBinding{}, "false"},
		{"GET", []*api.PathBinding{binding("GET")}, "true"},
		{"PUT", []*api.PathBinding{binding("PUT")}, "true"},
		{"DELETE", []*api.PathBinding{binding("DELETE")}, "true"},
		{"POST", []*api.PathBinding{binding("POST")}, "false"},
		{"PATCH", []*api.PathBinding{binding("PATCH")}, "false"},
		{"GET_GET", []*api.PathBinding{binding("GET"), binding("GET")}, "true"},
		{"GET_POST", []*api.PathBinding{binding("GET"), binding("POST")}, "false"},
		{"POST_POST", []*api.PathBinding{binding("POST"), binding("POST")}, "false"},
	} {
		t.Run(test.name, func(t *testing.T) {
			request := api.NewTestMessage("Request").WithPackage("test.v1")
			response := api.NewTestMessage("Response").WithPackage("test.v1")
			method := api.NewTestMethod("GetResource").
				WithInput(request).
				WithOutput(response)
			method.PathInfo = &api.PathInfo{
				Bindings: test.Bindings,
			}
			service := api.NewTestService("ResourceService").
				WithPackage("test.v1").
				WithMethods(method)

			model := api.NewTestAPI(
				[]*api.Message{request, response},
				[]*api.Enum{},
				[]*api.Service{service})
			api.CrossReference(model)
			codec := newTestCodec(t, libconfig.SpecProtobuf, "test.v1", map[string]string{
				"include-grpc-only-methods": "true",
			})
			annotateModel(model, codec)

			pathInfoAnn := method.PathInfo.Codec.(*pathInfoAnnotation)
			if pathInfoAnn.IsIdempotent != test.DefaultIdempotency {
				t.Errorf("fail")
			}
		})
	}
}

func TestPathBindingAnnotations(t *testing.T) {
	f_name := api.NewTestField("name").WithType(api.TypezString)
	f_project := api.NewTestField("project").WithType(api.TypezString)
	f_location := api.NewTestField("location").WithType(api.TypezString)
	f_id := api.NewTestField("id").WithType(api.TypezUint64)
	f_optional := api.NewTestField("optional").WithType(api.TypezString).WithOptional()

	// A field also of type `Request`. We want to test nested path
	// parameters, and this saves us from having to define a new
	// `api.Message`, with all of its fields.
	f_child := api.NewTestField("child").
		WithType(api.TypezMessage).
		WithTypezID(".test.Request").
		WithOptional()

	f_oneof := api.NewTestField("oneofField").WithType(api.TypezString)
	f_oneof.IsOneOf = true

	request := api.NewTestMessage("Request").
		WithFields(
			f_name,
			f_project,
			f_location,
			f_id,
			f_optional,
			f_child,
			f_oneof,
		)
	response := api.NewTestMessage("Response")

	b0 := &api.PathBinding{
		Verb: "POST",
		PathTemplate: (&api.PathTemplate{}).
			WithLiteral("v2").
			WithVariable(api.NewPathVariable("name").
				WithLiteral("projects").
				WithMatch().
				WithLiteral("locations").
				WithMatch()).
			WithVerb("create"),
		QueryParameters: map[string]bool{
			"id": true,
		},
	}
	want_b0 := &pathBindingAnnotation{
		PathFmt:     "/v2/{}:create",
		QueryParams: []*api.Field{f_id},
		Substitutions: []*bindingSubstitution{
			{
				FieldAccessor:  "Some(&req).map(|m| &m.name).map(|s| s.as_str())",
				PathExtraction: &PathExtraction{FieldTake: "Some(&mut req).map(|m| std::mem::take(&mut m.name))"},
				FieldName:      "name",
				Template:       []string{"projects", "*", "locations", "*"},
			},
		},
	}

	b1 := &api.PathBinding{
		Verb: "POST",
		PathTemplate: (&api.PathTemplate{}).
			WithLiteral("v1").
			WithLiteral("projects").
			WithVariableNamed("project").
			WithLiteral("locations").
			WithVariableNamed("location").
			WithLiteral("ids").
			WithVariableNamed("id").
			WithVerb("action"),
	}
	want_b1 := &pathBindingAnnotation{
		PathFmt: "/v1/projects/{}/locations/{}/ids/{}:action",
		Substitutions: []*bindingSubstitution{
			{
				FieldAccessor:  "Some(&req).map(|m| &m.project).map(|s| s.as_str())",
				PathExtraction: &PathExtraction{FieldTake: "Some(&mut req).map(|m| std::mem::take(&mut m.project))"},
				FieldName:      "project",
				Template:       []string{"*"},
			},
			{
				FieldAccessor:  "Some(&req).map(|m| &m.location).map(|s| s.as_str())",
				PathExtraction: &PathExtraction{FieldTake: "Some(&mut req).map(|m| std::mem::take(&mut m.location))"},
				FieldName:      "location",
				Template:       []string{"*"},
			},
			{
				FieldAccessor:  "Some(&req).map(|m| &m.id)",
				PathExtraction: &PathExtraction{FieldTake: "Some(&mut req).map(|m| std::mem::take(&mut m.id))"},
				FieldName:      "id",
				Template:       []string{"*"},
			},
		},
	}
	b2 := &api.PathBinding{
		Verb: "POST",
		PathTemplate: (&api.PathTemplate{}).
			WithLiteral("v1").
			WithLiteral("projects").
			WithVariableNamed("child", "project").
			WithLiteral("locations").
			WithVariableNamed("child", "location").
			WithLiteral("ids").
			WithVariableNamed("child", "id").
			WithVerb("actionOnChild"),
	}
	want_b2 := &pathBindingAnnotation{
		PathFmt: "/v1/projects/{}/locations/{}/ids/{}:actionOnChild",
		Substitutions: []*bindingSubstitution{
			{
				FieldAccessor:  "Some(&req).and_then(|m| m.child.as_ref()).map(|m| &m.project).map(|s| s.as_str())",
				PathExtraction: &PathExtraction{FieldTake: "Some(&mut req).and_then(|m| m.child.as_mut()).map(|m| std::mem::take(&mut m.project))"},
				FieldName:      "child.project",
				Template:       []string{"*"},
			},
			{
				FieldAccessor:  "Some(&req).and_then(|m| m.child.as_ref()).map(|m| &m.location).map(|s| s.as_str())",
				PathExtraction: &PathExtraction{FieldTake: "Some(&mut req).and_then(|m| m.child.as_mut()).map(|m| std::mem::take(&mut m.location))"},
				FieldName:      "child.location",
				Template:       []string{"*"},
			},
			{
				FieldAccessor:  "Some(&req).and_then(|m| m.child.as_ref()).map(|m| &m.id)",
				PathExtraction: &PathExtraction{FieldTake: "Some(&mut req).and_then(|m| m.child.as_mut()).map(|m| std::mem::take(&mut m.id))"},
				FieldName:      "child.id",
				Template:       []string{"*"},
			},
		},
	}
	b3 := &api.PathBinding{
		Verb: "GET",
		PathTemplate: (&api.PathTemplate{}).
			WithLiteral("v2").
			WithLiteral("foos"),
		QueryParameters: map[string]bool{
			"name":     true,
			"optional": true,
			"child":    true,
		},
	}
	want_b3 := &pathBindingAnnotation{
		PathFmt:     "/v2/foos",
		QueryParams: []*api.Field{f_name, f_optional, f_child},
	}
	b4 := &api.PathBinding{
		Verb: "POST",
		PathTemplate: (&api.PathTemplate{}).
			WithLiteral("v1").
			WithLiteral("projects").
			WithVariableNamed("oneofField").
			WithVerb("actionOnOneof"),
	}
	want_b4 := &pathBindingAnnotation{
		PathFmt: "/v1/projects/{}:actionOnOneof",
		Substitutions: []*bindingSubstitution{
			{
				FieldAccessor: "Some(&req).and_then(|m| m.oneof_field()).map(|s| s.as_str())",
				FieldName:     "oneof_field",
				Template:      []string{"*"},
			},
		},
	}
	method := api.NewTestMethod("DoFoo").
		WithInput(request).
		WithOutput(response)
	method.PathInfo = &api.PathInfo{
		Bindings:      []*api.PathBinding{b0, b1, b2, b3},
		BodyFieldPath: "*",
	}
	methodBar := api.NewTestMethod("DoBar").
		WithInput(request).
		WithOutput(response)
	methodBar.PathInfo = &api.PathInfo{
		Bindings:      []*api.PathBinding{b4},
		BodyFieldPath: "",
	}
	service := api.NewTestService("FooService").
		WithMethods(method, methodBar)

	model := api.NewTestAPI(
		[]*api.Message{request, response},
		[]*api.Enum{},
		[]*api.Service{service})
	api.CrossReference(model)
	codec := newTestCodec(t, libconfig.SpecProtobuf, "", map[string]string{})
	annotateModel(model, codec)

	if diff := cmp.Diff(want_b0, b0.Codec, cmpopts.IgnoreUnexported(pathBindingAnnotation{}, bindingSubstitution{})); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(want_b1, b1.Codec, cmpopts.IgnoreUnexported(pathBindingAnnotation{}, bindingSubstitution{})); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(want_b2, b2.Codec, cmpopts.IgnoreUnexported(pathBindingAnnotation{}, bindingSubstitution{})); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}

	if diff := cmp.Diff(want_b3, b3.Codec, cmpopts.IgnoreUnexported(pathBindingAnnotation{}, bindingSubstitution{})); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}

	if diff := cmp.Diff(want_b4, b4.Codec, cmpopts.IgnoreUnexported(pathBindingAnnotation{}, bindingSubstitution{})); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestPathBindingAnnotationsDetailedTracing(t *testing.T) {
	f_name := api.NewTestField("name").WithType(api.TypezString)
	request := api.NewTestMessage("Request").WithFields(f_name)
	response := api.NewTestMessage("Response")
	binding := &api.PathBinding{
		Verb: "POST",
		PathTemplate: (&api.PathTemplate{}).
			WithLiteral("v2").
			WithVariable(api.NewPathVariable("name").
				WithLiteral("projects").
				WithMatch()).
			WithVerb("create"),
	}
	method := api.NewTestMethod("DoFoo").
		WithInput(request).
		WithOutput(response)
	method.PathInfo = &api.PathInfo{
		Bindings: []*api.PathBinding{binding},
	}
	service := api.NewTestService("FooService").WithMethods(method)
	model := api.NewTestAPI(
		[]*api.Message{request, response},
		[]*api.Enum{},
		[]*api.Service{service})
	api.CrossReference(model)
	codec := newTestCodec(t, libconfig.SpecProtobuf, "", map[string]string{
		"detailed-tracing-attributes": "true",
	})
	annotateModel(model, codec)

	got := binding.Codec.(*pathBindingAnnotation)
	if !got.DetailedTracingAttributes {
		t.Errorf("pathBindingAnnotation.DetailedTracingAttributes = %v, want %v", got.DetailedTracingAttributes, true)
	}
}

func TestPathBindingAnnotationsStyle(t *testing.T) {
	for _, test := range []struct {
		FieldName     string
		WantFieldName string
		WantAccessor  string
		WantClear     string
	}{
		{"machine", "machine", "Some(&req).map(|m| &m.machine).map(|s| s.as_str())", "Some(&mut req).map(|m| std::mem::take(&mut m.machine))"},
		{"machineType", "machine_type", "Some(&req).map(|m| &m.machine_type).map(|s| s.as_str())", "Some(&mut req).map(|m| std::mem::take(&mut m.machine_type))"},
		{"machine_type", "machine_type", "Some(&req).map(|m| &m.machine_type).map(|s| s.as_str())", "Some(&mut req).map(|m| std::mem::take(&mut m.machine_type))"},
		{"type", "type", "Some(&req).map(|m| &m.r#type).map(|s| s.as_str())", "Some(&mut req).map(|m| std::mem::take(&mut m.r#type))"},
	} {
		field := api.NewTestField(test.FieldName).WithType(api.TypezString)
		field.JSONName = test.FieldName
		request := api.NewTestMessage("Request").WithFields(field)
		response := api.NewTestMessage("Response")
		binding := &api.PathBinding{
			Verb: "GET",
			PathTemplate: (&api.PathTemplate{}).
				WithLiteral("v1").
				WithLiteral("machines").
				WithVariable(api.NewPathVariable(test.FieldName).
					WithMatch()).
				WithVerb("create"),
			QueryParameters: map[string]bool{},
		}
		wantBinding := &pathBindingAnnotation{
			PathFmt: "/v1/machines/{}:create",
			Substitutions: []*bindingSubstitution{
				{
					FieldAccessor:  test.WantAccessor,
					FieldName:      test.WantFieldName,
					PathExtraction: &PathExtraction{FieldTake: test.WantClear},
					Template:       []string{"*"},
				},
			},
		}
		method := api.NewTestMethod("Create").
			WithInput(request).
			WithOutput(response)
		method.PathInfo = &api.PathInfo{
			Bindings:      []*api.PathBinding{binding},
			BodyFieldPath: "*",
		}
		service := api.NewTestService("Service").WithMethods(method)
		model := api.NewTestAPI(
			[]*api.Message{request, response},
			[]*api.Enum{},
			[]*api.Service{service})
		api.CrossReference(model)
		codec := newTestCodec(t, libconfig.SpecProtobuf, "", map[string]string{})
		annotateModel(model, codec)
		if diff := cmp.Diff(wantBinding, binding.Codec, cmpopts.IgnoreUnexported(pathBindingAnnotation{}, bindingSubstitution{})); diff != "" {
			t.Errorf("mismatch (-want +got):\n%s", diff)
		}

	}
}

func TestPathBindingAnnotationsErrors(t *testing.T) {
	field := api.NewTestField("field").WithType(api.TypezString)
	request := api.NewTestMessage("Request").WithFields(field)
	method := api.NewTestMethod("Create").WithInput(request)
	if got, err := makeAccessors([]string{"not-a-field-name"}, method); err == nil {
		t.Errorf("expected an error in makeAccessors() for an invalid field name, got=%v", got)
	}
}

func TestPathTemplateGeneration(t *testing.T) {
	for _, test := range []struct {
		name    string
		binding *pathBindingAnnotation
		want    string
	}{
		{
			name: "Simple Literal",
			binding: &pathBindingAnnotation{
				PathFmt: "/v1/things",
			},
			want: "/v1/things",
		},
		{
			name: "Single Variable",
			binding: &pathBindingAnnotation{
				PathFmt: "/v1/things/{}",
				Substitutions: []*bindingSubstitution{
					{FieldName: "thing_id"},
				},
			},
			want: "/v1/things/{thing_id}",
		},
		{
			name: "Multiple Variables",
			binding: &pathBindingAnnotation{
				PathFmt: "/v1/projects/{}/locations/{}",
				Substitutions: []*bindingSubstitution{
					{FieldName: "project"},
					{FieldName: "location"},
				},
			},
			want: "/v1/projects/{project}/locations/{location}",
		},
		{
			name: "Variable with Complex Segment Match",
			binding: &pathBindingAnnotation{
				PathFmt: "/v1/{}/databases",
				Substitutions: []*bindingSubstitution{
					{FieldName: "name"},
				},
			},
			want: "/v1/{name}/databases",
		},
		{
			name: "Variable Capturing Remaining Path",
			binding: &pathBindingAnnotation{
				PathFmt: "/v1/objects/{}",
				Substitutions: []*bindingSubstitution{
					{FieldName: "object"},
				},
			},
			want: "/v1/objects/{object}",
		},
		{
			name: "Top-Level Single Wildcard",
			binding: &pathBindingAnnotation{
				PathFmt: "/{}",
				Substitutions: []*bindingSubstitution{
					{FieldName: "field"},
				},
			},
			want: "/{field}",
		},
		{
			name: "Path with Custom Verb",
			binding: &pathBindingAnnotation{
				PathFmt: "/v1/things/{}:customVerb",
				Substitutions: []*bindingSubstitution{
					{FieldName: "thing_id"},
				},
			},
			want: "/v1/things/{thing_id}:customVerb",
		},
		{
			name: "Nested fields",
			binding: &pathBindingAnnotation{
				PathFmt: "/v1/projects/{}/locations/{}/ids/{}:actionOnChild",
				Substitutions: []*bindingSubstitution{
					{FieldName: "child.project"},
					{FieldName: "child.location"},
					{FieldName: "child.id"},
				},
			},
			want: "/v1/projects/{child.project}/locations/{child.location}/ids/{child.id}:actionOnChild",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := test.binding.PathTemplate(); got != test.want {
				t.Errorf("PathTemplate() = %v, want %v", got, test.want)
			}
		})
	}
}

func TestBindingSubstitutionTemplates(t *testing.T) {
	b := bindingSubstitution{
		Template: []string{"projects", "*", "locations", "*", "**"},
	}

	got := b.TemplateAsString()
	want := "projects/*/locations/*/**"

	if want != got {
		t.Errorf("TemplateAsString() failed. want=%q, got=%q", want, got)
	}

	got = b.TemplateAsArray()
	want = `&[Segment::Literal("projects/"), Segment::SingleWildcard, Segment::Literal("/locations/"), Segment::SingleWildcard, Segment::TrailingMultiWildcard]`

	if want != got {
		t.Errorf("TemplateAsArray() failed. want=`%s`, got=`%s`", want, got)
	}
}
