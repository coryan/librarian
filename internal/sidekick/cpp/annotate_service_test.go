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
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/sidekick/api"
)

func TestAnnotateService_NamespacesAndClassNames(t *testing.T) {
	req := api.NewTestMessage("Req").WithPackage("golden.v1")
	resp := api.NewTestMessage("Resp").WithPackage("golden.v1")
	m1 := api.NewTestMethod("Method1").WithInput(req).WithOutput(resp)
	m2 := api.NewTestMethod("Method2").WithInput(req).WithOutput(resp)
	svc := api.NewTestService("GoldenThingAdmin").WithPackage("golden.v1").WithMethods(m1, m2)
	model := api.NewTestAPI([]*api.Message{req, resp}, nil, []*api.Service{svc})
	if err := api.CrossReference(model); err != nil {
		t.Fatal(err)
	}

	lib := &config.Library{
		CopyrightYear: "2023",
		Cpp: &config.CppLibrary{
			CppDefault: config.CppDefault{
				ProductPath: "generator/integration_tests/golden/v1",
			},
		},
	}

	ann := annotateService(svc, lib, model)

	if diff := cmp.Diff("GoldenThingAdmin", ann.ServiceName); diff != "" {
		t.Errorf("ServiceName mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff("golden_v1", ann.Namespace()); diff != "" {
		t.Errorf("Namespace mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff("golden_v1_internal", ann.InternalNamespace()); diff != "" {
		t.Errorf("InternalNamespace mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff("golden_v1_mocks", ann.MocksNamespace()); diff != "" {
		t.Errorf("MocksNamespace mismatch (-want +got):\n%s", diff)
	}

	// Class names
	if diff := cmp.Diff("GoldenThingAdminClient", ann.ClientClassName()); diff != "" {
		t.Errorf("ClientClassName mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff("GoldenThingAdminConnection", ann.ConnectionClassName()); diff != "" {
		t.Errorf("ConnectionClassName mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff("GoldenThingAdminConnectionImpl", ann.ConnectionImplClassName()); diff != "" {
		t.Errorf("ConnectionImplClassName mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff("MockGoldenThingAdminConnection", ann.MockConnectionClassName()); diff != "" {
		t.Errorf("MockConnectionClassName mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff("GoldenThingAdminStub", ann.StubClassName()); diff != "" {
		t.Errorf("StubClassName mismatch (-want +got):\n%s", diff)
	}

	// Include guard
	expectedGuard := "GOOGLE_CLOUD_CPP_GENERATOR_INTEGRATION_TESTS_GOLDEN_V1_GOLDEN_THING_ADMIN_CLIENT_H"
	if diff := cmp.Diff(expectedGuard, ann.ClientHeaderIncludeGuard()); diff != "" {
		t.Errorf("ClientHeaderIncludeGuard mismatch (-want +got):\n%s", diff)
	}

	// Decorators
	if !ann.HasAuthDecorator || !ann.HasLoggingDecorator || !ann.HasMetadataDecorator || !ann.HasTracingDecorator {
		t.Errorf("expected standard decorators to be enabled")
	}
	if ann.HasRoundRobinDecorator {
		t.Errorf("expected HasRoundRobinDecorator=false by default")
	}

	if svc.Codec != ann {
		t.Errorf("svc.Codec not set to annotations")
	}
	if ann.Comments == nil || ann.Comments.ClassComment == "" {
		t.Errorf("expected ann.Comments.ClassComment to be populated")
	}
	if ann.Options == nil || ann.Options.OptionsClassName() == "" {
		t.Errorf("expected ann.Options.OptionsClassName() to be populated")
	}
}

func TestAnnotateService_RoundRobinDecorator(t *testing.T) {
	svc := api.NewTestService("KitchenSink").WithPackage("golden.v1")
	model := api.NewTestAPI(nil, nil, []*api.Service{svc})
	if err := api.CrossReference(model); err != nil {
		t.Fatal(err)
	}

	lib := &config.Library{
		Cpp: &config.CppLibrary{
			CppDefault: config.CppDefault{
				ProductPath:                 "generator/integration_tests/golden/v1",
				GenerateRoundRobinDecorator: true,
			},
		},
	}

	ann := annotateService(svc, lib, model)
	if !ann.HasRoundRobinDecorator {
		t.Errorf("expected HasRoundRobinDecorator=true")
	}
}

func TestAnnotateService_MethodFiltering(t *testing.T) {
	req := api.NewTestMessage("Req").WithPackage("test.v1")
	resp := api.NewTestMessage("Resp").WithPackage("test.v1")
	m1 := api.NewTestMethod("KeepMethod").WithInput(req).WithOutput(resp)
	m2 := api.NewTestMethod("OmittedMethod").WithInput(req).WithOutput(resp)
	m3 := api.NewTestMethod("AsyncMethod").WithInput(req).WithOutput(resp)
	svc := api.NewTestService("TestService").WithPackage("test.v1").WithMethods(m1, m2, m3)
	model := api.NewTestAPI([]*api.Message{req, resp}, nil, []*api.Service{svc})
	if err := api.CrossReference(model); err != nil {
		t.Fatal(err)
	}

	lib := &config.Library{
		Cpp: &config.CppLibrary{
			CppDefault: config.CppDefault{
				ProductPath: "generator/integration_tests/golden/v1",
			},
			OmittedRPCs:  []string{"OmittedMethod"},
			GenAsyncRPCs: []string{"AsyncMethod"},
		},
	}

	ann := annotateService(svc, lib, model)

	if len(ann.Methods) != 2 {
		t.Fatalf("got %d methods, want 2", len(ann.Methods))
	}
	for _, m := range ann.Methods {
		if m.Name == "OmittedMethod" {
			t.Errorf("OmittedMethod was not filtered out")
		}
	}

	if len(ann.AsyncMethods) != 1 || ann.AsyncMethods[0].Name != "AsyncMethod" {
		t.Errorf("got async methods %v, want ['AsyncMethod']", ann.AsyncMethods)
	}
}

func TestAnnotateService_Rest(t *testing.T) {
	svc := api.NewTestService("FooBar").WithPackage("test.v1")
	model := api.NewTestAPI(nil, nil, []*api.Service{svc})
	if err := api.CrossReference(model); err != nil {
		t.Fatal(err)
	}

	lib := &config.Library{
		Cpp: &config.CppLibrary{
			CppDefault: config.CppDefault{
				ProductPath:                   "test/v1",
				PreserveProtoFieldNamesInJson: true,
			},
		},
	}

	ann := annotateService(svc, lib, model)
	if diff := cmp.Diff("FooBarRestStub", ann.StubRestClassName()); diff != "" {
		t.Errorf("StubRestClassName mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff("FooBarRestConnectionImpl", ann.ConnectionImplRestClassName()); diff != "" {
		t.Errorf("ConnectionImplRestClassName mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff("FooBarRestLogging", ann.LoggingRestClassName()); diff != "" {
		t.Errorf("LoggingRestClassName mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff("FooBarRestMetadata", ann.MetadataRestClassName()); diff != "" {
		t.Errorf("MetadataRestClassName mismatch (-want +got):\n%s", diff)
	}
	if !ann.PreserveProtoFieldNamesInJson {
		t.Errorf("expected PreserveProtoFieldNamesInJson=true")
	}
	if diff := cmp.Diff("test/v1/foo_bar_rest_connection.h", ann.ConnectionRestHeaderPath()); diff != "" {
		t.Errorf("ConnectionRestHeaderPath mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff("GOOGLE_CLOUD_CPP_TEST_V1_FOO_BAR_REST_CONNECTION_H", ann.ConnectionRestHeaderIncludeGuard()); diff != "" {
		t.Errorf("ConnectionRestHeaderIncludeGuard mismatch (-want +got):\n%s", diff)
	}
}
