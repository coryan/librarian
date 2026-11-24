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

package swift

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/sidekick/api"
)

func TestResolveRoutingFieldPath(t *testing.T) {
	statusEnum := api.NewTestEnum("Status").
		WithPackage("google.cloud.test.v1").
		WithValues(
			api.NewTestEnumValue("STATUS_UNSPECIFIED", 0),
			api.NewTestEnumValue("STATUS_ACTIVE", 1),
		)
	extEnum := api.NewTestEnum("ExtMode").
		WithPackage("google.type").
		WithValues(
			api.NewTestEnumValue("EXT_MODE_UNSPECIFIED", 0),
			api.NewTestEnumValue("EXT_MODE_ON", 1),
		)
	unconfiguredEnum := api.NewTestEnum("UnconfiguredEnum").
		WithPackage("google.unconfigured").
		WithValues(
			api.NewTestEnumValue("UNCONFIGURED_UNSPECIFIED", 0),
		)

	extContainerMsg := api.NewTestMessage("ExtContainer").
		WithPackage("google.type").
		WithFields(
			api.NewTestField("rank").WithType(api.TypezInt32),
			api.NewTestField("unconfigured_sibling").WithEnumType(unconfiguredEnum),
		)

	innerTargetOneOf := api.NewTestOneOf("target").
		WithFields(
			api.NewTestField("resource").WithType(api.TypezString),
			api.NewTestField("target_id").WithType(api.TypezInt64),
		)
	innerMsg := api.NewTestMessage("Inner").
		WithPackage("google.cloud.test.v1").
		WithFields(
			api.NewTestField("count").WithType(api.TypezInt32),
			api.NewTestField("opt_count").WithType(api.TypezInt32).WithOptional(),
			api.NewTestField("status").WithEnumType(statusEnum),
			api.NewTestField("opt_status").WithEnumType(statusEnum).WithOptional(),
		).
		WithOneOfs(innerTargetOneOf)

	nodeMsg := api.NewTestMessage("Node").
		WithPackage("google.cloud.test.v1").
		WithFields(
			api.NewTestField("label").WithType(api.TypezString),
			api.NewTestField("rank").WithType(api.TypezInt32),
		).
		WithOneOfs(
			api.NewTestOneOf("node_target").WithFields(
				api.NewTestField("node_id").WithType(api.TypezInt64),
			),
		)
	nodeMsg.WithFields(
		api.NewTestField("child").WithMessageType(nodeMsg).WithOptional(),
	)

	indirectAMsg := api.NewTestMessage("IndirectA").
		WithPackage("google.cloud.test.v1").
		WithFields(
			api.NewTestField("code").WithType(api.TypezInt32),
		)
	indirectBMsg := api.NewTestMessage("IndirectB").
		WithPackage("google.cloud.test.v1").
		WithFields(
			api.NewTestField("to_a").WithMessageType(indirectAMsg).WithOptional(),
		)
	indirectAMsg.WithFields(
		api.NewTestField("to_b").WithMessageType(indirectBMsg).WithOptional(),
	)

	requestMsg := api.NewTestMessage("Request").
		WithPackage("google.cloud.test.v1").
		WithFields(
			api.NewTestField("name").WithType(api.TypezString),
			api.NewTestField("opt_name").WithType(api.TypezString).WithOptional(),
			api.NewTestField("page_size").WithType(api.TypezInt32),
			api.NewTestField("opt_page_size").WithType(api.TypezInt32).WithOptional(),
			api.NewTestField("ratio").WithType(api.TypezDouble),
			api.NewTestField("enabled").WithType(api.TypezBool),
			api.NewTestField("status").WithEnumType(statusEnum),
			api.NewTestField("opt_status").WithEnumType(statusEnum).WithOptional(),
			api.NewTestField("ext_mode").WithEnumType(extEnum),
			api.NewTestField("ext_container").WithMessageType(extContainerMsg).WithOptional(),
			api.NewTestField("req_inner").WithMessageType(innerMsg),
			api.NewTestField("info").WithMessageType(innerMsg).WithOptional(),
			api.NewTestField("node").WithMessageType(nodeMsg).WithOptional(),
			api.NewTestField("chain").WithMessageType(indirectAMsg).WithOptional(),
			api.NewTestField("str_wrapper").WithType(api.TypezMessage).WithTypezID(api.WktStringValueID).WithOptional(),
			api.NewTestField("int64_wrapper").WithType(api.TypezMessage).WithTypezID(api.WktInt64ValueID).WithOptional(),
			api.NewTestField("bool_wrapper").WithType(api.TypezMessage).WithTypezID(api.WktBoolValueID).WithOptional(),
		).
		WithOneOfs(
			api.NewTestOneOf("source").WithFields(
				api.NewTestField("bucket").WithType(api.TypezString),
				api.NewTestField("project_num").WithType(api.TypezInt64),
				api.NewTestField("oneof_status").WithEnumType(statusEnum),
				api.NewTestField("nested_inner").WithMessageType(innerMsg),
				api.NewTestField("oneof_wrapper").WithType(api.TypezMessage).WithTypezID(api.WktInt64ValueID),
			),
		)

	makeCodec := func(specFormat string) *codec {
		model := api.NewTestAPI(
			[]*api.Message{innerMsg, nodeMsg, indirectAMsg, indirectBMsg, requestMsg},
			[]*api.Enum{statusEnum, extEnum},
			nil,
		).WithPackageName("google.cloud.test.v1")
		model.AddMessage(extContainerMsg)
		model.AddEnum(unconfiguredEnum)
		if err := api.CrossReference(model); err != nil {
			t.Fatal(err)
		}
		api.LabelRecursiveFields(model)
		lib := &config.Library{
			SpecificationFormat: specFormat,
			Swift: swiftConfig(t, []config.SwiftDependency{
				{Name: "GoogleType", ApiPackage: "google.type"},
			}),
		}
		c := newTestCodec(t, model, lib)
		if err := c.annotateModel(); err != nil {
			t.Fatal(err)
		}
		return c
	}

	protoCodec := makeCodec(config.SpecProtobuf)
	discoveryCodec := makeCodec(config.SpecDiscovery)

	for _, test := range []struct {
		name      string
		c         *codec
		fieldPath []string
		want      *routingParamVariant
	}{
		{
			name:      "empty field path",
			c:         protoCodec,
			fieldPath: nil,
			want:      nil,
		},
		{
			name:      "non-optional string",
			c:         protoCodec,
			fieldPath: []string{"name"},
			want: &routingParamVariant{
				FieldSteps: []*routingFieldStep{{PropertyName: "name"}},
			},
		},
		{
			name:      "optional string",
			c:         protoCodec,
			fieldPath: []string{"opt_name"},
			want: &routingParamVariant{
				FieldSteps: []*routingFieldStep{{PropertyName: "optName"}},
				IsOptional: true,
			},
		},
		{
			name:      "non-optional int32 scalar",
			c:         protoCodec,
			fieldPath: []string{"page_size"},
			want: &routingParamVariant{
				FieldSteps:      []*routingFieldStep{{PropertyName: "pageSize"}},
				ConvertToString: true,
			},
		},
		{
			name:      "optional int32 scalar",
			c:         protoCodec,
			fieldPath: []string{"opt_page_size"},
			want: &routingParamVariant{
				FieldSteps:      []*routingFieldStep{{PropertyName: "optPageSize"}},
				ConvertToString: true,
				IsOptional:      true,
			},
		},
		{
			name:      "double scalar",
			c:         protoCodec,
			fieldPath: []string{"ratio"},
			want: &routingParamVariant{
				FieldSteps:      []*routingFieldStep{{PropertyName: "ratio"}},
				ConvertToString: true,
			},
		},
		{
			name:      "bool scalar",
			c:         protoCodec,
			fieldPath: []string{"enabled"},
			want: &routingParamVariant{
				FieldSteps:      []*routingFieldStep{{PropertyName: "enabled"}},
				ConvertToString: true,
			},
		},
		{
			name:      "non-optional protobuf enum",
			c:         protoCodec,
			fieldPath: []string{"status"},
			want: &routingParamVariant{
				FieldSteps: []*routingFieldStep{{PropertyName: "status"}},
				IsEnum:     true,
			},
		},
		{
			name:      "optional protobuf enum",
			c:         protoCodec,
			fieldPath: []string{"opt_status"},
			want: &routingParamVariant{
				FieldSteps: []*routingFieldStep{{PropertyName: "optStatus"}},
				IsEnum:     true,
				IsOptional: true,
			},
		},
		{
			name:      "external enum records required package",
			c:         protoCodec,
			fieldPath: []string{"ext_mode"},
			want: &routingParamVariant{
				FieldSteps:       []*routingFieldStep{{PropertyName: "extMode"}},
				IsEnum:           true,
				RequiredPackages: []string{"google.type"},
			},
		},
		{
			name:      "external unannotated intermediate ignores unconfigured sibling",
			c:         protoCodec,
			fieldPath: []string{"ext_container", "rank"},
			want: &routingParamVariant{
				FieldSteps: []*routingFieldStep{
					{PropertyName: "extContainer"},
					{PropertyName: "rank", ChainOptional: true},
				},
				ConvertToString:  true,
				IsOptional:       true,
				NeedsParens:      true,
				RequiredPackages: []string{"google.type"},
			},
		},
		{
			name:      "discovery string enum non-optional",
			c:         discoveryCodec,
			fieldPath: []string{"status"},
			want: &routingParamVariant{
				FieldSteps:   []*routingFieldStep{{PropertyName: "status"}},
				IsEnum:       true,
				IsStringEnum: true,
			},
		},
		{
			name:      "discovery string enum optional",
			c:         discoveryCodec,
			fieldPath: []string{"opt_status"},
			want: &routingParamVariant{
				FieldSteps:   []*routingFieldStep{{PropertyName: "optStatus"}},
				IsEnum:       true,
				IsStringEnum: true,
				IsOptional:   true,
			},
		},
		{
			name:      "oneof string leaf",
			c:         protoCodec,
			fieldPath: []string{"bucket"},
			want: &routingParamVariant{
				FieldSteps: []*routingFieldStep{
					{
						PropertyName:   "source",
						IsOneOf:        true,
						OneOfCaseName:  "bucket",
						OneOfValueType: "Swift.String",
					},
				},
				IsOptional: true,
			},
		},
		{
			name:      "oneof int64 leaf",
			c:         protoCodec,
			fieldPath: []string{"project_num"},
			want: &routingParamVariant{
				FieldSteps: []*routingFieldStep{
					{
						PropertyName:   "source",
						IsOneOf:        true,
						OneOfCaseName:  "projectNum",
						OneOfValueType: "Swift.Int64",
					},
				},
				ConvertToString: true,
				IsOptional:      true,
			},
		},
		{
			name:      "oneof enum leaf",
			c:         protoCodec,
			fieldPath: []string{"oneof_status"},
			want: &routingParamVariant{
				FieldSteps: []*routingFieldStep{
					{
						PropertyName:   "source",
						IsOneOf:        true,
						OneOfCaseName:  "oneofStatus",
						OneOfValueType: "Status",
					},
				},
				IsEnum:     true,
				IsOptional: true,
			},
		},
		{
			name:      "oneof message intermediate to nested scalar",
			c:         protoCodec,
			fieldPath: []string{"nested_inner", "count"},
			want: &routingParamVariant{
				FieldSteps: []*routingFieldStep{
					{
						PropertyName:   "source",
						IsOneOf:        true,
						OneOfCaseName:  "nestedInner",
						OneOfValueType: "Inner",
					},
					{
						PropertyName:  "count",
						ChainOptional: true,
					},
				},
				ConvertToString: true,
				IsOptional:      true,
				NeedsParens:     true,
			},
		},
		{
			name:      "oneof message intermediate to nested oneof int64",
			c:         protoCodec,
			fieldPath: []string{"nested_inner", "target_id"},
			want: &routingParamVariant{
				FieldSteps: []*routingFieldStep{
					{
						PropertyName:   "source",
						IsOneOf:        true,
						OneOfCaseName:  "nestedInner",
						OneOfValueType: "Inner",
					},
					{
						PropertyName:   "target",
						ChainOptional:  true,
						IsOneOf:        true,
						OneOfCaseName:  "targetId",
						OneOfValueType: "Swift.Int64",
					},
				},
				ConvertToString: true,
				IsOptional:      true,
				NeedsParens:     true,
			},
		},
		{
			name:      "non-optional message to optional scalar does not need parens",
			c:         protoCodec,
			fieldPath: []string{"req_inner", "opt_count"},
			want: &routingParamVariant{
				FieldSteps: []*routingFieldStep{
					{PropertyName: "reqInner"},
					{PropertyName: "optCount"},
				},
				ConvertToString: true,
				IsOptional:      true,
				NeedsParens:     false,
			},
		},
		{
			name:      "nested optional message to scalar",
			c:         protoCodec,
			fieldPath: []string{"info", "count"},
			want: &routingParamVariant{
				FieldSteps: []*routingFieldStep{
					{PropertyName: "info"},
					{PropertyName: "count", ChainOptional: true},
				},
				ConvertToString: true,
				IsOptional:      true,
				NeedsParens:     true,
			},
		},
		{
			name:      "nested optional message to enum",
			c:         protoCodec,
			fieldPath: []string{"info", "opt_status"},
			want: &routingParamVariant{
				FieldSteps: []*routingFieldStep{
					{PropertyName: "info"},
					{PropertyName: "optStatus", ChainOptional: true},
				},
				IsEnum:      true,
				IsOptional:  true,
				NeedsParens: true,
			},
		},
		{
			name:      "recursive message chain through WKTRecursive",
			c:         protoCodec,
			fieldPath: []string{"node", "child", "rank"},
			want: &routingParamVariant{
				FieldSteps: []*routingFieldStep{
					{PropertyName: "node"},
					{PropertyName: "child", ChainOptional: true, UnwrapRecursive: true},
					{PropertyName: "rank"},
				},
				ConvertToString:  true,
				IsOptional:       true,
				NeedsParens:      true,
				RequiredPackages: []string{"google.protobuf"},
			},
		},
		{
			name:      "recursive WKTRecursive to oneof leaf",
			c:         protoCodec,
			fieldPath: []string{"node", "child", "node_id"},
			want: &routingParamVariant{
				FieldSteps: []*routingFieldStep{
					{PropertyName: "node"},
					{PropertyName: "child", ChainOptional: true, UnwrapRecursive: true},
					{
						PropertyName:   "nodeTarget",
						IsOneOf:        true,
						OneOfCaseName:  "nodeId",
						OneOfValueType: "Swift.Int64",
					},
				},
				ConvertToString:  true,
				IsOptional:       true,
				NeedsParens:      true,
				RequiredPackages: []string{"google.protobuf"},
			},
		},
		{
			name:      "indirect recursion labeled via api.LabelRecursiveFields",
			c:         protoCodec,
			fieldPath: []string{"chain", "to_b", "to_a", "code"},
			want: &routingParamVariant{
				FieldSteps: []*routingFieldStep{
					{PropertyName: "chain"},
					{PropertyName: "toB", ChainOptional: true, UnwrapRecursive: true},
					{PropertyName: "toA", UnwrapRecursive: true},
					{PropertyName: "code"},
				},
				ConvertToString:  true,
				IsOptional:       true,
				NeedsParens:      true,
				RequiredPackages: []string{"google.protobuf"},
			},
		},
		{
			name:      "WKT StringValue wrapper direct",
			c:         protoCodec,
			fieldPath: []string{"str_wrapper"},
			want: &routingParamVariant{
				FieldSteps:       []*routingFieldStep{{PropertyName: "strWrapper"}},
				IsOptional:       true,
				RequiredPackages: []string{"google.protobuf"},
			},
		},
		{
			name:      "WKT Int64Value wrapper direct",
			c:         protoCodec,
			fieldPath: []string{"int64_wrapper"},
			want: &routingParamVariant{
				FieldSteps:       []*routingFieldStep{{PropertyName: "int64Wrapper"}},
				ConvertToString:  true,
				IsOptional:       true,
				RequiredPackages: []string{"google.protobuf"},
			},
		},
		{
			name:      "WKT BoolValue wrapper via .value",
			c:         protoCodec,
			fieldPath: []string{"bool_wrapper", "value"},
			want: &routingParamVariant{
				FieldSteps:       []*routingFieldStep{{PropertyName: "boolWrapper"}},
				ConvertToString:  true,
				IsOptional:       true,
				RequiredPackages: []string{"google.protobuf"},
			},
		},
		{
			name:      "WKT Int64Value inside oneof",
			c:         protoCodec,
			fieldPath: []string{"oneof_wrapper"},
			want: &routingParamVariant{
				FieldSteps: []*routingFieldStep{
					{
						PropertyName:   "source",
						IsOneOf:        true,
						OneOfCaseName:  "oneofWrapper",
						OneOfValueType: "GoogleWKT.WKTInt64Value",
					},
				},
				ConvertToString:  true,
				IsOptional:       true,
				RequiredPackages: []string{"google.protobuf"},
			},
		},
		{
			name:      "WKT Int64Value inside oneof via .value",
			c:         protoCodec,
			fieldPath: []string{"oneof_wrapper", "value"},
			want: &routingParamVariant{
				FieldSteps: []*routingFieldStep{
					{
						PropertyName:   "source",
						IsOneOf:        true,
						OneOfCaseName:  "oneofWrapper",
						OneOfValueType: "GoogleWKT.WKTInt64Value",
					},
				},
				ConvertToString:  true,
				IsOptional:       true,
				RequiredPackages: []string{"google.protobuf"},
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := test.c.resolveRoutingFieldPath(requestMsg, test.fieldPath)
			if err != nil {
				t.Fatalf("resolveRoutingFieldPath() unexpected error: %v", err)
			}
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestResolveRoutingFieldPath_Error(t *testing.T) {
	innerMsg := api.NewTestMessage("Inner").
		WithPackage("google.cloud.test.v1").
		WithFields(
			api.NewTestField("name").WithType(api.TypezString),
		)
	stringMap := api.NewTestMap("LabelsEntry", api.TypezString, api.TypezString)
	requestMsg := api.NewTestMessage("Request").
		WithPackage("google.cloud.test.v1").
		WithFields(
			api.NewTestField("name").WithType(api.TypezString),
			api.NewTestField("tags").WithType(api.TypezString).WithRepeated(),
			api.NewTestField("labels").WithMapType(stringMap).WithMap(),
			api.NewTestField("payload").WithType(api.TypezBytes),
			api.NewTestField("bytes_wrapper").WithType(api.TypezMessage).WithTypezID(api.WktBytesValueID).WithOptional(),
			api.NewTestField("int64_wrapper").WithType(api.TypezMessage).WithTypezID(api.WktInt64ValueID).WithOptional(),
			api.NewTestField("null_value").WithType(api.TypezEnum).WithTypezID(api.WktNullValueID).WithOptional(),
			api.NewTestField("inner").WithMessageType(innerMsg).WithOptional(),
		)
	model := api.NewTestAPI([]*api.Message{innerMsg, requestMsg}, nil, nil).
		WithPackageName("google.cloud.test.v1").
		WithMaps(stringMap)
	if err := api.CrossReference(model); err != nil {
		t.Fatal(err)
	}
	c := newTestCodec(t, model, &config.Library{Swift: swiftConfig(t, nil)})
	if err := c.annotateModel(); err != nil {
		t.Fatal(err)
	}

	for _, test := range []struct {
		name      string
		message   *api.Message
		fieldPath []string
	}{
		{
			name:      "nil message",
			message:   nil,
			fieldPath: []string{"name"},
		},
		{
			name:      "missing field",
			message:   requestMsg,
			fieldPath: []string{"does_not_exist"},
		},
		{
			name:      "repeated field",
			message:   requestMsg,
			fieldPath: []string{"tags"},
		},
		{
			name:      "map field",
			message:   requestMsg,
			fieldPath: []string{"labels"},
		},
		{
			name:      "bytes field",
			message:   requestMsg,
			fieldPath: []string{"payload"},
		},
		{
			name:      "WKT BytesValue field",
			message:   requestMsg,
			fieldPath: []string{"bytes_wrapper"},
		},
		{
			name:      "WKT NullValue field",
			message:   requestMsg,
			fieldPath: []string{"null_value"},
		},
		{
			name:      "non-value child after WKT wrapper",
			message:   requestMsg,
			fieldPath: []string{"int64_wrapper", "other"},
		},
		{
			name:      "non-message intermediate field",
			message:   requestMsg,
			fieldPath: []string{"name", "sub"},
		},
		{
			name:      "non-wrapper message leaf",
			message:   requestMsg,
			fieldPath: []string{"inner"},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got, err := c.resolveRoutingFieldPath(test.message, test.fieldPath); err == nil {
				t.Errorf("resolveRoutingFieldPath(%v) succeeded with %+v, want error", test.fieldPath, got)
			}
		})
	}
}
