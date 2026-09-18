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
		WithOperationInfo(&api.OperationInfo{
			ResponseTypeID: ".test.Foo",
			MetadataTypeID: ".test.CreateFooMetadata",
		})

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
		WithOperationInfo(&api.OperationInfo{
			ResponseTypeID: ".google.protobuf.Empty",
			MetadataTypeID: ".test.UpdateDdlMetadata",
		})
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
		WithOperationInfo(&api.OperationInfo{
			ResponseTypeID: ".google.protobuf.Empty",
			MetadataTypeID: ".google.protobuf.Empty",
		})
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
	m := api.NewTestMethod("DoFoo").WithInput(in)
	m.Signatures = []*api.MethodSignature{
		{Names: []string{"name"}},
		{Names: []string{"tag"}},   // Same type (std::string const&) -> should be dropped (first match wins)!
		{Names: []string{"count"}}, // Omitted via Cpp.OmittedRPCs!
		{Names: []string{"name", "count"}},
	}

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
		WithSignatures(&api.MethodSignature{Names: []string{"resource"}})

	setM := api.NewTestMethod("SetIamPolicy").
		WithInput(inSet).
		WithOutput(outPolicy).
		WithSignatures(&api.MethodSignature{Names: []string{"resource", "policy"}})

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
