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
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/sidekick/api"
)

func TestAnnotateService_GrpcAndRest(t *testing.T) {
	svc := api.NewTestService("GoldenKitchenSink")
	model := api.NewTestAPI(nil, nil, []*api.Service{svc})

	trueVal := true
	lib := &config.Library{
		Name:   "golden",
		Output: "generator/integration_tests/golden/v1",
		Cpp: &config.CppLibrary{
			ProductPath:                 "generator/integration_tests/golden/v1",
			ForwardingProductPath:       "generator/integration_tests/golden",
			InitialCopyrightYear:        "2022",
			GenerateRestTransport:       true,
			GenerateGrpcTransport:       &trueVal,
			GenerateRoundRobinDecorator: true,
			RetryableStatusCodes:        []string{"kUnavailable"},
		},
	}

	c := newCodec(model, "generator/integration_tests/golden/v1", lib)
	if err := c.annotateModel(); err != nil {
		t.Fatalf("annotateModel() failed: %v", err)
	}

	ann, ok := svc.Codec.(*serviceAnnotations)
	if !ok || ann == nil {
		t.Fatalf("expected *serviceAnnotations on service.Codec")
	}

	if ann.BaseFileName != "golden_kitchen_sink" {
		t.Errorf("BaseFileName mismatch: want 'golden_kitchen_sink', got %q", ann.BaseFileName)
	}
	if !ann.HasGrpc {
		t.Errorf("HasGrpc mismatch: want true, got false")
	}
	if !ann.HasRest {
		t.Errorf("HasRest mismatch: want true, got false")
	}
	if !ann.HasRoundRobin {
		t.Errorf("HasRoundRobin mismatch: want true, got false")
	}
	if !ann.HasRetryTraits {
		t.Errorf("HasRetryTraits mismatch: want true, got false")
	}

	files := ann.generatedFiles("..")
	// 42 files in v1 + 5 forwarding headers = 47 files
	if len(files) != 47 {
		t.Errorf("generatedFiles count mismatch: want 47, got %d", len(files))
	}

	// Verify forwarding header output paths
	var forwardingCount int
	for _, f := range files {
		if strings.HasPrefix(f.OutputPath, "..") {
			forwardingCount++
		}
	}
	if forwardingCount != 5 {
		t.Errorf("forwarding headers count mismatch: want 5, got %d", forwardingCount)
	}
}

func TestAnnotateService_RestOnly(t *testing.T) {
	svc := api.NewTestService("GoldenRestOnly")
	model := api.NewTestAPI(nil, nil, []*api.Service{svc})

	falseVal := false
	lib := &config.Library{
		Name:   "golden-test2",
		Output: "generator/integration_tests/golden/v1",
		Cpp: &config.CppLibrary{
			ProductPath:           "generator/integration_tests/golden/v1",
			GenerateRestTransport: true,
			GenerateGrpcTransport: &falseVal,
			RetryableStatusCodes:  []string{"kUnavailable"},
		},
	}

	c := newCodec(model, "generator/integration_tests/golden/v1", lib)
	if err := c.annotateModel(); err != nil {
		t.Fatalf("annotateModel() failed: %v", err)
	}

	ann, ok := svc.Codec.(*serviceAnnotations)
	if !ok || ann == nil {
		t.Fatalf("expected *serviceAnnotations on service.Codec")
	}

	if ann.HasGrpc {
		t.Errorf("HasGrpc mismatch: want false, got true")
	}
	if !ann.HasRest {
		t.Errorf("HasRest mismatch: want true, got false")
	}

	files := ann.generatedFiles("")
	// 26 files for REST-only
	if len(files) != 26 {
		t.Errorf("generatedFiles count mismatch: want 26, got %d", len(files))
	}
}

func TestAnnotateService_OmittedService(t *testing.T) {
	svc1 := api.NewTestService("KeepService")
	svc2 := api.NewTestService("OmittedService")
	model := api.NewTestAPI(nil, nil, []*api.Service{svc1, svc2})

	lib := &config.Library{
		Name:   "test-lib",
		Output: "out",
		Cpp: &config.CppLibrary{
			OmittedServices: []string{"OmittedService"},
		},
	}

	c := newCodec(model, "out", lib)
	if err := c.annotateModel(); err != nil {
		t.Fatalf("annotateModel() failed: %v", err)
	}

	if svc1.Codec == nil {
		t.Errorf("expected KeepService to be annotated")
	}
	if svc2.Codec != nil {
		t.Errorf("expected OmittedService to NOT be annotated")
	}
}

func TestAnnotateService_DerivedFileNames(t *testing.T) {
	ann := &serviceAnnotations{
		BaseFileName: "my_service",
	}

	for _, test := range []struct {
		name string
		got  string
		want string
	}{
		{name: "ClientHeader", got: ann.ClientHeader(), want: "my_service_client.h"},
		{name: "ClientSource", got: ann.ClientSource(), want: "my_service_client.cc"},
		{name: "ConnectionHeader", got: ann.ConnectionHeader(), want: "my_service_connection.h"},
		{name: "ConnectionSource", got: ann.ConnectionSource(), want: "my_service_connection.cc"},
		{name: "MockConnectionHeader", got: ann.MockConnectionHeader(), want: filepath.Join("mocks", "mock_my_service_connection.h")},
		{name: "StubHeader", got: ann.StubHeader(), want: filepath.Join("internal", "my_service_stub.h")},
		{name: "SourcesSource", got: ann.SourcesSource(), want: filepath.Join("internal", "my_service_sources.cc")},
	} {
		t.Run(test.name, func(t *testing.T) {
			if diff := cmp.Diff(test.want, test.got); diff != "" {
				t.Errorf("%s mismatch (-want +got):\n%s", test.name, diff)
			}
		})
	}
}

func TestAnnotateService_NonStreamingMethods(t *testing.T) {
	unary := api.NewTestMethod("Unary")
	serverStreaming := api.NewTestMethod("ServerStreaming").WithServerSideStreaming()
	clientStreaming := api.NewTestMethod("ClientStreaming").WithClientSideStreaming()
	bidiStreaming := api.NewTestMethod("BidiStreaming").WithBidiStreaming()

	svc := api.NewTestService("StreamingTestService").WithMethods(unary, serverStreaming, clientStreaming, bidiStreaming)
	model := api.NewTestAPI(nil, nil, []*api.Service{svc})

	lib := &config.Library{
		Name:   "test-lib",
		Output: "out",
		Cpp:    &config.CppLibrary{},
	}

	c := newCodec(model, "out", lib)
	if err := c.annotateModel(); err != nil {
		t.Fatalf("annotateModel() failed: %v", err)
	}

	ann, ok := svc.Codec.(*serviceAnnotations)
	if !ok || ann == nil {
		t.Fatalf("expected *serviceAnnotations on service.Codec")
	}

	nonStreaming := ann.NonStreamingMethods()
	if len(nonStreaming) != 1 || nonStreaming[0].Name != "Unary" {
		t.Errorf("NonStreamingMethods() want [Unary], got %+v", nonStreaming)
	}

	streamingRead := ann.StreamingReadMethods()
	if len(streamingRead) != 1 || streamingRead[0].Name != "ServerStreaming" {
		t.Errorf("StreamingReadMethods() want [ServerStreaming], got %+v", streamingRead)
	}

	if !ann.HasStreamingReadMethod() {
		t.Errorf("HasStreamingReadMethod() want true, got false")
	}

	if !ann.HasBidirStreamingMethod() {
		t.Errorf("HasBidirStreamingMethod() want true, got false")
	}
}

func TestServiceAnnotations_Layer36_Helpers(t *testing.T) {
	testSvc := api.NewTestService("TestService").
		WithPackage("google.test.v1").
		WithDefaultHost("test.googleapis.com")
	locSvc := api.NewTestService("Locations")

	s := &serviceAnnotations{
		Name:                  "TestService",
		BaseFileName:          "test_service",
		ServiceEndpointEnvVar: "GOOGLE_CLOUD_CPP_TEST_SERVICE_ENDPOINT",
		Service:               testSvc,
		Methods: []*methodAnnotations{
			{
				Method: api.NewTestMethod("Method1").WithAPIVersion("v1_20240101"),
				Signatures: []*signatureAnnotations{
					{
						Params: []*signatureParamAnnotations{
							{Field: api.NewTestField("dep_field").WithDeprecated(true)},
						},
					},
				},
			},
		},
		StubMethods: []*methodAnnotations{
			{
				Method: api.NewTestMethod("LocationMethod").
					WithSourceService(locSvc).
					WithService(testSvc),
			},
		},
	}

	if !s.MethodSignatureUsesDeprecatedField() {
		t.Errorf("MethodSignatureUsesDeprecatedField: want true, got false")
	}
	if got := s.ServiceAuthorityEnvVar(); got != "GOOGLE_CLOUD_CPP_TEST_SERVICE_AUTHORITY" {
		t.Errorf("ServiceAuthorityEnvVar: want 'GOOGLE_CLOUD_CPP_TEST_SERVICE_AUTHORITY', got %q", got)
	}
	if got := s.DefaultEndpoint(); got != "test.googleapis.com" {
		t.Errorf("DefaultEndpoint: want 'test.googleapis.com', got %q", got)
	}
	if got := s.ServiceGrpcFqn(); got != "google.test.v1.TestService" {
		t.Errorf("ServiceGrpcFqn: want 'google.test.v1.TestService', got %q", got)
	}
	if got := s.StreamingUpdaterFunctionName(); got != "TestServiceStreamingReadStreamingUpdater" {
		t.Errorf("StreamingUpdaterFunctionName: want 'TestServiceStreamingReadStreamingUpdater', got %q", got)
	}
	if got := s.ApiVersion(); got != "v1_20240101" {
		t.Errorf("ApiVersion: want 'v1_20240101', got %q", got)
	}
	if !s.HasLocationsMixin() {
		t.Errorf("HasLocationsMixin: want true, got false")
	}
	if s.HasIamMixin() {
		t.Errorf("HasIamMixin: want false, got true")
	}
	if s.HasOperationsMixin() {
		t.Errorf("HasOperationsMixin: want false, got true")
	}

	stubCode := s.StubFactoryMakeDefaultStub()
	if !strings.Contains(stubCode, "service_locations_stub") {
		t.Errorf("StubFactoryMakeDefaultStub expected to contain service_locations_stub, got:\n%s", stubCode)
	}
}
