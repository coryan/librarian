// Copyright 2025 Google LLC
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

package dart

import (
	"fmt"
	"maps"
	"math/rand"
	"slices"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/sample"
	"github.com/googleapis/librarian/internal/sidekick/api"
)

var (
	requiredConfig = map[string]string{
		"api-keys-environment-variables": "GOOGLE_API_KEY,GEMINI_API_KEY",
		"issue-tracker-url":              "http://www.example.com/issues",
		"package:google_cloud_rpc":       "^1.2.3",
		"package:http":                   "^4.5.6",
		"package:google_cloud_protobuf":  "^7.8.9",
	}
)

func TestAnnotateModel(t *testing.T) {
	model := api.NewTestAPI([]*api.Message{}, []*api.Enum{}, []*api.Service{})
	model.PackageName = "test"

	options := maps.Clone(requiredConfig)
	maps.Copy(options, map[string]string{"package:google_cloud_rpc": "^1.2.3"})

	annotate := newAnnotateModel(model)
	err := annotate.annotateModel(options)
	if err != nil {
		t.Fatal(err)
	}

	codec := model.Codec.(*modelAnnotations)

	if diff := cmp.Diff("google_cloud_test", codec.PackageName); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff("test.dart", codec.MainFileNameWithExtension); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestAnnotateModel_HasDocLines(t *testing.T) {
	modelWithDesc := api.NewTestAPI([]*api.Message{}, []*api.Enum{}, []*api.Service{})
	modelWithDesc.PackageName = "test"
	modelWithDesc.Description = "Has a description"

	modelWithoutDesc := api.NewTestAPI([]*api.Message{}, []*api.Enum{}, []*api.Service{})
	modelWithoutDesc.PackageName = "test"
	modelWithoutDesc.Description = ""

	options := maps.Clone(requiredConfig)

	annotate1 := newAnnotateModel(modelWithDesc)
	if err := annotate1.annotateModel(options); err != nil {
		t.Fatal(err)
	}
	codec1 := modelWithDesc.Codec.(*modelAnnotations)
	if !codec1.HasDocLines() {
		t.Errorf("Expected HasDocLines() to be true when description is provided")
	}

	annotate2 := newAnnotateModel(modelWithoutDesc)
	if err := annotate2.annotateModel(options); err != nil {
		t.Fatal(err)
	}
	codec2 := modelWithoutDesc.Codec.(*modelAnnotations)
	if codec2.HasDocLines() {
		t.Errorf("Expected HasDocLines() to be false when description is empty")
	}
}

func TestAnnotateModel_FakeList(t *testing.T) {
	service1 := api.NewTestService("SecretManagerService").WithPackage("google.cloud.secretmanager")
	service2 := api.NewTestService("AccessApprovalService").WithPackage("google.cloud.accessapproval")
	model := api.NewTestAPI([]*api.Message{}, []*api.Enum{}, []*api.Service{service1, service2})
	model.PackageName = "test"

	options := maps.Clone(requiredConfig)

	annotate := newAnnotateModel(model)
	err := annotate.annotateModel(options)
	if err != nil {
		t.Fatal(err)
	}

	codec := model.Codec.(*modelAnnotations)

	want := "FakeAccessApprovalService, FakeSecretManagerService"
	if diff := cmp.Diff(want, codec.FakeList); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestAnnotateModel_Options(t *testing.T) {
	model := api.NewTestAPI([]*api.Message{}, []*api.Enum{}, []*api.Service{})

	var tests = []struct {
		options map[string]string
		verify  func(*testing.T, *annotateModel)
	}{
		{
			map[string]string{"library-path-override": "src/buffers.dart"},
			func(t *testing.T, am *annotateModel) {
				codec := model.Codec.(*modelAnnotations)
				if diff := cmp.Diff("src/buffers.dart", codec.MainFileNameWithExtension); diff != "" {
					t.Errorf("mismatch (-want +got):\n%s", diff)
				}
			},
		},
		{
			map[string]string{"package-name-override": "google-cloud-type"},
			func(t *testing.T, am *annotateModel) {
				codec := model.Codec.(*modelAnnotations)
				if diff := cmp.Diff("google-cloud-type", codec.PackageName); diff != "" {
					t.Errorf("mismatch (-want +got):\n%s", diff)
				}
			},
		},
		{
			map[string]string{"dev-dependencies": "test,mockito"},
			func(t *testing.T, am *annotateModel) {
				codec := model.Codec.(*modelAnnotations)
				if diff := cmp.Diff([]string{"mockito", "test"}, codec.DevDependencies); diff != "" {
					t.Errorf("mismatch (-want +got):\n%s", diff)
				}
			},
		},
		{
			map[string]string{
				"dependencies":             "google_cloud_foo, google_cloud_bar",
				"package:google_cloud_bar": "^1.2.3",
				"package:google_cloud_foo": "^4.5.6"},
			func(t *testing.T, am *annotateModel) {
				codec := model.Codec.(*modelAnnotations)
				if !slices.Contains(codec.PackageDependencies, packageDependency{Name: "google_cloud_foo", Constraint: "^4.5.6"}) {
					t.Errorf("missing 'google_cloud_foo' in Codec.PackageDependencies, got %v", codec.PackageDependencies)
				}
				if !slices.Contains(codec.PackageDependencies, packageDependency{Name: "google_cloud_bar", Constraint: "^1.2.3"}) {
					t.Errorf("missing 'google_cloud_bar' in Codec.PackageDependencies, got %v", codec.PackageDependencies)
				}
			},
		},
		{
			map[string]string{"extra-exports": "export 'package:google_cloud_gax/gax.dart' show Any; export 'package:google_cloud_gax/gax.dart' show Status;"},
			func(t *testing.T, am *annotateModel) {
				codec := model.Codec.(*modelAnnotations)
				if diff := cmp.Diff([]string{
					"export 'package:google_cloud_gax/gax.dart' show Any",
					"export 'package:google_cloud_gax/gax.dart' show Status"}, codec.Exports); diff != "" {
					t.Errorf("mismatch (-want +got):\n%s", diff)
				}
			},
		},
		{
			map[string]string{"extra-imports": "dart:math; package:my_package/my_file.dart", "package:my_package": "^1.0.0"},
			func(t *testing.T, am *annotateModel) {
				codec := model.Codec.(*modelAnnotations)
				if !slices.Contains(codec.Imports, "import 'dart:math';") {
					t.Errorf("missing 'dart:math' in Codec.Imports, got %v", codec.Imports)
				}
				if !slices.Contains(codec.Imports, "import 'package:my_package/my_file.dart';") {
					t.Errorf("missing 'package:my_package/my_file.dart' in Codec.Imports, got %v", codec.Imports)
				}
			},
		},
		{
			map[string]string{"version": "1.2.3"},
			func(t *testing.T, am *annotateModel) {
				codec := model.Codec.(*modelAnnotations)
				if diff := cmp.Diff("1.2.3", codec.PackageVersion); diff != "" {
					t.Errorf("mismatch (-want +got):\n%s", diff)
				}
			},
		},
		{
			map[string]string{"part-file": "src/test.p.dart"},
			func(t *testing.T, am *annotateModel) {
				codec := model.Codec.(*modelAnnotations)
				if diff := cmp.Diff("src/test.p.dart", codec.PartFileReference); diff != "" {
					t.Errorf("mismatch (-want +got):\n%s", diff)
				}
			},
		},
		{
			map[string]string{"readme-after-title-text": "> [!TIP] Still beta!"},
			func(t *testing.T, am *annotateModel) {
				codec := model.Codec.(*modelAnnotations)
				if diff := cmp.Diff("> [!TIP] Still beta!", codec.ReadMeAfterTitleText); diff != "" {
					t.Errorf("mismatch (-want +got):\n%s", diff)
				}
			},
		},
		{
			map[string]string{"readme-quickstart-text": "## Getting Started\n..."},
			func(t *testing.T, am *annotateModel) {
				codec := model.Codec.(*modelAnnotations)
				if diff := cmp.Diff("## Getting Started\n...", codec.ReadMeQuickstartText); diff != "" {
					t.Errorf("mismatch (-want +got):\n%s", diff)
				}
			},
		},
		{
			map[string]string{"repository-url": "http://example.com/repo"},
			func(t *testing.T, am *annotateModel) {
				codec := model.Codec.(*modelAnnotations)
				if diff := cmp.Diff("http://example.com/repo", codec.RepositoryURL); diff != "" {
					t.Errorf("mismatch (-want +got):\n%s", diff)
				}
			},
		},
		{
			map[string]string{"issue-tracker-url": "http://example.com/issues"},
			func(t *testing.T, am *annotateModel) {
				codec := model.Codec.(*modelAnnotations)
				if diff := cmp.Diff("http://example.com/issues", codec.IssueTrackerURL); diff != "" {
					t.Errorf("mismatch (-want +got):\n%s", diff)
				}
			},
		},
		{
			map[string]string{"google_cloud_rpc": "^1.2.3", "package:http": "1.2.0"},
			func(t *testing.T, am *annotateModel) {
				if diff := cmp.Diff(map[string]string{
					"google_cloud_rpc":      "^1.2.3",
					"google_cloud_protobuf": "^7.8.9",
					"http":                  "1.2.0"},
					am.dependencyConstraints); diff != "" {
					t.Errorf("mismatch (-want +got):\n%s", diff)
				}
			},
		},
	}

	for _, test := range tests {
		annotate := newAnnotateModel(model)
		options := maps.Clone(requiredConfig)
		maps.Copy(options, test.options)
		err := annotate.annotateModel(maps.Clone(options))
		if err != nil {
			t.Fatal(err)
		}
		test.verify(t, annotate)
	}
}

func TestAnnotateModel_Options_MissingRequired(t *testing.T) {
	method := sample.MethodListSecretVersions()
	service := api.NewTestService(sample.ServiceName).
		WithPackage(sample.Package).
		WithMethods(method)
	service.Documentation = sample.APIDescription
	service.DefaultHost = sample.DefaultHost
	model := api.NewTestAPI(
		[]*api.Message{sample.ListSecretVersionsRequest(), sample.ListSecretVersionsResponse(),
			sample.Secret(), sample.SecretVersion(), sample.Replication(), sample.Automatic(),
			sample.CustomerManagedEncryption()},
		[]*api.Enum{sample.EnumState()},
		[]*api.Service{service},
	)

	var tests = []string{
		"api-keys-environment-variables",
		"issue-tracker-url",
	}

	for _, test := range tests {
		annotate := newAnnotateModel(model)
		options := maps.Clone(requiredConfig)
		delete(options, test)

		err := annotate.annotateModel(options)
		if err == nil {
			t.Fatalf("expected error when missing %q", test)
		}
	}
}

func TestAnnotateModel_HasMethods(t *testing.T) {
	method := sample.MethodListSecretVersions()
	serviceWithMethods := api.NewTestService("ServiceWithMethods").
		WithPackage(sample.Package).
		WithMethods(method)
	serviceWithoutMethods := api.NewTestService("ServiceWithoutMethods").
		WithPackage(sample.Package)
	model := api.NewTestAPI(
		[]*api.Message{sample.ListSecretVersionsRequest(), sample.ListSecretVersionsResponse(),
			sample.Secret(), sample.SecretVersion(), sample.Replication(), sample.Automatic(),
			sample.CustomerManagedEncryption()},
		[]*api.Enum{sample.EnumState()},
		[]*api.Service{serviceWithMethods, serviceWithoutMethods},
	)
	api.Validate(model)
	annotate := newAnnotateModel(model)
	err := annotate.annotateModel(requiredConfig)
	if err != nil {
		t.Fatal(err)
	}

	codec1 := serviceWithMethods.Codec.(*serviceAnnotations)
	if !codec1.HasMethods {
		t.Errorf("Expected HasMethods to be true for ServiceWithMethods")
	}

	codec2 := serviceWithoutMethods.Codec.(*serviceAnnotations)
	if codec2.HasMethods {
		t.Errorf("Expected HasMethods to be false for ServiceWithoutMethods")
	}
}

func TestAnnotateModel_Examples_ValidMethod(t *testing.T) {
	method := sample.MethodListSecretVersions()
	method.IsSimple = true
	serviceWithMethods := api.NewTestService("ServiceWithMethods").
		WithPackage(sample.Package).
		WithMethods(method)
	model := api.NewTestAPI(
		[]*api.Message{sample.ListSecretVersionsRequest(), sample.ListSecretVersionsResponse(),
			sample.Secret(), sample.SecretVersion(), sample.Replication(), sample.Automatic(),
			sample.CustomerManagedEncryption()},
		[]*api.Enum{sample.EnumState()},
		[]*api.Service{serviceWithMethods},
	)
	api.Validate(model)
	annotate := newAnnotateModel(model)
	err := annotate.annotateModel(requiredConfig)
	if err != nil {
		t.Fatal(err)
	}

	codec := model.Codec.(*modelAnnotations)

	if diff := cmp.Diff("ServiceWithMethods", codec.ExampleServiceName); diff != "" {
		t.Errorf("mismatch ExampleServiceName (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff("listSecretVersions", codec.ExampleMethodName); diff != "" {
		t.Errorf("mismatch ExampleMethodName (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff("ListSecretVersionRequest", codec.ExampleMethodRequestType); diff != "" {
		t.Errorf("mismatch ExampleMethodRequestType (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff("ListSecretVersionsResponse", codec.ExampleMethodResponseType); diff != "" {
		t.Errorf("mismatch ExampleMethodResponseType (-want +got):\n%s", diff)
	}
	if !codec.ExampleMethodReturnsValue {
		t.Errorf("expected ExampleMethodReturnsValue to be true")
	}
}

func TestAnnotateModel_Examples_NoMethod(t *testing.T) {
	serviceWithoutMethods := api.NewTestService("ServiceWithMethods").
		WithPackage(sample.Package)
	model := api.NewTestAPI(
		[]*api.Message{},
		[]*api.Enum{},
		[]*api.Service{serviceWithoutMethods},
	)
	api.Validate(model)
	annotate := newAnnotateModel(model)
	err := annotate.annotateModel(requiredConfig)
	if err != nil {
		t.Fatal(err)
	}

	codec := model.Codec.(*modelAnnotations)

	if diff := cmp.Diff("", codec.ExampleServiceName); diff != "" {
		t.Errorf("mismatch ExampleServiceName (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff("", codec.ExampleMethodName); diff != "" {
		t.Errorf("mismatch ExampleMethodName (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff("", codec.ExampleMethodRequestType); diff != "" {
		t.Errorf("mismatch ExampleMethodRequestType (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff("", codec.ExampleMethodResponseType); diff != "" {
		t.Errorf("mismatch ExampleMethodResponseType (-want +got):\n%s", diff)
	}
	if codec.ExampleMethodReturnsValue {
		t.Errorf("expected ExampleMethodReturnsValue to be false")
	}
}

func TestAnnotateMethod(t *testing.T) {
	method := sample.MethodListSecretVersions()
	service := api.NewTestService(sample.ServiceName).
		WithPackage(sample.Package).
		WithMethods(method)
	service.Documentation = sample.APIDescription
	service.DefaultHost = sample.DefaultHost
	model := api.NewTestAPI(
		[]*api.Message{sample.ListSecretVersionsRequest(), sample.ListSecretVersionsResponse(),
			sample.Secret(), sample.SecretVersion(), sample.Replication(), sample.Automatic(),
			sample.CustomerManagedEncryption()},
		[]*api.Enum{sample.EnumState()},
		[]*api.Service{service},
	)
	api.Validate(model)
	annotate := newAnnotateModel(model)
	err := annotate.annotateModel(requiredConfig)
	if err != nil {
		t.Fatal(err)
	}

	annotate.annotateMethod(method)
	codec := method.Codec.(*methodAnnotation)

	got := codec.Name
	want := "listSecretVersions"
	if got != want {
		t.Errorf("mismatched name, got=%q, want=%q", got, want)
	}

	got = codec.RequestType
	want = "ListSecretVersionRequest"
	if got != want {
		t.Errorf("mismatched type, got=%q, want=%q", got, want)
	}

	got = codec.ResponseType
	want = "ListSecretVersionsResponse"
	if got != want {
		t.Errorf("mismatched type, got=%q, want=%q", got, want)
	}
}

func TestAnnotateMethod_IsLast(t *testing.T) {
	notLastMethod := sample.MethodListSecretVersions()
	lastMethod := sample.MethodListSecretVersions()
	lastMethod.Name = "ListSecretVersions2"
	lastMethod.ID = notLastMethod.ID + "2"

	service := api.NewTestService(sample.ServiceName).
		WithPackage(sample.Package).
		WithMethods(notLastMethod, lastMethod)
	service.Documentation = sample.APIDescription
	service.DefaultHost = sample.DefaultHost
	model := api.NewTestAPI(
		[]*api.Message{sample.ListSecretVersionsRequest(), sample.ListSecretVersionsResponse(),
			sample.Secret(), sample.SecretVersion(), sample.Replication(), sample.Automatic(),
			sample.CustomerManagedEncryption()},
		[]*api.Enum{sample.EnumState()},
		[]*api.Service{service},
	)
	api.Validate(model)
	annotate := newAnnotateModel(model)
	err := annotate.annotateModel(requiredConfig)
	if err != nil {
		t.Fatal(err)
	}

	codec1 := notLastMethod.Codec.(*methodAnnotation)
	if codec1.IsLast {
		t.Errorf("Expected IsLast to be false for method1")
	}

	codec2 := lastMethod.Codec.(*methodAnnotation)
	if !codec2.IsLast {
		t.Errorf("Expected IsLast to be true for method2")
	}
}

func TestCalculatePubPackages(t *testing.T) {
	for _, test := range []struct {
		imports map[string]bool
		want    map[string]bool
	}{
		{imports: map[string]bool{"dart:typed_data": true},
			want: map[string]bool{}},
		{imports: map[string]bool{"dart:typed_data as typed_data": true},
			want: map[string]bool{}},
		{imports: map[string]bool{"package:http/http.dart": true},
			want: map[string]bool{"http": true}},
		{imports: map[string]bool{"package:http/http.dart as http": true},
			want: map[string]bool{"http": true}},
		{imports: map[string]bool{"package:google_cloud_protobuf/src/encoding.dart": true},
			want: map[string]bool{"google_cloud_protobuf": true}},
		{imports: map[string]bool{"package:google_cloud_protobuf/src/encoding.dart as encoding": true},
			want: map[string]bool{"google_cloud_protobuf": true}},
		{imports: map[string]bool{"package:http/http.dart": true, "package:http/http.dart as http": true},
			want: map[string]bool{"http": true}},
		{imports: map[string]bool{
			"package:google_cloud_protobuf/src/encoding.dart": true,
			"package:http/http.dart":                          true,
			"dart:typed_data":                                 true},
			want: map[string]bool{"google_cloud_protobuf": true, "http": true}},
	} { // package:http/http.dart as http
		got := calculatePubPackages(test.imports)

		if !maps.Equal(got, test.want) {
			t.Errorf("calculatePubPackages(%v) = %v, want %v", test.imports, got, test.want)
		}
	}
}

func TestCalculateDependencies(t *testing.T) {
	for _, test := range []struct {
		testName    string
		packages    map[string]bool
		constraints map[string]string
		packageName string
		want        []packageDependency
		wantErr     bool
	}{
		{
			testName:    "empty",
			packages:    map[string]bool{},
			constraints: map[string]string{},
			packageName: "google_cloud_bar",
			want:        []packageDependency{},
		},
		{
			testName:    "self dependency",
			packages:    map[string]bool{"google_cloud_bar": true},
			constraints: map[string]string{},
			packageName: "google_cloud_bar",
			want:        []packageDependency{},
		},
		{
			testName:    "separate dependency",
			packages:    map[string]bool{"google_cloud_foo": true},
			constraints: map[string]string{"google_cloud_foo": "^1.2.3"},
			packageName: "google_cloud_bar",
			want:        []packageDependency{{Name: "google_cloud_foo", Constraint: "^1.2.3"}},
		},
		{
			testName:    "missing constraint",
			packages:    map[string]bool{"google_cloud_foo": true},
			constraints: map[string]string{},
			packageName: "google_cloud_bar",
			wantErr:     true,
		},
		{
			testName:    "multiple dependencies",
			packages:    map[string]bool{"google_cloud_bar": true, "google_cloud_baz": true, "google_cloud_foo": true},
			constraints: map[string]string{"google_cloud_baz": "^1.2.3", "google_cloud_foo": "^4.5.6"},
			packageName: "google_cloud_bar",
			want: []packageDependency{
				{Name: "google_cloud_baz", Constraint: "^1.2.3"},
				{Name: "google_cloud_foo", Constraint: "^4.5.6"}},
		},
	} {
		t.Run(test.testName, func(t *testing.T) {
			got, err := calculateDependencies(test.packages, test.constraints, test.packageName)
			if (err != nil) != test.wantErr {
				t.Errorf("calculateDependencies(%v, %v, %v) error = %v, want error presence = %t",
					test.packages, test.constraints, test.packageName, err, test.wantErr)
			}

			if err != nil {
				return
			}

			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("calculateDependencies(%v, %v, %v) = %v, want %v",
					test.packages, test.constraints, test.packageName, got, test.want)
			}
		})
	}
}

func TestCalculateImports(t *testing.T) {
	for _, test := range []struct {
		name         string
		imports      []string
		packageName  string
		mainFileName string
		want         []string
	}{
		{
			name:         "dart: import",
			imports:      []string{"dart:typed_data"},
			packageName:  "google_cloud_bar",
			mainFileName: "bar.dart",
			want:         []string{"import 'dart:typed_data';"},
		},
		{
			name:         "dart: import with prefix",
			imports:      []string{"dart:typed_data as td"},
			packageName:  "google_cloud_bar",
			mainFileName: "bar.dart",
			want:         []string{"import 'dart:typed_data' as td;"},
		},
		{
			name:         "package: import",
			imports:      []string{"package:http/http.dart"},
			packageName:  "google_cloud_bar",
			mainFileName: "bar.dart",
			want:         []string{"import 'package:http/http.dart';"},
		},
		{
			name:         "package: import with prefix",
			imports:      []string{"package:http/http.dart as http"},
			packageName:  "google_cloud_bar",
			mainFileName: "bar.dart",
			want:         []string{"import 'package:http/http.dart' as http;"},
		},
		{
			name:         "dart: and package: imports",
			imports:      []string{"dart:typed_data", "package:http/http.dart"},
			packageName:  "google_cloud_bar",
			mainFileName: "bar.dart",
			want: []string{
				"import 'dart:typed_data';",
				"",
				"import 'package:http/http.dart';",
			},
		},
		{
			name:         "same file import",
			imports:      []string{"package:google_cloud_bar/bar.dart"},
			packageName:  "google_cloud_bar",
			mainFileName: "bar.dart",
			want:         nil,
		},
		{
			name:         "same file import with prefix",
			imports:      []string{"package:google_cloud_bar/bar.dart as bar"},
			packageName:  "google_cloud_bar",
			mainFileName: "bar.dart",
			want:         nil,
		},
		{
			name:         "same package import",
			imports:      []string{"package:google_cloud_bar/baz.dart"},
			packageName:  "google_cloud_bar",
			mainFileName: "bar.dart",
			want:         []string{"import 'baz.dart';"},
		},
		{
			name:         "same package import, src directory",
			imports:      []string{"package:google_cloud_bar/src/baz.dart"},
			packageName:  "google_cloud_bar",
			mainFileName: "bar.dart",
			want:         []string{"import 'baz.dart';"},
		},
		{
			name:         "same package import with prefix",
			imports:      []string{"package:google_cloud_bar/baz.dart as baz"},
			packageName:  "google_cloud_bar",
			mainFileName: "bar.dart",
			want:         []string{"import 'baz.dart' as baz;"},
		},
		{
			name: "many imports", imports: []string{
				"package:google_cloud_foo/foo.dart",
				"package:google_cloud_bar/bar.dart as bar",
				"package:google_cloud_bar/src/bing.dart",
				"package:google_cloud_bar/src/foo.dart as foo",
				"package:google_cloud_bar/baz.dart",
				"dart:core",
				"dart:io as io",
			},
			packageName:  "google_cloud_bar",
			mainFileName: "bar.dart",
			want: []string{
				"import 'dart:core';",
				"import 'dart:io' as io;",
				"",
				"import 'package:google_cloud_foo/foo.dart';",
				"",
				"import 'baz.dart';",
				"import 'bing.dart';",
				"import 'foo.dart' as foo;",
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			deps := map[string]bool{}
			for _, imp := range test.imports {
				deps[imp] = true
			}
			got := calculateImports(deps, test.packageName, test.mainFileName)

			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestAnnotateMessage_ToString(t *testing.T) {
	model := api.NewTestAPI(
		[]*api.Message{sample.Secret(), sample.SecretVersion(), sample.Replication(),
			sample.Automatic(), sample.CustomerManagedEncryption()},
		[]*api.Enum{sample.EnumState()},
		[]*api.Service{},
	)
	annotate := newAnnotateModel(model)
	annotate.annotateModel(map[string]string{})

	for _, test := range []struct {
		message  *api.Message
		expected int
	}{
		// Expect the number of fields less the number of message fields.
		{message: sample.Secret(), expected: 1},
		{message: sample.SecretVersion(), expected: 2},
		{message: sample.Replication(), expected: 0},
		{message: sample.Automatic(), expected: 0},
	} {
		t.Run(test.message.Name, func(t *testing.T) {
			annotate.annotateMessage(test.message)

			codec := test.message.Codec.(*messageAnnotation)
			actual := codec.ToStringLines

			if len(actual) != test.expected {
				t.Errorf("Expected list of length %d, got %d", test.expected, len(actual))
			}
		})
	}
}

func TestAnnotateMessage_HasFields(t *testing.T) {
	model := api.NewTestAPI(
		[]*api.Message{sample.Secret()},
		[]*api.Enum{},
		[]*api.Service{},
	)
	annotate := newAnnotateModel(model)
	if err := annotate.annotateModel(requiredConfig); err != nil {
		t.Fatal(err)
	}

	emptyMessage := api.NewTestMessage("EmptyMessage").
		WithPackage("google.cloud.foo").
		WithID("google.cloud.foo.EmptyMessage")

	t.Run("has fields", func(t *testing.T) {
		secret := sample.Secret()
		annotate.annotateMessage(secret)
		codec := secret.Codec.(*messageAnnotation)
		if !codec.HasFields() {
			t.Errorf("mismatch got = %v, want true", codec.HasFields())
		}
	})

	t.Run("no fields", func(t *testing.T) {
		annotate.annotateMessage(emptyMessage)
		codec := emptyMessage.Codec.(*messageAnnotation)
		if codec.HasFields() {
			t.Errorf("mismatch got = %v, want false", codec.HasFields())
		}
	})
}

// Tests that messages that are allowlisted as not being generated are, in fact, not generated.
func TestAnnotateMessage_OmitGeneration_Allowlisted(t *testing.T) {
	status := api.NewTestMessage("Status").
		WithPackage("google.rpc").
		WithID(".google.rpc.Status")
	message := api.NewTestMessage("Operation").
		WithPackage("google.longrunning").
		WithID(".google.longrunning.Operation").
		WithFields(
			api.NewTestField("error").
				WithMessageType(status),
		)
	model := api.NewTestAPI([]*api.Message{message, status}, []*api.Enum{}, []*api.Service{})
	annotate := newAnnotateModel(model)
	annotate.annotateMessage(message)

	codec := message.Codec.(*messageAnnotation)
	if !codec.OmitGeneration {
		t.Errorf("Expected OmitGeneration to be true for .google.longrunning.Operation")
	}

	if len(annotate.imports) != 0 {
		// The `error` field is of type `google.rpc.Status`, which would normally require that the
		// `google.rpc` package be imported. However, since the message is not generated, the import
		// should not be added.
		t.Errorf("Expected no imports for .google.longrunning.Operation")
	}
}

// Tests that map messages are not generated but that there key value types generate imports.
func TestAnnotateMessage_OmitGeneration_Map(t *testing.T) {
	status := api.NewTestMessage("Status").
		WithPackage("google.rpc").
		WithID(".google.rpc.Status")
	message := api.NewTestMessage("HasMap").
		WithPackage("some.package").
		WithID(".some.package.HasMap").
		WithFields(
			api.NewTestField("map_field").
				WithType(api.TypezMessage).
				WithTypezID(".some.package.HasMap.MapFieldEntry"),
		)
	mapMessage := api.NewTestMessage("Entry").
		WithPackage("some.package").
		WithID(".some.package.HasMap.MapFieldEntry").
		WithFields(
			api.NewTestField("key").WithType(api.TypezString),
			api.NewTestField("value").WithMessageType(status),
		)
	mapMessage.IsMap = true
	model := api.NewTestAPI([]*api.Message{message}, []*api.Enum{}, []*api.Service{})
	model.AddMessage(status)
	model.AddMessage(mapMessage)
	annotate := newAnnotateModel(model)

	annotate.annotateModel(map[string]string{
		"proto:google.rpc": "package:google_cloud_rpc/google_cloud_rpc.dart",
	})
	annotate.annotateMessage(message)

	codec := message.Codec.(*messageAnnotation)
	if codec.OmitGeneration {
		t.Errorf("Expected OmitGeneration to be true for map entry")
	}

	if !annotate.imports["package:google_cloud_rpc/google_cloud_rpc.dart"] {
		t.Errorf("Expected import for google.rpc")
	}
}

func withJSONName(f *api.Field, jsonName string) *api.Field {
	f.JSONName = jsonName
	return f
}

func withOptional(f *api.Field) *api.Field {
	f.Optional = true
	return f
}

func TestBuildQueryLines_Primitives(t *testing.T) {
	for _, test := range []struct {
		field *api.Field
		want  []string
	}{
		// primitives
		{
			api.NewTestField("bool").WithType(api.TypezBool),
			[]string{"if (result.bool$ case final $1 when $1.isNotDefault) 'bool': '${$1}'"},
		}, {
			api.NewTestField("bytes").WithType(api.TypezBytes),
			[]string{"if (result.bytes case final $1 when $1.isNotDefault) 'bytes': encodeBytes($1)!"},
		}, {
			api.NewTestField("int32").WithType(api.TypezInt32),
			[]string{"if (result.int32 case final $1 when $1.isNotDefault) 'int32': '${$1}'"},
		}, {
			api.NewTestField("fixed32").WithType(api.TypezFixed32),
			[]string{"if (result.fixed32 case final $1 when $1.isNotDefault) 'fixed32': '${$1}'"},
		}, {
			api.NewTestField("sfixed32").WithType(api.TypezSfixed32),
			[]string{"if (result.sfixed32 case final $1 when $1.isNotDefault) 'sfixed32': '${$1}'"},
		}, {
			api.NewTestField("int64").WithType(api.TypezInt64),
			[]string{"if (result.int64 case final $1 when $1.isNotDefault) 'int64': '${$1}'"},
		}, {
			api.NewTestField("fixed64").WithType(api.TypezFixed64),
			[]string{"if (result.fixed64 case final $1 when $1.isNotDefault) 'fixed64': '${$1}'"},
		}, {
			api.NewTestField("sfixed64").WithType(api.TypezSfixed64),
			[]string{"if (result.sfixed64 case final $1 when $1.isNotDefault) 'sfixed64': '${$1}'"},
		}, {
			api.NewTestField("double").WithType(api.TypezDouble),
			[]string{"if (result.double$ case final $1 when $1.isNotDefault) 'double': '${$1}'"},
		}, {
			api.NewTestField("string").WithType(api.TypezString),
			[]string{"if (result.string case final $1 when $1.isNotDefault) 'string': $1"},
		},

		// optional primitives
		{
			withJSONName(api.NewTestField("bool_opt").WithType(api.TypezBool).WithOptional(), "bool"),
			[]string{"if (result.boolOpt case final $1?) 'bool': '${$1}'"},
		}, {
			withJSONName(api.NewTestField("bytes_opt").WithType(api.TypezBytes).WithOptional(), "bytes"),
			[]string{"if (result.bytesOpt case final $1?) 'bytes': encodeBytes($1)!"},
		}, {
			withJSONName(api.NewTestField("int32_opt").WithType(api.TypezInt32).WithOptional(), "int32"),
			[]string{"if (result.int32Opt case final $1?) 'int32': '${$1}'"},
		}, {
			withJSONName(api.NewTestField("fixed32_opt").WithType(api.TypezFixed32).WithOptional(), "fixed32"),
			[]string{"if (result.fixed32Opt case final $1?) 'fixed32': '${$1}'"},
		}, {
			withJSONName(api.NewTestField("sfixed32_opt").WithType(api.TypezSfixed32).WithOptional(), "sfixed32"),
			[]string{"if (result.sfixed32Opt case final $1?) 'sfixed32': '${$1}'"},
		}, {
			withJSONName(api.NewTestField("int64_opt").WithType(api.TypezInt64).WithOptional(), "int64"),
			[]string{"if (result.int64Opt case final $1?) 'int64': '${$1}'"},
		}, {
			withJSONName(api.NewTestField("fixed64_opt").WithType(api.TypezFixed64).WithOptional(), "fixed64"),
			[]string{"if (result.fixed64Opt case final $1?) 'fixed64': '${$1}'"},
		}, {
			withJSONName(api.NewTestField("sfixed64_opt").WithType(api.TypezSfixed64).WithOptional(), "sfixed64"),
			[]string{"if (result.sfixed64Opt case final $1?) 'sfixed64': '${$1}'"},
		}, {
			withJSONName(api.NewTestField("double_opt").WithType(api.TypezDouble).WithOptional(), "double"),
			[]string{"if (result.doubleOpt case final $1?) 'double': '${$1}'"},
		}, {
			withJSONName(api.NewTestField("string_opt").WithType(api.TypezString).WithOptional(), "string"),
			[]string{"'string': ?result.stringOpt"},
		},

		// one ofs
		{
			api.NewTestOneOf("oneof").WithFields(api.NewTestField("bool").WithType(api.TypezBool)).Fields[0],
			[]string{"if (result.bool$ case final $1?) 'bool': '${$1}'"},
		},

		// repeated primitives
		{
			api.NewTestField("boolList").WithType(api.TypezBool).WithRepeated(),
			[]string{"if (result.boolList case final $1 when $1.isNotDefault) 'boolList': $1.map((e) => '$e')"},
		}, {
			api.NewTestField("bytesList").WithType(api.TypezBytes).WithRepeated(),
			[]string{"if (result.bytesList case final $1 when $1.isNotDefault) 'bytesList': $1.map((e) => encodeBytes(e)!)"},
		}, {
			api.NewTestField("int32List").WithType(api.TypezInt32).WithRepeated(),
			[]string{"if (result.int32List case final $1 when $1.isNotDefault) 'int32List': $1.map((e) => '$e')"},
		}, {
			api.NewTestField("int64List").WithType(api.TypezInt64).WithRepeated(),
			[]string{"if (result.int64List case final $1 when $1.isNotDefault) 'int64List': $1.map((e) => '$e')"},
		}, {
			api.NewTestField("doubleList").WithType(api.TypezDouble).WithRepeated(),
			[]string{"if (result.doubleList case final $1 when $1.isNotDefault) 'doubleList': $1.map((e) => '$e')"},
		}, {
			api.NewTestField("stringList").WithType(api.TypezString).WithRepeated(),
			[]string{"if (result.stringList case final $1 when $1.isNotDefault) 'stringList': $1"},
		},

		// repeated primitives w/ optional
		{
			withJSONName(withOptional(api.NewTestField("int32List_opt").WithType(api.TypezInt32).WithRepeated()), "int32List"),
			[]string{"if (result.int32ListOpt case final $1 when $1.isNotDefault) 'int32List': $1.map((e) => '$e')"},
		},
	} {
		t.Run(test.field.Name, func(t *testing.T) {
			message := api.NewTestMessage("UpdateSecretRequest").
				WithPackage(sample.Package).
				WithID("..UpdateRequest").
				WithFields(test.field)
			model := api.NewTestAPI([]*api.Message{message}, []*api.Enum{}, []*api.Service{})
			annotate := newAnnotateModel(model)
			annotate.annotateModel(map[string]string{})

			got := annotate.buildQueryLines([]string{}, "result.", false, "", test.field)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestBuildQueryLines_Enums(t *testing.T) {
	r := sample.Replication()
	a := sample.Automatic()
	enum := sample.EnumState()
	foreignEnumState := api.NewTestEnum("ForeignEnum").
		WithPackage("google.cloud.foo").
		WithID("google.cloud.foo.ForeignEnum").
		WithValues(
			api.NewTestEnumValue("Enabled", 1),
		)

	model := api.NewTestAPI(
		[]*api.Message{r, a, sample.CustomerManagedEncryption()},
		[]*api.Enum{enum, foreignEnumState},
		[]*api.Service{})
	model.PackageName = "test"
	annotate := newAnnotateModel(model)
	annotate.annotateModel(map[string]string{
		"prefix:google.cloud.foo": "foo",
	})
	for _, test := range []struct {
		enumField *api.Field
		want      []string
	}{
		{
			withJSONName(
				api.NewTestField("enumName").
					WithType(api.TypezEnum).
					WithTypezID(enum.ID),
				"jsonEnumName",
			),
			[]string{"if (result.enumName case final $1 when $1.isNotDefault) 'jsonEnumName': $1.value"},
		},
		{
			withJSONName(
				api.NewTestField("optionalEnum").
					WithType(api.TypezEnum).
					WithTypezID(enum.ID).
					WithOptional(),
				"optionalJsonEnum",
			),
			[]string{"'optionalJsonEnum': ?result.optionalEnum?.value"},
		},
		{
			withJSONName(
				api.NewTestField("enumName").
					WithType(api.TypezEnum).
					WithTypezID(foreignEnumState.ID),
				"jsonEnumName",
			),
			[]string{"if (result.enumName case final $1 when $1.isNotDefault) 'jsonEnumName': $1.value"},
		},
	} {
		t.Run(test.enumField.Name, func(t *testing.T) {
			got := annotate.buildQueryLines([]string{}, "result.", false, "", test.enumField)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestBuildQueryLines_Messages(t *testing.T) {
	r := sample.Replication()
	a := sample.Automatic()
	secretVersion := sample.SecretVersion()
	updateRequest := sample.UpdateRequest()
	payload := sample.SecretPayload()
	model := api.NewTestAPI(
		[]*api.Message{r, a, sample.CustomerManagedEncryption(), secretVersion,
			updateRequest, sample.Secret(), payload},
		[]*api.Enum{sample.EnumState()},
		[]*api.Service{})
	model.PackageName = "test"
	annotate := newAnnotateModel(model)
	annotate.annotateModel(map[string]string{})

	messageField1 := api.NewTestField("message1").
		WithMessageType(secretVersion)
	messageField2 := api.NewTestField("message2").
		WithMessageType(payload)
	messageField3 := api.NewTestField("message3").
		WithMessageType(updateRequest)
	fieldMaskField := api.NewTestField("field_mask").
		WithType(api.TypezMessage).
		WithTypezID(".google.protobuf.FieldMask")

	durationField := api.NewTestField("duration").
		WithType(api.TypezMessage).
		WithTypezID(".google.protobuf.Duration")

	timestampField := api.NewTestField("time").
		WithType(api.TypezMessage).
		WithTypezID(".google.protobuf.Timestamp")

	// messages
	got := annotate.buildQueryLines([]string{}, "result.", false, "", messageField1)
	want := []string{
		"if (result.message1?.name case final $1? when $1.isNotDefault) 'message1.name': $1",
		"if (result.message1?.state case final $1? when $1.isNotDefault) 'message1.state': $1.value",
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}

	got = annotate.buildQueryLines([]string{}, "result.", false, "", messageField2)
	want = []string{
		"if (result.message2?.data case final $1?) 'message2.data': encodeBytes($1)!",
		"if (result.message2?.dataCrc32C case final $1?) 'message2.dataCrc32c': '${$1}'",
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}

	// nested messages
	got = annotate.buildQueryLines([]string{}, "result.", false, "", messageField3)
	want = []string{
		"if (result.message3?.secret?.name case final $1? when $1.isNotDefault) 'message3.secret.name': $1",
		"if (result.message3?.fieldMask case final $1?) 'message3.fieldMask': $1.toJson()",
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}

	// custom encoded messages
	got = annotate.buildQueryLines([]string{}, "result.", false, "", fieldMaskField)
	want = []string{
		"if (result.fieldMask case final $1?) 'fieldMask': $1.toJson()",
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}

	got = annotate.buildQueryLines([]string{}, "result.", false, "", durationField)
	want = []string{
		"if (result.duration case final $1?) 'duration': $1.toJson()",
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}

	got = annotate.buildQueryLines([]string{}, "result.", false, "", timestampField)
	want = []string{
		"if (result.time case final $1?) 'time': $1.toJson()",
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestCreateFromJsonLine(t *testing.T) {
	secret := sample.Secret()
	enumState := sample.EnumState()

	foreignMessage := api.NewTestMessage("Foo").
		WithPackage("google.cloud.foo").
		WithID("google.cloud.foo.Foo")
	foreignEnumState := api.NewTestEnum("ForeignEnum").
		WithPackage("google.cloud.foo").
		WithID("google.cloud.foo.ForeignEnum").
		WithValues(
			api.NewTestEnumValue("Enabled", 1),
		)
	mapStringToBytes := api.NewTestMessage("$StringToBytes").
		WithID("..$StringToBytes").
		WithFields(
			api.NewTestField("key").WithType(api.TypezString),
			api.NewTestField("value").WithType(api.TypezBytes),
		)
	mapStringToBytes.IsMap = true
	mapInt32ToBytes := api.NewTestMessage("$Int32ToBytes").
		WithID("..$Int32ToBytes").
		WithFields(
			api.NewTestField("key").WithType(api.TypezInt32),
			api.NewTestField("value").WithType(api.TypezBytes),
		)
	mapInt32ToBytes.IsMap = true
	nullValueEnum := api.NewTestEnum("NullValue").
		WithPackage("google.protobuf").
		WithID(".google.protobuf.NullValue").
		WithValues(
			api.NewTestEnumValue("NULL_VALUE", 0),
		)
	valueMessage := api.NewTestMessage("Value").
		WithPackage("google.protobuf").
		WithID(".google.protobuf.Value")

	for _, test := range []struct {
		field *api.Field
		want  string
	}{
		// primitives
		{
			api.NewTestField("bool").WithType(api.TypezBool),
			"switch (json['bool']) { null => false, Object $1 => decodeBool($1)}",
		}, {
			api.NewTestField("bytes").WithType(api.TypezBytes),
			"switch (json['bytes']) { null => Uint8List(0), Object $1 => decodeBytes($1)}",
		}, {
			api.NewTestField("double").WithType(api.TypezDouble),
			"switch (json['double']) { null => 0, Object $1 => decodeDouble($1)}",
		}, {
			api.NewTestField("fixed32").WithType(api.TypezFixed32),
			"switch (json['fixed32']) { null => 0, Object $1 => decodeInt($1)}",
		}, {
			api.NewTestField("fixed64").WithType(api.TypezFixed64),
			"switch (json['fixed64']) { null => BigInt.zero, Object $1 => decodeUint64($1)}",
		}, {
			api.NewTestField("float").WithType(api.TypezFloat),
			"switch (json['float']) { null => 0, Object $1 => decodeDouble($1)}",
		}, {
			api.NewTestField("int32").WithType(api.TypezInt32),
			"switch (json['int32']) { null => 0, Object $1 => decodeInt($1)}",
		}, {
			api.NewTestField("int64").WithType(api.TypezInt64),
			"switch (json['int64']) { null => 0, Object $1 => decodeInt64($1)}",
		}, {
			api.NewTestField("sfixed32").WithType(api.TypezSfixed32),
			"switch (json['sfixed32']) { null => 0, Object $1 => decodeInt($1)}",
		}, {
			api.NewTestField("sfixed64").WithType(api.TypezSfixed64),
			"switch (json['sfixed64']) { null => 0, Object $1 => decodeInt64($1)}",
		}, {
			api.NewTestField("sint64").WithType(api.TypezSint64),
			"switch (json['sint64']) { null => 0, Object $1 => decodeInt64($1)}",
		}, {
			api.NewTestField("string").WithType(api.TypezString),
			"switch (json['string']) { null => '', Object $1 => decodeString($1)}",
		}, {
			api.NewTestField("uint32").WithType(api.TypezUint32),
			"switch (json['uint32']) { null => 0, Object $1 => decodeInt($1)}",
		}, {
			api.NewTestField("uint64").WithType(api.TypezUint64),
			"switch (json['uint64']) { null => BigInt.zero, Object $1 => decodeUint64($1)}",
		},

		// optional primitives
		{
			withJSONName(api.NewTestField("bool_opt").WithType(api.TypezBool).WithOptional(), "bool"),
			"switch (json['bool']) { null => null, Object $1 => decodeBool($1)}",
		}, {
			withJSONName(api.NewTestField("bytes_opt").WithType(api.TypezBytes).WithOptional(), "bytes"),
			"switch (json['bytes']) { null => null, Object $1 => decodeBytes($1)}",
		}, {
			withJSONName(api.NewTestField("double_opt").WithType(api.TypezDouble).WithOptional(), "double"),
			"switch (json['double']) { null => null, Object $1 => decodeDouble($1)}",
		}, {
			withJSONName(api.NewTestField("fixed64_opt").WithType(api.TypezFixed64).WithOptional(), "fixed64"),
			"switch (json['fixed64']) { null => null, Object $1 => decodeUint64($1)}",
		}, {
			withJSONName(api.NewTestField("float_opt").WithType(api.TypezFloat).WithOptional(), "float"),
			"switch (json['float']) { null => null, Object $1 => decodeDouble($1)}",
		}, {
			withJSONName(api.NewTestField("int32_opt").WithType(api.TypezInt32).WithOptional(), "int32"),
			"switch (json['int32']) { null => null, Object $1 => decodeInt($1)}",
		}, {
			withJSONName(api.NewTestField("int64_opt").WithType(api.TypezInt64).WithOptional(), "int64"),
			"switch (json['int64']) { null => null, Object $1 => decodeInt64($1)}",
		}, {
			withJSONName(api.NewTestField("sfixed32_opt").WithType(api.TypezSfixed32).WithOptional(), "sfixed32"),
			"switch (json['sfixed32']) { null => null, Object $1 => decodeInt($1)}",
		}, {
			withJSONName(api.NewTestField("sfixed64_opt").WithType(api.TypezSfixed64).WithOptional(), "sfixed64"),
			"switch (json['sfixed64']) { null => null, Object $1 => decodeInt64($1)}",
		}, {
			withJSONName(api.NewTestField("sint64_opt").WithType(api.TypezSint64).WithOptional(), "sint64"),
			"switch (json['sint64']) { null => null, Object $1 => decodeInt64($1)}",
		}, {
			withJSONName(api.NewTestField("string_opt").WithType(api.TypezString).WithOptional(), "string"),
			"switch (json['string']) { null => null, Object $1 => decodeString($1)}",
		}, {
			withJSONName(api.NewTestField("uint32_opt").WithType(api.TypezUint32).WithOptional(), "uint32"),
			"switch (json['uint32']) { null => null, Object $1 => decodeInt($1)}",
		}, {
			withJSONName(api.NewTestField("uint64_opt").WithType(api.TypezUint64).WithOptional(), "uint64"),
			"switch (json['uint64']) { null => null, Object $1 => decodeUint64($1)}",
		},

		// one ofs
		{
			api.NewTestOneOf("oneof").WithFields(api.NewTestField("bool").WithType(api.TypezBool)).Fields[0],
			"switch (json['bool']) { null => null, Object $1 => decodeBool($1)}",
		},

		// repeated primitives
		{
			api.NewTestField("boolList").WithType(api.TypezBool).WithRepeated(),
			"switch (json['boolList']) { null => [], List<Object?> $1 => [for (final i in $1) decodeBool(i)], _ => throw const FormatException('\"boolList\" is not a list') }",
		}, {
			api.NewTestField("bytesList").WithType(api.TypezBytes).WithRepeated(),
			"switch (json['bytesList']) { null => [], List<Object?> $1 => [for (final i in $1) decodeBytes(i)], _ => throw const FormatException('\"bytesList\" is not a list') }",
		}, {
			api.NewTestField("doubleList").WithType(api.TypezDouble).WithRepeated(),
			"switch (json['doubleList']) { null => [], List<Object?> $1 => [for (final i in $1) decodeDouble(i)], _ => throw const FormatException('\"doubleList\" is not a list') }",
		}, {
			api.NewTestField("fixed32List").WithType(api.TypezFixed32).WithRepeated(),
			"switch (json['fixed32List']) { null => [], List<Object?> $1 => [for (final i in $1) decodeInt(i)], _ => throw const FormatException('\"fixed32List\" is not a list') }",
		}, {
			api.NewTestField("int32List").WithType(api.TypezInt32).WithRepeated(),
			"switch (json['int32List']) { null => [], List<Object?> $1 => [for (final i in $1) decodeInt(i)], _ => throw const FormatException('\"int32List\" is not a list') }",
		}, {
			api.NewTestField("stringList").WithType(api.TypezString).WithRepeated(),
			"switch (json['stringList']) { null => [], List<Object?> $1 => [for (final i in $1) decodeString(i)], _ => throw const FormatException('\"stringList\" is not a list') }",
		},

		// repeated primitives w/ optional
		{
			withJSONName(withOptional(api.NewTestField("int32List_opt").WithType(api.TypezInt32).WithRepeated()), "int32List"),
			"switch (json['int32List']) { null => [], List<Object?> $1 => [for (final i in $1) decodeInt(i)], _ => throw const FormatException('\"int32List\" is not a list') }",
		},

		// enums
		{
			api.NewTestField("message").WithType(api.TypezEnum).WithTypezID(enumState.ID),
			"switch (json['message']) { null => State.$default, Object $1 => State.fromJson($1)}",
		},
		{
			api.NewTestField("message").WithType(api.TypezEnum).WithTypezID(foreignEnumState.ID),
			"switch (json['message']) { null => foo.ForeignEnum.$default, Object $1 => foo.ForeignEnum.fromJson($1)}",
		},

		// messages
		{
			api.NewTestField("message").WithMessageType(secret),
			"switch (json['message']) { null => null, Object $1 => Secret.fromJson($1)}",
		},
		{
			api.NewTestField("message").WithMessageType(foreignMessage),
			"switch (json['message']) { null => null, Object $1 => foo.Foo.fromJson($1)}",
		},
		{
			// Custom encoding.
			api.NewTestField("message").WithType(api.TypezMessage).WithTypezID(".google.protobuf.Duration"),
			"switch (json['message']) { null => null, Object $1 => Duration.fromJson($1)}",
		},
		// canBeNull exceptions
		{
			api.NewTestField("nullValue").WithType(api.TypezEnum).WithTypezID(".google.protobuf.NullValue"),
			"switch ((json.containsKey('nullValue'), json['nullValue'])) {(false,_) => NullValue.$default, (true, Object? $1) => NullValue.fromJson($1)}",
		},
		{
			api.NewTestField("value").WithType(api.TypezMessage).WithTypezID(".google.protobuf.Value"),
			"switch ((json.containsKey('value'), json['value'])) {(false,_) => null, (true, Object? $1) => Value.fromJson($1)}",
		},

		// maps
		{
			// string -> bytes
			api.NewTestField("message").WithMap().WithMessageType(mapStringToBytes),
			"switch (json['message']) { null => {}, Map<String, Object?> $1 => {for (final e in $1.entries) decodeString(e.key): decodeBytes(e.value)}, _ => throw const FormatException('\"message\" is not an object') }",
		},
		{
			// int32 -> bytes
			api.NewTestField("message").WithMap().WithMessageType(mapInt32ToBytes),
			"switch (json['message']) { null => {}, Map<String, Object?> $1 => {for (final e in $1.entries) decodeIntKey(e.key): decodeBytes(e.value)}, _ => throw const FormatException('\"message\" is not an object') }",
		},
	} {
		t.Run(test.field.Name, func(t *testing.T) {
			message := api.NewTestMessage("UpdateSecretRequest").
				WithPackage(sample.Package).
				WithID("..UpdateRequest").
				WithFields(test.field)
			model := api.NewTestAPI([]*api.Message{message,
				secret, foreignMessage, mapStringToBytes, mapInt32ToBytes, valueMessage},
				[]*api.Enum{enumState, foreignEnumState, nullValueEnum},
				[]*api.Service{})
			annotate := newAnnotateModel(model)
			annotate.annotateModel(map[string]string{
				"prefix:google.cloud.foo": "foo",
			})
			codec := test.field.Codec.(*fieldAnnotation)

			got := annotate.createFromJsonLine(test.field, codec.Required)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestToJson(t *testing.T) {
	secret := sample.Secret()
	enum := sample.EnumState()

	foreignMessage := api.NewTestMessage("Foo").
		WithPackage("google.cloud.foo").
		WithID("google.cloud.foo.Foo")
	foreignEnumState := api.NewTestEnum("ForeignEnum").
		WithPackage("google.cloud.foo").
		WithID("google.cloud.foo.ForeignEnum").
		WithValues(
			api.NewTestEnumValue("Enabled", 1),
		)

	mapStringToString := api.NewTestMessage("$StringToString").
		WithID("..$StringToString").
		WithFields(
			api.NewTestField("key").WithType(api.TypezString),
			api.NewTestField("value").WithType(api.TypezString),
		)
	mapStringToString.IsMap = true
	mapInt32ToString := api.NewTestMessage("$Int32ToString").
		WithID("..$Int32ToString").
		WithFields(
			api.NewTestField("key").WithType(api.TypezInt32),
			api.NewTestField("value").WithType(api.TypezString),
		)
	mapInt32ToString.IsMap = true
	mapBoolToString := api.NewTestMessage("$BoolToString").
		WithID("..$BoolToString").
		WithFields(
			api.NewTestField("key").WithType(api.TypezBool),
			api.NewTestField("value").WithType(api.TypezString),
		)
	mapBoolToString.IsMap = true
	mapStringToInt64 := api.NewTestMessage("$StringToInt64").
		WithID("..$StringToInt64").
		WithFields(
			api.NewTestField("key").WithType(api.TypezString),
			api.NewTestField("value").WithType(api.TypezInt64),
		)
	mapStringToInt64.IsMap = true
	mapInt64ToString := api.NewTestMessage("$Int64ToString").
		WithID("..$Int64ToString").
		WithFields(
			api.NewTestField("key").WithType(api.TypezInt64),
			api.NewTestField("value").WithType(api.TypezString),
		)
	mapInt64ToString.IsMap = true
	mapUint32ToString := api.NewTestMessage("$Uint32ToString").
		WithID("..$Uint32ToString").
		WithFields(
			api.NewTestField("key").WithType(api.TypezUint32),
			api.NewTestField("value").WithType(api.TypezString),
		)
	mapUint32ToString.IsMap = true
	mapUint64ToString := api.NewTestMessage("$Uint64ToString").
		WithID("..$Uint64ToString").
		WithFields(
			api.NewTestField("key").WithType(api.TypezUint64),
			api.NewTestField("value").WithType(api.TypezString),
		)
	mapUint64ToString.IsMap = true
	mapSint32ToString := api.NewTestMessage("$Sint32ToString").
		WithID("..$Sint32ToString").
		WithFields(
			api.NewTestField("key").WithType(api.TypezSint32),
			api.NewTestField("value").WithType(api.TypezString),
		)
	mapSint32ToString.IsMap = true
	mapSint64ToString := api.NewTestMessage("$Sint64ToString").
		WithID("..$Sint64ToString").
		WithFields(
			api.NewTestField("key").WithType(api.TypezSint64),
			api.NewTestField("value").WithType(api.TypezString),
		)
	mapSint64ToString.IsMap = true
	mapFixed32ToString := api.NewTestMessage("$Fixed32ToString").
		WithID("..$Fixed32ToString").
		WithFields(
			api.NewTestField("key").WithType(api.TypezFixed32),
			api.NewTestField("value").WithType(api.TypezString),
		)
	mapFixed32ToString.IsMap = true
	mapFixed64ToString := api.NewTestMessage("$Fixed64ToString").
		WithID("..$Fixed64ToString").
		WithFields(
			api.NewTestField("key").WithType(api.TypezFixed64),
			api.NewTestField("value").WithType(api.TypezString),
		)
	mapFixed64ToString.IsMap = true
	mapSfixed32ToString := api.NewTestMessage("$Sfixed32ToString").
		WithID("..$Sfixed32ToString").
		WithFields(
			api.NewTestField("key").WithType(api.TypezSfixed32),
			api.NewTestField("value").WithType(api.TypezString),
		)
	mapSfixed32ToString.IsMap = true
	mapSfixed64ToString := api.NewTestMessage("$Sfixed64ToString").
		WithID("..$Sfixed64ToString").
		WithFields(
			api.NewTestField("key").WithType(api.TypezSfixed64),
			api.NewTestField("value").WithType(api.TypezString),
		)
	mapSfixed64ToString.IsMap = true

	for _, test := range []struct {
		field *api.Field
		want  string
	}{
		// primitives
		{
			api.NewTestField("bool").WithType(api.TypezBool),
			"if (bool$.isNotDefault) 'bool': bool$",
		}, {
			api.NewTestField("bytes").WithType(api.TypezBytes),
			"if (bytes.isNotDefault) 'bytes': encodeBytes(bytes)",
		}, {
			api.NewTestField("double").WithType(api.TypezDouble),
			"if (double$.isNotDefault) 'double': encodeDouble(double$)",
		}, {
			api.NewTestField("fixed32").WithType(api.TypezFixed32),
			"if (fixed32.isNotDefault) 'fixed32': fixed32",
		}, {
			api.NewTestField("fixed64").WithType(api.TypezFixed64),
			"if (fixed64.isNotDefault) 'fixed64': fixed64.toString()",
		}, {
			api.NewTestField("float").WithType(api.TypezFloat),
			"if (float.isNotDefault) 'float': encodeDouble(float)",
		}, {
			api.NewTestField("int32").WithType(api.TypezInt32),
			"if (int32.isNotDefault) 'int32': int32",
		}, {
			api.NewTestField("int64").WithType(api.TypezInt64),
			"if (int64.isNotDefault) 'int64': int64.toString()",
		}, {
			api.NewTestField("sfixed32").WithType(api.TypezSfixed32),
			"if (sfixed32.isNotDefault) 'sfixed32': sfixed32",
		}, {
			api.NewTestField("sfixed64").WithType(api.TypezSfixed64),
			"if (sfixed64.isNotDefault) 'sfixed64': sfixed64.toString()",
		}, {
			api.NewTestField("sint32").WithType(api.TypezSint32),
			"if (sint32.isNotDefault) 'sint32': sint32",
		}, {
			api.NewTestField("sint64").WithType(api.TypezSint64),
			"if (sint64.isNotDefault) 'sint64': sint64.toString()",
		}, {
			api.NewTestField("string").WithType(api.TypezString),
			"if (string.isNotDefault) 'string': string",
		}, {
			api.NewTestField("uint32").WithType(api.TypezUint32),
			"if (uint32.isNotDefault) 'uint32': uint32",
		}, {
			api.NewTestField("uint64").WithType(api.TypezUint64),
			"if (uint64.isNotDefault) 'uint64': uint64.toString()",
		},

		// optional / nullable primitives (which use createNullableToJson)
		{
			withJSONName(api.NewTestField("bool_opt").WithType(api.TypezBool).WithOptional(), "bool"),
			"'bool': ?boolOpt",
		}, {
			withJSONName(api.NewTestField("string_opt").WithType(api.TypezString).WithOptional(), "string"),
			"'string': ?stringOpt",
		}, {
			withJSONName(api.NewTestField("double_opt").WithType(api.TypezDouble).WithOptional(), "double"),
			"if (doubleOpt case final $1?) 'double': encodeDouble($1)",
		}, {
			withJSONName(api.NewTestField("bytes_opt").WithType(api.TypezBytes).WithOptional(), "bytes"),
			"if (bytesOpt case final $1?) 'bytes': encodeBytes($1)",
		},

		// enums (implicitly non-nullable unless optional)
		{
			api.NewTestField("enum1").WithType(api.TypezEnum).WithTypezID(enum.ID),
			"if (enum1.isNotDefault) 'enum1': enum1.toJson()",
		},
		{
			api.NewTestField("enum_opt").WithType(api.TypezEnum).WithTypezID(enum.ID).WithOptional(),
			"'enumOpt': ?enumOpt?.toJson()",
		},

		// messages (always nullable in proto3 singular message fields)
		{
			api.NewTestField("message").WithMessageType(secret),
			"'message': ?message?.toJson()",
		},
		{
			// Required message (but still nullable since it's a message!)
			api.NewTestField("message").WithMessageType(secret).WithBehavior(api.FieldBehaviorRequired),
			"'message': ?message?.toJson()",
		},

		// repeated primitives
		{
			api.NewTestField("boolList").WithType(api.TypezBool).WithRepeated(),
			"if (boolList.isNotDefault) 'boolList': boolList",
		}, {
			api.NewTestField("bytesList").WithType(api.TypezBytes).WithRepeated(),
			"if (bytesList.isNotDefault) 'bytesList': [for (final i in bytesList) encodeBytes(i)]",
		}, {
			api.NewTestField("doubleList").WithType(api.TypezDouble).WithRepeated(),
			"if (doubleList.isNotDefault) 'doubleList': [for (final i in doubleList) encodeDouble(i)]",
		}, {
			api.NewTestField("fixed32List").WithType(api.TypezFixed32).WithRepeated(),
			"if (fixed32List.isNotDefault) 'fixed32List': fixed32List",
		}, {
			api.NewTestField("fixed64List").WithType(api.TypezFixed64).WithRepeated(),
			"if (fixed64List.isNotDefault) 'fixed64List': [for (final i in fixed64List) i.toString()]",
		}, {
			api.NewTestField("floatList").WithType(api.TypezFloat).WithRepeated(),
			"if (floatList.isNotDefault) 'floatList': [for (final i in floatList) encodeDouble(i)]",
		}, {
			api.NewTestField("int32List").WithType(api.TypezInt32).WithRepeated(),
			"if (int32List.isNotDefault) 'int32List': int32List",
		}, {
			api.NewTestField("int64List").WithType(api.TypezInt64).WithRepeated(),
			"if (int64List.isNotDefault) 'int64List': [for (final i in int64List) i.toString()]",
		}, {
			api.NewTestField("sfixed32List").WithType(api.TypezSfixed32).WithRepeated(),
			"if (sfixed32List.isNotDefault) 'sfixed32List': sfixed32List",
		}, {
			api.NewTestField("sfixed64List").WithType(api.TypezSfixed64).WithRepeated(),
			"if (sfixed64List.isNotDefault) 'sfixed64List': [for (final i in sfixed64List) i.toString()]",
		}, {
			api.NewTestField("sint32List").WithType(api.TypezSint32).WithRepeated(),
			"if (sint32List.isNotDefault) 'sint32List': sint32List",
		}, {
			api.NewTestField("sint64List").WithType(api.TypezSint64).WithRepeated(),
			"if (sint64List.isNotDefault) 'sint64List': [for (final i in sint64List) i.toString()]",
		}, {
			api.NewTestField("stringList").WithType(api.TypezString).WithRepeated(),
			"if (stringList.isNotDefault) 'stringList': stringList",
		}, {
			api.NewTestField("uint32List").WithType(api.TypezUint32).WithRepeated(),
			"if (uint32List.isNotDefault) 'uint32List': uint32List",
		}, {
			api.NewTestField("uint64List").WithType(api.TypezUint64).WithRepeated(),
			"if (uint64List.isNotDefault) 'uint64List': [for (final i in uint64List) i.toString()]",
		},

		// repeated enums
		{
			api.NewTestField("enumList").WithType(api.TypezEnum).WithTypezID(enum.ID).WithRepeated(),
			"if (enumList.isNotDefault) 'enumList': [for (final i in enumList) i.toJson()]",
		},

		// repeated messages
		{
			api.NewTestField("messageList").WithMessageType(secret).WithRepeated(),
			"if (messageList.isNotDefault) 'messageList': [for (final i in messageList) i.toJson()]",
		},

		// maps
		{
			api.NewTestField("map_string_to_string").WithMap().WithMessageType(mapStringToString),
			"if (mapStringToString.isNotDefault) 'mapStringToString': mapStringToString",
		},
		{
			api.NewTestField("map_int32_to_string").WithMap().WithMessageType(mapInt32ToString),
			"if (mapInt32ToString.isNotDefault) 'mapInt32ToString': {for (final e in mapInt32ToString.entries) e.key.toString(): e.value}",
		},
		{
			api.NewTestField("map_bool_to_string").WithMap().WithMessageType(mapBoolToString),
			"if (mapBoolToString.isNotDefault) 'mapBoolToString': {for (final e in mapBoolToString.entries) e.key.toString(): e.value}",
		},
		{
			api.NewTestField("map_string_to_int64").WithMap().WithMessageType(mapStringToInt64),
			"if (mapStringToInt64.isNotDefault) 'mapStringToInt64': {for (final e in mapStringToInt64.entries) e.key: e.value.toString()}",
		},
		{
			api.NewTestField("map_int64_to_string").WithMap().WithMessageType(mapInt64ToString),
			"if (mapInt64ToString.isNotDefault) 'mapInt64ToString': {for (final e in mapInt64ToString.entries) e.key.toString(): e.value}",
		},
		{
			api.NewTestField("map_uint32_to_string").WithMap().WithMessageType(mapUint32ToString),
			"if (mapUint32ToString.isNotDefault) 'mapUint32ToString': {for (final e in mapUint32ToString.entries) e.key.toString(): e.value}",
		},
		{
			api.NewTestField("map_uint64_to_string").WithMap().WithMessageType(mapUint64ToString),
			"if (mapUint64ToString.isNotDefault) 'mapUint64ToString': {for (final e in mapUint64ToString.entries) e.key.toString(): e.value}",
		},
		{
			api.NewTestField("map_sint32_to_string").WithMap().WithMessageType(mapSint32ToString),
			"if (mapSint32ToString.isNotDefault) 'mapSint32ToString': {for (final e in mapSint32ToString.entries) e.key.toString(): e.value}",
		},
		{
			api.NewTestField("map_sint64_to_string").WithMap().WithMessageType(mapSint64ToString),
			"if (mapSint64ToString.isNotDefault) 'mapSint64ToString': {for (final e in mapSint64ToString.entries) e.key.toString(): e.value}",
		},
		{
			api.NewTestField("map_fixed32_to_string").WithMap().WithMessageType(mapFixed32ToString),
			"if (mapFixed32ToString.isNotDefault) 'mapFixed32ToString': {for (final e in mapFixed32ToString.entries) e.key.toString(): e.value}",
		},
		{
			api.NewTestField("map_fixed64_to_string").WithMap().WithMessageType(mapFixed64ToString),
			"if (mapFixed64ToString.isNotDefault) 'mapFixed64ToString': {for (final e in mapFixed64ToString.entries) e.key.toString(): e.value}",
		},
		{
			api.NewTestField("map_sfixed32_to_string").WithMap().WithMessageType(mapSfixed32ToString),
			"if (mapSfixed32ToString.isNotDefault) 'mapSfixed32ToString': {for (final e in mapSfixed32ToString.entries) e.key.toString(): e.value}",
		},
		{
			api.NewTestField("map_sfixed64_to_string").WithMap().WithMessageType(mapSfixed64ToString),
			"if (mapSfixed64ToString.isNotDefault) 'mapSfixed64ToString': {for (final e in mapSfixed64ToString.entries) e.key.toString(): e.value}",
		},
	} {
		t.Run(test.field.Name, func(t *testing.T) {
			message := api.NewTestMessage("UpdateSecretRequest").
				WithPackage(sample.Package).
				WithID("..UpdateRequest").
				WithFields(test.field)
			model := api.NewTestAPI([]*api.Message{
				message, secret, foreignMessage,
				mapStringToString, mapInt32ToString, mapBoolToString, mapStringToInt64,
				mapInt64ToString, mapUint32ToString, mapUint64ToString,
				mapSint32ToString, mapSint64ToString,
				mapFixed32ToString, mapFixed64ToString,
				mapSfixed32ToString, mapSfixed64ToString,
			}, []*api.Enum{enum, foreignEnumState}, []*api.Service{})
			annotate := newAnnotateModel(model)
			annotate.annotateModel(map[string]string{
				"prefix:google.cloud.foo": "foo",
			})

			annotate.annotateField(test.field)
			got := test.field.Codec.(*fieldAnnotation).ToJson
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestAnnotateEnum(t *testing.T) {
	type wantedValueAnnotation struct {
		wantValueName string
	}

	enumValueSimple := api.NewTestEnumValue("NAME", 0).WithID(".test.v1.SomeMessage.SomeEnum.NAME")
	enumValueReservedName := api.NewTestEnumValue("in", 0).WithID(".test.v1.SomeMessage.SomeEnum.in")
	enumValueCompound := api.NewTestEnumValue("ENUM_VALUE", 0).WithID(".test.v1.SomeMessage.SomeEnum.ENUM_VALUE")
	enumValueNameDifferentCaseOnly := api.NewTestEnumValue("name", 0).WithID(".test.v1.SomeMessage.SomeEnum.name")
	someEnum := api.NewTestEnum("SomeEnum").
		WithPackage("test.v1").
		WithID(".test.v1.SomeMessage.SomeEnum").
		WithValues(enumValueSimple, enumValueReservedName, enumValueCompound)
	noValuesEnum := api.NewTestEnum("NoValuesEnum").
		WithPackage("test.v1").
		WithID(".test.v1.NoValuesEnum")
	someEnumNameDifferentCaseOnly := api.NewTestEnum("DifferentCaseOnlyEnum").
		WithPackage("test.v1").
		WithID(".test.v1.SomeMessage.SomeDifferentCaseOnlyEnum").
		WithValues(enumValueSimple, enumValueNameDifferentCaseOnly)

	model := api.NewTestAPI(
		[]*api.Message{},
		[]*api.Enum{someEnum, noValuesEnum, someEnumNameDifferentCaseOnly},
		[]*api.Service{})
	model.PackageName = "test"
	annotate := newAnnotateModel(model)

	for _, test := range []struct {
		enum                 *api.Enum
		wantEnumName         string
		wantEnumDefaultValue string
		wantValueAnnotations []wantedValueAnnotation
	}{
		{enum: someEnum,
			wantEnumName:         "SomeEnum",
			wantEnumDefaultValue: "name",
			wantValueAnnotations: []wantedValueAnnotation{{"name"}, {"in$"}, {"enumValue"}},
		},
		{enum: noValuesEnum,
			wantEnumName:         "NoValuesEnum",
			wantEnumDefaultValue: "",
			wantValueAnnotations: []wantedValueAnnotation{},
		},
		{enum: someEnumNameDifferentCaseOnly,
			wantEnumName:         "DifferentCaseOnlyEnum",
			wantEnumDefaultValue: "NAME",
			wantValueAnnotations: []wantedValueAnnotation{{"NAME"}, {"name"}},
		},
	} {
		t.Run(test.wantEnumName, func(t *testing.T) {
			annotate.annotateEnum(test.enum)
			codec := test.enum.Codec.(*enumAnnotation)
			gotEnumName := codec.Name
			gotEnumDefaultValue := codec.DefaultValue

			if diff := cmp.Diff(test.wantEnumName, gotEnumName); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(test.wantEnumDefaultValue, gotEnumDefaultValue); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}

			for i, value := range test.enum.Values {
				testName := fmt.Sprintf("TestAnnotateEnum(%q) [value annotation %d]", test.enum.Name, i)
				t.Run(testName, func(t *testing.T) {
					wantValueAnnotation := test.wantValueAnnotations[i]
					gotValueAnnotation := value.Codec.(*enumValueAnnotation)
					if diff := cmp.Diff(wantValueAnnotation.wantValueName, gotValueAnnotation.Name); diff != "" {
						t.Errorf("mismatch (-want +got):\n%s", diff)
					}
				})
			}
		})
	}
}

func TestAnnotateField(t *testing.T) {
	enumState := api.NewTestEnum("State").WithID("State")
	message := api.NewTestMessage("Message").WithID("Message")
	mapMessage := api.NewTestMessage("MapMessage").
		WithID("..MapMessage").
		WithFields(
			api.NewTestField("key").WithType(api.TypezString),
			api.NewTestField("value").WithType(api.TypezInt32),
		)
	mapMessage.IsMap = true

	for _, test := range []struct {
		name  string
		field *api.Field
		want  *fieldAnnotation
	}{
		{
			name:  "implicit presence primitive",
			field: api.NewTestField("int32_field").WithType(api.TypezInt32),
			want: &fieldAnnotation{
				Name:                  "int32Field",
				Type:                  "int",
				DocLines:              []string{},
				Required:              true,
				Nullable:              false,
				FieldBehaviorRequired: false,
				DefaultValue:          "0",
				ConstDefault:          true,
			},
		},
		{
			name: "required primitive",
			field: api.NewTestField("int32_field").
				WithType(api.TypezInt32).
				WithBehavior(api.FieldBehaviorRequired),
			want: &fieldAnnotation{
				Name:                  "int32Field",
				Type:                  "int",
				DocLines:              []string{},
				Required:              true,
				Nullable:              false,
				FieldBehaviorRequired: true,
				DefaultValue:          "",
				ConstDefault:          true,
			},
		},
		{
			name:  "optional primitive",
			field: api.NewTestField("int32_field").WithType(api.TypezInt32).WithOptional(),
			want: &fieldAnnotation{
				Name:                  "int32Field",
				Type:                  "int",
				DocLines:              []string{},
				Required:              false,
				Nullable:              true,
				FieldBehaviorRequired: false,
				DefaultValue:          "",
				ConstDefault:          true,
			},
		},
		{
			name:  "repeated",
			field: api.NewTestField("int32_list").WithType(api.TypezInt32).WithRepeated(),
			want: &fieldAnnotation{
				Name:                  "int32List",
				Type:                  "List<int>",
				DocLines:              []string{},
				Required:              true,
				Nullable:              false,
				FieldBehaviorRequired: false,
				DefaultValue:          "const []",
				ConstDefault:          true,
			},
		},
		{
			name: "map",
			field: api.NewTestField("map_field").
				WithType(api.TypezMessage).
				WithTypezID("..MapMessage").
				WithMap(),
			want: &fieldAnnotation{
				Name:                  "mapField",
				Type:                  "Map<String, int>",
				DocLines:              []string{},
				Required:              true,
				Nullable:              false,
				FieldBehaviorRequired: false,
				DefaultValue:          "const {}",
				ConstDefault:          true,
			},
		},
		{
			name:  "message",
			field: api.NewTestField("message_field").WithMessageType(message),
			want: &fieldAnnotation{
				Name:                  "messageField",
				Type:                  "Message",
				DocLines:              []string{},
				Required:              false,
				Nullable:              true,
				FieldBehaviorRequired: false,
				DefaultValue:          "",
				ConstDefault:          true,
			},
		},
		{
			name: "required message",
			field: api.NewTestField("message_field").
				WithMessageType(message).
				WithBehavior(api.FieldBehaviorRequired),
			want: &fieldAnnotation{
				Name:                  "messageField",
				Type:                  "Message",
				DocLines:              []string{},
				Required:              false,
				Nullable:              true,
				FieldBehaviorRequired: true,
				DefaultValue:          "",
				ConstDefault:          true,
			},
		},
		{
			name:  "enum",
			field: api.NewTestField("enum_field").WithType(api.TypezEnum).WithTypezID("State"),
			want: &fieldAnnotation{
				Name:                  "enumField",
				Type:                  "State",
				DocLines:              []string{},
				Required:              true,
				Nullable:              false,
				FieldBehaviorRequired: false,
				DefaultValue:          "State.$default",
				ConstDefault:          true,
			},
		},
		{
			name: "required enum",
			field: api.NewTestField("enum_field").
				WithType(api.TypezEnum).
				WithTypezID("State").
				WithBehavior(api.FieldBehaviorRequired),
			want: &fieldAnnotation{
				Name:                  "enumField",
				Type:                  "State",
				DocLines:              []string{},
				Required:              true,
				Nullable:              false,
				FieldBehaviorRequired: true,
				DefaultValue:          "",
				ConstDefault:          true,
			},
		},
		{
			// `google.protobuf.Empty` is a special because, in some cases, it is
			// converted to the `void` Dart type. `void` is not nullable in Dart.
			name:  "google.protobuf.Empty",
			field: api.NewTestField("empty_field").WithType(api.TypezMessage).WithTypezID(".google.protobuf.Empty"),
			want: &fieldAnnotation{
				Name:                  "emptyField",
				Type:                  "Empty",
				DocLines:              []string{},
				Required:              false,
				Nullable:              true,
				FieldBehaviorRequired: false,
				DefaultValue:          "",
				ConstDefault:          true,
			},
		},
		{
			name:  "float",
			field: api.NewTestField("float_field").WithType(api.TypezFloat).WithOptional(),
			want: &fieldAnnotation{
				Name:                  "floatField",
				Type:                  "double",
				DocLines:              []string{},
				Required:              false,
				Nullable:              true,
				FieldBehaviorRequired: false,
				DefaultValue:          "",
				ConstDefault:          true,
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			model := api.NewTestAPI([]*api.Message{message}, []*api.Enum{enumState}, []*api.Service{})
			model.AddMessage(mapMessage)
			annotate := newAnnotateModel(model)
			registerMissingWkt(annotate.model)

			annotate.annotateField(test.field)
			got := test.field.Codec.(*fieldAnnotation)
			// `FromJson` and `ToJson` have their own tests.
			// Clear them rather than using `IgnoreFields` so that they do not appear in the diff.
			got.FromJson = ""
			got.ToJson = ""

			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestFindExampleMethod(t *testing.T) {
	requiredField := api.NewTestField("required").WithBehavior(api.FieldBehaviorRequired)
	optionalField := api.NewTestField("optional")

	msgWithRequired := api.NewTestMessage("WithRequired").
		WithPackage("mypackage").
		WithFields(requiredField)
	msgWithoutRequired := api.NewTestMessage("WithoutRequired").
		WithPackage("mypackage").
		WithFields(optionalField)

	differentPackageMsg := api.NewTestMessage("DifferentPackage").
		WithPackage("otherpackage").
		WithFields(optionalField)

	lroMethod := api.NewTestMethod("LroMethod").WithOperationInfo(&api.OperationInfo{})

	streamingMethod := api.NewTestMethod("StreamingMethod")
	streamingMethod.IsSimple = false

	voidMethod := api.NewTestMethod("VoidMethod").
		WithInput(msgWithoutRequired)
	voidMethod.IsSimple = true
	voidMethod.ReturnsEmpty = true

	requiredInputMethod := api.NewTestMethod("RequiredInputMethod").
		WithInput(msgWithRequired).
		WithOutput(msgWithoutRequired)
	requiredInputMethod.IsSimple = true

	requiredOutputMethod := api.NewTestMethod("RequiredOutputMethod").
		WithInput(msgWithoutRequired).
		WithOutput(msgWithRequired)
	requiredOutputMethod.IsSimple = true

	differentPackageMethod := api.NewTestMethod("DifferentPackageMethod").
		WithInput(differentPackageMsg).
		WithOutput(differentPackageMsg)
	differentPackageMethod.IsSimple = true

	perfectMethod := api.NewTestMethod("PerfectMethod").
		WithInput(msgWithoutRequired).
		WithOutput(msgWithoutRequired)
	perfectMethod.IsSimple = true

	for _, test := range []struct {
		name    string
		methods []*api.Method
		want    *api.Method
	}{
		{
			name:    "perfect method exists",
			methods: []*api.Method{perfectMethod, requiredInputMethod, requiredOutputMethod, voidMethod, differentPackageMethod, streamingMethod, lroMethod},
			want:    perfectMethod,
		},
		{
			name:    "best method has a request type with a required field",
			methods: []*api.Method{requiredInputMethod, requiredOutputMethod, voidMethod, differentPackageMethod, streamingMethod, lroMethod},
			want:    requiredInputMethod,
		},
		{
			name:    "best method has a response type with a required field",
			methods: []*api.Method{requiredOutputMethod, voidMethod, differentPackageMethod, streamingMethod, lroMethod},
			want:    requiredOutputMethod,
		},
		{
			name:    "best method has void return",
			methods: []*api.Method{voidMethod, differentPackageMethod, streamingMethod, lroMethod},
			want:    voidMethod,
		},
		{
			name:    "best method has types from other packages",
			methods: []*api.Method{differentPackageMethod, streamingMethod, lroMethod},
			want:    differentPackageMethod,
		},
		{
			name:    "only streaming and LRO methods exist",
			methods: []*api.Method{streamingMethod, lroMethod},
			want:    nil,
		},
		{
			name:    "only LRO method exists",
			methods: []*api.Method{lroMethod},
			want:    nil,
		},
	} {
		r := rand.New(rand.NewSource(42))
		t.Run(test.name, func(t *testing.T) {
			methods := slices.Clone(test.methods)
			r.Shuffle(len(methods), func(i, j int) {
				methods[i], methods[j] = methods[j], methods[i]
			})
			s := api.NewTestService("Service").WithPackage("mypackage")
			s.Codec = &serviceAnnotations{
				Methods: methods,
			}
			_, got := findExampleMethod([]*api.Service{s})
			if got != test.want {
				gotName := "nil"
				if got != nil {
					gotName = got.Name
				}
				wantName := "nil"
				if test.want != nil {
					wantName = test.want.Name
				}
				t.Errorf("mismatch, got %v, want %v", gotName, wantName)
			}
		})
	}
}
