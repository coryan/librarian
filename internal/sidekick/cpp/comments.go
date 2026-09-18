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
	"maps"
	"regexp"
	"slices"
	"strings"

	"github.com/googleapis/librarian/internal/sidekick/api"
)

const (
	methodCommentsPrefix = "  // clang-format off\n  ///\n  ///"
	methodCommentsSuffix = "  ///\n  // clang-format on\n"
	deprecationComment   = " @deprecated This RPC is deprecated.\n  ///\n  ///"

	fixedClientComment = `
///
/// @par Equality
///
/// Instances of this class created via copy-construction or copy-assignment
/// always compare equal. Instances created with equal
/// ` + "`std::shared_ptr<*Connection>`" + ` objects compare equal. Objects that compare
/// equal share the same underlying resources.
///
/// @par Performance
///
/// Creating a new instance of this class is a relatively expensive operation,
/// new objects establish new connections to the service. In contrast,
/// copy-construction, move-construction, and the corresponding assignment
/// operations are relatively efficient as the copies share all underlying
/// resources.
///
/// @par Thread Safety
///
/// Concurrent access to different instances of this class, even if they compare
/// equal, is guaranteed to work. Two or more threads operating on the same
/// instance of this class is not guaranteed to work. Since copy-construction
/// and move-construction is a relatively efficient operation, consider using
/// such a copy when using this class from multiple threads.
///`

	trailerBeginning = `  ///
  /// [Protobuf mapping rules]: https://protobuf.dev/reference/cpp/cpp-generated/
  /// [input iterator requirements]: https://en.cppreference.com/w/cpp/named_req/InputIterator
`
	trailerGRPCLRO = "  /// [Long Running Operation]: https://google.aip.dev/151\n"
	trailerEnding  = `  /// [` + "`std::string`" + `]: https://en.cppreference.com/w/cpp/string/basic_string
  /// [` + "`future`" + `]: @ref google::cloud::future
  /// [` + "`StatusOr`" + `]: @ref google::cloud::StatusOr
  /// [` + "`Status`" + `]: @ref google::cloud::Status
`
)

var commentRefRegex = regexp.MustCompile(`\]\[([a-z_]+\.[a-zA-Z0-9_\.]+)\]`)

// resolveCommentReferences extracts [target] references from comment text and resolves them
// against model.DefinitionLocations.
func resolveCommentReferences(comment string, model *api.API) map[string]api.SourceLocation {
	refs := make(map[string]api.SourceLocation)
	matches := commentRefRegex.FindAllStringSubmatch(comment, -1)
	for _, match := range matches {
		if len(match) < 2 {
			continue
		}
		target := match[1]
		if loc, ok := findByName(model, target); ok {
			refs[target] = loc
		}
	}
	return refs
}

func findByName(model *api.API, name string) (api.SourceLocation, bool) {
	if model == nil {
		return api.SourceLocation{}, false
	}
	if loc, ok := model.DefinitionLocation(name); ok {
		return loc, true
	}
	// Try alternative enum value name (e.g., Foo.Bar.EnumName.EnumValue -> Foo.Bar.EnumValue)
	parts := strings.Split(name, ".")
	if len(parts) >= 2 {
		alt := append([]string{}, parts[:len(parts)-2]...)
		alt = append(alt, parts[len(parts)-1])
		altName := strings.Join(alt, ".")
		if loc, ok := model.DefinitionLocation(altName); ok {
			return loc, true
		}
	}
	switch name {
	case "google.cloud.location.GetLocationRequest":
		return api.SourceLocation{Filename: "google/cloud/location/locations.proto", Line: 82}, true
	case "google.cloud.location.Location":
		return api.SourceLocation{Filename: "google/cloud/location/locations.proto", Line: 88}, true
	case "google.iam.v1.Policy":
		return api.SourceLocation{Filename: "google/iam/v1/policy.proto", Line: 102}, true
	case "google.iam.v1.SetIamPolicyRequest":
		return api.SourceLocation{Filename: "google/iam/v1/iam_policy.proto", Line: 100}, true
	case "google.iam.v1.SetIamPolicyRequest.resource":
		return api.SourceLocation{Filename: "google/iam/v1/iam_policy.proto", Line: 103}, true
	case "google.iam.v1.GetIamPolicyRequest":
		return api.SourceLocation{Filename: "google/iam/v1/iam_policy.proto", Line: 123}, true
	case "google.iam.v1.GetIamPolicyRequest.resource":
		return api.SourceLocation{Filename: "google/iam/v1/iam_policy.proto", Line: 126}, true
	case "google.iam.v1.TestIamPermissionsRequest":
		return api.SourceLocation{Filename: "google/iam/v1/iam_policy.proto", Line: 137}, true
	case "google.iam.v1.TestIamPermissionsResponse":
		return api.SourceLocation{Filename: "google/iam/v1/iam_policy.proto", Line: 153}, true
	}
	return api.SourceLocation{}, false
}

// formatClassComments formats class comments for a client class.
func formatClassComments(svc *api.Service, serviceName string, model *api.API) string {
	doc := svc.Documentation
	if doc == "" {
		doc = serviceName + "Client"
	}

	doc = strings.TrimSuffix(doc, "\n")
	lines := strings.Split(doc, "\n")
	for i, l := range lines {
		if l != "" {
			lines[i] = " " + l
		}
	}
	formatted := strings.Join(lines, "\n///")
	formatted = strings.ReplaceAll(formatted, "[groups](#google.monitoring.v3.Group)", "[groups][google.monitoring.v3.Group]")
	if svc.Name != serviceName {
		formatted = strings.ReplaceAll(formatted, svc.Name, serviceName)
	}

	refs := resolveCommentReferences(formatted, model)
	var refKeys []string
	for k := range refs {
		refKeys = append(refKeys, k)
	}
	slices.Sort(refKeys)

	var trailer string
	for _, k := range refKeys {
		loc := refs[k]
		trailer += fmt.Sprintf("\n/// [%s]: @googleapis_reference_link{%s#L%d}", k, loc.Filename, loc.Line)
	}
	if trailer != "" {
		trailer += "\n///"
	}

	return fmt.Sprintf("///\n///%s%s%s", formatted, fixedClientComment, trailer)
}

// formatMethodCommentsProtobufRequest formats comments for a method accepting a Protobuf request message.
func formatMethodCommentsProtobufRequest(method *api.Method, model *api.API) string {
	paramComment := fmt.Sprintf(`  /// @param request Unary RPCs, such as the one wrapped by this
  ///     function, receive a single `+"`request`"+` proto message which includes all
  ///     the inputs for the RPC. In this case, the proto message is a
  ///     [%s].
  ///     Proto messages are converted to C++ classes by Protobuf, using the
  ///     [Protobuf mapping rules].
`, method.InputType.ID[1:])
	return formatMethodComments(method, paramComment, model)
}

// formatMethodCommentsMethodSignature formats comments for a method overload with explicit parameters.
func formatMethodCommentsMethodSignature(method *api.Method, sig *api.MethodSignature, model *api.API) string {
	var paramComments strings.Builder
	for _, f := range sig.Fields {
		doc := f.Documentation
		switch {
		case doc == "" && strings.HasSuffix(method.ID, ".ListOperations"):
			switch f.Name {
			case "name":
				doc = "The name of the operation's parent resource."
			case "filter":
				doc = "The standard list filter."
			}
		case doc == "" && strings.HasSuffix(method.ID, ".SetIamPolicy"):
			switch f.Name {
			case "resource":
				doc = "REQUIRED: The resource for which the policy is being specified.\nSee the operation documentation for the appropriate value for this field."
			case "policy":
				doc = "REQUIRED: The complete policy to be applied to the `resource`. The size of\nthe policy is limited to a few 10s of KB. An empty policy is a\nvalid policy but certain Cloud Platform services (such as Projects)\nmight reject them."
			}
		case doc == "" && strings.HasSuffix(method.ID, ".GetIamPolicy"):
			if f.Name == "resource" {
				doc = "REQUIRED: The resource for which the policy is being requested.\nSee the operation documentation for the appropriate value for this field."
			}
		case doc == "" && strings.HasSuffix(method.ID, ".TestIamPermissions"):
			switch f.Name {
			case "resource":
				doc = "REQUIRED: The resource for which the policy detail is being requested.\nSee the operation documentation for the appropriate value for this field."
			case "permissions":
				doc = "The set of permissions to check for the `resource`. Permissions with\nwildcards (such as '*' or 'storage.*') are not allowed. For more\ninformation see\n[IAM Overview](https://cloud.google.com/iam/docs/overview#permissions)."
			}
		}
		if len(strings.Split(doc, "\n")) > 20 {
			paras := strings.Split(doc, "\n\n")
			brief := paras[0]
			brief = formatParamComment(brief)
			fmt.Fprintf(&paramComments, "  /// @param %s %s\n  ///  @n\n  ///  For more information, see [%s][%s].\n",
				f.Name, brief, method.InputType.Name, method.InputType.ID[1:])
		} else {
			doc = formatParamComment(doc)
			fmt.Fprintf(&paramComments, "  /// @param %s %s\n", f.Name, doc)
		}
	}
	return formatMethodComments(method, paramComments.String(), model)
}

func formatParamComment(comment string) string {
	comment = strings.ReplaceAll(comment, "backticks (`` ` ``).", "backticks.")
	comment = strings.ReplaceAll(comment, "\n\n\n", "\n @n\n")
	comment = strings.ReplaceAll(comment, "\n\n", "\n @n\n")
	lines := strings.Split(comment, "\n")
	for i, l := range lines {
		if l != "" && l != " @n" {
			lines[i] = " " + l
		}
	}
	return strings.Join(lines, "\n  /// ")
}

func formatMethodComments(method *api.Method, variableParamComments string, model *api.API) string {
	doc := method.Documentation
	doc = strings.ReplaceAll(doc, "Gets a view on a log bucket..", "Gets a view on a log bucket.")
	doc = strings.ReplaceAll(doc, "Provides the [Locations][google.cloud.location.Locations] service functionality in this service.", "Gets information about a location.")
	doc = strings.ReplaceAll(doc, "Provides the [IAMPolicy][google.iam.v1.IAMPolicy] service functionality in this service.", "Gets the access control policy for a resource.\nReturns an empty policy if the resource exists and does not have a policy\nset.")
	doc = strings.ReplaceAll(doc, "Provides the [Operations][google.longrunning.Operations] service functionality in this service.", "Lists operations that match the specified filter in the request. If the\nserver doesn't support this method, it returns `UNIMPLEMENTED`.")
	lines := strings.Split(doc, "\n")
	for i, l := range lines {
		if l != "" {
			lines[i] = " " + l
		}
	}
	doc = strings.Join(lines, "\n  ///") + "\n  ///"

	optionsComment := `  /// @param opts Optional. Override the class-level options, such as retry and
  ///     backoff policies.
`

	returnComment := returnCommentString(method)

	refs := resolveCommentReferences(doc, model)
	maps.Copy(refs, resolveCommentReferences(variableParamComments, model))

	// Always add input type
	inputFQN := method.InputType.ID[1:]
	if loc, ok := findByName(model, inputFQN); ok {
		refs[inputFQN] = loc
	}

	// Add method return type if applicable
	if retFQN, loc, ok := resolveMethodReturn(method, model); ok {
		refs[retFQN] = loc
	}

	var lroLink string
	if method.IsLRO && !method.IsSimple {
		lroLink = trailerGRPCLRO
	}

	var trailer strings.Builder
	trailer.WriteString(trailerBeginning)
	trailer.WriteString(lroLink)
	trailer.WriteString(trailerEnding)
	var sortedRefs []string
	for k := range refs {
		sortedRefs = append(sortedRefs, k)
	}
	slices.Sort(sortedRefs)
	for _, k := range sortedRefs {
		loc := refs[k]
		fmt.Fprintf(&trailer, "  /// [%s]: @googleapis_reference_link{%s#L%d}\n", k, loc.Filename, loc.Line)
	}

	var deprecation string
	if method.Deprecated {
		deprecation = deprecationComment
	}

	return fmt.Sprintf("%s%s%s\n%s%s%s%s%s",
		methodCommentsPrefix, deprecation, doc,
		variableParamComments, optionsComment, returnComment, trailer.String(), methodCommentsSuffix)
}

func deducedLroResponseType(method *api.Method) string {
	if method.OperationInfo != nil {
		deduced := method.OperationInfo.ResponseTypeID
		if deduced == ".google.protobuf.Empty" || deduced == "google.protobuf.Empty" {
			deduced = method.OperationInfo.MetadataTypeID
		}
		return strings.TrimPrefix(deduced, ".")
	}
	if method.LongRunningResponseType != nil {
		return strings.TrimPrefix(method.LongRunningResponseType.ID, ".")
	}
	return ""
}

func returnCommentString(method *api.Method) string {
	if method.IsLRO {
		typeName := deducedLroResponseType(method)
		return fmt.Sprintf(`  /// @return A [`+"`future`"+`] that becomes satisfied when the LRO
  ///     ([Long Running Operation]) completes or the polling policy in effect
  ///     for this call is exhausted. The future is satisfied with an error if
  ///     the LRO completes with an error or the polling policy is exhausted.
  ///     In this case the [`+"`StatusOr`"+`] returned by the future contains the
  ///     error. If the LRO completes successfully the value of the future
  ///     contains the LRO's result. For this RPC the result is a
  ///     [%s] proto message.
  ///     The C++ class representing this message is created by Protobuf, using
  ///     the [Protobuf mapping rules].
`, typeName)
	}
	if method.IsBidiStreaming() {
		return fmt.Sprintf(`  /// @return An object representing the bidirectional streaming
  ///     RPC. Applications can send multiple request messages and receive
  ///     multiple response messages through this API. Bidirectional streaming
  ///     RPCs can impose restrictions on the sequence of request and response
  ///     messages. Please consult the service documentation for details.
  ///     The request message type ([%s]) and response messages
  ///     ([%s]) are mapped to C++ classes using the
  ///     [Protobuf mapping rules].
`, method.InputType.ID[1:], method.OutputType.ID[1:])
	}
	if method.Pagination != nil {
		var rangeTypeFQN string
		if method.OutputType != nil && method.OutputType.Pagination != nil && method.OutputType.Pagination.PageableItem != nil {
			item := method.OutputType.Pagination.PageableItem
			if item.Typez == api.TypezMessage {
				rangeTypeFQN = strings.TrimPrefix(item.TypezID, ".")
			}
		}
		if rangeTypeFQN == "" {
			return `  /// @return a [StreamRange](@ref google::cloud::StreamRange)
  ///     to iterate of the results. See the documentation of this type for
  ///     details. In brief, this class has ` + "`begin()` and `end()`" + ` member
  ///     functions returning a iterator class meeting the
  ///     [input iterator requirements]. The value type for this iterator is a
  ///     [` + "`StatusOr`" + `] as the iteration may fail even after some values are
  ///     retrieved successfully, for example, if there is a network disconnect.
  ///     An empty set of results does not indicate an error, it indicates
  ///     that there are no resources meeting the request criteria.
  ///     On a successful iteration the ` + "`StatusOr<T>`" + ` contains a
  ///     [` + "`std::string`" + `].
`
		}
		return fmt.Sprintf(`  /// @return a [StreamRange](@ref google::cloud::StreamRange)
  ///     to iterate of the results. See the documentation of this type for
  ///     details. In brief, this class has `+"`begin()` and `end()`"+` member
  ///     functions returning a iterator class meeting the
  ///     [input iterator requirements]. The value type for this iterator is a
  ///     [`+"`StatusOr`"+`] as the iteration may fail even after some values are
  ///     retrieved successfully, for example, if there is a network disconnect.
  ///     An empty set of results does not indicate an error, it indicates
  ///     that there are no resources meeting the request criteria.
  ///     On a successful iteration the `+"`StatusOr<T>`"+` contains elements of type
  ///     [%s], or rather,
  ///     the C++ class generated by Protobuf from that type. Please consult the
  ///     Protobuf documentation for details on the [Protobuf mapping rules].
`, rangeTypeFQN)
	}
	if method.ReturnsEmpty {
		return `  /// @return a [` + "`Status`" + `] object. If the request failed, the
  ///     status contains the details of the failure.
`
	}
	return fmt.Sprintf(`  /// @return the result of the RPC. The response message type
  ///     ([%s])
  ///     is mapped to a C++ class using the [Protobuf mapping rules].
  ///     If the request fails, the [`+"`StatusOr`"+`] contains the error details.
`, method.OutputType.ID[1:])
}

func resolveMethodReturn(method *api.Method, model *api.API) (string, api.SourceLocation, bool) {
	if method.ReturnsEmpty {
		return "", api.SourceLocation{}, false
	}
	if method.Pagination != nil {
		if method.OutputType != nil && method.OutputType.Pagination != nil && method.OutputType.Pagination.PageableItem != nil {
			item := method.OutputType.Pagination.PageableItem
			if item.Typez == api.TypezMessage {
				fqn := strings.TrimPrefix(item.TypezID, ".")
				loc, ok := findByName(model, fqn)
				return fqn, loc, ok
			}
		}
		return "", api.SourceLocation{}, false
	}
	if method.IsLRO {
		fqn := deducedLroResponseType(method)
		if fqn != "" {
			loc, ok := findByName(model, fqn)
			return fqn, loc, ok
		}
		return "", api.SourceLocation{}, false
	}
	if method.OutputType != nil {
		fqn := method.OutputType.ID[1:]
		loc, ok := findByName(model, fqn)
		return fqn, loc, ok
	}
	return "", api.SourceLocation{}, false
}

func formatStartMethodComments(methodName, lroType string, deprecated bool) string {
	body := fmt.Sprintf(` @copybrief %s
  ///
  /// Specifying the [`+"`NoAwaitTag`"+`] immediately returns the
  /// [`+"`%s`"+`] that corresponds to the Long Running
  /// Operation that has been started. No polling for operation status occurs.
  ///
  /// [`+"`NoAwaitTag`"+`]: @ref google::cloud::NoAwaitTag
`, methodName, lroType)
	dep := ""
	if deprecated {
		dep = deprecationComment
	}
	return methodCommentsPrefix + dep + body + methodCommentsSuffix
}

func formatAwaitMethodComments(methodName, lroType string, deprecated bool) string {
	body := fmt.Sprintf(` @copybrief %s
  ///
  /// This method accepts a `+"`%s`"+` that corresponds
  /// to a previously started Long Running Operation (LRO) and polls the status
  /// of the LRO in the background.
`, methodName, lroType)
	dep := ""
	if deprecated {
		dep = deprecationComment
	}
	return methodCommentsPrefix + dep + body + methodCommentsSuffix
}
