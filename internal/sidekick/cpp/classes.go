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
	"strings"
)

// ClientClassName returns the C++ class name for the service client
// (e.g. GoldenKitchenSinkClient).
func (ann *serviceAnnotations) ClientClassName() string {
	return ann.Name + "Client"
}

// ConnectionClassName returns the C++ class name for the service connection interface
// (e.g. GoldenKitchenSinkConnection).
func (ann *serviceAnnotations) ConnectionClassName() string {
	return ann.Name + "Connection"
}

// IdempotencyPolicyClassName returns the C++ class name for the connection idempotency
// policy (e.g. GoldenKitchenSinkConnectionIdempotencyPolicy).
func (ann *serviceAnnotations) IdempotencyPolicyClassName() string {
	return ann.Name + "ConnectionIdempotencyPolicy"
}

// RetryPolicyClassName returns the C++ class name for the retry policy interface
// (e.g. GoldenKitchenSinkRetryPolicy).
func (ann *serviceAnnotations) RetryPolicyClassName() string {
	return ann.Name + "RetryPolicy"
}

// LimitedErrorCountRetryPolicyClassName returns the C++ class name for the error-count
// retry policy (e.g. GoldenKitchenSinkLimitedErrorCountRetryPolicy).
func (ann *serviceAnnotations) LimitedErrorCountRetryPolicyClassName() string {
	return ann.Name + "LimitedErrorCountRetryPolicy"
}

// LimitedTimeRetryPolicyClassName returns the C++ class name for the elapsed-time
// retry policy (e.g. GoldenKitchenSinkLimitedTimeRetryPolicy).
func (ann *serviceAnnotations) LimitedTimeRetryPolicyClassName() string {
	return ann.Name + "LimitedTimeRetryPolicy"
}

// RetryTraitsClassName returns the C++ struct name for the retry traits
// (e.g. GoldenKitchenSinkRetryTraits).
func (ann *serviceAnnotations) RetryTraitsClassName() string {
	return ann.Name + "RetryTraits"
}

// StubClassName returns the C++ class name for the gRPC stub interface
// (e.g. GoldenKitchenSinkStub).
func (ann *serviceAnnotations) StubClassName() string {
	return ann.Name + "Stub"
}

// DefaultStubClassName returns the C++ class name for the default gRPC stub implementation
// (e.g. DefaultGoldenKitchenSinkStub).
func (ann *serviceAnnotations) DefaultStubClassName() string {
	return "Default" + ann.Name + "Stub"
}

// StubFactoryFunctionName returns the C++ function name for creating the default gRPC stub
// (e.g. CreateDefaultGoldenKitchenSinkStub).
func (ann *serviceAnnotations) StubFactoryFunctionName() string {
	return "CreateDefault" + ann.Name + "Stub"
}

// ConnectionImplClassName returns the C++ class name for the concrete connection
// implementation (e.g. GoldenKitchenSinkConnectionImpl).
func (ann *serviceAnnotations) ConnectionImplClassName() string {
	return ann.Name + "ConnectionImpl"
}

// AuthDecoratorClassName returns the C++ class name for the auth decorator
// (e.g. GoldenKitchenSinkAuth).
func (ann *serviceAnnotations) AuthDecoratorClassName() string {
	return ann.Name + "Auth"
}

// LoggingDecoratorClassName returns the C++ class name for the logging decorator
// (e.g. GoldenKitchenSinkLogging).
func (ann *serviceAnnotations) LoggingDecoratorClassName() string {
	return ann.Name + "Logging"
}

// MetadataDecoratorClassName returns the C++ class name for the metadata decorator
// (e.g. GoldenKitchenSinkMetadata).
func (ann *serviceAnnotations) MetadataDecoratorClassName() string {
	return ann.Name + "Metadata"
}

// RoundRobinDecoratorClassName returns the C++ class name for the round-robin decorator
// (e.g. GoldenKitchenSinkRoundRobin).
func (ann *serviceAnnotations) RoundRobinDecoratorClassName() string {
	return ann.Name + "RoundRobin"
}

// TracingStubClassName returns the C++ class name for the tracing stub decorator
// (e.g. GoldenKitchenSinkTracingStub).
func (ann *serviceAnnotations) TracingStubClassName() string {
	return ann.Name + "TracingStub"
}

// TracingConnectionClassName returns the C++ class name for the tracing connection
// decorator (e.g. GoldenKitchenSinkTracingConnection).
func (ann *serviceAnnotations) TracingConnectionClassName() string {
	return ann.Name + "TracingConnection"
}

// MockConnectionClassName returns the C++ class name for the mock connection
// (e.g. MockGoldenKitchenSinkConnection).
func (ann *serviceAnnotations) MockConnectionClassName() string {
	return "Mock" + ann.Name + "Connection"
}

// DefaultOptionsFunctionName returns the C++ function name for default options
// (e.g. GoldenKitchenSinkDefaultOptions).
func (ann *serviceAnnotations) DefaultOptionsFunctionName() string {
	return ann.Name + "DefaultOptions"
}

// MakeConnectionFunctionName returns the C++ factory function name for creating the
// connection (e.g. MakeGoldenKitchenSinkConnection).
func (ann *serviceAnnotations) MakeConnectionFunctionName() string {
	return "Make" + ann.Name + "Connection"
}

// RetryPolicyOptionClassName returns the C++ struct name for the retry policy option
// (e.g. GoldenKitchenSinkRetryPolicyOption).
func (ann *serviceAnnotations) RetryPolicyOptionClassName() string {
	return ann.Name + "RetryPolicyOption"
}

// BackoffPolicyOptionClassName returns the C++ struct name for the backoff policy option
// (e.g. GoldenKitchenSinkBackoffPolicyOption).
func (ann *serviceAnnotations) BackoffPolicyOptionClassName() string {
	return ann.Name + "BackoffPolicyOption"
}

// ConnectionIdempotencyPolicyOptionClassName returns the C++ struct name for the
// idempotency policy option (e.g. GoldenKitchenSinkConnectionIdempotencyPolicyOption).
func (ann *serviceAnnotations) ConnectionIdempotencyPolicyOptionClassName() string {
	return ann.Name + "ConnectionIdempotencyPolicyOption"
}

// PollingPolicyOptionClassName returns the C++ struct name for the polling policy option
// (e.g. GoldenKitchenSinkPollingPolicyOption).
func (ann *serviceAnnotations) PollingPolicyOptionClassName() string {
	return ann.Name + "PollingPolicyOption"
}

// PolicyOptionListClassName returns the C++ type alias name for the policy option list
// (e.g. GoldenKitchenSinkPolicyOptionList).
func (ann *serviceAnnotations) PolicyOptionListClassName() string {
	return ann.Name + "PolicyOptionList"
}

// RestStubClassName returns the C++ class name for the REST stub interface
// (e.g. GoldenKitchenSinkRestStub).
func (ann *serviceAnnotations) RestStubClassName() string {
	return ann.Name + "RestStub"
}

// DefaultRestStubClassName returns the C++ class name for the default REST stub
// implementation (e.g. DefaultGoldenKitchenSinkRestStub).
func (ann *serviceAnnotations) DefaultRestStubClassName() string {
	return "Default" + ann.Name + "RestStub"
}

// MakeRestStubFunctionName returns the C++ function name for creating the default REST
// stub (e.g. CreateDefaultGoldenKitchenSinkRestStub).
func (ann *serviceAnnotations) MakeRestStubFunctionName() string {
	return "CreateDefault" + ann.Name + "RestStub"
}

// RestConnectionImplClassName returns the C++ class name for the concrete REST
// connection implementation (e.g. GoldenKitchenSinkRestConnectionImpl).
func (ann *serviceAnnotations) RestConnectionImplClassName() string {
	return ann.Name + "RestConnectionImpl"
}

// RestLoggingDecoratorClassName returns the C++ class name for the REST logging decorator
// (e.g. GoldenKitchenSinkRestLogging).
func (ann *serviceAnnotations) RestLoggingDecoratorClassName() string {
	return ann.Name + "RestLogging"
}

// RestMetadataDecoratorClassName returns the C++ class name for the REST metadata decorator
// (e.g. GoldenKitchenSinkRestMetadata).
func (ann *serviceAnnotations) RestMetadataDecoratorClassName() string {
	return ann.Name + "RestMetadata"
}

// MakeRestConnectionFunctionName returns the C++ factory function name for creating the
// REST connection (e.g. MakeGoldenKitchenSinkConnectionRest).
// In google-cloud-cpp generator conventions, this is "Make" + ServiceName + "ConnectionRest".
func (ann *serviceAnnotations) MakeRestConnectionFunctionName() string {
	return "Make" + ann.Name + "ConnectionRest"
}

// MakeConnectionRestFunctionName is an alias for MakeRestConnectionFunctionName.
func (ann *serviceAnnotations) MakeConnectionRestFunctionName() string {
	return ann.MakeRestConnectionFunctionName()
}

// MakeDefaultIdempotencyPolicyFunctionName returns the C++ factory function name for the
// default idempotency policy (e.g. MakeDefaultGoldenKitchenSinkConnectionIdempotencyPolicy).
func (ann *serviceAnnotations) MakeDefaultIdempotencyPolicyFunctionName() string {
	return "MakeDefault" + ann.IdempotencyPolicyClassName()
}

// MakeDefaultIdempotencyPolicyClassName is an alias for MakeDefaultIdempotencyPolicyFunctionName.
func (ann *serviceAnnotations) MakeDefaultIdempotencyPolicyClassName() string {
	return ann.MakeDefaultIdempotencyPolicyFunctionName()
}

// MakeTracingStubFunctionName returns the C++ factory function name for the tracing stub
// decorator (e.g. MakeGoldenKitchenSinkTracingStub).
func (ann *serviceAnnotations) MakeTracingStubFunctionName() string {
	return "Make" + ann.Name + "TracingStub"
}

// MakeTracingConnectionFunctionName returns the C++ factory function name for the tracing
// connection decorator (e.g. MakeGoldenKitchenSinkTracingConnection).
func (ann *serviceAnnotations) MakeTracingConnectionFunctionName() string {
	return "Make" + ann.Name + "TracingConnection"
}

// GrpcStubType returns the C++ protobuf service stub interface type prefix
// (e.g. google::test::admin::database::v1::GoldenKitchenSink).
func (ann *serviceAnnotations) GrpcStubType() string {
	if ann.Service == nil || ann.Service.Package == "" {
		return ann.Name
	}
	pkg := strings.ReplaceAll(ann.Service.Package, ".", "::")
	return pkg + "::" + ann.Name
}

// IsDeprecated returns true if the service is marked deprecated in the protobuf spec.
func (ann *serviceAnnotations) IsDeprecated() bool {
	return ann.Service != nil && ann.Service.Deprecated
}

// RetryTraitsExpression returns the C++ boolean expression checking for permanent failure in RetryTraits.
func (ann *serviceAnnotations) RetryTraitsExpression() string {
	var b strings.Builder
	b.WriteString("status.code() != StatusCode::kOk")
	for _, code := range ann.RetryableStatusCodes {
		b.WriteString(" && status.code() != StatusCode::")
		b.WriteString(code)
	}
	return b.String()
}
