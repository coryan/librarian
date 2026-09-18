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
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/sidekick/api"
)

func TestFormatStartMethodComments(t *testing.T) {
	got := formatStartMethodComments("CreateFoo", false)
	want := "  // clang-format off\n" +
		"  ///\n" +
		"  /// @copybrief CreateFoo\n" +
		"  ///\n" +
		"  /// Specifying the [`NoAwaitTag`] immediately returns the\n" +
		"  /// [`google::longrunning::Operation`] that corresponds to the Long Running\n" +
		"  /// Operation that has been started. No polling for operation status occurs.\n" +
		"  ///\n" +
		"  /// [`NoAwaitTag`]: @ref google::cloud::NoAwaitTag\n" +
		"  ///\n" +
		"  // clang-format on\n"

	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("formatStartMethodComments mismatch (-want +got):\n%s", diff)
	}

	gotDep := formatStartMethodComments("CreateFoo", true)
	if !strings.Contains(gotDep, "@deprecated This RPC is deprecated.") {
		t.Errorf("expected deprecation in comment, got: %s", gotDep)
	}
}

func TestFormatAwaitMethodComments(t *testing.T) {
	got := formatAwaitMethodComments("CreateFoo", false)
	want := "  // clang-format off\n" +
		"  ///\n" +
		"  /// @copybrief CreateFoo\n" +
		"  ///\n" +
		"  /// This method accepts a `google::longrunning::Operation` that corresponds\n" +
		"  /// to a previously started Long Running Operation (LRO) and polls the status\n" +
		"  /// of the LRO in the background.\n" +
		"  ///\n" +
		"  // clang-format on\n"

	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("formatAwaitMethodComments mismatch (-want +got):\n%s", diff)
	}
}

func TestFormatIamUpdaterComments(t *testing.T) {
	got := formatIamUpdaterComments("google::iam::v1::Policy")
	if !strings.Contains(got, "Updates the IAM policy for @p resource") {
		t.Errorf("expected IAM policy updater explanation, got: %s", got)
	}
	if !strings.Contains(got, "@return google::iam::v1::Policy") {
		t.Errorf("expected return type in IAM comment, got: %s", got)
	}
}

func TestFormatMethodCommentsProtobufRequest_Unary(t *testing.T) {
	in := api.NewTestMessage("GenerateAccessTokenRequest").WithPackage("google.test.admin.database.v1")
	out := api.NewTestMessage("GenerateAccessTokenResponse").WithPackage("google.test.admin.database.v1")
	m := api.NewTestMethod("GenerateAccessToken").
		WithDocumentation("Generates an OAuth 2.0 access token for a service account.").
		WithInput(in).
		WithOutput(out)
	got := formatMethodCommentsProtobufRequest(m, false)
	if !strings.Contains(got, "// clang-format off") {
		t.Errorf("missing clang-format off: %s", got)
	}
	if !strings.Contains(got, "Generates an OAuth 2.0 access token for a service account.") {
		t.Errorf("missing method documentation: %s", got)
	}
	if !strings.Contains(got, "@param request Unary RPCs") {
		t.Errorf("missing @param request: %s", got)
	}
	if !strings.Contains(got, "@param opts Optional.") {
		t.Errorf("missing @param opts: %s", got)
	}
	if !strings.Contains(got, "@return the result of the RPC.") {
		t.Errorf("missing @return: %s", got)
	}
	if !strings.Contains(got, "[google.test.admin.database.v1.GenerateAccessTokenRequest]: @googleapis_reference_link{generator/integration_tests/test.proto#L969}") {
		t.Errorf("missing input type reference link: %s", got)
	}
	if !strings.Contains(got, "[google.test.admin.database.v1.GenerateAccessTokenResponse]: @googleapis_reference_link{generator/integration_tests/test.proto#L1010}") {
		t.Errorf("missing output type reference link: %s", got)
	}
}

func TestFormatMethodCommentsProtobufRequest_LRO(t *testing.T) {
	in := api.NewTestMessage("CreateDatabaseRequest").WithPackage("google.test.admin.database.v1")
	out := api.NewTestMessage("Operation").WithPackage("google.longrunning")
	m := api.NewTestMethod("CreateDatabase").
		WithDocumentation("Creates a new database.").
		WithInput(in).
		WithOutput(out).
		WithOperationInfo(api.NewTestOperationInfo(
			".google.test.admin.database.v1.Database",
			".google.test.admin.database.v1.CreateDatabaseMetadata",
		))
	got := formatMethodCommentsProtobufRequest(m, false)
	if !strings.Contains(got, "@return A [`future`] that becomes satisfied when the LRO") {
		t.Errorf("missing LRO return comment: %s", got)
	}
	if !strings.Contains(got, "[Long Running Operation]: https://google.aip.dev/151") {
		t.Errorf("missing AIP-151 link: %s", got)
	}
	if !strings.Contains(got, "[google.test.admin.database.v1.Database]: @googleapis_reference_link{generator/integration_tests/test.proto#L365}") {
		t.Errorf("missing LRO deduced return reference: %s", got)
	}
}

func TestFormatMethodCommentsProtobufRequest_Pagination(t *testing.T) {
	in := api.NewTestMessage("ListDatabasesRequest").WithPackage("google.test.admin.database.v1")
	pageTokenField := api.NewTestField("page_token").WithType(api.TypezString)
	nextPageTokenField := api.NewTestField("next_page_token").WithType(api.TypezString)
	pageableField := api.NewTestField("databases").
		WithType(api.TypezMessage).
		WithTypezID(".google.test.admin.database.v1.Database")
	out := api.NewTestMessage("ListDatabasesResponse").
		WithPackage("google.test.admin.database.v1").
		WithPagination(nextPageTokenField, pageableField)

	m := api.NewTestMethod("ListDatabases").
		WithDocumentation("Lists databases.").
		WithInput(in).
		WithOutput(out).
		WithPagination(pageTokenField)
	got := formatMethodCommentsProtobufRequest(m, false)
	if !strings.Contains(got, "@return a [StreamRange](@ref google::cloud::StreamRange)") {
		t.Errorf("missing StreamRange return comment: %s", got)
	}
}

func TestFormatMethodComments_GetOperation(t *testing.T) {
	in := api.NewTestMessage("GetOperationRequest").WithPackage("google.longrunning")
	out := api.NewTestMessage("Operation").WithPackage("google.longrunning")
	nameField := api.NewTestField("name").WithType(api.TypezString)
	in.WithFields(nameField)

	m := api.NewTestMethod("GetOperation").
		WithDocumentation("Provides the [google.longrunning.Operations] service.").
		WithInput(in).
		WithOutput(out)

	got := formatMethodCommentsProtobufRequest(m, false)
	if !strings.Contains(got, "Gets the latest state of a long-running operation.") {
		t.Errorf("expected standard GetOperation doc comment, got: %s", got)
	}

	gotParam := formatParameterComment(m, nameField, "name")
	if !strings.Contains(gotParam, "@param name  The name of the operation resource.") {
		t.Errorf("expected GetOperation param comment for name, got: %s", gotParam)
	}
}
