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

func TestNewCodec(t *testing.T) {
	reqMsg := api.NewTestMessage("GetDatabaseRequest")
	respMsg := api.NewTestMessage("Database")
	method := api.NewTestMethod("GetDatabase").WithInput(reqMsg).WithOutput(respMsg)
	svc := api.NewTestService("GoldenKitchenSink").WithMethods(method)
	model := api.NewTestAPI([]*api.Message{reqMsg, respMsg}, nil, []*api.Service{svc})
	if err := api.CrossReference(model); err != nil {
		t.Fatalf("api.CrossReference failed: %v", err)
	}

	lib := &config.Library{
		Name: "golden_kitchen_sink",
		Cpp: &config.CppLibrary{
			CppDefault: config.CppDefault{
				ProductPath: "generator/integration_tests/golden/v1",
			},
			ServiceEndpointEnvVar: "GOLDEN_KITCHEN_SINK_ENDPOINT",
		},
	}

	codec, err := newCodec(model, lib, "/tmp/out")
	if err != nil {
		t.Fatalf("newCodec failed: %v", err)
	}
	if codec.Model != model {
		t.Errorf("got model %v, want %v", codec.Model, model)
	}
	if codec.Library != lib {
		t.Errorf("got library %v, want %v", codec.Library, lib)
	}
	if codec.OutDir != "/tmp/out" {
		t.Errorf("got outdir %q, want %q", codec.OutDir, "/tmp/out")
	}
	if err := codec.annotateModel(); err != nil {
		t.Fatalf("annotateModel failed: %v", err)
	}
}

func TestNewCodec_RichModel(t *testing.T) {
	// Construct messages with fields, oneofs, and pagination using api.NewTest*()
	nameField := api.NewTestField("name").WithType(api.TypezString)
	pageTokenField := api.NewTestField("page_token").WithType(api.TypezString)
	pageSizeField := api.NewTestField("page_size").WithType(api.TypezInt32)
	reqMsg := api.NewTestMessage("ListDatabasesRequest").WithFields(nameField, pageTokenField, pageSizeField)

	databaseMsg := api.NewTestMessage("Database").WithFields(nameField)
	itemField := api.NewTestField("databases").WithMessageType(databaseMsg).WithRepeated()
	nextPageTokenField := api.NewTestField("next_page_token").WithType(api.TypezString)
	respMsg := api.NewTestMessage("ListDatabasesResponse").WithPagination(nextPageTokenField, itemField)

	stateEnum := api.NewTestEnum("DatabaseState", "STATE_UNSPECIFIED", "CREATING", "READY")

	// Create methods: unary, paging, streaming, LRO
	listMethod := api.NewTestMethod("ListDatabases").
		WithInput(reqMsg).
		WithOutput(respMsg).
		WithPagination(pageTokenField)

	bidiMethod := api.NewTestMethod("StreamingChat").
		WithInput(reqMsg).
		WithOutput(respMsg).
		WithBidiStreaming()

	lroMethod := api.NewTestMethod("CreateDatabase").
		WithInput(reqMsg).
		WithOutput(databaseMsg).
		WithOperationInfo(&api.OperationInfo{
			ResponseTypeID: ".test.Database",
			MetadataTypeID: ".test.CreateDatabaseMetadata",
		})

	svc := api.NewTestService("DatabaseAdmin").WithMethods(listMethod, bidiMethod, lroMethod)
	model := api.NewTestAPI([]*api.Message{reqMsg, respMsg, databaseMsg}, []*api.Enum{stateEnum}, []*api.Service{svc})
	if err := api.CrossReference(model); err != nil {
		t.Fatalf("api.CrossReference failed: %v", err)
	}

	lib := &config.Library{
		Name: "database_admin",
		Cpp: &config.CppLibrary{
			CppDefault: config.CppDefault{
				ProductPath: "generator/integration_tests/golden/v1",
			},
		},
	}

	codec, err := newCodec(model, lib, "/tmp/out")
	if err != nil {
		t.Fatalf("newCodec failed: %v", err)
	}
	if err := codec.annotateModel(); err != nil {
		t.Fatalf("annotateModel failed: %v", err)
	}
}

func TestNewCodec_NilModel(t *testing.T) {
	_, err := newCodec(nil, nil, "/tmp/out")
	if err == nil {
		t.Errorf("expected error for nil model, got nil")
	}
}
