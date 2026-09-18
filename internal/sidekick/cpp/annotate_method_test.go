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

func TestMethodAnnotations_UnaryVoidAndNonVoid(t *testing.T) {
	svc := api.NewTestService("TestService")
	emptyMsg := api.NewTestMessage("Empty").WithPackage("google.protobuf")
	voidMethod := api.NewTestMethod("DoNothing").
		WithInput(emptyMsg).
		WithOutput(emptyMsg)
	voidMethod.ReturnsEmpty = true

	fooReq := api.NewTestMessage("GetFooRequest")
	fooResp := api.NewTestMessage("Foo")
	nonVoidMethod := api.NewTestMethod("GetFoo").
		WithInput(fooReq).
		WithOutput(fooResp)

	c := &codec{}
	sAnn := &serviceAnnotations{Name: "TestService", Service: svc}

	vAnn := c.annotateMethod(svc, voidMethod, sAnn)
	if !vAnn.IsVoid() {
		t.Errorf("expected voidMethod to be void")
	}
	if vAnn.PlainReturnType() != "Status" {
		t.Errorf("expected PlainReturnType to be 'Status', got %q", vAnn.PlainReturnType())
	}
	if !vAnn.IsPlainUnary() {
		t.Errorf("expected voidMethod to be PlainUnary")
	}

	nvAnn := c.annotateMethod(svc, nonVoidMethod, sAnn)
	if nvAnn.IsVoid() {
		t.Errorf("expected nonVoidMethod not to be void")
	}
	if nvAnn.PlainReturnType() != "StatusOr<test::Foo>" {
		t.Errorf("expected PlainReturnType to be 'StatusOr<test::Foo>', got %q", nvAnn.PlainReturnType())
	}
	if !nvAnn.IsPlainUnary() {
		t.Errorf("expected nonVoidMethod to be PlainUnary")
	}
}

func TestMethodAnnotations_Pagination(t *testing.T) {
	svc := api.NewTestService("TestService")
	in := api.NewTestMessage("ListFoosRequest")
	pageTokenField := api.NewTestField("page_token").WithType(api.TypezString)
	nextPageTokenField := api.NewTestField("next_page_token").WithType(api.TypezString)
	pageableField := api.NewTestField("foos").
		WithType(api.TypezMessage).
		WithTypezID(".test.Foo")
	out := api.NewTestMessage("ListFoosResponse").
		WithPagination(nextPageTokenField, pageableField)

	m := api.NewTestMethod("ListFoos").
		WithInput(in).
		WithOutput(out).
		WithPagination(pageTokenField)

	c := &codec{}
	sAnn := &serviceAnnotations{Name: "TestService", Service: svc}
	mAnn := c.annotateMethod(svc, m, sAnn)

	if !mAnn.IsPaginated {
		t.Errorf("expected IsPaginated to be true")
	}
	if mAnn.IsPlainUnary() {
		t.Errorf("expected IsPlainUnary to be false for paginated method")
	}
	if mAnn.RangeOutputType() != "test::Foo" {
		t.Errorf("expected RangeOutputType to be 'test::Foo', got %q", mAnn.RangeOutputType())
	}
}

func TestMethodAnnotations_LRO(t *testing.T) {
	svc := api.NewTestService("TestService")
	in := api.NewTestMessage("CreateFooRequest")
	out := api.NewTestMessage("Operation").WithPackage("google.longrunning")
	m := api.NewTestMethod("CreateFoo").
		WithInput(in).
		WithOutput(out).
		WithOperationInfo(api.NewTestOperationInfo(".test.Foo", ".test.CreateFooMetadata"))

	c := &codec{}
	sAnn := &serviceAnnotations{Name: "TestService", Service: svc}
	mAnn := c.annotateMethod(svc, m, sAnn)

	if !mAnn.IsLRO {
		t.Errorf("expected IsLRO to be true")
	}
	if mAnn.IsLroVoid() {
		t.Errorf("expected IsLroVoid to be false")
	}
	if mAnn.DeducedResponseType() != "test::Foo" {
		t.Errorf("expected DeducedResponseType to be 'test::Foo', got %q", mAnn.DeducedResponseType())
	}

	// LRO with Empty response using metadata
	inEmpty := api.NewTestMessage("UpdateDdlRequest")
	mEmpty := api.NewTestMethod("UpdateDdl").
		WithInput(inEmpty).
		WithOutput(out).
		WithOperationInfo(api.NewTestOperationInfo(".google.protobuf.Empty", ".test.UpdateDdlMetadata"))
	mEmptyAnn := c.annotateMethod(svc, mEmpty, sAnn)
	if mEmptyAnn.IsLroVoid() {
		t.Errorf("expected IsLroVoid to be false when metadata is present")
	}
	if mEmptyAnn.DeducedResponseType() != "test::UpdateDdlMetadata" {
		t.Errorf("expected DeducedResponseType to be 'test::UpdateDdlMetadata', got %q", mEmptyAnn.DeducedResponseType())
	}

	// Void LRO (both empty)
	inVoid := api.NewTestMessage("DeleteDdlRequest")
	mVoid := api.NewTestMethod("DeleteDdl").
		WithInput(inVoid).
		WithOutput(out).
		WithOperationInfo(api.NewTestOperationInfo(".google.protobuf.Empty", ".google.protobuf.Empty"))
	mVoidAnn := c.annotateMethod(svc, mVoid, sAnn)
	if !mVoidAnn.IsLroVoid() {
		t.Errorf("expected IsLroVoid to be true when both response and metadata are empty")
	}
}

func TestMethodAnnotations_Streaming(t *testing.T) {
	svc := api.NewTestService("TestService")
	sAnn := &serviceAnnotations{Name: "TestService", Service: svc}
	c := &codec{}

	// Server streaming
	mRead := api.NewTestMethod("StreamRead").WithServerSideStreaming()
	annRead := c.annotateMethod(svc, mRead, sAnn)
	if !annRead.IsStreamingRead() || annRead.IsStreamingWrite() || annRead.IsBidiStreaming() {
		t.Errorf("mismatch in server streaming detection: %+v", annRead)
	}

	// Client streaming
	mWrite := api.NewTestMethod("StreamWrite").WithClientSideStreaming()
	annWrite := c.annotateMethod(svc, mWrite, sAnn)
	if !annWrite.IsStreamingWrite() || annWrite.IsStreamingRead() || annWrite.IsBidiStreaming() {
		t.Errorf("mismatch in client streaming detection: %+v", annWrite)
	}

	// Bidi streaming
	mBidi := api.NewTestMethod("StreamBidi").WithBidiStreaming()
	annBidi := c.annotateMethod(svc, mBidi, sAnn)
	if !annBidi.IsBidiStreaming() || annBidi.IsStreamingRead() || annBidi.IsStreamingWrite() {
		t.Errorf("mismatch in bidi streaming detection: %+v", annBidi)
	}
}

func TestMethodAnnotations_Deprecated(t *testing.T) {
	svc := api.NewTestService("TestService")
	sAnn := &serviceAnnotations{Name: "TestService", Service: svc}
	c := &codec{}

	mDep := api.NewTestMethod("DeprecatedMethod").WithDeprecated(true)
	annDep := c.annotateMethod(svc, mDep, sAnn)
	if !annDep.IsDeprecated() {
		t.Errorf("expected IsDeprecated to be true")
	}
	if annDep.DeprecationMacro() == "" {
		t.Errorf("expected non-empty DeprecationMacro")
	}

	mNotDep := api.NewTestMethod("ActiveMethod")
	annNotDep := c.annotateMethod(svc, mNotDep, sAnn)
	if annNotDep.IsDeprecated() {
		t.Errorf("expected IsDeprecated to be false")
	}
	if annNotDep.DeprecationMacro() != "" {
		t.Errorf("expected empty DeprecationMacro, got %q", annNotDep.DeprecationMacro())
	}
}

func TestMethodAnnotations_SignaturesAndConflictResolution(t *testing.T) {
	svc := api.NewTestService("TestService")
	sAnn := &serviceAnnotations{Name: "TestService", Service: svc}
	c := &codec{
		Cpp: &config.CppLibrary{
			OmittedRPCs: []string{"DoFoo(std::int32_t)"},
		},
	}

	in := api.NewTestMessage("DoFooRequest").WithFields(
		api.NewTestField("name").WithType(api.TypezString),
		api.NewTestField("count").WithType(api.TypezInt32),
		api.NewTestField("tag").WithType(api.TypezString),
	)
	m := api.NewTestMethod("DoFoo").
		WithInput(in).
		WithSignatures(
			api.NewTestMethodSignature("name"),
			api.NewTestMethodSignature("tag"),   // Same type (std::string const&) -> should be dropped (first match wins)!
			api.NewTestMethodSignature("count"), // Omitted via Cpp.OmittedRPCs!
			api.NewTestMethodSignature("name", "count"),
		)

	mAnn := c.annotateMethod(svc, m, sAnn)
	if len(mAnn.Signatures) != 2 {
		t.Fatalf("expected 2 signatures after conflict resolution and omission, got %d", len(mAnn.Signatures))
	}

	// First signature: name
	sig0 := mAnn.Signatures[0]
	if len(sig0.Params) != 1 || sig0.Params[0].Name != "name" {
		t.Errorf("expected sig0 to have param 'name', got %+v", sig0.Params)
	}
	if sig0.ParamsString() != "std::string const& name, " {
		t.Errorf("expected ParamsString 'std::string const& name, ', got %q", sig0.ParamsString())
	}

	// Second signature: name, count
	sig1 := mAnn.Signatures[1]
	if len(sig1.Params) != 2 || sig1.Params[0].Name != "name" || sig1.Params[1].Name != "count" {
		t.Errorf("expected sig1 to have params name, count, got %+v", sig1.Params)
	}
	if sig1.ParamsString() != "std::string const& name, std::int32_t count, " {
		t.Errorf("expected ParamsString 'std::string const& name, std::int32_t count, ', got %q", sig1.ParamsString())
	}
}

func TestMethodAnnotations_Idempotency(t *testing.T) {
	svc := api.NewTestService("TestService")
	sAnn := &serviceAnnotations{Name: "TestService", Service: svc}
	c := &codec{
		Cpp: &config.CppLibrary{
			IdempotencyOverrides: []config.IdempotencyRule{
				{RPCName: "CustomRpc", Idempotency: "IDEMPOTENT"},
				{RPCName: "TestService.OverrideNonIdempotent", Idempotency: "NON_IDEMPOTENT"},
			},
		},
	}

	mGet := api.NewTestMethod("GetThing").WithVerb("GET")
	annGet := c.annotateMethod(svc, mGet, sAnn)
	if annGet.Idempotency != "kIdempotent" {
		t.Errorf("expected GET to be kIdempotent, got %q", annGet.Idempotency)
	}

	mPost := api.NewTestMethod("CreateThing").WithVerb("POST")
	annPost := c.annotateMethod(svc, mPost, sAnn)
	if annPost.Idempotency != "kNonIdempotent" {
		t.Errorf("expected POST to be kNonIdempotent, got %q", annPost.Idempotency)
	}

	mOverride := api.NewTestMethod("CustomRpc").WithVerb("POST")
	annOverride := c.annotateMethod(svc, mOverride, sAnn)
	if annOverride.Idempotency != "kIdempotent" {
		t.Errorf("expected override to be kIdempotent, got %q", annOverride.Idempotency)
	}
}

func TestMethodAnnotations_IAMOptimisticConcurrency(t *testing.T) {
	inGet := api.NewTestMessage("GetIamPolicyRequest").
		WithPackage("google.iam.v1").
		WithFields(api.NewTestField("resource").WithType(api.TypezString))
	inSet := api.NewTestMessage("SetIamPolicyRequest").
		WithPackage("google.iam.v1").
		WithFields(
			api.NewTestField("resource").WithType(api.TypezString),
			api.NewTestField("policy").WithType(api.TypezMessage),
		)
	outPolicy := api.NewTestMessage("Policy").WithPackage("google.iam.v1")

	getM := api.NewTestMethod("GetIamPolicy").
		WithInput(inGet).
		WithOutput(outPolicy).
		WithSignatures(api.NewTestMethodSignature("resource"))

	setM := api.NewTestMethod("SetIamPolicy").
		WithInput(inSet).
		WithOutput(outPolicy).
		WithSignatures(api.NewTestMethodSignature("resource", "policy"))

	svc := api.NewTestService("DatabaseAdmin").WithMethods(getM, setM)
	sAnn := &serviceAnnotations{Name: "DatabaseAdmin", Service: svc}
	c := &codec{}

	_ = c.annotateMethod(svc, getM, sAnn)
	setAnn := c.annotateMethod(svc, setM, sAnn)

	if !setAnn.IsIamSetMethodWithUpdater {
		t.Errorf("expected IsIamSetMethodWithUpdater to be true")
	}
	if diff := cmp.Diff("google::iam::v1::Policy", setAnn.IamUpdaterResponse()); diff != "" {
		t.Errorf("IamUpdaterResponse mismatch (-want +got):\n%s", diff)
	}
}

func TestMethodAnnotations_Layer36_Helpers(t *testing.T) {
	// RangeOutputFieldName
	outMsg := api.NewTestMessage("Response").WithPagination(nil, api.NewTestField("items"))
	paginatedM := &methodAnnotations{
		Method: api.NewTestMethod("List").WithOutput(outMsg),
	}
	if got := paginatedM.RangeOutputFieldName(); got != "items" {
		t.Errorf("RangeOutputFieldName: want 'items', got %q", got)
	}

	// HasRequestId and RequestIdFieldName
	mWithReqId := &methodAnnotations{
		Method: api.NewTestMethod("Create").WithAutoPopulated(api.NewTestField("request_id")),
	}
	if !mWithReqId.HasRequestId() {
		t.Errorf("HasRequestId: want true, got false")
	}
	if got := mWithReqId.RequestIdFieldName(); got != "request_id" {
		t.Errorf("RequestIdFieldName: want 'request_id', got %q", got)
	}

	mNoReqId := &methodAnnotations{Method: api.NewTestMethod("Create")}
	if mNoReqId.HasRequestId() {
		t.Errorf("HasRequestId: want false, got true")
	}

	// IsSetIamPolicy
	svc := &serviceAnnotations{Name: "TestService"}
	mSetIam := &methodAnnotations{
		Name: "SetIamPolicy",
		Method: api.NewTestMethod("SetIamPolicy").
			WithInputTypeID("google.iam.v1.SetIamPolicyRequest").
			WithOutputTypeID("google.iam.v1.Policy"),
		RequestType:  "google::iam::v1::SetIamPolicyRequest",
		ResponseType: "google::iam::v1::Policy",
		Service:      svc,
	}
	if !mSetIam.IsSetIamPolicy() {
		t.Errorf("IsSetIamPolicy: want true, got false")
	}

	// GrpcStub
	sourceLocations := api.NewTestService("Locations")
	parentSvc := api.NewTestService("ParentService")
	mLocations := &methodAnnotations{
		Method: api.NewTestMethod("LocationsOp").
			WithSourceService(sourceLocations).
			WithService(parentSvc),
	}
	if got := mLocations.GrpcStub(); got != "locations_stub_" {
		t.Errorf("GrpcStub: want 'locations_stub_', got %q", got)
	}

	mParent := &methodAnnotations{
		Method: api.NewTestMethod("ParentOp").
			WithSourceService(parentSvc).
			WithService(parentSvc),
	}
	if got := mParent.GrpcStub(); got != "grpc_stub_" {
		t.Errorf("GrpcStub: want 'grpc_stub_', got %q", got)
	}

	// LRO helpers
	mLro := &methodAnnotations{
		IsLRO: true,
		Method: api.NewTestMethod("LroOp").
			WithOperationInfo(api.NewTestOperationInfo(".google.test.v1.TestResponse", ".google.test.v1.TestMetadata")),
		ResponseType: "google::test::v1::TestResponse",
	}
	if mLro.IsLongrunningMetadataTypeUsedAsResponse() {
		t.Errorf("IsLongrunningMetadataTypeUsedAsResponse: want false, got true")
	}
	if got := mLro.OperationMetadataType(); got != "google::test::v1::TestMetadata" {
		t.Errorf("OperationMetadataType: want 'google::test::v1::TestMetadata', got %q", got)
	}
	if got := mLro.ExtractLongRunningResultFunction(); got != "&google::cloud::internal::ExtractLongRunningResultResponse<google::test::v1::TestResponse>," {
		t.Errorf("ExtractLongRunningResultFunction: want ExtractLongRunningResultResponse, got %q", got)
	}

	mLroEmptyRes := &methodAnnotations{
		IsLRO: true,
		Method: api.NewTestMethod("LroOpEmpty").
			WithOperationInfo(api.NewTestOperationInfo("google.protobuf.Empty", ".google.test.v1.TestMetadata")),
		ResponseType: "google::test::v1::TestMetadata",
	}
	if !mLroEmptyRes.IsLongrunningMetadataTypeUsedAsResponse() {
		t.Errorf("IsLongrunningMetadataTypeUsedAsResponse: want true, got false")
	}
	if got := mLroEmptyRes.ExtractLongRunningResultFunction(); got != "&google::cloud::internal::ExtractLongRunningResultMetadata<google::test::v1::TestMetadata>," {
		t.Errorf("ExtractLongRunningResultFunction: want ExtractLongRunningResultMetadata, got %q", got)
	}

	// StreamingUpdaterFunctionName
	mStream := &methodAnnotations{
		Name:    "ReadRows",
		Service: &serviceAnnotations{Name: "BigQueryRead"},
	}
	if got := mStream.StreamingUpdaterFunctionName(); got != "BigQueryReadReadRowsStreamingUpdater" {
		t.Errorf("StreamingUpdaterFunctionName: want 'BigQueryReadReadRowsStreamingUpdater', got %q", got)
	}

	// RequestSetters
	sig := &signatureAnnotations{
		Params: []*signatureParamAnnotations{
			{Name: "parent", Field: api.NewTestField("parent").WithType(api.TypezString)},
			{Name: "tags", Field: api.NewTestField("tags").WithType(api.TypezString).WithRepeated()},
			{Name: "config", Field: api.NewTestField("config").WithType(api.TypezMessage)},
		},
	}
	wantSetters := "  request.set_parent(parent);\n  *request.mutable_tags() = {tags.begin(), tags.end()};\n  *request.mutable_config() = config;\n"
	if got := sig.RequestSetters(); got != wantSetters {
		t.Errorf("RequestSetters mismatch (-want +got):\n%s", cmp.Diff(wantSetters, got))
	}
}
