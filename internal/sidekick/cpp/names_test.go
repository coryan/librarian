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

func TestCamelCaseToSnakeCase(t *testing.T) {
	for _, test := range []struct {
		input string
		want  string
	}{
		{input: "GoldenKitchenSink", want: "golden_kitchen_sink"},
		{input: "GoldenThingAdmin", want: "golden_thing_admin"},
		{input: "GoldenRestOnly", want: "golden_rest_only"},
		{input: "Deprecated", want: "deprecated"},
		{input: "RequestId", want: "request_id"},
		{input: "BigQuery", want: "bigquery"},
		{input: "BigQueryAdmin", want: "bigquery_admin"},
		{input: "HTTPClient", want: "http_client"},
		{input: "Test2Foo", want: "test2_foo"},
	} {
		t.Run(test.input, func(t *testing.T) {
			got := camelCaseToSnakeCase(test.input)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("camelCaseToSnakeCase(%q) mismatch (-want +got):\n%s", test.input, diff)
			}
		})
	}
}

func TestServiceNameToFileName(t *testing.T) {
	for _, test := range []struct {
		input string
		want  string
	}{
		{input: "GoldenKitchenSink", want: "golden_kitchen_sink"},
		{input: "GoldenThingAdmin", want: "golden_thing_admin"},
		{input: "GoldenRestOnly", want: "golden_rest_only"},
		{input: "DeprecatedService", want: "deprecated"},
		{input: "RequestIdService", want: "request_id"},
		{input: "Service", want: ""},
		{input: "google.cloud.SecretManagerService", want: "google/cloud/secret_manager"},
	} {
		t.Run(test.input, func(t *testing.T) {
			got := serviceNameToFileName(test.input)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("serviceNameToFileName(%q) mismatch (-want +got):\n%s", test.input, diff)
			}
		})
	}
}
