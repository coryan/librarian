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
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/sidekick/api"
	"github.com/googleapis/librarian/internal/sidekick/parser"
)

func TestGenerateService_Files(t *testing.T) {
	outDir := t.TempDir()

	// We need explicit Package and ID fields because we generate both messages
	// and services.
	iam := api.NewTestService("IAM").WithPackage("test")
	secretManager := api.NewTestService("SecretManagerService").WithPackage("test")
	clash0 := api.NewTestMessage("InstanceSettings").WithPackage("test")
	clash1 := api.NewTestService("instanceSettings").WithPackage("test")

	model := api.NewTestAPI([]*api.Message{clash0}, nil, []*api.Service{iam, secretManager, clash1})
	model.PackageName = "test"

	library := &config.Library{Swift: swiftConfig(t, nil)}
	if err := Generate(t.Context(), model, outDir, library, nil); err != nil {
		t.Fatal(err)
	}

	expectedDir := filepath.Join(outDir, "Sources", "Test")
	wantFiles := []string{
		"IAM.swift",
		"Clients.swift",
		"SecretManagerService.swift",
		"SecretManagerService+Stub.swift",
		"SecretManagerService+Transport.swift",
		"SecretManagerService+Logging.swift",
		"SecretManagerService+Retry.swift",
		"InstanceSettings.swift",
		"instanceSettings+000.swift",
		"instanceSettings+Stub.swift",
		"instanceSettings+Transport.swift",
		"instanceSettings+Logging.swift",
		"instanceSettings+Retry.swift",
	}
	for _, expected := range wantFiles {
		filename := filepath.Join(expectedDir, expected)
		if _, err := os.Stat(filename); err != nil {
			t.Error(err)
		}
	}
}

func TestGenerateServiceSwift_SnippetReference(t *testing.T) {
	outDir := t.TempDir()

	// "Protocol" is a reserved word that gets mangled to "Protocol_"
	service := api.NewTestService("Protocol")

	model := api.NewTestAPI(nil, nil, []*api.Service{service})
	model.PackageName = "google.cloud.test.v1"

	library := &config.Library{
		Swift: swiftConfig(t, nil),
	}
	if err := Generate(t.Context(), model, outDir, library, nil); err != nil {
		t.Fatal(err)
	}

	// The file name uses the unmangled name
	filename := filepath.Join(outDir, "Sources", "GoogleCloudTestV1", "Protocol.swift")
	content, err := os.ReadFile(filename)
	if err != nil {
		t.Fatal(err)
	}
	contentStr := string(content)

	gotBlock := extractBlock(t, contentStr, "public protocol ", "{")
	wantBlock := `public protocol ProtocolProtocol {`
	if diff := cmp.Diff(wantBlock, gotBlock); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestGenerateService_Delegation(t *testing.T) {
	outDir := t.TempDir()

	request := api.NewTestMessage("Request").WithPackage("test")
	response := api.NewTestMessage("Response").WithPackage("test")
	iam := api.NewTestService("IAM").
		WithPackage("test").
		WithMethods(
			api.NewTestMethod("CreateRole").
				WithInput(request).
				WithOutput(response).
				WithVerb("POST").
				WithPathTemplate((&api.PathTemplate{}).WithLiteral("v1")),
		)

	model := api.NewTestAPI([]*api.Message{request, response}, nil, []*api.Service{iam})
	model.PackageName = "google.cloud.test.v1"

	swiftCfg := swiftConfig(t, []config.SwiftDependency{
		{
			Name:       "SomeTestPackage",
			ApiPackage: "test",
		},
	})
	library := &config.Library{
		Swift: swiftCfg,
	}
	if err := Generate(t.Context(), model, outDir, library, nil); err != nil {
		t.Fatal(err)
	}

	filename := filepath.Join(outDir, "Sources", "GoogleCloudTestV1", "IAM.swift")
	content, err := os.ReadFile(filename)
	if err != nil {
		t.Fatal(err)
	}
	contentStr := string(content)

	for _, want := range []string{
		"let inner: any Clients.IAMStub",
		"var inner: any Clients.IAMStub = try Clients.IAMTransport(options)",
		"try await self.inner.createRole(request: request, options: options)",
	} {
		if !strings.Contains(contentStr, want) {
			t.Errorf("expected %q in IAM.swift, got:\n%s", want, contentStr)
		}
	}

	transportFilename := filepath.Join(outDir, "Sources", "GoogleCloudTestV1", "IAM+Transport.swift")
	transportContent, err := os.ReadFile(transportFilename)
	if err != nil {
		t.Fatal(err)
	}
	wantNewRequest := "var req = try await self.inner.newRequest(percentEncodedPath: path, query: query, options: options)"
	if !bytes.Contains(transportContent, []byte(wantNewRequest)) {
		t.Errorf("expected %q in IAM+Transport.swift, got:\n%s", wantNewRequest, string(transportContent))
	}
}

func TestGenerateService_SnippetFiles(t *testing.T) {
	outDir := t.TempDir()

	packageName := "google.cloud.test.v1"
	dummyMessage := api.NewTestMessage("DummyMessage").WithPackage(packageName)
	iam := api.NewTestService("IAM").WithPackage(packageName).WithMethods(
		api.NewTestMethod("CreateRole").
			WithInput(dummyMessage).
			WithVerb("POST").
			WithPathTemplate(&api.PathTemplate{}),
	)
	secretManager := api.NewTestService("SecretManagerService").WithPackage(packageName).WithMethods(
		api.NewTestMethod("GetSecret").
			WithInput(dummyMessage).
			WithVerb("GET").
			WithPathTemplate(&api.PathTemplate{}),
	)

	model := api.NewTestAPI([]*api.Message{dummyMessage}, nil, []*api.Service{iam, secretManager})
	model.PackageName = packageName

	library := &config.Library{
		Swift: swiftConfig(t, nil),
	}
	if err := Generate(t.Context(), model, outDir, library, nil); err != nil {
		t.Fatal(err)
	}

	expectedDir := filepath.Join(outDir, "Snippets")
	expectedFiles := []string{
		"IAMQuickstart.swift",
		"SecretManagerServiceQuickstart.swift",
		"IAM_CreateRole.swift",
		"SecretManagerService_GetSecret.swift",
	}
	for _, expected := range expectedFiles {
		filename := filepath.Join(expectedDir, expected)
		if _, err := os.Stat(filename); err != nil {
			t.Error(err)
		}
	}
}

func TestGenerateService_WithImports(t *testing.T) {
	outDir := t.TempDir()

	externalMessage := api.NewTestMessage("ExternalMessage").
		WithPackage("google.cloud.external.v1")

	inputMessage := api.NewTestMessage("LocalMessage").
		WithPackage("google.cloud.test.v1").
		WithFields(
			api.NewTestField("ext_field").
				WithType(api.TypezMessage).
				WithTypezID(".google.cloud.external.v1.ExternalMessage"),
		)

	iam := api.NewTestService("IAM").
		WithPackage("google.cloud.test.v1").
		WithMethods(
			api.NewTestMethod("TestMethod").
				WithInput(inputMessage).
				WithOutput(externalMessage).
				WithVerb("POST").
				WithPathTemplate(&api.PathTemplate{}),
		)

	model := api.NewTestAPI([]*api.Message{inputMessage}, nil, []*api.Service{iam})
	model.PackageName = "google.cloud.test.v1"
	model.AddMessage(externalMessage)

	swiftCfg := swiftConfig(t, []config.SwiftDependency{
		{
			Name:               "GoogleGax",
			RequiredByServices: true,
		},
		{
			Name:               "GoogleAuth",
			RequiredByServices: true,
		},
		{
			ApiPackage: "google.cloud.external.v1",
			Name:       "GoogleCloudExternalV1",
		},
	})

	library := &config.Library{
		Swift: swiftCfg,
	}
	if err := Generate(t.Context(), model, outDir, library, nil); err != nil {
		t.Fatal(err)
	}

	expectedDir := filepath.Join(outDir, "Sources", "GoogleCloudTestV1")
	filename := filepath.Join(expectedDir, "IAM.swift")
	content, err := os.ReadFile(filename)
	if err != nil {
		t.Fatal(err)
	}
	contentStr := string(content)

	expectedImports := `import GoogleCloudExternalV1
import GoogleWKT
import GoogleGax`

	if !strings.Contains(contentStr, expectedImports) {
		t.Errorf("expected imports block not found in %s. Got content:\n%s", filename, contentStr)
	}
}

func TestGenerateService_PathParameters(t *testing.T) {
	for _, test := range []struct {
		name      string
		path      *api.PathTemplate
		wantBlock string
	}{
		{
			name: "Nested",
			path: (&api.PathTemplate{}).
				WithLiteral("v1").
				WithVariableNamed("secret", "name"),
			wantBlock: `let (path, query, configure) = try { () throws -> (Swift.String, [URLQueryItem], (inout GoogleGax._HTTPClientRequest) -> Void) in
        if let candidate = try { () throws -> (Swift.String, [URLQueryItem])? in
          guard
            let pathVariable0 = try GoogleGax._RoutingMatcher.pathValue(
              request.secret.map({ $0.name }),
              matching: [.singleWildcard],
              fieldName: "secret.name")
          else {
            return nil
          }
          let path = "/v1/\(pathVariable0)"
          let query = [
            URLQueryItem(name: "$alt", value: "json;enum-encoding=int"),
          ]
          return (path, query)
        }() {
          return (candidate.0, candidate.1, { $0.setMethod(.POST) })
        }
        var paths: [GoogleGax.PathMismatch] = []
        do {
          var builder = GoogleGax._PathMismatchBuilder()
          builder.maybeAdd(
            request.secret.map({ $0.name }),
            matching: [.singleWildcard],
            fieldName: "secret.name",
            expecting: "*"
          )
          paths.append(builder.build())
        }
        throw GoogleGax.RequestError.binding(GoogleGax.BindingError(paths: paths))
      }()`,
		},
		{
			name: "Plain",
			path: (&api.PathTemplate{}).
				WithLiteral("v1").
				WithVariableNamed("name"),
			wantBlock: `let (path, query, configure) = try { () throws -> (Swift.String, [URLQueryItem], (inout GoogleGax._HTTPClientRequest) -> Void) in
        if let candidate = try { () throws -> (Swift.String, [URLQueryItem])? in
          guard
            let pathVariable0 = try GoogleGax._RoutingMatcher.pathValue(
              request.name as Swift.String?,
              matching: [.singleWildcard],
              fieldName: "name")
          else {
            return nil
          }
          let path = "/v1/\(pathVariable0)"
          let query = [
            URLQueryItem(name: "$alt", value: "json;enum-encoding=int"),
          ]
          return (path, query)
        }() {
          return (candidate.0, candidate.1, { $0.setMethod(.POST) })
        }
        var paths: [GoogleGax.PathMismatch] = []
        do {
          var builder = GoogleGax._PathMismatchBuilder()
          builder.maybeAdd(
            request.name as Swift.String?,
            matching: [.singleWildcard],
            fieldName: "name",
            expecting: "*"
          )
          paths.append(builder.build())
        }
        throw GoogleGax.RequestError.binding(GoogleGax.BindingError(paths: paths))
      }()`,
		},
		{
			name: "Multiple strings",
			path: (&api.PathTemplate{}).
				WithLiteral("v1").
				WithLiteral("projects").
				WithVariableNamed("project").
				WithLiteral("locations").
				WithVariableNamed("location"),
			wantBlock: `let (path, query, configure) = try { () throws -> (Swift.String, [URLQueryItem], (inout GoogleGax._HTTPClientRequest) -> Void) in
        if let candidate = try { () throws -> (Swift.String, [URLQueryItem])? in
          guard
            let pathVariable0 = try GoogleGax._RoutingMatcher.pathValue(
              request.project as Swift.String?,
              matching: [.singleWildcard],
              fieldName: "project")
          else {
            return nil
          }
          guard
            let pathVariable1 = try GoogleGax._RoutingMatcher.pathValue(
              request.location,
              matching: [.singleWildcard],
              fieldName: "location")
          else {
            return nil
          }
          let path = "/v1/projects/\(pathVariable0)/locations/\(pathVariable1)"
          let query = [
            URLQueryItem(name: "$alt", value: "json;enum-encoding=int"),
          ]
          return (path, query)
        }() {
          return (candidate.0, candidate.1, { $0.setMethod(.POST) })
        }
        var paths: [GoogleGax.PathMismatch] = []
        do {
          var builder = GoogleGax._PathMismatchBuilder()
          builder.maybeAdd(
            request.project as Swift.String?,
            matching: [.singleWildcard],
            fieldName: "project",
            expecting: "*"
          )
          builder.maybeAdd(
            request.location,
            matching: [.singleWildcard],
            fieldName: "location",
            expecting: "*"
          )
          paths.append(builder.build())
        }
        throw GoogleGax.RequestError.binding(GoogleGax.BindingError(paths: paths))
      }()`,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			outDir := t.TempDir()

			secretMessage := api.NewTestMessage("Secret").
				WithPackage("google.cloud.secretmanager.v1").
				WithFields(
					api.NewTestField("name").WithType(api.TypezString),
				)

			requestMessage := api.NewTestMessage("CreateSecretRequest").
				WithPackage("google.cloud.secretmanager.v1").
				WithFields(
					api.NewTestField("name").WithType(api.TypezString),
					api.NewTestField("secret").
						WithType(api.TypezMessage).
						WithTypezID(".google.cloud.secretmanager.v1.Secret").
						WithOptional(),
					api.NewTestField("project").WithType(api.TypezString),
					api.NewTestField("location").
						WithType(api.TypezString).
						WithOptional(),
				)

			iam := api.NewTestService("SecretManagerService").
				WithPackage("google.cloud.secretmanager.v1").
				WithMethods(
					api.NewTestMethod("CreateSecret").
						WithInput(requestMessage).
						WithVerb("POST").
						WithPathTemplate(test.path),
				)

			model := api.NewTestAPI([]*api.Message{requestMessage, secretMessage}, nil, []*api.Service{iam})
			model.PackageName = "google.cloud.secretmanager.v1"

			library := &config.Library{
				Swift: swiftConfig(t, nil),
			}
			if err := Generate(t.Context(), model, outDir, library, nil); err != nil {
				t.Fatal(err)
			}

			filename := filepath.Join(outDir, "Sources", "GoogleCloudSecretmanagerV1", "SecretManagerService+Transport.swift")
			content, err := os.ReadFile(filename)
			if err != nil {
				t.Fatal(err)
			}
			contentStr := string(content)

			gotBlock := extractBlock(t, contentStr, "let (path, query, configure) = try { () throws -> (Swift.String, [URLQueryItem], (inout GoogleGax._HTTPClientRequest) -> Void) in", "\n      }()")
			if diff := cmp.Diff(test.wantBlock, gotBlock); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestGenerateService_Pagination(t *testing.T) {
	for _, test := range []struct {
		name              string
		optional          bool
		wantNextPageToken string
	}{
		{
			name:     "Required",
			optional: false,
			wantNextPageToken: `public func _nextPageToken() -> Swift.String {
    return self.nextPageToken
  }`,
		},
		{
			name:     "Optional",
			optional: true,
			wantNextPageToken: `public func _nextPageToken() -> Swift.String {
    return self.nextPageToken ?? ""
  }`,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			outDir := t.TempDir()

			pageSizeField := api.NewTestField("page_size").WithType(api.TypezInt32)
			pageTokenField := api.NewTestField("page_token").WithType(api.TypezString)
			inputType := api.NewTestMessage("ListSecretsRequest").
				WithPackage("google.cloud.secretmanager.v1").
				WithFields(pageSizeField, pageTokenField)

			itemField := api.NewTestField("secrets").
				WithType(api.TypezMessage).
				WithTypezID(".google.cloud.secretmanager.v1.Secret").
				WithRepeated()
			nextPageTokenField := api.NewTestField("next_page_token").
				WithType(api.TypezString)
			if test.optional {
				nextPageTokenField.WithOptional()
			}
			outputType := api.NewTestMessage("ListSecretsResponse").
				WithPackage("google.cloud.secretmanager.v1").
				WithFields(itemField, nextPageTokenField).
				WithPagination(nextPageTokenField, itemField)

			secretType := api.NewTestMessage("Secret").
				WithPackage("google.cloud.secretmanager.v1")

			listSecrets := api.NewTestMethod("ListSecrets").
				WithInput(inputType).
				WithOutput(outputType).
				WithVerb("GET").
				WithPathTemplate((&api.PathTemplate{}).WithLiteral("v1").WithLiteral("secrets")).
				WithPagination(pageTokenField)
			listSecrets.Documentation = "Lists secrets."

			iam := api.NewTestService("SecretManagerService").
				WithPackage("google.cloud.secretmanager.v1").
				WithMethods(listSecrets)

			model := api.NewTestAPI([]*api.Message{inputType, outputType, secretType}, nil, []*api.Service{iam})
			model.PackageName = "google.cloud.secretmanager.v1"

			swiftCfg := swiftConfig(t, []config.SwiftDependency{
				{
					Name:               "GoogleGax",
					RequiredByServices: true,
				},
				{
					Name:               "GoogleAuth",
					RequiredByServices: true,
				},
			})

			library := &config.Library{
				Swift: swiftCfg,
			}
			if err := Generate(t.Context(), model, outDir, library, nil); err != nil {
				t.Fatal(err)
			}

			verifyGeneratedService(t, outDir)
			verifyGeneratedRequest(t, outDir)
			verifyGeneratedResponse(t, outDir, test.wantNextPageToken)
			verifyGeneratedMessage(t, outDir)
		})
	}
}

func verifyGeneratedService(t *testing.T, outDir string) {
	t.Helper()
	// Verify generated Service source code
	filename := filepath.Join(outDir, "Sources", "GoogleCloudSecretmanagerV1", "SecretManagerService.swift")
	content, err := os.ReadFile(filename)
	if err != nil {
		t.Fatal(err)
	}
	contentStr := string(content)

	gotMethodOverload := extractBlock(t, contentStr, `  public func listSecrets(
    byItem: ListSecretsRequest, options: `, "\n  }")
	wantMethodOverload := `  public func listSecrets(
    byItem: ListSecretsRequest, options: GoogleGax.RequestOptions
) throws -> any AsyncSequence<Secret, Swift.Error>
 {
    let listRpc = { (token: Swift.String) async throws -> GoogleCloudSecretmanagerV1.ListSecretsResponse in
      var request = byItem
      request.pageToken = token
      return try await self.listSecrets(request: request, options: options)
    }
    return GoogleGax.PaginatedResponseSequence(listRpc: listRpc)
  }`
	if diff := cmp.Diff(wantMethodOverload, gotMethodOverload); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func verifyGeneratedRequest(t *testing.T, outDir string) {
	t.Helper()
	// Verify generated Request and Response Messages source code
	msgFilename := filepath.Join(outDir, "Sources", "GoogleCloudSecretmanagerV1", "ListSecretsRequest.swift")
	msgContent, err := os.ReadFile(msgFilename)
	if err != nil {
		t.Fatal(err)
	}
	msgContentStr := string(msgContent)

	gotRequestMessage := extractBlock(t, msgContentStr, "public struct ListSecretsRequest: ", "{")
	for _, p := range []string{"Codable", "Equatable", "GoogleWKT._AnyPackable", "Sendable"} {
		if !strings.Contains(gotRequestMessage, p) {
			t.Errorf("expected %q in ListSecretsRequest declaration, got: %s", p, gotRequestMessage)
		}
	}

}

func verifyGeneratedResponse(t *testing.T, outDir string, wantNextPageToken string) {
	t.Helper()
	respFilename := filepath.Join(outDir, "Sources", "GoogleCloudSecretmanagerV1", "ListSecretsResponse.swift")
	respContent, err := os.ReadFile(respFilename)
	if err != nil {
		t.Fatal(err)
	}
	respContentStr := string(respContent)

	gotResponseMessage := extractBlock(t, respContentStr, "public struct ListSecretsResponse: ", "{")
	for _, p := range []string{"Codable", "Equatable", "GoogleWKT._AnyPackable", "GoogleGax._PaginatedResponse", "Sendable"} {
		if !strings.Contains(gotResponseMessage, p) {
			t.Errorf("expected %q in ListSecretsResponse declaration, got: %s", p, gotResponseMessage)
		}
	}

	gotGetItems := extractBlock(t, respContentStr, "public func _getPaginatedItems()", "  }")
	wantGetItems := `public func _getPaginatedItems() -> [Secret] {
    return self.secrets
  }`
	if diff := cmp.Diff(wantGetItems, gotGetItems); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}

	gotNextPageToken := extractBlock(t, respContentStr, "public func _nextPageToken()", "  }")
	if diff := cmp.Diff(wantNextPageToken, gotNextPageToken); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}

	if !strings.Contains(respContentStr, "import GoogleGax") {
		t.Errorf("expected ListSecretsResponse.swift to import GoogleGax, got:\n%s", respContentStr)
	}
}

func verifyGeneratedMessage(t *testing.T, outDir string) {
	t.Helper()
	secretFilename := filepath.Join(outDir, "Sources", "GoogleCloudSecretmanagerV1", "Secret.swift")
	secretContent, err := os.ReadFile(secretFilename)
	if err != nil {
		t.Fatal(err)
	}
	secretContentStr := string(secretContent)
	if strings.Contains(secretContentStr, "import GoogleGax") {
		t.Errorf("expected Secret.swift to NOT import GoogleGax, got:\n%s", secretContentStr)
	}
}

func TestGenerateService_LRO(t *testing.T) {
	outDir := t.TempDir()

	operationType := api.NewTestMessage("Operation").
		WithPackage("google.longrunning")

	workflowType := api.NewTestMessage("Workflow").
		WithPackage("google.cloud.workflows.v1")

	metadataType := api.NewTestMessage("OperationMetadata").
		WithPackage("google.cloud.workflows.v1")

	inputType := api.NewTestMessage("CreateWorkflowRequest").
		WithPackage("google.cloud.workflows.v1")

	getOperationInputType := api.NewTestMessage("GetOperationRequest").
		WithPackage("google.longrunning")

	createWorkflow := api.NewTestMethod("CreateWorkflow").
		WithInput(inputType).
		WithOutput(operationType).
		WithVerb("POST").
		WithPathTemplate((&api.PathTemplate{}).WithLiteral("v1").WithLiteral("workflows")).
		WithOperationInfo(&api.OperationInfo{
			ResponseTypeID: workflowType.ID,
			MetadataTypeID: metadataType.ID,
		})
	createWorkflow.Documentation = "Creates a workflow."

	getOperation := api.NewTestMethod("GetOperation").
		WithInput(getOperationInputType).
		WithOutput(operationType).
		WithVerb("GET").
		WithPathTemplate((&api.PathTemplate{}).WithLiteral("v1").WithLiteral("operations"))

	workflows := api.NewTestService("WorkflowsService").
		WithPackage("google.cloud.workflows.v1").
		WithMethods(createWorkflow, getOperation)

	model := api.NewTestAPI([]*api.Message{inputType, workflowType, metadataType, operationType, getOperationInputType}, nil, []*api.Service{workflows})
	model.PackageName = "google.cloud.workflows.v1"

	swiftCfg := swiftConfig(t, []config.SwiftDependency{
		{
			Name:               "GoogleGax",
			RequiredByServices: true,
		},
		{
			Name:               "GoogleAuth",
			RequiredByServices: true,
		},
		{
			ApiPackage: "google.longrunning",
			Name:       "GoogleCloudLongrunningV1",
		},
		{
			ApiPackage: "google.rpc",
			Name:       "GoogleRpc",
		},
	})

	library := &config.Library{
		Swift: swiftCfg,
	}
	if err := Generate(t.Context(), model, outDir, library, nil); err != nil {
		t.Fatal(err)
	}

	filename := filepath.Join(outDir, "Sources", "GoogleCloudWorkflowsV1", "WorkflowsService.swift")
	contentBytes, err := os.ReadFile(filename)
	if err != nil {
		t.Fatal(err)
	}
	content := string(contentBytes)

	wantContains := []string{
		"import GoogleRpc",
		"public func createWorkflow(withPolling: CreateWorkflowRequest) async throws -> any GoogleGax.PollableOperation<Workflow>",
		"self.getOperation(request: .init().with { $0.name = rawOp.name }, options: options)",
	}
	for _, want := range wantContains {
		if !strings.Contains(content, want) {
			t.Errorf("expected %q in WorkflowsService.swift, got:\n%s", want, content)
		}
	}

	got := extractBlock(t, content, "GoogleGax._PollableOperationImpl(", "\n    )")
	want := `GoogleGax._PollableOperationImpl(
      initialState: initialState,
      polling: options.pollingErrorPolicy ?? self.pollingErrorPolicy,
      backoff: options.pollingBackoffPolicy ?? self.pollingBackoffPolicy,
      poll: poll,
    )`
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestGenerateService_LRO_Empty(t *testing.T) {
	outDir := t.TempDir()

	operationType := api.NewTestMessage("Operation").
		WithPackage("google.longrunning")

	metadataType := api.NewTestMessage("OperationMetadata").
		WithPackage("google.cloud.workflows.v1")

	inputType := api.NewTestMessage("DeleteWorkflowRequest").
		WithPackage("google.cloud.workflows.v1")

	getOperationInputType := api.NewTestMessage("GetOperationRequest").
		WithPackage("google.longrunning")

	deleteWorkflow := api.NewTestMethod("DeleteWorkflow").
		WithInput(inputType).
		WithOutput(operationType).
		WithVerb("DELETE").
		WithPathTemplate((&api.PathTemplate{}).WithLiteral("v1").WithLiteral("workflows")).
		WithOperationInfo(&api.OperationInfo{
			ResponseTypeID: ".google.protobuf.Empty",
			MetadataTypeID: metadataType.ID,
		})
	deleteWorkflow.Documentation = "Deletes a workflow."

	getOperation := api.NewTestMethod("GetOperation").
		WithInput(getOperationInputType).
		WithOutput(operationType).
		WithVerb("GET").
		WithPathTemplate((&api.PathTemplate{}).WithLiteral("v1").WithLiteral("operations"))

	workflows := api.NewTestService("WorkflowsService").
		WithPackage("google.cloud.workflows.v1").
		WithMethods(deleteWorkflow, getOperation)

	model := api.NewTestAPI([]*api.Message{inputType, metadataType, operationType, getOperationInputType}, nil, []*api.Service{workflows})
	model.PackageName = "google.cloud.workflows.v1"

	swiftCfg := swiftConfig(t, []config.SwiftDependency{
		{
			Name:               "GoogleGax",
			RequiredByServices: true,
		},
		{
			Name:               "GoogleAuth",
			RequiredByServices: true,
		},
		{
			ApiPackage: "google.longrunning",
			Name:       "GoogleCloudLongrunningV1",
		},
		{
			ApiPackage: "google.rpc",
			Name:       "GoogleRpc",
		},
	})

	library := &config.Library{
		Swift: swiftCfg,
	}
	if err := Generate(t.Context(), model, outDir, library, nil); err != nil {
		t.Fatal(err)
	}

	filename := filepath.Join(outDir, "Sources", "GoogleCloudWorkflowsV1", "WorkflowsService.swift")
	content, err := os.ReadFile(filename)
	if err != nil {
		t.Fatal(err)
	}
	contentStr := string(content)

	wantContains := []string{
		"public func deleteWorkflow(withPolling: DeleteWorkflowRequest) async throws -> any GoogleGax.PollableOperation<Swift.Void>",
		"GoogleGax._PollableOperationImpl<Swift.Void>",
	}
	for _, want := range wantContains {
		if !strings.Contains(contentStr, want) {
			t.Errorf("expected %q in WorkflowsService.swift, got:\n%s", want, contentStr)
		}
	}
}

func TestGenerateDiscoveryService_Files(t *testing.T) {
	testdataDir, err := filepath.Abs("../../testdata")
	if err != nil {
		t.Fatal(err)
	}
	outDir := t.TempDir()

	cfg := &parser.ModelConfig{
		SpecificationFormat: config.SpecDiscovery,
		ServiceConfig:       filepath.Join(testdataDir, "googleapis/google/cloud/compute/v1/small-compute_v1.yaml"),
		SpecificationSource: filepath.Join(testdataDir, "discovery/small-compute.v1.json"),
	}
	model, err := parser.CreateModel(cfg)
	if err != nil {
		t.Fatal(err)
	}

	swiftCfg := swiftConfig(t, []config.SwiftDependency{
		{
			Name:               "GoogleGax",
			RequiredByServices: true,
		},
		{
			Name:               "GoogleAuth",
			RequiredByServices: true,
		},
	})
	library := &config.Library{
		SpecificationFormat: config.SpecDiscovery,
		Swift:               swiftCfg,
	}
	if err := Generate(t.Context(), model, outDir, library, nil); err != nil {
		t.Fatal(err)
	}

	// Verify files
	expectedDir := filepath.Join(outDir, "Sources", "GoogleCloudComputeV1")

	for _, test := range []struct {
		filename    string
		serviceName string
		structName  string
	}{
		{
			filename:    "AcceleratorTypes+Requests.swift",
			serviceName: "AcceleratorTypes",
			structName:  "ListRequest",
		},
		{
			filename:    "Addresses+Requests.swift",
			serviceName: "Addresses",
			structName:  "DeleteRequest",
		},
		{
			filename:    "instances+Requests.swift",
			serviceName: "Instances",
			structName:  "GetRequest",
		},
	} {
		t.Run(test.serviceName, func(t *testing.T) {
			filename := filepath.Join(expectedDir, test.filename)
			content, err := os.ReadFile(filename)
			if err != nil {
				t.Fatal(err)
			}

			// Verify it contains an extension to the right ${ServiceName}Client type.
			wantExtension := fmt.Appendf(nil, "extension %sClient {", test.serviceName)
			if !bytes.Contains(content, wantExtension) {
				t.Errorf("expected extension %q in %s, got:\n%s", wantExtension, filename, content)
			}

			// Verify the request struct definition appears in that file.
			wantStruct := fmt.Appendf(nil, "public struct %s: ", test.structName)
			if !bytes.Contains(content, wantStruct) {
				t.Errorf("expected struct %q in %s, got:\n%s", wantStruct, filename, content)
			}
		})
	}
}

func TestGenerateService_WildcardBodyOmitsPathFields(t *testing.T) {
	outDir := t.TempDir()

	secretMessage := api.NewTestMessage("Secret").
		WithPackage("google.cloud.secretmanager.v1").
		WithFields(
			api.NewTestField("name").WithType(api.TypezString),
		)
	requestMessage := api.NewTestMessage("SecretRequest").
		WithPackage("google.cloud.secretmanager.v1").
		WithFields(
			api.NewTestField("parent").WithType(api.TypezString),
			api.NewTestField("alternative_parent").WithType(api.TypezString),
			api.NewTestField("secret").
				WithType(api.TypezMessage).
				WithTypezID(".google.cloud.secretmanager.v1.Secret").
				WithOptional(),
		)

	createSecret := api.NewTestMethod("CreateSecret").
		WithInput(requestMessage)
	createSecret.PathInfo = &api.PathInfo{
		BodyFieldPath: "*",
		Bindings: []*api.PathBinding{{
			Verb:         "POST",
			PathTemplate: (&api.PathTemplate{}).WithLiteral("v1").WithVariableNamed("parent"),
		}},
	}

	updateSecret := api.NewTestMethod("UpdateSecret").
		WithInput(requestMessage)
	updateSecret.PathInfo = &api.PathInfo{
		BodyFieldPath: "*",
		Bindings: []*api.PathBinding{{
			Verb:         "PATCH",
			PathTemplate: (&api.PathTemplate{}).WithLiteral("v1").WithVariableNamed("secret", "name"),
		}},
	}

	addSecretVersion := api.NewTestMethod("AddSecretVersion").
		WithInput(requestMessage)
	addSecretVersion.PathInfo = &api.PathInfo{
		BodyFieldPath: "*",
		Bindings: []*api.PathBinding{
			{
				Verb:         "POST",
				PathTemplate: (&api.PathTemplate{}).WithLiteral("v1").WithVariableNamed("parent"),
			},
			{
				Verb:         "POST",
				PathTemplate: (&api.PathTemplate{}).WithLiteral("v1").WithVariableNamed("alternative_parent"),
			},
		},
	}

	patchSecret := api.NewTestMethod("PatchSecret").
		WithInput(requestMessage)
	patchSecret.PathInfo = &api.PathInfo{
		BodyFieldPath: "secret",
		Bindings: []*api.PathBinding{{
			Verb:         "PATCH",
			PathTemplate: (&api.PathTemplate{}).WithLiteral("v1").WithVariableNamed("parent"),
		}},
	}

	service := api.NewTestService("SecretManagerService").
		WithPackage("google.cloud.secretmanager.v1").
		WithMethods(createSecret, updateSecret, addSecretVersion, patchSecret)

	model := api.NewTestAPI([]*api.Message{requestMessage, secretMessage}, nil, []*api.Service{service})
	model.PackageName = "google.cloud.secretmanager.v1"

	library := &config.Library{
		Swift: swiftConfig(t, nil),
	}
	if err := Generate(t.Context(), model, outDir, library, nil); err != nil {
		t.Fatal(err)
	}

	filename := filepath.Join(outDir, "Sources", "GoogleCloudSecretmanagerV1", "SecretManagerService+Transport.swift")
	content, err := os.ReadFile(filename)
	if err != nil {
		t.Fatal(err)
	}

	// The fields bound by the path template are not part of the request body.
	want := []string{
		`let (path, query, configure, omitted) = try { () throws -> (Swift.String, [URLQueryItem], (inout GoogleGax._HTTPClientRequest) -> Void, [Swift.String]) in`,
		`return (candidate.0, candidate.1, { $0.setMethod(.POST) }, ["parent"])`,
		`return (candidate.0, candidate.1, { $0.setMethod(.PATCH) }, ["secret.name"])`,
		`return (candidate.0, candidate.1, { $0.setMethod(.POST) }, ["alternativeParent"])`,
		`try req.setBody(json: request, omitting: omitted)`,
	}
	for _, w := range want {
		if !bytes.Contains(content, []byte(w)) {
			t.Errorf("expected %q in %s, got:\n%s", w, filename, content)
		}
	}

	// Methods with a named body field keep the previous shape: the body is a separate field, it
	// cannot contain the path parameters.
	want = []string{
		`let (path, query, configure) = try { () throws -> (Swift.String, [URLQueryItem], (inout GoogleGax._HTTPClientRequest) -> Void) in`,
		`return (candidate.0, candidate.1, { $0.setMethod(.PATCH) })`,
		`try req.setBody(json: body)`,
	}
	for _, w := range want {
		if !bytes.Contains(content, []byte(w)) {
			t.Errorf("expected %q in %s, got:\n%s", w, filename, content)
		}
	}

	if bytes.Contains(content, []byte("try req.setBody(json: request)\n")) {
		t.Errorf("unexpected unfiltered request body in %s, got:\n%s", filename, content)
	}
}
