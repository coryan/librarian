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

func generateOptionDefaultsHeader(_ *api.Service, ann *serviceAnnotations, _ *config.Library) (string, string) {
	headerPath := ann.OptionDefaultsHeaderPath()
	guard := ann.OptionDefaultsHeaderIncludeGuard()

	locationStyle := ann.EndpointLocationStyle
	isLocationDependent := locationStyle == "LOCATION_DEPENDENT" ||
		locationStyle == "LOCATION_DEPENDENT_COMPAT" ||
		locationStyle == "LOCATION_OPTIONALLY_DEPENDENT"

	data := map[string]any{
		"header_include_guard":       guard,
		"copyright_year":             ann.CopyrightYear,
		"proto_file_name":            ann.ProtoFileName,
		"product_internal_namespace": ann.InternalNamespace(),
		"service_name":               ann.ServiceName,
		"is_location_dependent":      isLocationDependent,
	}

	content, err := renderTemplate("templates/internal/option_defaults.h.mustache", data)
	if err != nil {
		panic(err)
	}

	return filepath.Clean(headerPath), content
}

func generateOptionDefaultsCc(_ *api.Service, ann *serviceAnnotations, methods []*api.Method, _ *config.Library) (string, string) {
	ccPath := ann.OptionDefaultsCcPath()

	locationStyle := ann.EndpointLocationStyle
	isLocationDependent := locationStyle == "LOCATION_DEPENDENT" ||
		locationStyle == "LOCATION_DEPENDENT_COMPAT" ||
		locationStyle == "LOCATION_OPTIONALLY_DEPENDENT"

	includes := []string{
		ann.ConnectionHeaderPath(),
		ann.OptionsHeaderPath(),
		"google/cloud/internal/populate_common_options.h",
		"google/cloud/internal/populate_grpc_options.h",
	}
	slices.Sort(includes)
	localIncludes := append([]string{ann.OptionDefaultsHeaderPath()}, includes...)
	if isLocationDependent {
		localIncludes = append(localIncludes, "google/cloud/internal/absl_str_cat_quiet.h")
	}

	var endpointExpr string
	switch locationStyle {
	case "LOCATION_DEPENDENT":
		endpointExpr = `absl::StrCat(location, "-", "` + ann.ServiceEndpoint + `")`
	case "LOCATION_DEPENDENT_COMPAT":
		endpointExpr = `absl::StrCat(location, location.empty() ? "" : "-", "` + ann.ServiceEndpoint + `")`
	case "LOCATION_OPTIONALLY_DEPENDENT":
		endpointExpr = `// optional location tag for generating docs
      absl::StrCat(location, location.empty() ? "" : "-", "` + ann.ServiceEndpoint + `")`
	default:
		endpointExpr = `"` + ann.ServiceEndpoint + `"`
	}

	data := map[string]any{
		"copyright_year":                 ann.CopyrightYear,
		"proto_file_name":                ann.ProtoFileName,
		"product_internal_namespace":     ann.InternalNamespace(),
		"product_namespace":              ann.Namespace(),
		"service_name":                   ann.ServiceName,
		"service_endpoint_env_var":       ann.ServiceEndpointEnvVar,
		"emulator_endpoint_env_var":      ann.EmulatorEndpointEnvVar,
		"service_authority_env_var":      ann.ServiceAuthorityEnvVar,
		"endpoint_expression":            endpointExpr,
		"retry_policy_name":              ann.RetryPolicyName(),
		"limited_time_retry_policy_name": ann.LimitedTimeRetryPolicyName(),
		"idempotency_class_name":         ann.IdempotencyClassName(),
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
