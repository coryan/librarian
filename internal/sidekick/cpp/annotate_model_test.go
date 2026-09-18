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

func TestAnnotateModel_Success(t *testing.T) {
	field1 := api.NewTestField("name").WithType(api.TypezString)
	field2 := api.NewTestField("count").WithType(api.TypezInt32)
	req := api.NewTestMessage("TestRequest").
		WithPackage("test.v1").
		WithFields(field1, field2)
	resp := api.NewTestMessage("TestResponse").
		WithPackage("test.v1")

	m := api.NewTestMethod("TestMethod").
		WithInput(req).
		WithOutput(resp)

	svc := api.NewTestService("TestService").
		WithPackage("test.v1").
		WithMethods(m)

	model := api.NewTestAPI([]*api.Message{req, resp}, nil, []*api.Service{svc})
	if err := api.CrossReference(model); err != nil {
		t.Fatalf("cross reference failed: %v", err)
	}

	lib := &config.Library{
		CopyrightYear: "2026",
		Cpp: &config.CppLibrary{
			CppDefault: config.CppDefault{
				ProductPath: "google/cloud/test/v1",
			},
		},
	}

	if err := annotateModel(model, lib); err != nil {
		t.Fatalf("annotateModel returned unexpected error: %v", err)
	}

	// Verify field annotations
	if field1.Codec == nil {
		t.Errorf("expected field1.Codec to be populated")
	} else if fa, ok := field1.Codec.(*FieldAnnotations); !ok || fa.CppType != "std::string" {
		t.Errorf("expected FieldAnnotations with CppType std::string, got %#v", field1.Codec)
	}

	// Verify method annotations
	if m.Codec == nil {
		t.Errorf("expected m.Codec to be populated")
	} else if ma, ok := m.Codec.(*MethodAnnotations); !ok || ma.MethodName != "TestMethod" {
		t.Errorf("expected MethodAnnotations with MethodName TestMethod, got %#v", m.Codec)
	}

	// Verify service annotations
	if svc.Codec == nil {
		t.Errorf("expected svc.Codec to be populated")
	} else if sa, ok := svc.Codec.(*ServiceAnnotations); !ok || sa.ServiceName != "TestService" {
		t.Errorf("expected ServiceAnnotations with ServiceName TestService, got %#v", svc.Codec)
	}
}
