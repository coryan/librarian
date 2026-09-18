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
	"github.com/googleapis/librarian/internal/sidekick/api"
)

func TestClassAndFunctionNamingHelpers(t *testing.T) {
	svc := api.NewTestService("GoldenKitchenSink")
	svc.Package = "google.test.admin.database.v1"
	ann := &serviceAnnotations{
		Name:                 "GoldenKitchenSink",
		RetryableStatusCodes: []string{"kInternal", "kUnavailable"},
		Service:              svc,
	}

	tests := []struct {
		name string
		got  string
		want string
	}{
		{name: "ClientClassName", got: ann.ClientClassName(), want: "GoldenKitchenSinkClient"},
		{name: "ConnectionClassName", got: ann.ConnectionClassName(), want: "GoldenKitchenSinkConnection"},
		{name: "IdempotencyPolicyClassName", got: ann.IdempotencyPolicyClassName(), want: "GoldenKitchenSinkConnectionIdempotencyPolicy"},
		{name: "RetryPolicyClassName", got: ann.RetryPolicyClassName(), want: "GoldenKitchenSinkRetryPolicy"},
		{name: "LimitedErrorCountRetryPolicyClassName", got: ann.LimitedErrorCountRetryPolicyClassName(), want: "GoldenKitchenSinkLimitedErrorCountRetryPolicy"},
		{name: "LimitedTimeRetryPolicyClassName", got: ann.LimitedTimeRetryPolicyClassName(), want: "GoldenKitchenSinkLimitedTimeRetryPolicy"},
		{name: "RetryTraitsClassName", got: ann.RetryTraitsClassName(), want: "GoldenKitchenSinkRetryTraits"},
		{name: "StubClassName", got: ann.StubClassName(), want: "GoldenKitchenSinkStub"},
		{name: "DefaultStubClassName", got: ann.DefaultStubClassName(), want: "DefaultGoldenKitchenSinkStub"},
		{name: "StubFactoryFunctionName", got: ann.StubFactoryFunctionName(), want: "CreateDefaultGoldenKitchenSinkStub"},
		{name: "ConnectionImplClassName", got: ann.ConnectionImplClassName(), want: "GoldenKitchenSinkConnectionImpl"},
		{name: "AuthDecoratorClassName", got: ann.AuthDecoratorClassName(), want: "GoldenKitchenSinkAuth"},
		{name: "LoggingDecoratorClassName", got: ann.LoggingDecoratorClassName(), want: "GoldenKitchenSinkLogging"},
		{name: "MetadataDecoratorClassName", got: ann.MetadataDecoratorClassName(), want: "GoldenKitchenSinkMetadata"},
		{name: "RoundRobinDecoratorClassName", got: ann.RoundRobinDecoratorClassName(), want: "GoldenKitchenSinkRoundRobin"},
		{name: "TracingStubClassName", got: ann.TracingStubClassName(), want: "GoldenKitchenSinkTracingStub"},
		{name: "TracingConnectionClassName", got: ann.TracingConnectionClassName(), want: "GoldenKitchenSinkTracingConnection"},
		{name: "MockConnectionClassName", got: ann.MockConnectionClassName(), want: "MockGoldenKitchenSinkConnection"},
		{name: "DefaultOptionsFunctionName", got: ann.DefaultOptionsFunctionName(), want: "GoldenKitchenSinkDefaultOptions"},
		{name: "MakeConnectionFunctionName", got: ann.MakeConnectionFunctionName(), want: "MakeGoldenKitchenSinkConnection"},
		{name: "RetryPolicyOptionClassName", got: ann.RetryPolicyOptionClassName(), want: "GoldenKitchenSinkRetryPolicyOption"},
		{name: "BackoffPolicyOptionClassName", got: ann.BackoffPolicyOptionClassName(), want: "GoldenKitchenSinkBackoffPolicyOption"},
		{name: "ConnectionIdempotencyPolicyOptionClassName", got: ann.ConnectionIdempotencyPolicyOptionClassName(), want: "GoldenKitchenSinkConnectionIdempotencyPolicyOption"},
		{name: "PollingPolicyOptionClassName", got: ann.PollingPolicyOptionClassName(), want: "GoldenKitchenSinkPollingPolicyOption"},
		{name: "PolicyOptionListClassName", got: ann.PolicyOptionListClassName(), want: "GoldenKitchenSinkPolicyOptionList"},
		{name: "RestStubClassName", got: ann.RestStubClassName(), want: "GoldenKitchenSinkRestStub"},
		{name: "DefaultRestStubClassName", got: ann.DefaultRestStubClassName(), want: "DefaultGoldenKitchenSinkRestStub"},
		{name: "MakeRestStubFunctionName", got: ann.MakeRestStubFunctionName(), want: "CreateDefaultGoldenKitchenSinkRestStub"},
		{name: "RestConnectionImplClassName", got: ann.RestConnectionImplClassName(), want: "GoldenKitchenSinkRestConnectionImpl"},
		{name: "RestLoggingDecoratorClassName", got: ann.RestLoggingDecoratorClassName(), want: "GoldenKitchenSinkRestLogging"},
		{name: "RestMetadataDecoratorClassName", got: ann.RestMetadataDecoratorClassName(), want: "GoldenKitchenSinkRestMetadata"},
		{name: "MakeRestConnectionFunctionName", got: ann.MakeRestConnectionFunctionName(), want: "MakeGoldenKitchenSinkConnectionRest"},
		{name: "MakeConnectionRestFunctionName", got: ann.MakeConnectionRestFunctionName(), want: "MakeGoldenKitchenSinkConnectionRest"},
		{name: "MakeDefaultIdempotencyPolicyFunctionName", got: ann.MakeDefaultIdempotencyPolicyFunctionName(), want: "MakeDefaultGoldenKitchenSinkConnectionIdempotencyPolicy"},
		{name: "MakeDefaultIdempotencyPolicyClassName", got: ann.MakeDefaultIdempotencyPolicyClassName(), want: "MakeDefaultGoldenKitchenSinkConnectionIdempotencyPolicy"},
		{name: "MakeTracingStubFunctionName", got: ann.MakeTracingStubFunctionName(), want: "MakeGoldenKitchenSinkTracingStub"},
		{name: "MakeTracingConnectionFunctionName", got: ann.MakeTracingConnectionFunctionName(), want: "MakeGoldenKitchenSinkTracingConnection"},
		{name: "GrpcStubType", got: ann.GrpcStubType(), want: "google::test::admin::database::v1::GoldenKitchenSink"},
		{name: "RetryTraitsExpression", got: ann.RetryTraitsExpression(), want: "status.code() != StatusCode::kOk && status.code() != StatusCode::kInternal && status.code() != StatusCode::kUnavailable"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if diff := cmp.Diff(tc.want, tc.got); diff != "" {
				t.Errorf("%s mismatch (-want +got):\n%s", tc.name, diff)
			}
		})
	}

	if ann.HasEndpointLocation() {
		t.Errorf("HasEndpointLocation() for empty endpoint location style: want false, got true")
	}
	ann.EndpointLocationStyle = "LOCATION_OPTIONALLY_DEPENDENT"
	if !ann.HasEndpointLocation() {
		t.Errorf("HasEndpointLocation() want true, got false")
	}

	if ann.HasLRO() {
		t.Errorf("HasLRO() for service without LRO: want false, got true")
	}
	method := api.NewTestMethod("TestLRO").WithOperationInfo(api.NewTestOperationInfo("google.protobuf.Empty", ""))
	svc.WithMethods(method)
	if !ann.HasLRO() {
		t.Errorf("HasLRO() for service with LRO: want true, got false")
	}
}

func TestClassNamingHelpers_DeprecatedService(t *testing.T) {
	svc := api.NewTestService("DeprecatedService")
	svc.Deprecated = true
	ann := &serviceAnnotations{
		Name:    "DeprecatedService",
		Service: svc,
	}

	if !ann.IsDeprecated() {
		t.Errorf("IsDeprecated() want true, got false")
	}
	if ann.ClientClassName() != "DeprecatedServiceClient" {
		t.Errorf("ClientClassName() mismatch: want DeprecatedServiceClient, got %q", ann.ClientClassName())
	}
	if ann.ConnectionClassName() != "DeprecatedServiceConnection" {
		t.Errorf("ConnectionClassName() mismatch: want DeprecatedServiceConnection, got %q", ann.ConnectionClassName())
	}
	if ann.MakeRestConnectionFunctionName() != "MakeDeprecatedServiceConnectionRest" {
		t.Errorf("MakeRestConnectionFunctionName() mismatch: want MakeDeprecatedServiceConnectionRest, got %q", ann.MakeRestConnectionFunctionName())
	}
}

func TestClassNamingHelpers_EmptyService(t *testing.T) {
	ann := &serviceAnnotations{
		Name: "EmptyService",
	}

	if ann.IsDeprecated() {
		t.Errorf("IsDeprecated() for nil Service: want false, got true")
	}
	if ann.GrpcStubType() != "EmptyService" {
		t.Errorf("GrpcStubType() for nil Service: want EmptyService, got %q", ann.GrpcStubType())
	}
	if ann.RetryTraitsExpression() != "status.code() != StatusCode::kOk" {
		t.Errorf("RetryTraitsExpression() for empty codes: want 'status.code() != StatusCode::kOk', got %q", ann.RetryTraitsExpression())
	}
}
