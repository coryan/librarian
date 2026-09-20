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
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/sidekick/api"
)

func TestAnnotateProto_Grouping(t *testing.T) {
	msg1 := api.NewTestMessage("Request").
		WithPackage("google.iam.credentials.v1")
	msg2 := api.NewTestMessage("Response").
		WithPackage("google.iam.credentials.v1")

	model := api.NewTestAPI([]*api.Message{msg1, msg2}, nil, nil).
		WithDefinitionLocation(msg1.ID, "google/iam/credentials/v1/common.proto", 10).
		WithDefinitionLocation(msg2.ID, "google/iam/credentials/v1/common.proto", 20)

	lib := &config.Library{
		Name:    "google-iam-credentials",
		Version: "1.0.0",
		Python: &config.PythonPackage{
			DefaultVersion: "v1",
		},
	}

	c, err := newCodec(model, lib, "/tmp/out")
	if err != nil {
		t.Fatalf("newCodec failed: %v", err)
	}

	if err := c.annotateModel(); err != nil {
		t.Fatalf("annotateModel failed: %v", err)
	}

	mAnn, ok := model.Codec.(*ModelAnnotations)
	if !ok || mAnn == nil {
		t.Fatalf("model.Codec is not *ModelAnnotations")
	}

	if len(mAnn.Protos) != 1 {
		t.Fatalf("got %d protos, want 1", len(mAnn.Protos))
	}

	proto := mAnn.Protos[0]
	if proto.ModuleName != "common" {
		t.Errorf("proto.ModuleName = %q, want %q", proto.ModuleName, "common")
	}
	if len(proto.TopLevelMessages) != 2 {
		t.Errorf("got %d messages, want 2", len(proto.TopLevelMessages))
	}
	wantManifest := []string{"Request", "Response"}
	if diff := cmp.Diff(wantManifest, proto.Manifest); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestComputeModuleAlias(t *testing.T) {
	for _, test := range []struct {
		name     string
		pkgParts []string
		version  string
		module   string
		want     string
	}{
		{
			name:     "asset module alias",
			pkgParts: []string{"google", "cloud", "asset", "v1"},
			version:  "v1",
			module:   "assets",
			want:     "gca_assets",
		},
		{
			name:     "redis module alias",
			pkgParts: []string{"google", "cloud", "redis", "v1"},
			version:  "v1",
			module:   "cloud_redis",
			want:     "gcr_cloud_redis",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := computeModuleAlias(test.pkgParts, test.version, test.module)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
