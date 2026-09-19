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

package python

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/sidekick/api"
)

func TestAnnotateModel(t *testing.T) {
	msg := api.NewTestMessage("Secret").WithFields(
		api.NewTestField("name").WithType(api.TypezString),
	)
	enum := api.NewTestEnum("SecretStatus").WithValues(
		api.NewTestEnumValue("ENABLED", 1),
	)
	svc := api.NewTestService("SecretManagerService")

	model := api.NewTestAPI([]*api.Message{msg}, []*api.Enum{enum}, []*api.Service{svc})
	model.Name = "google-cloud-secretmanager"

	lib := &config.Library{
		Name:          "google-cloud-secretmanager",
		Version:       "1.0.0",
		CopyrightYear: "2026",
		APIs: []*config.API{
			{Path: "google/cloud/secretmanager/v1"},
		},
		Python: &config.PythonPackage{
			DefaultVersion: "v1",
		},
	}

	codec := newTestCodec(t, model, lib)
	if err := codec.annotateModel(); err != nil {
		t.Fatalf("annotateModel() failed: %v", err)
	}

	ann, ok := model.Codec.(*ModelAnnotations)
	if !ok {
		t.Fatalf("expected ModelAnnotations, got %T", model.Codec)
	}

	want := &ModelAnnotations{
		CopyrightYear:    "2026",
		PackageName:      "google-cloud-secretmanager",
		PackageVersion:   "1.0.0",
		DefaultVersion:   "v1",
		LibraryPackage:   "google.cloud.secretmanager_v1",
		ProtoPackage:     "test",
		GAPICNamespace:   "google.cloud",
		GAPICName:        "secretmanager",
		PackageDirectory: "google/cloud/secretmanager_v1",
	}

	if diff := cmp.Diff(want, ann,
		cmpopts.IgnoreFields(ModelAnnotations{}, "BoilerPlate", "Messages", "Enums", "Services"),
	); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}

	if len(ann.Messages) != 1 || ann.Messages[0].Name != "Secret" {
		t.Errorf("unexpected Messages in model annotation: %+v", ann.Messages)
	}
	if len(ann.Enums) != 1 || ann.Enums[0].Name != "SecretStatus" {
		t.Errorf("unexpected Enums in model annotation: %+v", ann.Enums)
	}
	if len(ann.Services) != 1 || ann.Services[0].Name != "SecretManagerService" {
		t.Errorf("unexpected Services in model annotation: %+v", ann.Services)
	}
}
