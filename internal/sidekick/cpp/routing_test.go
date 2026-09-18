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
	"github.com/googleapis/librarian/internal/sidekick/api"
)

func TestFormatRoutingPattern(t *testing.T) {
	for _, test := range []struct {
		name        string
		variant     *api.RoutingInfoVariant
		wantPattern string
		wantFull    bool
	}{
		{
			name:        "full match",
			variant:     api.NewTestRoutingInfoVariant([]string{"field"}, nil, []string{"**"}, nil),
			wantPattern: "(.*)",
			wantFull:    true,
		},
		{
			name:        "single segment matching",
			variant:     api.NewTestRoutingInfoVariant([]string{"field"}, []string{"profiles"}, []string{"*"}, nil),
			wantPattern: "profiles/([^/]+)",
			wantFull:    false,
		},
		{
			name:        "prefix matching and suffix",
			variant:     api.NewTestRoutingInfoVariant([]string{"field"}, []string{"projects", "*"}, []string{"instances", "*"}, []string{"tables", "*"}),
			wantPattern: "projects/[^/]+/(instances/[^/]+)/tables/[^/]+",
			wantFull:    false,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			gotPattern, gotFull := formatRoutingPattern(test.variant)
			if gotPattern != test.wantPattern {
				t.Errorf("formatRoutingPattern() pattern = %q, want %q", gotPattern, test.wantPattern)
			}
			if gotFull != test.wantFull {
				t.Errorf("formatRoutingPattern() fullMatch = %v, want %v", gotFull, test.wantFull)
			}
		})
	}
}

func TestParseExplicitRoutingParams_Ordering(t *testing.T) {
	// Tests that "routing_id" is sorted after other params, and prefix length orders the rest.
	vProject := api.NewTestRoutingInfoVariant([]string{"db"}, nil, []string{"projects", "*"}, []string{"instances", "*", "databases", "*"})
	vInstance := api.NewTestRoutingInfoVariant([]string{"db"}, []string{"projects", "*"}, []string{"instances", "*"}, []string{"databases", "*"})
	vDatabase := api.NewTestRoutingInfoVariant([]string{"db"}, []string{"projects", "*", "instances", "*"}, []string{"databases", "*"}, nil)

	rDatabase := api.NewTestRoutingInfo("database", vDatabase)
	rInstance := api.NewTestRoutingInfo("instance", vInstance)
	rProject := api.NewTestRoutingInfo("project", vProject)

	m := api.NewTestMethod("DropDatabase").
		WithRouting(rDatabase, rInstance, rProject)

	params := parseExplicitRoutingParams(m)
	var gotNames []string
	for _, p := range params {
		gotNames = append(gotNames, p.name)
	}
	wantNames := []string{"project", "instance", "database"}
	if diff := cmp.Diff(wantNames, gotNames); diff != "" {
		t.Errorf("parseExplicitRoutingParams ordering mismatch (-want +got):\n%s", diff)
	}

	// Tests that routing_id is sorted after resource location
	vLocation := api.NewTestRoutingInfoVariant([]string{"table"}, []string{"projects", "*"}, []string{"instances", "*"}, []string{"tables", "*"})
	vRoutingID := api.NewTestRoutingInfoVariant([]string{"profile"}, nil, []string{"**"}, nil)
	rLocation := api.NewTestRoutingInfo("table_location", vLocation)
	rRoutingID := api.NewTestRoutingInfo("routing_id", vRoutingID)

	mExplicit := api.NewTestMethod("Explicit1").
		WithRouting(rRoutingID, rLocation)

	paramsExplicit := parseExplicitRoutingParams(mExplicit)
	var gotExplicitNames []string
	for _, p := range paramsExplicit {
		gotExplicitNames = append(gotExplicitNames, p.name)
	}
	wantExplicitNames := []string{"table_location", "routing_id"}
	if diff := cmp.Diff(wantExplicitNames, gotExplicitNames); diff != "" {
		t.Errorf("parseExplicitRoutingParams ordering mismatch (-want +got):\n%s", diff)
	}
}

func TestFormatMetadataDecoratorSetMetadata(t *testing.T) {
	t.Run("NoRouting_NoPathVariables", func(t *testing.T) {
		m := api.NewTestMethod("Simple")
		ann := &methodAnnotations{
			Name:   "Simple",
			Method: m,
		}
		got := formatMetadataDecoratorSetMetadata(ann, "context", "options")
		want := "  SetMetadata(context, options);"
		if got != want {
			t.Errorf("want %q, got %q", want, got)
		}
	})

	t.Run("NoRouting_WithPathVariable", func(t *testing.T) {
		m := api.NewTestMethod("Get").
			WithPathTemplate(api.NewTestPathTemplate(api.ParseTemplateForTest("v1/{name}")...))
		ann := &methodAnnotations{
			Name:   "Get",
			Method: m,
		}
		got := formatMetadataDecoratorSetMetadata(ann, "context", "options")
		want := `  SetMetadata(context, options, absl::StrCat("name=", internal::UrlEncode(request.name())));`
		if got != want {
			t.Errorf("want %q, got %q", want, got)
		}
	})

	t.Run("ExplicitRouting_AllFullMatch", func(t *testing.T) {
		v := api.NewTestRoutingInfoVariant([]string{"parent"}, nil, []string{"**"}, nil)
		r := api.NewTestRoutingInfo("parent", v)
		m := api.NewTestMethod("CreateFoo").
			WithRouting(r)
		ann := &methodAnnotations{
			Name:        "CreateFoo",
			Method:      m,
			RequestType: "google::test::CreateFooRequest",
		}
		got := formatMetadataDecoratorSetMetadata(ann, "context", "options")
		if !strings.Contains(got, "if (!request.parent().empty()) {") {
			t.Errorf("expected simple if statement, got:\n%s", got)
		}
		if !strings.Contains(got, `params.push_back(absl::StrCat("parent=", internal::UrlEncode(request.parent())));`) {
			t.Errorf("expected params.push_back with UrlEncode, got:\n%s", got)
		}
		if strings.Contains(got, "RoutingMatcher") {
			t.Errorf("did not expect RoutingMatcher for all full match, got:\n%s", got)
		}
	})

	t.Run("ExplicitRouting_WithRegexMatcher", func(t *testing.T) {
		v := api.NewTestRoutingInfoVariant([]string{"name"}, nil, []string{"projects", "*", "parents", "*"}, []string{"**"})
		r := api.NewTestRoutingInfo("parent", v)
		m := api.NewTestMethod("RenameFoo").
			WithRouting(r)
		ann := &methodAnnotations{
			Name:        "RenameFoo",
			Method:      m,
			RequestType: "google::test::RenameFooRequest",
		}
		got := formatMetadataDecoratorSetMetadata(ann, "*context", "*options")
		if !strings.Contains(got, "static auto* parent_matcher = []{") {
			t.Errorf("expected parent_matcher definition, got:\n%s", got)
		}
		if !strings.Contains(got, "RoutingMatcher<google::test::RenameFooRequest>") {
			t.Errorf("expected RoutingMatcher type, got:\n%s", got)
		}
		if !strings.Contains(got, `std::regex{"(projects/[^/]+/parents/[^/]+)/.*", std::regex::optimize}`) {
			t.Errorf("expected regex in matcher, got:\n%s", got)
		}
		if !strings.Contains(got, "SetMetadata(*context, *options, absl::StrJoin(params, \"&\"));") {
			t.Errorf("expected async SetMetadata call, got:\n%s", got)
		}
	})
}
