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

	"github.com/google/go-cmp/cmp"
)

func TestPrinter_Substitution(t *testing.T) {
	p := newPrinter(map[string]string{
		"class_name": "MyClass",
		"foo":        "bar",
	})
	p.Print("class $class_name$ {\n  $foo$;\n  $$100;\n};\n")
	got := p.String()
	want := "class MyClass {\n  bar;\n  $100;\n};\n"
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestPrinter_Includes(t *testing.T) {
	p := newPrinter(nil)
	p.HeaderLocalIncludes([]string{"b.h", "a.h", "", "c.h"})
	p.SystemIncludes([]string{"vector", "memory", "string"})
	p.ProtobufIncludes([]string{"foo.pb.h"})

	got := p.String()
	want := `#include "a.h"
#include "b.h"
#include "c.h"
#include <memory>
#include <string>
#include <vector>
#include <foo.pb.h>
`
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestPrinter_CcLocalIncludes(t *testing.T) {
	p := newPrinter(nil)
	p.CcLocalIncludes([]string{"client.h", "z.h", "a.h", "b.h"})
	got := p.String()
	want := `#include "client.h"
#include "a.h"
#include "b.h"
#include "z.h"
`
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}
