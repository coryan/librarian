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

func TestAnnotateOptions_Defaults(t *testing.T) {
	svc := api.NewTestService("KitchenSink").WithPackage("golden.v1")
	svc.DefaultHost = "kitchensink.googleapis.com"

	ann := annotateOptions(svc, nil)

	if diff := cmp.Diff("KitchenSinkConnectionOptions", ann.OptionsClassName()); diff != "" {
		t.Errorf("OptionsClassName mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff("kitchensink.googleapis.com", ann.DefaultEndpoint); diff != "" {
		t.Errorf("DefaultEndpoint mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff("443", ann.DefaultPort); diff != "" {
		t.Errorf("DefaultPort mismatch (-want +got):\n%s", diff)
	}
	if ann.HasEndpointEnvVar() || ann.HasEmulatorEnvVar() {
		t.Errorf("expected no env vars by default")
	}
}

func TestAnnotateOptions_WithEnvVars(t *testing.T) {
	svc := api.NewTestService("KitchenSink").WithPackage("golden.v1")
	svc.DefaultHost = "kitchensink.googleapis.com:8443"

	lib := &config.Library{
		Cpp: &config.CppLibrary{
			CppDefault: config.CppDefault{
				ProductPath: "generator/integration_tests/golden/v1",
			},
			ServiceEndpointEnvVar:  "KITCHEN_SINK_ENDPOINT",
			EmulatorEndpointEnvVar: "KITCHEN_SINK_EMULATOR_HOST",
		},
	}

	ann := annotateOptions(svc, lib)

	if !ann.HasEndpointEnvVar() || ann.ServiceEndpointEnvVar != "KITCHEN_SINK_ENDPOINT" {
		t.Errorf("got ServiceEndpointEnvVar %q, want 'KITCHEN_SINK_ENDPOINT'", ann.ServiceEndpointEnvVar)
	}
	if !ann.HasEmulatorEnvVar() || ann.EmulatorEndpointEnvVar != "KITCHEN_SINK_EMULATOR_HOST" {
		t.Errorf("got EmulatorEndpointEnvVar %q, want 'KITCHEN_SINK_EMULATOR_HOST'", ann.EmulatorEndpointEnvVar)
	}
	if diff := cmp.Diff("8443", ann.DefaultPort); diff != "" {
		t.Errorf("DefaultPort mismatch (-want +got):\n%s", diff)
	}
}

func TestAnnotateOptions_LocationStyle(t *testing.T) {
	svc := api.NewTestService("LocationSvc").WithPackage("golden.v1")
	lib := &config.Library{
		Cpp: &config.CppLibrary{
			CppDefault: config.CppDefault{
				ProductPath:           "generator/integration_tests/golden/v1",
				EndpointLocationStyle: "LOCATION_DEPENDENT",
			},
		},
	}

	ann := annotateOptions(svc, lib)
	if diff := cmp.Diff("LOCATION_DEPENDENT", ann.EndpointLocationStyle); diff != "" {
		t.Errorf("EndpointLocationStyle mismatch (-want +got):\n%s", diff)
	}
}
