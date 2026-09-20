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

func testCodec(t *testing.T, model *api.API) *codec {
	t.Helper()
	lib := &config.Library{
		Name:          "google-cloud-test",
		Version:       "1.0.0",
		CopyrightYear: "2026",
		APIs: []*config.API{
			{Path: "google/cloud/test/v1"},
		},
		Python: &config.PythonPackage{
			DefaultVersion: "v1",
		},
	}
	return newTestCodec(t, model, lib)
}

func TestAnnotatePagers_NoPagers(t *testing.T) {
	svc := api.NewTestService("SimpleService").WithMethods(
		api.NewTestMethod("GetItem").
			WithInput(api.NewTestMessage("GetItemRequest")).
			WithOutput(api.NewTestMessage("Item")),
	)
	model := api.NewTestAPI(nil, nil, []*api.Service{svc})
	c := testCodec(t, model)
	got := c.annotatePagers(svc)

	if len(got.Pagers) != 0 {
		t.Errorf("got %d pagers, want 0", len(got.Pagers))
	}
	if len(got.TypeImports) != 0 {
		t.Errorf("got %d type imports, want 0", len(got.TypeImports))
	}
	if got.VersionPackage != "google.cloud.test_v1" {
		t.Errorf("VersionPackage = %q, want %q", got.VersionPackage, "google.cloud.test_v1")
	}
}

func TestAnnotatePagers_StandardList(t *testing.T) {
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
	got := c.annotatePagers(svc)

	want := &PagersAnnotations{
		VersionPackage: "google.cloud.test_v1",
		TypeImports:    []string{"item_service", "items"},
		Pagers: []*PagerAnnotations{
			{
				MethodName:      "ListItems",
				MethodSnakeName: "list_items",
				RequestIdent:    "item_service.ListItemsRequest",
				ResponseIdent:   "item_service.ListItemsResponse",
				RequestDocType:  "google.cloud.test_v1.types.ListItemsRequest",
				ResponseDocType: "google.cloud.test_v1.types.ListItemsResponse",
				ItemFieldName:   "items",
				ItemTypeIdent:   "items.Item",
				IsMap:           false,
				HasNext:         false,
			},
		},
	}

	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestAnnotatePagers_MultiplePagersWithHasNext(t *testing.T) {
	item1Msg := api.NewTestMessage("Item1").WithPackage("google.cloud.test.v1")
	items1Field := api.NewTestField("items1").WithMessageType(item1Msg).WithRepeated()

	item2Msg := api.NewTestMessage("Item2").WithPackage("google.cloud.test.v1")
	items2Field := api.NewTestField("items2").WithMessageType(item2Msg).WithRepeated()

	req1 := api.NewTestMessage("List1Request").WithPackage("google.cloud.test.v1")
	resp1 := api.NewTestMessage("List1Response").WithPackage("google.cloud.test.v1").WithFields(items1Field)
	resp1.Pagination = &api.PaginationInfo{PageableItem: items1Field}

	req2 := api.NewTestMessage("List2Request").WithPackage("google.cloud.test.v1")
	resp2 := api.NewTestMessage("List2Response").WithPackage("google.cloud.test.v1").WithFields(items2Field)
	resp2.Pagination = &api.PaginationInfo{PageableItem: items2Field}

	tokenField := api.NewTestField("page_token").WithType(api.TypezString)
	m1 := api.NewTestMethod("List1").WithInput(req1).WithOutput(resp1)
	m1.Pagination = tokenField
	m2 := api.NewTestMethod("List2").WithInput(req2).WithOutput(resp2)
	m2.Pagination = tokenField

	svc := api.NewTestService("MultiService").WithMethods(m1, m2)
	model := api.NewTestAPI([]*api.Message{item1Msg, item2Msg, req1, resp1, req2, resp2}, nil, []*api.Service{svc}).
		WithDefinitionLocation(req1.ID, "google/cloud/test/v1/service.proto", 10).
		WithDefinitionLocation(resp1.ID, "google/cloud/test/v1/service.proto", 20).
		WithDefinitionLocation(item1Msg.ID, "google/cloud/test/v1/service.proto", 30).
		WithDefinitionLocation(req2.ID, "google/cloud/test/v1/service.proto", 40).
		WithDefinitionLocation(resp2.ID, "google/cloud/test/v1/service.proto", 50).
		WithDefinitionLocation(item2Msg.ID, "google/cloud/test/v1/service.proto", 60)

	c := testCodec(t, model)
	got := c.annotatePagers(svc)

	if len(got.Pagers) != 2 {
		t.Fatalf("got %d pagers, want 2", len(got.Pagers))
	}
	if !got.Pagers[0].HasNext {
		t.Errorf("pagers[0].HasNext = false, want true")
	}
	if got.Pagers[1].HasNext {
		t.Errorf("pagers[1].HasNext = true, want false")
	}
	if diff := cmp.Diff([]string{"service"}, got.TypeImports); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestAnnotatePagers_NestedMessageItem(t *testing.T) {
	respMsg := api.NewTestMessage("AnalyzePoliciesResponse").WithPackage("google.cloud.test.v1")

	nestedMsg := api.NewTestMessage("PolicyResult")
	nestedMsg.ID = ".google.cloud.test.v1.AnalyzePoliciesResponse.PolicyResult"
	nestedMsg.Parent = respMsg

	resultsField := api.NewTestField("results").WithMessageType(nestedMsg).WithRepeated()

	respMsg.Fields = append(respMsg.Fields, resultsField)
	respMsg.Pagination = &api.PaginationInfo{PageableItem: resultsField}

	reqMsg := api.NewTestMessage("AnalyzePoliciesRequest").WithPackage("google.cloud.test.v1")

	tokenField := api.NewTestField("page_token").WithType(api.TypezString)
	method := api.NewTestMethod("AnalyzePolicies").WithInput(reqMsg).WithOutput(respMsg)
	method.Pagination = tokenField

	svc := api.NewTestService("PolicyService").WithMethods(method)
	model := api.NewTestAPI([]*api.Message{respMsg, nestedMsg, reqMsg}, nil, []*api.Service{svc}).
		WithDefinitionLocation(reqMsg.ID, "google/cloud/test/v1/policy.proto", 10).
		WithDefinitionLocation(respMsg.ID, "google/cloud/test/v1/policy.proto", 20).
		WithDefinitionLocation(nestedMsg.ID, "google/cloud/test/v1/policy.proto", 30)

	c := testCodec(t, model)
	got := c.annotatePagers(svc)

	if len(got.Pagers) != 1 {
		t.Fatalf("got %d pagers, want 1", len(got.Pagers))
	}
	p := got.Pagers[0]
	if p.ItemTypeIdent != "policy.AnalyzePoliciesResponse.PolicyResult" {
		t.Errorf("ItemTypeIdent = %q, want %q", p.ItemTypeIdent, "policy.AnalyzePoliciesResponse.PolicyResult")
	}
	if p.ResponseDocType != "google.cloud.test_v1.types.AnalyzePoliciesResponse" {
		t.Errorf("ResponseDocType = %q, want %q", p.ResponseDocType, "google.cloud.test_v1.types.AnalyzePoliciesResponse")
	}
}

func TestAnnotatePagers_MapItem(t *testing.T) {
	valMsg := api.NewTestMessage("MetricValue").WithPackage("google.cloud.test.v1")

	entryKey := api.NewTestField("key").WithType(api.TypezString)
	entryVal := api.NewTestField("value").WithMessageType(valMsg)

	entryMsg := api.NewTestMessage("MetricsEntry").WithPackage("google.cloud.test.v1").WithFields(entryKey, entryVal)
	entryMsg.IsMap = true

	mapField := api.NewTestField("metrics").WithMessageType(entryMsg).WithMap()

	reqMsg := api.NewTestMessage("GetMetricsRequest").WithPackage("google.cloud.test.v1")

	respMsg := api.NewTestMessage("GetMetricsResponse").WithPackage("google.cloud.test.v1").WithFields(mapField)
	respMsg.Pagination = &api.PaginationInfo{PageableItem: mapField}

	tokenField := api.NewTestField("page_token").WithType(api.TypezString)
	method := api.NewTestMethod("GetMetrics").WithInput(reqMsg).WithOutput(respMsg)
	method.Pagination = tokenField

	svc := api.NewTestService("MetricsService").WithMethods(method)
	model := api.NewTestAPI([]*api.Message{reqMsg, respMsg, entryMsg, valMsg}, nil, []*api.Service{svc}).
		WithDefinitionLocation(reqMsg.ID, "google/cloud/test/v1/metrics.proto", 10).
		WithDefinitionLocation(respMsg.ID, "google/cloud/test/v1/metrics.proto", 20).
		WithDefinitionLocation(valMsg.ID, "google/cloud/test/v1/metrics.proto", 30)

	c := testCodec(t, model)
	got := c.annotatePagers(svc)

	if len(got.Pagers) != 1 {
		t.Fatalf("got %d pagers, want 1", len(got.Pagers))
	}
	p := got.Pagers[0]
	if !p.IsMap {
		t.Errorf("p.IsMap = false, want true")
	}
	if p.ItemTypeIdent != "metrics.MetricValue" {
		t.Errorf("ItemTypeIdent = %q, want %q", p.ItemTypeIdent, "metrics.MetricValue")
	}
}

func TestAnnotatePagers_PrimitiveItem(t *testing.T) {
	for _, test := range []struct {
		name     string
		typez    api.Typez
		wantType string
	}{
		{"string", api.TypezString, "str"},
		{"bytes", api.TypezBytes, "bytes"},
		{"int32", api.TypezInt32, "int"},
		{"int64", api.TypezInt64, "int"},
		{"uint32", api.TypezUint32, "int"},
		{"bool", api.TypezBool, "bool"},
		{"float", api.TypezFloat, "float"},
		{"double", api.TypezDouble, "float"},
	} {
		t.Run(test.name, func(t *testing.T) {
			itemField := api.NewTestField("values").WithType(test.typez).WithRepeated()
			reqMsg := api.NewTestMessage("ListValuesRequest").WithPackage("google.cloud.test.v1")
			respMsg := api.NewTestMessage("ListValuesResponse").WithPackage("google.cloud.test.v1").WithFields(itemField)
			respMsg.Pagination = &api.PaginationInfo{PageableItem: itemField}

			tokenField := api.NewTestField("page_token").WithType(api.TypezString)
			method := api.NewTestMethod("ListValues").WithInput(reqMsg).WithOutput(respMsg)
			method.Pagination = tokenField

			svc := api.NewTestService("ValueService").WithMethods(method)
			model := api.NewTestAPI([]*api.Message{reqMsg, respMsg}, nil, []*api.Service{svc}).
				WithDefinitionLocation(reqMsg.ID, "google/cloud/test/v1/values.proto", 10).
				WithDefinitionLocation(respMsg.ID, "google/cloud/test/v1/values.proto", 20)

			c := testCodec(t, model)
			got := c.annotatePagers(svc)

			if len(got.Pagers) != 1 {
				t.Fatalf("got %d pagers, want 1", len(got.Pagers))
			}
			p := got.Pagers[0]
			if p.ItemTypeIdent != test.wantType {
				t.Errorf("ItemTypeIdent = %q, want %q", p.ItemTypeIdent, test.wantType)
			}
		})
	}
}

func TestAnnotatePagers_EnumItem(t *testing.T) {
	enum := api.NewTestEnum("Status").WithPackage("google.cloud.test.v1")
	enumField := api.NewTestField("statuses").WithType(api.TypezEnum).WithRepeated()
	enumField.TypezID = enum.ID
	enumField.EnumType = enum

	reqMsg := api.NewTestMessage("ListStatusesRequest").WithPackage("google.cloud.test.v1")
	respMsg := api.NewTestMessage("ListStatusesResponse").WithPackage("google.cloud.test.v1").WithFields(enumField)
	respMsg.Pagination = &api.PaginationInfo{PageableItem: enumField}

	tokenField := api.NewTestField("page_token").WithType(api.TypezString)
	method := api.NewTestMethod("ListStatuses").WithInput(reqMsg).WithOutput(respMsg)
	method.Pagination = tokenField

	svc := api.NewTestService("StatusService").WithMethods(method)
	model := api.NewTestAPI([]*api.Message{reqMsg, respMsg}, []*api.Enum{enum}, []*api.Service{svc}).
		WithDefinitionLocation(reqMsg.ID, "google/cloud/test/v1/status.proto", 10).
		WithDefinitionLocation(respMsg.ID, "google/cloud/test/v1/status.proto", 20).
		WithDefinitionLocation(enum.ID, "google/cloud/test/v1/status.proto", 30)

	c := testCodec(t, model)
	got := c.annotatePagers(svc)

	if len(got.Pagers) != 1 {
		t.Fatalf("got %d pagers, want 1", len(got.Pagers))
	}
	p := got.Pagers[0]
	if p.ItemTypeIdent != "status.Status" {
		t.Errorf("ItemTypeIdent = %q, want %q", p.ItemTypeIdent, "status.Status")
	}
}

func TestAnnotatePagers_IgnoresMixins(t *testing.T) {
	itemMsg := api.NewTestMessage("Operation").WithPackage("google.longrunning")
	itemsField := api.NewTestField("operations").WithMessageType(itemMsg).WithRepeated()

	reqMsg := api.NewTestMessage("ListOperationsRequest").WithPackage("google.longrunning")
	respMsg := api.NewTestMessage("ListOperationsResponse").WithPackage("google.longrunning").WithFields(itemsField)
	respMsg.Pagination = &api.PaginationInfo{PageableItem: itemsField}

	tokenField := api.NewTestField("page_token").WithType(api.TypezString)
	mixinMethod := api.NewTestMethod("ListOperations").WithInput(reqMsg).WithOutput(respMsg)
	mixinMethod.Pagination = tokenField
	mixinMethod.SourceServiceID = "google.longrunning.Operations"

	svc := api.NewTestService("MyService")
	svc.ID = "google.cloud.test.v1.MyService"
	svc.WithMethods(mixinMethod)

	model := api.NewTestAPI([]*api.Message{itemMsg, reqMsg, respMsg}, nil, []*api.Service{svc})
	c := testCodec(t, model)
	got := c.annotatePagers(svc)

	if len(got.Pagers) != 0 {
		t.Errorf("got %d pagers, want 0 (mixin methods should be skipped)", len(got.Pagers))
	}
}

func TestAnnotatePagers_HelperFunctions(t *testing.T) {
	t.Run("primitiveTypeName", func(t *testing.T) {
		for _, test := range []struct {
			typez api.Typez
			want  string
		}{
			{api.TypezString, "str"},
			{api.TypezBytes, "bytes"},
			{api.TypezBool, "bool"},
			{api.TypezFloat, "float"},
			{api.TypezDouble, "float"},
			{api.TypezInt32, "int"},
			{api.TypezInt64, "int"},
			{api.TypezUint32, "int"},
			{api.TypezUint64, "int"},
			{api.TypezSint32, "int"},
			{api.TypezSint64, "int"},
			{api.TypezFixed32, "int"},
			{api.TypezFixed64, "int"},
			{api.TypezSfixed32, "int"},
			{api.TypezSfixed64, "int"},
			{api.TypezMessage, "Any"},
		} {
			if got := primitiveTypeName(test.typez); got != test.want {
				t.Errorf("primitiveTypeName(%v) = %q, want %q", test.typez, got, test.want)
			}
		}
	})

	t.Run("resolveMessageRelativeName", func(t *testing.T) {
		if got := resolveMessageRelativeName(nil); got != "" {
			t.Errorf("resolveMessageRelativeName(nil) = %q, want empty", got)
		}

		root := api.NewTestMessage("Root")
		if got := resolveMessageRelativeName(root); got != "Root" {
			t.Errorf("resolveMessageRelativeName(root) = %q, want Root", got)
		}

		child := api.NewTestMessage("Child")
		child.Parent = root
		if got := resolveMessageRelativeName(child); got != "Root.Child" {
			t.Errorf("resolveMessageRelativeName(child) = %q, want Root.Child", got)
		}

		grandchild := api.NewTestMessage("Grandchild")
		grandchild.Parent = child
		if got := resolveMessageRelativeName(grandchild); got != "Root.Child.Grandchild" {
			t.Errorf("resolveMessageRelativeName(grandchild) = %q, want Root.Child.Grandchild", got)
		}
	})

	t.Run("resolveEnumRelativeName", func(t *testing.T) {
		if got := resolveEnumRelativeName(nil); got != "" {
			t.Errorf("resolveEnumRelativeName(nil) = %q, want empty", got)
		}

		rootEnum := api.NewTestEnum("StandaloneEnum")
		if got := resolveEnumRelativeName(rootEnum); got != "StandaloneEnum" {
			t.Errorf("resolveEnumRelativeName(rootEnum) = %q, want StandaloneEnum", got)
		}

		parentMsg := api.NewTestMessage("ParentMsg")
		nestedEnum := api.NewTestEnum("NestedEnum")
		nestedEnum.Parent = parentMsg
		if got := resolveEnumRelativeName(nestedEnum); got != "ParentMsg.NestedEnum" {
			t.Errorf("resolveEnumRelativeName(nestedEnum) = %q, want ParentMsg.NestedEnum", got)
		}
	})

	t.Run("typeNameWithFallback", func(t *testing.T) {
		if got := typeNameWithFallback("", "Default"); got != "Default" {
			t.Errorf("typeNameWithFallback(\"\", \"Default\") = %q, want Default", got)
		}
		if got := typeNameWithFallback(".google.cloud.Foo", "Default"); got != "Foo" {
			t.Errorf("typeNameWithFallback(\".google.cloud.Foo\", \"Default\") = %q, want Foo", got)
		}
	})
}

func TestResolveTypeModule_AncestralFallback(t *testing.T) {
	model := api.NewTestAPI(nil, nil, nil).
		WithDefinitionLocation(".google.cloud.test.v1.ParentMessage", "google/cloud/test/v1/parent.proto", 10)
	c := testCodec(t, model)

	got := c.resolveTypeModule(".google.cloud.test.v1.ParentMessage.NestedMessage", nil)
	if got != "parent" {
		t.Errorf("resolveTypeModule() = %q, want %q", got, "parent")
	}

	gotDeep := c.resolveTypeModule(".google.cloud.test.v1.ParentMessage.NestedMessage.DeeplyNested", nil)
	if gotDeep != "parent" {
		t.Errorf("resolveTypeModule() = %q, want %q", gotDeep, "parent")
	}

	gotUnknown := c.resolveTypeModule(".unknown.pkg.Type", nil)
	if gotUnknown != "common" {
		t.Errorf("resolveTypeModule() = %q, want %q", gotUnknown, "common")
	}
}
