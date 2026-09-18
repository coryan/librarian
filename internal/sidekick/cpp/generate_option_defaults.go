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
	"path/filepath"
	"slices"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/sidekick/api"
)

func generateOptionDefaultsHeader(_ *api.Service, serviceVars map[string]string, lib *config.Library) (string, string) {
	headerPath := serviceVars["option_defaults_header_path"]
	guard := formatHeaderIncludeGuard(headerPath)

	locationStyle := ""
	if lib != nil && lib.Cpp != nil {
		locationStyle = lib.Cpp.EndpointLocationStyle
	}
	isLocationDependent := locationStyle == "LOCATION_DEPENDENT" ||
		locationStyle == "LOCATION_DEPENDENT_COMPAT" ||
		locationStyle == "LOCATION_OPTIONALLY_DEPENDENT"

	data := map[string]any{
		"header_include_guard":       guard,
		"copyright_year":             serviceVars["copyright_year"],
		"proto_file_name":            serviceVars["proto_file_name"],
		"product_internal_namespace": serviceVars["product_internal_namespace"],
		"service_name":               serviceVars["service_name"],
		"is_location_dependent":      isLocationDependent,
	}

	content, err := renderTemplate("templates/internal/option_defaults.h.mustache", data)
	if err != nil {
		panic(err)
	}

	return filepath.Clean(headerPath), content
}

func generateOptionDefaultsCc(_ *api.Service, serviceVars map[string]string, methods []*api.Method, lib *config.Library) (string, string) {
	ccPath := serviceVars["option_defaults_cc_path"]

	locationStyle := ""
	if lib != nil && lib.Cpp != nil {
		locationStyle = lib.Cpp.EndpointLocationStyle
	}
	isLocationDependent := locationStyle == "LOCATION_DEPENDENT" ||
		locationStyle == "LOCATION_DEPENDENT_COMPAT" ||
		locationStyle == "LOCATION_OPTIONALLY_DEPENDENT"

	includes := []string{
		serviceVars["connection_header_path"],
		serviceVars["options_header_path"],
		"google/cloud/internal/populate_common_options.h",
		"google/cloud/internal/populate_grpc_options.h",
	}
	slices.Sort(includes)
	localIncludes := append([]string{serviceVars["option_defaults_header_path"]}, includes...)
	if isLocationDependent {
		localIncludes = append(localIncludes, "google/cloud/internal/absl_str_cat_quiet.h")
	}

	var endpointExpr string
	switch locationStyle {
	case "LOCATION_DEPENDENT":
		endpointExpr = `absl::StrCat(location, "-", "` + serviceVars["service_endpoint"] + `")`
	case "LOCATION_DEPENDENT_COMPAT":
		endpointExpr = `absl::StrCat(location, location.empty() ? "" : "-", "` + serviceVars["service_endpoint"] + `")`
	case "LOCATION_OPTIONALLY_DEPENDENT":
		endpointExpr = `// optional location tag for generating docs
      absl::StrCat(location, location.empty() ? "" : "-", "` + serviceVars["service_endpoint"] + `")`
	default:
		endpointExpr = `"` + serviceVars["service_endpoint"] + `"`
	}

	data := map[string]any{
		"copyright_year":                 serviceVars["copyright_year"],
		"proto_file_name":                serviceVars["proto_file_name"],
		"product_internal_namespace":     serviceVars["product_internal_namespace"],
		"product_namespace":              serviceVars["product_namespace"],
		"service_name":                   serviceVars["service_name"],
		"service_endpoint_env_var":       serviceVars["service_endpoint_env_var"],
		"emulator_endpoint_env_var":      serviceVars["emulator_endpoint_env_var"],
		"service_authority_env_var":      serviceVars["service_authority_env_var"],
		"endpoint_expression":            endpointExpr,
		"retry_policy_name":              serviceVars["retry_policy_name"],
		"limited_time_retry_policy_name": serviceVars["limited_time_retry_policy_name"],
		"idempotency_class_name":         serviceVars["idempotency_class_name"],
		"is_location_dependent":          isLocationDependent,
		"has_lro":                        hasLongrunningMethod(methods),
		"local_includes":                 localIncludes,
	}

	content, err := renderTemplate("templates/internal/option_defaults.cc.mustache", data)
	if err != nil {
		panic(err)
	}

	return filepath.Clean(ccPath), content
}
