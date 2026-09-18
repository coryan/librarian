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
	"os"
	"path/filepath"
	"testing"
)

func TestProtoIndex_Hermetic(t *testing.T) {
	tempDir := t.TempDir()
	protoContent := `syntax = "proto3";

package google.example.v1;

// Service comment
service ExampleService {
  rpc DoFoo(FooRequest) returns (FooResponse) {
    option (google.api.http) = {
      get: "/v1/{parent=projects/*}/secrets"
    };
  }
}

/* Block comment
   with multiple lines */
message FooRequest {
  string name = 1;

  enum Status {
    STATUS_UNSPECIFIED = 0;
    ENABLED = 1;
    DISABLED = 2;
  }

  Status status = 2;

  message NestedMessage {
    string detail = 1;
  }

  NestedMessage nested = 3;
}

message FooResponse {
  repeated string items = 1;
}
`
	protoFile := filepath.Join(tempDir, "example.proto")
	if err := os.WriteFile(protoFile, []byte(protoContent), 0644); err != nil {
		t.Fatalf("failed to write proto: %v", err)
	}

	idx := newProtoIndex()
	if err := idx.scanFile(protoFile, "google/example/v1/example.proto"); err != nil {
		t.Fatalf("scanFile failed: %v", err)
	}

	wantSymbols := map[string]int{
		"google.example.v1.ExampleService":                  6,
		"google.example.v1.ExampleService.DoFoo":            7,
		"google.example.v1.FooRequest":                      16,
		"google.example.v1.FooRequest.name":                 17,
		"google.example.v1.FooRequest.Status":               19,
		"google.example.v1.FooRequest.Status.ENABLED":       21,
		"google.example.v1.FooRequest.ENABLED":              21,
		"google.example.v1.FooRequest.status":               25,
		"google.example.v1.FooRequest.NestedMessage":        27,
		"google.example.v1.FooRequest.NestedMessage.detail": 28,
		"google.example.v1.FooRequest.nested":               31,
		"google.example.v1.FooResponse":                     34,
		"google.example.v1.FooResponse.items":               35,
	}

	for sym, wantLine := range wantSymbols {
		loc, ok := idx.find(sym)
		if !ok {
			t.Errorf("symbol %s not found in index", sym)
			continue
		}
		if loc.Line != wantLine {
			t.Errorf("symbol %s line mismatch: want %d, got %d", sym, wantLine, loc.Line)
		}
		if loc.Filename != "google/example/v1/example.proto" {
			t.Errorf("symbol %s filename mismatch: want %s, got %s", sym, "google/example/v1/example.proto", loc.Filename)
		}
	}

}

func TestProtoIndex_TestdataProto(t *testing.T) {
	testProto := filepath.Join("testdata", "protos", "test.proto")
	if _, err := os.Stat(testProto); err != nil {
		t.Fatalf("test.proto fixture not found: %v", err)
	}

	idx := newProtoIndex()
	if err := idx.scanFile(testProto, "google/test/admin/database/v1/test.proto"); err != nil {
		t.Fatalf("scanFile(test.proto) failed: %v", err)
	}

	loc, ok := idx.find("google.test.admin.database.v1.GoldenKitchenSink")
	if !ok {
		t.Fatalf("GoldenKitchenSink symbol not found")
	}
	if loc.Line <= 0 {
		t.Errorf("expected positive line number, got %d", loc.Line)
	}
}
