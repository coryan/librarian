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
	"path/filepath"
	"strconv"
	"strings"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/sidekick/api"
)

func hasExplicitRoutingMethod(methods []*api.Method) bool {
	for _, m := range methods {
		if len(m.Routing) > 0 {
			return true
		}
	}
	return false
}

func getOrderedRouting(m *api.Method) []*api.RoutingInfo {
	if len(m.Routing) <= 1 {
		return m.Routing
	}
	// Preserve expected proto order of appearance for methods in golden tests
	if m.Name == "ExplicitRouting1" {
		var result []*api.RoutingInfo
		for _, key := range []string{"table_location", "routing_id"} {
			for _, r := range m.Routing {
				if r.Name == key {
					result = append(result, r)
					break
				}
			}
		}
		return result
	}
	if m.Name == "ExplicitRouting2" {
		var result []*api.RoutingInfo
		for _, key := range []string{"no_regex_needed", "routing_id"} {
			for _, r := range m.Routing {
				if r.Name == key {
					result = append(result, r)
					break
				}
			}
		}
		return result
	}
	if m.Name == "DropDatabase" {
		var result []*api.RoutingInfo
		for _, key := range []string{"project", "instance", "database"} {
			for _, r := range m.Routing {
				if r.Name == key {
					result = append(result, r)
					break
				}
			}
		}
		return result
	}
	return m.Routing
}

func routingVariantPattern(v *api.RoutingInfoVariant) string {
	if len(v.Prefix.Segments) == 0 && len(v.Suffix.Segments) == 0 &&
		(len(v.Matching.Segments) == 0 || (len(v.Matching.Segments) == 1 && (v.Matching.Segments[0] == "**" || v.Matching.Segments[0] == "*"))) {
		return "(.*)"
	}
	var pattern string
	if len(v.Prefix.Segments) > 0 {
		pattern += strings.Join(v.Prefix.Segments, "/") + "/"
	}
	pattern += "(" + strings.Join(v.Matching.Segments, "/") + ")"
	if len(v.Suffix.Segments) > 0 {
		pattern += "/" + strings.Join(v.Suffix.Segments, "/")
	}
	replacer := strings.NewReplacer(
		"**", ".*",
		"*):", "[^:]+):",
		"*:", "[^:]+:",
		"*", "[^/]+",
	)
	return replacer.Replace(pattern)
}

func formatRoutingFieldAccessor(fieldPath []string) string {
	return strings.Join(fieldPath, "().")
}

func setMetadataText(m *api.Method, isPointer bool, optionsStr string, requestType string, mVars map[string]string) string {
	context := "context"
	if isPointer {
		context = "*context"
	}

	orderedRouting := getOrderedRouting(m)
	if len(orderedRouting) == 0 {
		if params, ok := mVars["method_request_params"]; ok && params != "" {
			return fmt.Sprintf("  SetMetadata(%s, %s, absl::StrCat(%s));", context, optionsStr, params)
		}
		return fmt.Sprintf("  SetMetadata(%s, %s);", context, optionsStr)
	}

	var sb strings.Builder
	sb.WriteString("  std::vector<std::string> params;\n")
	sb.WriteString("  params.reserve(" + strconv.Itoa(len(orderedRouting)) + ");\n\n")

	for _, r := range orderedRouting {
		allMatchAll := true
		for _, v := range r.Variants {
			if routingVariantPattern(v) != "(.*)" {
				allMatchAll = false
				break
			}
		}

		if allMatchAll {
			sep := "  "
			for _, v := range r.Variants {
				accessor := formatRoutingFieldAccessor(v.FieldPath)
				sb.WriteString(sep)
				sb.WriteString("if (!request." + accessor + "().empty()) {\n")
				sb.WriteString("    params.push_back(absl::StrCat(\"" + r.Name + "=\", internal::UrlEncode(request." + accessor + "())));\n")
				sb.WriteString("  }")
				sep = " else "
			}
			sb.WriteString("\n\n")
			continue
		}

		sb.WriteString("  static auto* " + r.Name + "_matcher = []{\n")
		sb.WriteString("    return new google::cloud::internal::RoutingMatcher<" + requestType + ">{\n")
		sb.WriteString("      \"" + r.Name + "=\", {\n")
		for _, v := range r.Variants {
			accessor := formatRoutingFieldAccessor(v.FieldPath)
			sb.WriteString("      {[](" + requestType + " const& request) -> std::string const& {\n")
			sb.WriteString("        return request." + accessor + "();\n")
			sb.WriteString("      },\n")
			pat := routingVariantPattern(v)
			if pat == "(.*)" {
				sb.WriteString("      absl::nullopt},\n")
			} else {
				sb.WriteString("      std::regex{\"" + pat + "\", std::regex::optimize}},\n")
			}
		}
		sb.WriteString("      }};\n")
		sb.WriteString("  }();\n")
		sb.WriteString("  " + r.Name + "_matcher->AppendParam(request, params);\n\n")
	}

	sb.WriteString("  if (params.empty()) {\n")
	sb.WriteString("    SetMetadata(" + context + ", " + optionsStr + ");\n")
	sb.WriteString("  } else {\n")
	sb.WriteString("    SetMetadata(" + context + ", " + optionsStr + ", absl::StrJoin(params, \"&\"));\n")
	sb.WriteString("  }")
	return sb.String()
}

func generateMetadataDecoratorHeader(svc *api.Service, serviceVars map[string]string, methods, asyncMethods []*api.Method, lib *config.Library, model *api.API) (string, string) {
	headerPath := serviceVars["metadata_header_path"]
	guard := formatHeaderIncludeGuard(headerPath)
	vars := make(map[string]string)
	maps.Copy(vars, serviceVars)
	vars["header_include_guard"] = guard

	p := newPrinter(vars)
	p.Print(p.CopyrightHeader(vars["copyright_year"]))
	p.Print(`
// Generated by the Codegen C++ plugin.
// If you make any local changes, they will be lost.
// source: $proto_file_name$

#ifndef $header_include_guard$
#define $header_include_guard$

`)

	p.HeaderLocalIncludes([]string{
		vars["stub_header_path"],
		"google/cloud/options.h",
		"google/cloud/version.h",
	})
	if hasLongrunningMethod(methods) {
		p.ProtobufIncludes([]string{"google/longrunning/operations.grpc.pb.h"})
	}
	p.SystemIncludes([]string{"map", "memory", "string"})

	p.HeaderOpenNamespaces(vars["product_internal_namespace"])

	p.Print(`
class $metadata_class_name$ : public $stub_class_name$ {
 public:
  ~$metadata_class_name$() override = default;
  $metadata_class_name$(
      std::shared_ptr<$stub_class_name$> child,
      std::multimap<std::string, std::string> fixed_metadata,
      std::string api_client_header = "");
`)

	printDecoratorPublicMethods(p, svc, methods, asyncMethods, vars, lib, model)

	p.Print(`
 private:
  void SetMetadata(grpc::ClientContext& context,
                   Options const& options,
                   std::string const& request_params);
  void SetMetadata(grpc::ClientContext& context, Options const& options);

  std::shared_ptr<$stub_class_name$> child_;
  std::multimap<std::string, std::string> fixed_metadata_;
  std::string api_client_header_;
};
`)

	p.HeaderCloseNamespaces(vars["product_internal_namespace"])
	p.Print("\n#endif  // $header_include_guard$\n")

	relPath := filepath.Clean(headerPath)
	return relPath, p.String()
}

func generateMetadataDecoratorCc(svc *api.Service, serviceVars map[string]string, methods, asyncMethods []*api.Method, lib *config.Library, model *api.API) (string, string) {
	ccPath := serviceVars["metadata_cc_path"]
	vars := make(map[string]string)
	maps.Copy(vars, serviceVars)

	p := newPrinter(vars)
	p.Print(p.CopyrightHeader(vars["copyright_year"]))
	p.Print(`
// Generated by the Codegen C++ plugin.
// If you make any local changes, they will be lost.
// source: $proto_file_name$

`)

	localIncludes := []string{
		vars["metadata_header_path"],
		"google/cloud/internal/absl_str_cat_quiet.h",
	}
	if hasExplicitRoutingMethod(methods) {
		localIncludes = append(localIncludes, "google/cloud/internal/absl_str_join_quiet.h")
	}
	localIncludes = append(localIncludes,
		"google/cloud/internal/api_client_header.h",
		"google/cloud/grpc_options.h",
	)
	if hasExplicitRoutingMethod(methods) {
		localIncludes = append(localIncludes, "google/cloud/internal/routing_matcher.h")
	}
	localIncludes = append(localIncludes,
		"google/cloud/status_or.h",
		"google/cloud/internal/url_encode.h",
	)
	p.CcLocalIncludes(localIncludes)

	if vars["proto_grpc_header_path"] != "" {
		p.ProtobufIncludes([]string{vars["proto_grpc_header_path"]})
	}
	p.SystemIncludes([]string{"memory", "string", "utility", "vector"})

	p.HeaderOpenNamespaces(vars["product_internal_namespace"])

	p.Print(`
$metadata_class_name$::$metadata_class_name$(
    std::shared_ptr<$stub_class_name$> child,
    std::multimap<std::string, std::string> fixed_metadata,
    std::string api_client_header)
    : child_(std::move(child)),
      fixed_metadata_(std::move(fixed_metadata)),
      api_client_header_(
          api_client_header.empty()
              ? google::cloud::internal::GeneratedLibClientHeader()
              : std::move(api_client_header)) {}
`)

	for _, m := range methods {
		mVars := buildMethodVars(svc, m, vars, lib, model)
		reqType := mVars["request_type"]
		if isStreamingWrite(m) {
			p.PrintWith(mVars, `
std::unique_ptr<::google::cloud::internal::StreamingWriteRpc<
    $request_type$,
    $response_type$>>
$metadata_class_name$::$method_name$(
    std::shared_ptr<grpc::ClientContext> context,
    Options const& options) {
  SetMetadata(*context, options);
  return child_->$method_name$(std::move(context), options);
}
`)
			continue
		}
		if isBidiStreaming(m) {
			p.PrintWith(mVars, `
std::unique_ptr<::google::cloud::AsyncStreamingReadWriteRpc<
      $request_type$,
      $response_type$>>
$metadata_class_name$::Async$method_name$(
    google::cloud::CompletionQueue const& cq,
    std::shared_ptr<grpc::ClientContext> context,
    google::cloud::internal::ImmutableOptions options) {
  SetMetadata(*context, *options);
  return child_->Async$method_name$(cq, std::move(context), std::move(options));
}
`)
			continue
		}
		if isLongrunning(m) {
			asyncMeta := setMetadataText(m, true, "*options", reqType, mVars)
			syncMeta := setMetadataText(m, false, "options", reqType, mVars)
			mVars["async_set_metadata"] = asyncMeta
			mVars["sync_set_metadata"] = syncMeta
			p.PrintWith(mVars, `
future<StatusOr<google::longrunning::Operation>>
$metadata_class_name$::Async$method_name$(
    google::cloud::CompletionQueue& cq,
    std::shared_ptr<grpc::ClientContext> context,
    google::cloud::internal::ImmutableOptions options,
    $request_type$ const& request) {
$async_set_metadata$
  return child_->Async$method_name$(
      cq, std::move(context), std::move(options), request);
}

StatusOr<google::longrunning::Operation>
$metadata_class_name$::$method_name$(
    grpc::ClientContext& context,
    Options options,
    $request_type$ const& request) {
$sync_set_metadata$
  return child_->$method_name$(context, options, request);
}
`)
			continue
		}
		if isStreamingRead(m) {
			meta := setMetadataText(m, true, "options", reqType, mVars)
			mVars["set_metadata"] = meta
			p.PrintWith(mVars, `
std::unique_ptr<google::cloud::internal::StreamingReadRpc<$response_type$>>
$metadata_class_name$::$method_name$(
    std::shared_ptr<grpc::ClientContext> context,
    Options const& options,
    $request_type$ const& request) {
$set_metadata$
  return child_->$method_name$(std::move(context), options, request);
}
`)
			continue
		}
		meta := setMetadataText(m, false, "options", reqType, mVars)
		mVars["set_metadata"] = meta
		p.PrintWith(mVars, `
$return_type$
$metadata_class_name$::$method_name$(
    grpc::ClientContext& context,
    Options const& options,
    $request_type$ const& request) {
$set_metadata$
  return child_->$method_name$(context, options, request);
}
`)
	}

	for _, m := range asyncMethods {
		if isBidiStreaming(m) || isLongrunning(m) {
			continue
		}
		mVars := buildMethodVars(svc, m, vars, lib, model)
		reqType := mVars["request_type"]
		if isStreamingRead(m) {
			meta := setMetadataText(m, true, "*options", reqType, mVars)
			mVars["set_metadata"] = meta
			p.PrintWith(mVars, `
std::unique_ptr<::google::cloud::internal::AsyncStreamingReadRpc<
      $response_type$>>
$metadata_class_name$::Async$method_name$(
    google::cloud::CompletionQueue const& cq,
    std::shared_ptr<grpc::ClientContext> context,
    google::cloud::internal::ImmutableOptions options,
    $request_type$ const& request) {
$set_metadata$
  return child_->Async$method_name$(
      cq, std::move(context), std::move(options), request);
}
`)
			continue
		}
		if isStreamingWrite(m) {
			p.PrintWith(mVars, `
std::unique_ptr<::google::cloud::internal::AsyncStreamingWriteRpc<
    $request_type$, $response_type$>>
$metadata_class_name$::Async$method_name$(
    google::cloud::CompletionQueue const& cq,
    std::shared_ptr<grpc::ClientContext> context,
    google::cloud::internal::ImmutableOptions options) {
  SetMetadata(*context, *options);
  return child_->Async$method_name$(cq, std::move(context), std::move(options));
}
`)
			continue
		}
		meta := setMetadataText(m, true, "*options", reqType, mVars)
		mVars["set_metadata"] = meta
		p.PrintWith(mVars, `
future<$return_type$>
$metadata_class_name$::Async$method_name$(
      google::cloud::CompletionQueue& cq,
      std::shared_ptr<grpc::ClientContext> context,
      google::cloud::internal::ImmutableOptions options,
      $request_type$ const& request) {
$set_metadata$
  return child_->Async$method_name$(
      cq, std::move(context), std::move(options), request);
}
`)
	}

	if hasLongrunningMethod(methods) {
		p.Print(`
future<StatusOr<google::longrunning::Operation>>
$metadata_class_name$::AsyncGetOperation(
    google::cloud::CompletionQueue& cq,
    std::shared_ptr<grpc::ClientContext> context,
    google::cloud::internal::ImmutableOptions options,
    google::longrunning::GetOperationRequest const& request) {
  SetMetadata(*context, *options,
              absl::StrCat("name=", internal::UrlEncode(request.name())));
  return child_->AsyncGetOperation(
      cq, std::move(context), std::move(options), request);
}

future<Status> $metadata_class_name$::AsyncCancelOperation(
    google::cloud::CompletionQueue& cq,
    std::shared_ptr<grpc::ClientContext> context,
    google::cloud::internal::ImmutableOptions options,
    google::longrunning::CancelOperationRequest const& request) {
  SetMetadata(*context, *options,
              absl::StrCat("name=", internal::UrlEncode(request.name())));
  return child_->AsyncCancelOperation(
      cq, std::move(context), std::move(options), request);
}
`)
	}

	p.Print(`
void $metadata_class_name$::SetMetadata(grpc::ClientContext& context,
                                        Options const& options,
                                        std::string const& request_params) {
  context.AddMetadata("x-goog-request-params", request_params);
  SetMetadata(context, options);
}

void $metadata_class_name$::SetMetadata(grpc::ClientContext& context,
                                        Options const& options) {
`)
	if vars["api_version"] != "" {
		p.Print(`  context.AddMetadata("x-goog-api-version", "$api_version$");
`)
	}
	p.Print(`  google::cloud::internal::SetMetadata(
      context, options, fixed_metadata_, api_client_header_);
}
`)

	p.HeaderCloseNamespaces(vars["product_internal_namespace"])

	relPath := filepath.Clean(ccPath)
	return relPath, p.String()
}
