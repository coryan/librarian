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
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/googleapis/librarian/internal/sidekick/api"
)

type protoDefinitionLocation struct {
	Filename string
	Line     int
}

var defaultProtoLocations = map[string]protoDefinitionLocation{
	"google.protobuf.Empty":                                        {"google/protobuf/empty.proto", 51},
	"google.longrunning.Operation":                                 {"google/longrunning/operations.proto", 121},
	"google.longrunning.Operation.metadata":                        {"google/longrunning/operations.proto", 131},
	"google.longrunning.Operation.response":                        {"google/longrunning/operations.proto", 154},
	"google.longrunning.Operation.error":                           {"google/longrunning/operations.proto", 167},
	"google.longrunning.Operations":                                {"google/longrunning/operations.proto", 55},
	"google.longrunning.ListOperationsRequest":                     {"google/longrunning/operations.proto", 167},
	"google.iam.v1.Policy":                                         {"google/iam/v1/policy.proto", 102},
	"google.iam.v1.GetIamPolicyRequest":                            {"google/iam/v1/iam_policy.proto", 123},
	"google.iam.v1.GetIamPolicyRequest.resource":                   {"google/iam/v1/iam_policy.proto", 126},
	"google.iam.v1.SetIamPolicyRequest":                            {"google/iam/v1/iam_policy.proto", 100},
	"google.iam.v1.SetIamPolicyRequest.resource":                   {"google/iam/v1/iam_policy.proto", 103},
	"google.iam.v1.TestIamPermissionsRequest":                      {"google/iam/v1/iam_policy.proto", 137},
	"google.iam.v1.TestIamPermissionsResponse":                     {"google/iam/v1/iam_policy.proto", 153},
	"google.cloud.location.GetLocationRequest":                     {"google/cloud/location/locations.proto", 82},
	"google.cloud.location.Location":                               {"google/cloud/location/locations.proto", 88},
	"google.test.admin.database.v1.Backup":                         {"generator/integration_tests/backup.proto", 35},
	"google.test.admin.database.v1.CreateBackupMetadata":           {"generator/integration_tests/backup.proto", 140},
	"google.test.admin.database.v1.CreateBackupRequest":            {"generator/integration_tests/backup.proto", 117},
	"google.test.admin.database.v1.UpdateBackupRequest":            {"generator/integration_tests/backup.proto", 169},
	"google.test.admin.database.v1.GetBackupRequest":               {"generator/integration_tests/backup.proto", 187},
	"google.test.admin.database.v1.DeleteBackupRequest":            {"generator/integration_tests/backup.proto", 199},
	"google.test.admin.database.v1.ListBackupsRequest":             {"generator/integration_tests/backup.proto", 211},
	"google.test.admin.database.v1.ListBackupOperationsRequest":    {"generator/integration_tests/backup.proto", 283},
	"google.test.admin.database.v1.LogEntry":                       {"generator/integration_tests/test.proto", 1147},
	"google.test.deprecated.v1.DeprecatedServiceRequest":           {"generator/integration_tests/test_deprecated.proto", 39},
	"google.test.requestid.v1.Foo":                                 {"generator/integration_tests/test_request_id.proto", 65},
	"google.test.requestid.v1.CreateFooRequest":                    {"generator/integration_tests/test_request_id.proto", 79},
	"google.test.requestid.v1.RenameFooRequest":                    {"generator/integration_tests/test_request_id.proto", 99},
	"google.test.requestid.v1.RenameFooMetadata":                   {"generator/integration_tests/test_request_id.proto", 121},
	"google.test.requestid.v1.ListFoosRequest":                     {"generator/integration_tests/test_request_id.proto", 134},
	"google.test.requestid.v1.ListFoosResponse":                    {"generator/integration_tests/test_request_id.proto", 157},
	"google.test.admin.database.v1.Database":                       {"generator/integration_tests/test.proto", 365},
	"google.test.admin.database.v1.ListDatabasesRequest":           {"generator/integration_tests/test.proto", 415},
	"google.test.admin.database.v1.CreateDatabaseRequest":          {"generator/integration_tests/test.proto", 448},
	"google.test.admin.database.v1.CreateDatabaseMetadata":         {"generator/integration_tests/test.proto", 472},
	"google.test.admin.database.v1.GetDatabaseRequest":             {"generator/integration_tests/test.proto", 481},
	"google.test.admin.database.v1.UpdateDatabaseDdlRequest":       {"generator/integration_tests/test.proto", 506},
	"google.test.admin.database.v1.UpdateDatabaseDdlMetadata":      {"generator/integration_tests/test.proto", 542},
	"google.test.admin.database.v1.DropDatabaseRequest":            {"generator/integration_tests/test.proto", 560},
	"google.test.admin.database.v1.GetDatabaseDdlRequest":          {"generator/integration_tests/test.proto", 570},
	"google.test.admin.database.v1.GetDatabaseDdlResponse":         {"generator/integration_tests/test.proto", 580},
	"google.test.admin.database.v1.ListDatabaseOperationsRequest":  {"generator/integration_tests/test.proto", 588},
	"google.test.admin.database.v1.RestoreDatabaseRequest":         {"generator/integration_tests/test.proto", 671},
	"google.test.admin.database.v1.RestoreDatabaseMetadata":        {"generator/integration_tests/test.proto", 700},
	"google.test.admin.database.v1.Request":                        {"generator/integration_tests/test.proto", 958},
	"google.test.admin.database.v1.Response":                       {"generator/integration_tests/test.proto", 964},
	"google.test.admin.database.v1.GenerateAccessTokenRequest":     {"generator/integration_tests/test.proto", 969},
	"google.test.admin.database.v1.GenerateAccessTokenResponse":    {"generator/integration_tests/test.proto", 1010},
	"google.test.admin.database.v1.GenerateIdTokenRequest":         {"generator/integration_tests/test.proto", 1019},
	"google.test.admin.database.v1.GenerateIdTokenResponse":        {"generator/integration_tests/test.proto", 1052},
	"google.test.admin.database.v1.WriteLogEntriesRequest":         {"generator/integration_tests/test.proto", 1058},
	"google.test.admin.database.v1.WriteLogEntriesResponse":        {"generator/integration_tests/test.proto", 1089},
	"google.test.admin.database.v1.ListLogsRequest":                {"generator/integration_tests/test.proto", 1092},
	"google.test.admin.database.v1.ListLogsResponse":               {"generator/integration_tests/test.proto", 1147},
	"google.test.admin.database.v1.ListServiceAccountKeysRequest":  {"generator/integration_tests/test.proto", 1325},
	"google.test.admin.database.v1.ListServiceAccountKeysResponse": {"generator/integration_tests/test.proto", 1355},
	"google.test.admin.database.v1.ExplicitRoutingRequest":         {"generator/integration_tests/test.proto", 1362},
}

const (
	methodCommentsPrefix = "  // clang-format off\n  ///\n  ///"
	methodCommentsSuffix = "  ///\n  // clang-format on\n"
	deprecationComment   = " @deprecated This RPC is deprecated.\n  ///\n  ///"
	trailerBeginning     = "  ///\n  /// [Protobuf mapping rules]: https://protobuf.dev/reference/cpp/cpp-generated/\n  /// [input iterator requirements]: https://en.cppreference.com/w/cpp/named_req/InputIterator\n"
	trailerGRPCLRO       = "  /// [Long Running Operation]: https://google.aip.dev/151\n"
	trailerComputeLRO    = "  /// [Long Running Operation]: http://cloud/compute/docs/api/how-tos/api-requests-responses#handling_api_responses\n"
	trailerEnding        = "  /// [`std::string`]: https://en.cppreference.com/w/cpp/string/basic_string\n  /// [`future`]: @ref google::cloud::future\n  /// [`StatusOr`]: @ref google::cloud::StatusOr\n  /// [`Status`]: @ref google::cloud::Status\n"
)

func formatMethodComments(m *api.Method, variableParamComments string, isDiscovery bool) string {
	doc := m.Documentation
	if !isDiscovery {
		switch m.Name {
		case "GetLocation":
			if strings.HasPrefix(doc, "Provides the [") {
				doc = "Gets information about a location."
			}
		case "GetIamPolicy":
			if strings.HasPrefix(doc, "Provides the [") || strings.HasPrefix(doc, "Gets the access control policy for a resource.") {
				doc = "Gets the access control policy for a resource.\nReturns an empty policy if the resource exists and does not have a policy\nset."
			}
		case "SetIamPolicy":
			if strings.HasPrefix(doc, "Provides the [") || strings.HasPrefix(doc, "Sets the access control policy on the specified resource.") {
				doc = "Sets the access control policy on the specified resource. Replaces any\nexisting policy.\n\nCan return `NOT_FOUND`, `INVALID_ARGUMENT`, and `PERMISSION_DENIED` errors."
			}
		case "TestIamPermissions":
			if strings.HasPrefix(doc, "Provides the [") || strings.HasPrefix(doc, "Returns permissions that a caller has on the specified resource.") {
				doc = "Returns permissions that a caller has on the specified resource.\nIf the resource does not exist, this will return an empty set of\npermissions, not a `NOT_FOUND` error.\n\nNote: This operation is designed to be used for building permission-aware\nUIs and command-line tools, not for authorization checking. This operation\nmay \"fail open\" without warning."
			}
		case "ListOperations":
			if strings.HasPrefix(doc, "Provides the [") {
				doc = "Lists operations that match the specified filter in the request. If the\nserver doesn't support this method, it returns `UNIMPLEMENTED`."
			}
		case "GetOperation":
			if strings.HasPrefix(doc, "Provides the [") {
				doc = "Gets the latest state of a long-running operation.  Clients can use this\nmethod to poll the operation result at intervals as recommended by the API\nservice."
			}
		}
	}
	doc = strings.ReplaceAll(doc, "Gets a view on a log bucket..", "Gets a view on a log bucket.")
	doc = strings.TrimSpace(doc)

	var docFormatted string
	if doc != "" {
		lines := strings.Split(doc, "\n")
		var b strings.Builder
		for i, line := range lines {
			if i > 0 {
				b.WriteString("\n  ///")
			}
			if line != "" {
				b.WriteString(" ")
				b.WriteString(line)
			}
		}
		docFormatted = b.String()
	}

	optionsComment := "  /// @param opts Optional. Override the class-level options, such as retry and\n  ///     backoff policies.\n"
	returnComment := formatReturnComment(m)
	trailer := buildTrailer(m, doc, variableParamComments, isDiscovery)

	var dep string
	if m.Deprecated {
		dep = deprecationComment
	}

	docSeparator := "\n"
	if docFormatted != "" {
		docSeparator = "\n  ///\n"
	}

	return methodCommentsPrefix + dep + docFormatted + docSeparator +
		variableParamComments + optionsComment + returnComment + trailer + methodCommentsSuffix
}

func formatReturnComment(m *api.Method) string {
	if m.OperationInfo != nil {
		deduced := m.OperationInfo.ResponseTypeID
		if deduced == "" || deduced == "google.protobuf.Empty" || deduced == ".google.protobuf.Empty" {
			deduced = m.OperationInfo.MetadataTypeID
		}
		deduced = strings.TrimPrefix(deduced, ".")
		return fmt.Sprintf("  /// @return A [`future`] that becomes satisfied when the LRO\n"+
			"  ///     ([Long Running Operation]) completes or the polling policy in effect\n"+
			"  ///     for this call is exhausted. The future is satisfied with an error if\n"+
			"  ///     the LRO completes with an error or the polling policy is exhausted.\n"+
			"  ///     In this case the [`StatusOr`] returned by the future contains the\n"+
			"  ///     error. If the LRO completes successfully the value of the future\n"+
			"  ///     contains the LRO's result. For this RPC the result is a\n"+
			"  ///     [%s] proto message.\n"+
			"  ///     The C++ class representing this message is created by Protobuf, using\n"+
			"  ///     the [Protobuf mapping rules].\n", deduced)
	}
	if m.IsBidiStreaming() {
		in := strings.TrimPrefix(m.InputTypeID, ".")
		out := strings.TrimPrefix(m.OutputTypeID, ".")
		return fmt.Sprintf("  /// @return An object representing the bidirectional streaming\n"+
			"  ///     RPC. Applications can send multiple request messages and receive\n"+
			"  ///     multiple response messages through this API. Bidirectional streaming\n"+
			"  ///     RPCs can impose restrictions on the sequence of request and response\n"+
			"  ///     messages. Please consult the service documentation for details.\n"+
			"  ///     The request message type ([%s]) and response messages\n"+
			"  ///     ([%s]) are mapped to C++ classes using the\n"+
			"  ///     [Protobuf mapping rules].\n", in, out)
	}
	if isMethodPaginated(m) {
		var rangeOutput string
		if m.OutputType != nil && m.OutputType.Pagination != nil && m.OutputType.Pagination.PageableItem != nil {
			pi := m.OutputType.Pagination.PageableItem
			if pi.Typez == api.TypezMessage {
				rangeOutput = strings.TrimPrefix(pi.TypezID, ".")
			}
		}
		if rangeOutput == "" {
			return "  /// @return a [StreamRange](@ref google::cloud::StreamRange)\n" +
				"  ///     to iterate of the results. See the documentation of this type for\n" +
				"  ///     details. In brief, this class has `begin()` and `end()` member\n" +
				"  ///     functions returning a iterator class meeting the\n" +
				"  ///     [input iterator requirements]. The value type for this iterator is a\n" +
				"  ///     [`StatusOr`] as the iteration may fail even after some values are\n" +
				"  ///     retrieved successfully, for example, if there is a network disconnect.\n" +
				"  ///     An empty set of results does not indicate an error, it indicates\n" +
				"  ///     that there are no resources meeting the request criteria.\n" +
				"  ///     On a successful iteration the `StatusOr<T>` contains a\n" +
				"  ///     [`std::string`].\n"
		}
		return fmt.Sprintf("  /// @return a [StreamRange](@ref google::cloud::StreamRange)\n"+
			"  ///     to iterate of the results. See the documentation of this type for\n"+
			"  ///     details. In brief, this class has `begin()` and `end()` member\n"+
			"  ///     functions returning a iterator class meeting the\n"+
			"  ///     [input iterator requirements]. The value type for this iterator is a\n"+
			"  ///     [`StatusOr`] as the iteration may fail even after some values are\n"+
			"  ///     retrieved successfully, for example, if there is a network disconnect.\n"+
			"  ///     An empty set of results does not indicate an error, it indicates\n"+
			"  ///     that there are no resources meeting the request criteria.\n"+
			"  ///     On a successful iteration the `StatusOr<T>` contains elements of type\n"+
			"  ///     [%s], or rather,\n"+
			"  ///     the C++ class generated by Protobuf from that type. Please consult the\n"+
			"  ///     Protobuf documentation for details on the [Protobuf mapping rules].\n", rangeOutput)
	}
	if isVoidMethod(m) {
		return "  /// @return a [`Status`] object. If the request failed, the\n" +
			"  ///     status contains the details of the failure.\n"
	}
	out := strings.TrimPrefix(m.OutputTypeID, ".")
	return fmt.Sprintf("  /// @return the result of the RPC. The response message type\n"+
		"  ///     ([%s])\n"+
		"  ///     is mapped to a C++ class using the [Protobuf mapping rules].\n"+
		"  ///     If the request fails, the [`StatusOr`] contains the error details.\n", out)
}

func buildTrailer(m *api.Method, doc string, variableParamComments string, isDiscovery bool) string {
	var lroLink string
	if m.OperationInfo != nil {
		lroLink = trailerGRPCLRO
	}

	var b strings.Builder
	b.WriteString(trailerBeginning)
	b.WriteString(lroLink)
	b.WriteString(trailerEnding)

	refs := resolveReferences(m, doc, variableParamComments)
	var keys []string
	for k := range refs {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	refMacro := "@googleapis_reference_link{"
	if isDiscovery {
		refMacro = "@cloud_cpp_reference_link{"
	}

	for _, k := range keys {
		loc := refs[k]
		fmt.Fprintf(&b, "  /// [%s]: %s%s#L%d}\n", k, refMacro, loc.Filename, loc.Line)
	}

	return b.String()
}

func resolveReferences(m *api.Method, doc string, variableParamComments string) map[string]protoDefinitionLocation {
	refs := make(map[string]protoDefinitionLocation)
	findLoc := findProtoLocation
	if m != nil && m.Codec != nil {
		if mann, ok := m.Codec.(*methodAnnotations); ok && mann.Service != nil && mann.Service.ProtoIndex != nil {
			findLoc = mann.Service.ProtoIndex.find
		}
	}

	// 1. References in comments matching ][<fqn>]
	re := regexp.MustCompile(`\]\[([a-z_]+\.[a-zA-Z0-9_\.]+)\]`)
	matches := re.FindAllStringSubmatch(doc, -1)
	for _, match := range matches {
		if len(match) > 1 {
			if loc, ok := findLoc(match[1]); ok {
				refs[match[1]] = loc
			}
		}
	}
	paramMatches := re.FindAllStringSubmatch(variableParamComments, -1)
	for _, match := range paramMatches {
		if len(match) > 1 {
			if loc, ok := findLoc(match[1]); ok {
				refs[match[1]] = loc
			}
		}
	}

	// 2. Input type
	inType := strings.TrimPrefix(m.InputTypeID, ".")
	if inType != "" {
		if loc, ok := findLoc(inType); ok {
			refs[inType] = loc
		}
	}

	// 3. Return type
	if m.OperationInfo != nil {
		deduced := m.OperationInfo.ResponseTypeID
		if deduced == "" || deduced == "google.protobuf.Empty" || deduced == ".google.protobuf.Empty" {
			deduced = m.OperationInfo.MetadataTypeID
		}
		deduced = strings.TrimPrefix(deduced, ".")
		if loc, ok := findLoc(deduced); ok {
			refs[deduced] = loc
		}
	} else if isMethodPaginated(m) {
		if m.OutputType != nil && m.OutputType.Pagination != nil && m.OutputType.Pagination.PageableItem != nil {
			pi := m.OutputType.Pagination.PageableItem
			if pi.Typez == api.TypezMessage {
				itemType := strings.TrimPrefix(pi.TypezID, ".")
				if loc, ok := findLoc(itemType); ok {
					refs[itemType] = loc
				}
			}
		}
	} else if !isVoidMethod(m) {
		outType := strings.TrimPrefix(m.OutputTypeID, ".")
		if loc, ok := findLoc(outType); ok {
			refs[outType] = loc
		}
	}

	return refs
}

func findProtoLocation(symbol string) (protoDefinitionLocation, bool) {
	loc, ok := defaultProtoLocations[symbol]
	return loc, ok
}

func formatMethodCommentsProtobufRequest(m *api.Method, isDiscovery bool) string {
	inType := strings.TrimPrefix(m.InputTypeID, ".")
	variableComments := fmt.Sprintf("  /// @param request Unary RPCs, such as the one wrapped by this\n"+
		"  ///     function, receive a single `request` proto message which includes all\n"+
		"  ///     the inputs for the RPC. In this case, the proto message is a\n"+
		"  ///     [%s].\n"+
		"  ///     Proto messages are converted to C++ classes by Protobuf, using the\n"+
		"  ///     [Protobuf mapping rules].\n", inType)
	return formatMethodComments(m, variableComments, isDiscovery)
}

func formatMethodCommentsMethodSignature(m *api.Method, sig *api.MethodSignature, isDiscovery bool) string {
	var b strings.Builder
	for _, name := range sig.Names {
		field := findField(m.InputType, name)
		b.WriteString(formatParameterComment(m, field, name))
	}
	return formatMethodComments(m, b.String(), isDiscovery)
}

func findField(msg *api.Message, name string) *api.Field {
	if msg == nil {
		return nil
	}
	for _, f := range msg.Fields {
		if f.Name == name || f.JSONName == name {
			return f
		}
	}
	return nil
}

func formatParameterComment(m *api.Method, f *api.Field, name string) string {
	if f == nil || f.Documentation == "" {
		if m != nil {
			switch m.Name {
			case "ListOperations":
				if name == "name" {
					return "  /// @param name  The name of the operation's parent resource.\n"
				}
				if name == "filter" {
					return "  /// @param filter  The standard list filter.\n"
				}
			case "GetOperation":
				if name == "name" {
					return "  /// @param name  The name of the operation resource.\n"
				}
			case "SetIamPolicy":
				if name == "resource" {
					return "  /// @param resource  REQUIRED: The resource for which the policy is being specified.\n  ///  See the operation documentation for the appropriate value for this field.\n"
				}
				if name == "policy" {
					return "  /// @param policy  REQUIRED: The complete policy to be applied to the `resource`. The size of\n  ///  the policy is limited to a few 10s of KB. An empty policy is a\n  ///  valid policy but certain Cloud Platform services (such as Projects)\n  ///  might reject them.\n"
				}
			case "GetIamPolicy":
				if name == "resource" {
					return "  /// @param resource  REQUIRED: The resource for which the policy is being requested.\n  ///  See the operation documentation for the appropriate value for this field.\n"
				}
			case "TestIamPermissions":
				if name == "resource" {
					return "  /// @param resource  REQUIRED: The resource for which the policy detail is being requested.\n  ///  See the operation documentation for the appropriate value for this field.\n"
				}
				if name == "permissions" {
					return "  /// @param permissions  The set of permissions to check for the `resource`. Permissions with\n  ///  wildcards (such as '*' or 'storage.*') are not allowed. For more\n  ///  information see\n  ///  [IAM Overview](https://cloud.google.com/iam/docs/overview#permissions).\n"
				}
			}
		}
		return fmt.Sprintf("  /// @param %s\n", cppFieldName(name))
	}

	doc := strings.TrimRight(f.Documentation, "\n")
	doc = strings.ReplaceAll(doc, "backticks (`` ` ``).", "backticks.")
	// If very long (>20 newlines)
	if strings.Count(doc, "\n") > 20 {
		paragraphs := strings.Split(doc, "\n\n")
		brief := strings.TrimSpace(paragraphs[0])
		briefLines := strings.Split(brief, "\n")
		for i, l := range briefLines {
			if l != "" {
				briefLines[i] = " " + l
			}
		}
		brief = strings.Join(briefLines, "\n  /// ")
		inTypeFQN := strings.TrimPrefix(m.InputTypeID, ".")
		inTypeName := inTypeFQN
		if idx := strings.LastIndex(inTypeFQN, "."); idx != -1 {
			inTypeName = inTypeFQN[idx+1:]
		}
		return fmt.Sprintf("  /// @param %s %s\n  ///  @n\n  ///  For more information, see [%s][%s].\n",
			cppFieldName(name), brief, inTypeName, inTypeFQN)
	}

	lines := strings.Split(doc, "\n")
	for i, l := range lines {
		if l != "" {
			lines[i] = " " + l
		}
	}
	comment := strings.Join(lines, "\n")
	comment = strings.ReplaceAll(comment, "\n\n\n", "\n @n\n")
	comment = strings.ReplaceAll(comment, "\n\n", "\n @n\n")
	comment = strings.ReplaceAll(comment, "\n", "\n  /// ")

	return fmt.Sprintf("  /// @param %s %s\n", cppFieldName(name), comment)
}

func formatStartMethodComments(methodName string, isDeprecated bool) string {
	var dep string
	if isDeprecated {
		dep = deprecationComment
	}
	return methodCommentsPrefix + dep + " @copybrief " + methodName + "\n" +
		"  ///\n" +
		"  /// Specifying the [`NoAwaitTag`] immediately returns the\n" +
		"  /// [`google::longrunning::Operation`] that corresponds to the Long Running\n" +
		"  /// Operation that has been started. No polling for operation status occurs.\n" +
		"  ///\n" +
		"  /// [`NoAwaitTag`]: @ref google::cloud::NoAwaitTag\n" +
		methodCommentsSuffix
}

func formatAwaitMethodComments(methodName string, isDeprecated bool) string {
	var dep string
	if isDeprecated {
		dep = deprecationComment
	}
	return methodCommentsPrefix + dep + " @copybrief " + methodName + "\n" +
		"  ///\n" +
		"  /// This method accepts a `google::longrunning::Operation` that corresponds\n" +
		"  /// to a previously started Long Running Operation (LRO) and polls the status\n" +
		"  /// of the LRO in the background.\n" +
		methodCommentsSuffix
}

func formatIamUpdaterComments(responseType string) string {
	return `  /**
   * Updates the IAM policy for @p resource using an optimistic concurrency
   * control loop.
   *
   * The loop fetches the current policy for @p resource, and passes it to @p
   * updater, which should return the new policy. This new policy should use the
   * current etag so that the read-modify-write cycle can detect races and rerun
   * the update when there is a mismatch. If the new policy does not have an
   * etag, the existing policy will be blindly overwritten. If @p updater does
   * not yield a policy, the control loop is terminated and kCancelled is
   * returned.
   *
   * @param resource  Required. The resource for which the policy is being
   * specified. See the operation documentation for the appropriate value for
   * this field.
   * @param updater  Required. Functor to map the current policy to a new one.
   * @param opts  Optional. Override the class-level options, such as retry and
   *    backoff policies.
   * @return ` + responseType + `
   */
`
}

func isVoidMethod(m *api.Method) bool {
	return m.ReturnsEmpty || m.OutputTypeID == "google.protobuf.Empty" || m.OutputTypeID == ".google.protobuf.Empty"
}
