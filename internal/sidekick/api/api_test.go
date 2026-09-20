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

package api

import (
	"maps"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestAPIDefinitionLocation(t *testing.T) {
	t.Run("nil api", func(t *testing.T) {
		var a *API
		gotLoc, gotOK := a.DefinitionLocation("any")
		if gotOK || gotLoc != (SourceLocation{}) {
			t.Errorf("DefinitionLocation(%q) = (%v, %t), want (%v, %t)", "any", gotLoc, gotOK, SourceLocation{}, false)
		}
	})

	t.Run("empty api without map", func(t *testing.T) {
		a := &API{}
		gotLoc, gotOK := a.DefinitionLocation("any")
		if gotOK || gotLoc != (SourceLocation{}) {
			t.Errorf("DefinitionLocation(%q) = (%v, %t), want (%v, %t)", "any", gotLoc, gotOK, SourceLocation{}, false)
		}
	})

	t.Run("add and retrieve definition locations", func(t *testing.T) {
		a := &API{}
		a.AddDefinitionLocation(".test.MyService", SourceLocation{Filename: "service.proto", Line: 10})
		a.AddDefinitionLocation(".test.MyMessage", SourceLocation{Filename: "message.proto", Line: 25})

		for _, test := range []struct {
			name    string
			lookup  string
			wantLoc SourceLocation
			wantOK  bool
		}{
			{
				name:    "service location",
				lookup:  ".test.MyService",
				wantLoc: SourceLocation{Filename: "service.proto", Line: 10},
				wantOK:  true,
			},
			{
				name:    "message location",
				lookup:  ".test.MyMessage",
				wantLoc: SourceLocation{Filename: "message.proto", Line: 25},
				wantOK:  true,
			},
			{
				name:    "non-existent symbol",
				lookup:  ".test.NonExistent",
				wantLoc: SourceLocation{},
				wantOK:  false,
			},
		} {
			t.Run(test.name, func(t *testing.T) {
				gotLoc, gotOK := a.DefinitionLocation(test.lookup)
				if gotOK != test.wantOK {
					t.Errorf("DefinitionLocation(%q) ok = %t, want %t", test.lookup, gotOK, test.wantOK)
				}
				if diff := cmp.Diff(test.wantLoc, gotLoc); diff != "" {
					t.Errorf("DefinitionLocation(%q) mismatch (-want +got):\n%s", test.lookup, diff)
				}
			})
		}
	})

	t.Run("fluent builder WithDefinitionLocation", func(t *testing.T) {
		a := NewTestAPI(nil, nil, nil).
			WithDefinitionLocation(".test.Foo", "foo.proto", 15).
			WithDefinitionLocation(".test.Bar", "bar.proto", 30)

		for _, test := range []struct {
			name    string
			lookup  string
			wantLoc SourceLocation
			wantOK  bool
		}{
			{
				name:    "foo location",
				lookup:  ".test.Foo",
				wantLoc: SourceLocation{Filename: "foo.proto", Line: 15},
				wantOK:  true,
			},
			{
				name:    "bar location",
				lookup:  ".test.Bar",
				wantLoc: SourceLocation{Filename: "bar.proto", Line: 30},
				wantOK:  true,
			},
		} {
			t.Run(test.name, func(t *testing.T) {
				gotLoc, gotOK := a.DefinitionLocation(test.lookup)
				if gotOK != test.wantOK {
					t.Errorf("DefinitionLocation(%q) ok = %t, want %t", test.lookup, gotOK, test.wantOK)
				}
				if diff := cmp.Diff(test.wantLoc, gotLoc); diff != "" {
					t.Errorf("DefinitionLocation(%q) mismatch (-want +got):\n%s", test.lookup, diff)
				}
			})
		}
	})

	t.Run("count and iterator", func(t *testing.T) {
		var nilAPI *API
		if got := nilAPI.DefinitionLocationCount(); got != 0 {
			t.Errorf("DefinitionLocationCount() = %d, want 0", got)
		}
		for range nilAPI.AllDefinitionLocations() {
			t.Errorf("expected no iterations on nil API")
		}

		a := &API{}
		if got := a.DefinitionLocationCount(); got != 0 {
			t.Errorf("DefinitionLocationCount() = %d, want 0", got)
		}
		a.AddDefinitionLocation(".test.Foo", SourceLocation{Filename: "foo.proto", Line: 10})
		a.AddDefinitionLocation(".test.Bar", SourceLocation{Filename: "bar.proto", Line: 20})
		if got := a.DefinitionLocationCount(); got != 2 {
			t.Errorf("DefinitionLocationCount() = %d, want 2", got)
		}

		seen := maps.Collect(a.AllDefinitionLocations())
		wantSeen := map[string]SourceLocation{
			".test.Foo": {Filename: "foo.proto", Line: 10},
			".test.Bar": {Filename: "bar.proto", Line: 20},
		}
		if diff := cmp.Diff(wantSeen, seen); diff != "" {
			t.Errorf("AllDefinitionLocations mismatch (-want +got):\n%s", diff)
		}
	})
}
