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
	"github.com/googleapis/librarian/internal/sidekick/api"
)

func TestAnnotateClient_NilService(t *testing.T) {
	c := testCodec(t, api.NewTestAPI(nil, nil, nil))
	got := c.annotateClient(nil, nil)
	if got != nil {
		t.Errorf("annotateClient(nil, nil) = %v, want nil", got)
	}
}

func TestAnnotateClient_BasicService(t *testing.T) {
	itemMsg := api.NewTestMessage("Item").WithPackage("google.cloud.test.v1")
	reqMsg := api.NewTestMessage("GetItemRequest").WithPackage("google.cloud.test.v1")

	method := api.NewTestMethod("GetItem").
		WithInput(reqMsg).
		WithOutput(itemMsg)

	svc := api.NewTestService("ItemService").WithMethods(method)
	model := api.NewTestAPI([]*api.Message{itemMsg, reqMsg}, nil, []*api.Service{svc}).
		WithDefinitionLocation(reqMsg.ID, "google/cloud/test/v1/item_service.proto", 10).
		WithDefinitionLocation(itemMsg.ID, "google/cloud/test/v1/items.proto", 20)

	c := testCodec(t, model)
	if err := c.annotateModel(); err != nil {
		t.Fatal(err)
	}

	sAnn, ok := svc.Codec.(*ServiceAnnotations)
	if !ok || sAnn == nil || sAnn.Client == nil {
		t.Fatalf("expected ServiceAnnotations with Client, got %T", svc.Codec)
	}

	got := sAnn.Client
	type basicClientSummary struct {
		ClientClassName       string
		TransportClassName    string
		ServiceName           string
		ServiceTitleSnakeCase string
		ClientTitleSnakeCase  string
	}
	gotSummary := basicClientSummary{
		ClientClassName:       got.ClientClassName,
		TransportClassName:    got.TransportClassName,
		ServiceName:           got.ServiceName,
		ServiceTitleSnakeCase: got.ServiceTitleSnakeCase,
		ClientTitleSnakeCase:  got.ClientTitleSnakeCase,
	}
	wantSummary := basicClientSummary{
		ClientClassName:       "ItemServiceClient",
		TransportClassName:    "ItemServiceTransport",
		ServiceName:           "ItemService",
		ServiceTitleSnakeCase: "item service",
		ClientTitleSnakeCase:  "item service client",
	}
	if diff := cmp.Diff(wantSummary, gotSummary); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}

	if len(got.Methods) != 1 {
		t.Fatalf("got %d methods, want 1", len(got.Methods))
	}
	m := got.Methods[0]
	type basicMethodSummary struct {
		Name         string
		IsUnary      bool
		IsPaged      bool
		IsLRO        bool
		IsVoid       bool
		RequestType  string
		ReturnType   string
		ResultSphinx string
	}
	gotMethod := basicMethodSummary{
		Name:         m.Name,
		IsUnary:      m.IsUnary,
		IsPaged:      m.IsPaged,
		IsLRO:        m.IsLRO,
		IsVoid:       m.IsVoid,
		RequestType:  m.RequestType,
		ReturnType:   m.ReturnType,
		ResultSphinx: m.ResultSphinx,
	}
	wantMethod := basicMethodSummary{
		Name:         "get_item",
		IsUnary:      true,
		IsPaged:      false,
		IsLRO:        false,
		IsVoid:       false,
		RequestType:  "item_service.GetItemRequest",
		ReturnType:   "items.Item",
		ResultSphinx: "google.cloud.test_v1.types.Item",
	}
	if diff := cmp.Diff(wantMethod, gotMethod); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestAnnotateClient_PagedMethod(t *testing.T) {
	itemMsg := api.NewTestMessage("Item").WithPackage("google.cloud.test.v1")
	pageTokenField := api.NewTestField("page_token").WithType(api.TypezString)
	nextPageTokenField := api.NewTestField("next_page_token").WithType(api.TypezString)
	itemsField := api.NewTestField("items").WithMessageType(itemMsg).WithRepeated()

	reqMsg := api.NewTestMessage("ListItemsRequest").WithPackage("google.cloud.test.v1").WithFields(pageTokenField)
	respMsg := api.NewTestMessage("ListItemsResponse").WithPackage("google.cloud.test.v1").WithFields(itemsField, nextPageTokenField)
	respMsg.Pagination = &api.PaginationInfo{
		NextPageToken: nextPageTokenField,
		PageableItem:  itemsField,
	}

	method := api.NewTestMethod("ListItems").
		WithInput(reqMsg).
		WithOutput(respMsg)
	method.Pagination = pageTokenField

	svc := api.NewTestService("ItemService").WithMethods(method)
	model := api.NewTestAPI([]*api.Message{itemMsg, reqMsg, respMsg}, nil, []*api.Service{svc}).
		WithDefinitionLocation(reqMsg.ID, "google/cloud/test/v1/item_service.proto", 10).
		WithDefinitionLocation(respMsg.ID, "google/cloud/test/v1/item_service.proto", 20).
		WithDefinitionLocation(itemMsg.ID, "google/cloud/test/v1/items.proto", 30)

	c := testCodec(t, model)
	if err := c.annotateModel(); err != nil {
		t.Fatal(err)
	}

	got := svc.Codec.(*ServiceAnnotations).Client
	if len(got.Methods) != 1 {
		t.Fatalf("got %d methods, want 1", len(got.Methods))
	}
	m := got.Methods[0]
	type pagedMethodSummary struct {
		IsPaged        bool
		PagerClassName string
		ReturnType     string
		PagerSphinx    string
	}
	gotPaged := pagedMethodSummary{
		IsPaged:        m.IsPaged,
		PagerClassName: m.PagerClassName,
		ReturnType:     m.ReturnType,
		PagerSphinx:    m.PagerSphinx,
	}
	wantPaged := pagedMethodSummary{
		IsPaged:        true,
		PagerClassName: "ListItemsPager",
		ReturnType:     "pagers.ListItemsPager",
		PagerSphinx:    "google.cloud.test_v1.services.item_service.pagers.ListItemsPager",
	}
	if diff := cmp.Diff(wantPaged, gotPaged); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestAnnotateClient_LROMethod_WithEmptyResponse(t *testing.T) {
	reqMsg := api.NewTestMessage("DeleteItemRequest").WithPackage("google.cloud.test.v1")

	method := api.NewTestMethod("DeleteItem").
		WithInput(reqMsg)
	method.IsLRO = true
	method.OperationInfo = &api.OperationInfo{
		ResponseTypeID: ".google.protobuf.Empty",
		MetadataTypeID: ".google.protobuf.Empty",
	}

	svc := api.NewTestService("ItemService").WithMethods(method)
	model := api.NewTestAPI([]*api.Message{reqMsg}, nil, []*api.Service{svc}).
		WithDefinitionLocation(reqMsg.ID, "google/cloud/test/v1/item_service.proto", 10)

	c := testCodec(t, model)
	if err := c.annotateModel(); err != nil {
		t.Fatal(err)
	}

	got := svc.Codec.(*ServiceAnnotations).Client
	if len(got.Methods) != 1 {
		t.Fatalf("got %d methods, want 1", len(got.Methods))
	}
	m := got.Methods[0]
	type lroMethodSummary struct {
		IsLRO             bool
		ReturnType        string
		LROResponseType   string
		LROOnSameLine     bool
		HasFirstResultDoc bool
	}
	gotLRO := lroMethodSummary{
		IsLRO:             m.IsLRO,
		ReturnType:        m.ReturnType,
		LROResponseType:   m.LROResponseType,
		LROOnSameLine:     m.LROOnSameLine,
		HasFirstResultDoc: m.FirstResultDocLine != "",
	}
	wantLRO := lroMethodSummary{
		IsLRO:             true,
		ReturnType:        "operation.Operation",
		LROResponseType:   "empty_pb2.Empty",
		LROOnSameLine:     true,
		HasFirstResultDoc: true,
	}
	if diff := cmp.Diff(wantLRO, gotLRO); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestAnnotateClient_FlattenedParams_NestedEnum(t *testing.T) {
	outerMsg := api.NewTestMessage("OuterRequest").WithPackage("google.cloud.test.v1")
	enum := api.NewTestEnum("Status")
	outerMsg.WithEnums(enum)

	statusField := api.NewTestField("status").WithType(api.TypezEnum)
	statusField.TypezID = enum.ID
	nameField := api.NewTestField("name").WithType(api.TypezString)

	outerMsg.WithFields(nameField, statusField)

	method := api.NewTestMethod("DoAction").
		WithInput(outerMsg)
	method.Signatures = []*api.MethodSignature{
		{
			Fields: []*api.Field{nameField, statusField},
		},
	}

	svc := api.NewTestService("ActionService").WithPackage("google.cloud.test.v1").WithMethods(method)
	model := api.NewTestAPI([]*api.Message{outerMsg}, []*api.Enum{enum}, []*api.Service{svc}).
		WithDefinitionLocation(outerMsg.ID, "google/cloud/test/v1/action_service.proto", 10)

	c := testCodec(t, model)
	if err := c.annotateModel(); err != nil {
		t.Fatal(err)
	}

	got := svc.Codec.(*ServiceAnnotations).Client
	if len(got.Methods) != 1 {
		t.Fatalf("got %d methods, want 1", len(got.Methods))
	}
	m := got.Methods[0]
	if !m.HasFlattenedParams || len(m.FlattenedParams) != 2 {
		t.Fatalf("got %d flattened params, want 2", len(m.FlattenedParams))
	}
	wantParam := &ClientFlattenedParam{
		Name:       "status",
		Key:        "status",
		TypeIdent:  "action_service.OuterRequest.Status",
		SphinxType: "google.cloud.test_v1.types.OuterRequest.Status",
	}
	if diff := cmp.Diff(wantParam, m.FlattenedParams[1], cmpopts.IgnoreFields(ClientFlattenedParam{}, "DocLines")); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestAnnotateClient_RoutingHeaders(t *testing.T) {
	tableField := api.NewTestField("table").WithType(api.TypezString)
	appProfileField := api.NewTestField("app_profile_id").WithType(api.TypezString)
	reqMsg := api.NewTestMessage("ReadRowRequest").WithPackage("google.cloud.test.v1").WithFields(tableField, appProfileField)

	method := api.NewTestMethod("ReadRow").WithInput(reqMsg)
	method.Routing = []*api.RoutingInfo{
		{
			Name: "table_name",
			Variants: []*api.RoutingInfoVariant{
				{FieldPath: []string{"table"}},
			},
		},
		{
			Name: "app_profile_id",
			Variants: []*api.RoutingInfoVariant{
				{FieldPath: []string{"app_profile_id"}},
			},
		},
	}

	svc := api.NewTestService("DataService").WithMethods(method)
	model := api.NewTestAPI([]*api.Message{reqMsg}, nil, []*api.Service{svc}).
		WithDefinitionLocation(reqMsg.ID, "google/cloud/test/v1/data_service.proto", 10)

	c := testCodec(t, model)
	if err := c.annotateModel(); err != nil {
		t.Fatal(err)
	}

	got := svc.Codec.(*ServiceAnnotations).Client
	if len(got.Methods) != 1 {
		t.Fatalf("got %d methods, want 1", len(got.Methods))
	}
	m := got.Methods[0]
	if !m.HasRouting || len(m.RoutingHeaders) != 2 {
		t.Fatalf("got %d routing headers, want 2", len(m.RoutingHeaders))
	}
	want := []*ClientRoutingHeader{
		{HeaderKey: "table_name", FieldPath: "request.table"},
		{HeaderKey: "app_profile_id", FieldPath: "request.app_profile_id"},
	}
	if diff := cmp.Diff(want, m.RoutingHeaders, cmpopts.EquateEmpty()); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestAnnotateClient_ResourcePaths(t *testing.T) {
	res := &api.Resource{
		Type: "test.googleapis.com/Book",
		Patterns: []api.ResourcePattern{
			{
				{Literal: "shelves"},
				{Variable: api.NewPathVariable("shelf")},
				{Literal: "books"},
				{Variable: api.NewPathVariable("book")},
			},
			{
				{Literal: "libraries"},
				{Variable: api.NewPathVariable("library")},
				{Literal: "books"},
				{Variable: api.NewPathVariable("book")},
			},
		},
	}
	bookMsg := api.NewTestMessage("Book").WithResource(res)
	method := api.NewTestMethod("GetBook").WithOutput(bookMsg)

	svc := api.NewTestService("LibraryService").WithMethods(method)

	model := api.NewTestAPI([]*api.Message{bookMsg}, nil, []*api.Service{svc})
	c := testCodec(t, model)
	if err := c.annotateModel(); err != nil {
		t.Fatal(err)
	}

	got := svc.Codec.(*ServiceAnnotations).Client
	if len(got.ResourcePaths) != 1 {
		t.Fatalf("got %d resource paths, want 1 (first pattern only)", len(got.ResourcePaths))
	}
	wantPath := &ClientResourcePath{
		MethodName:   "book",
		PathPattern:  "shelves/{shelf}/books/{book}",
		FormatArgs:   []string{"shelf=shelf", "book=book"},
		HasArgs:      true,
		Args:         []string{"shelf", "book"},
		RegexPattern: "^shelves/(?P<shelf>.+?)/books/(?P<book>.+?)$",
	}
	if diff := cmp.Diff(wantPath, got.ResourcePaths[0]); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestAnnotateClient_OperationsMixin(t *testing.T) {
	method := api.NewTestMethod("GetOperation")
	method.SourceServiceID = ".google.longrunning.Operations"

	svc := api.NewTestService("ItemService").WithMethods(method)
	model := api.NewTestAPI(nil, nil, []*api.Service{svc})

	c := testCodec(t, model)
	if err := c.annotateModel(); err != nil {
		t.Fatal(err)
	}

	got := svc.Codec.(*ServiceAnnotations).Client
	type mixinSummary struct {
		HasOperationsMixin bool
		HasGetOperation    bool
		HasListOperations  bool
	}
	gotMixin := mixinSummary{
		HasOperationsMixin: got.HasOperationsMixin,
		HasGetOperation:    got.HasGetOperation,
		HasListOperations:  got.HasListOperations,
	}
	wantMixin := mixinSummary{
		HasOperationsMixin: true,
		HasGetOperation:    true,
		HasListOperations:  false,
	}
	if diff := cmp.Diff(wantMixin, gotMixin); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestAnnotateClient_SampleSnippet(t *testing.T) {
	subField := api.NewTestField("title").WithType(api.TypezString)
	subField.Behavior = []api.FieldBehavior{api.FieldBehaviorRequired}

	subMsg := api.NewTestMessage("SubItem").WithPackage("google.cloud.test.v1").WithFields(subField)

	itemField := api.NewTestField("item").WithMessageType(subMsg)
	itemField.Behavior = []api.FieldBehavior{api.FieldBehaviorRequired}

	nameField := api.NewTestField("name").WithType(api.TypezString)
	nameField.Behavior = []api.FieldBehavior{api.FieldBehaviorRequired}

	reqMsg := api.NewTestMessage("CreateItemRequest").WithPackage("google.cloud.test.v1").WithFields(nameField, itemField)

	method := api.NewTestMethod("CreateItem").WithInput(reqMsg)
	svc := api.NewTestService("ItemService").WithPackage("google.cloud.test.v1").WithMethods(method)

	model := api.NewTestAPI([]*api.Message{subMsg, reqMsg}, nil, []*api.Service{svc}).
		WithDefinitionLocation(reqMsg.ID, "google/cloud/test/v1/item_service.proto", 10).
		WithDefinitionLocation(subMsg.ID, "google/cloud/test/v1/item_service.proto", 20)

	c := testCodec(t, model)
	if err := c.annotateModel(); err != nil {
		t.Fatal(err)
	}

	got := svc.Codec.(*ServiceAnnotations).Client
	if len(got.Methods) != 1 {
		t.Fatalf("got %d methods, want 1", len(got.Methods))
	}
	m := got.Methods[0]

	wantSubmessages := []*ClientSampleSubmessage{
		{
			VarName:  "item",
			TypeName: "SubItem",
			Assignments: []*ClientSampleAssignment{
				{
					FieldPath: "title",
					Value:     `"title_value"`,
				},
			},
		},
	}
	if diff := cmp.Diff(wantSubmessages, m.SampleSubmessages); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}

	wantReqFields := []*ClientSampleRequestField{
		{FieldName: "name", Value: `"name_value"`},
		{FieldName: "item", Value: "item"},
	}
	if diff := cmp.Diff(wantReqFields, m.SampleRequestFields); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}
