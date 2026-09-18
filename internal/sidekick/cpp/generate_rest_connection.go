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
)

func generateRestConnectionHeader(serviceVars map[string]string, lib *config.Library) (string, string) {
	headerPath := serviceVars["connection_rest_header_path"]
	guard := formatHeaderIncludeGuard(headerPath)

	localIncludes := []string{
		serviceVars["connection_header_path"],
		"google/cloud/options.h",
		"google/cloud/version.h",
	}
	slices.Sort(localIncludes)

	locationStyle := ""
	if lib != nil && lib.Cpp != nil {
		locationStyle = lib.Cpp.EndpointLocationStyle
	}
	isLocationDependent := locationStyle == "LOCATION_DEPENDENT" ||
		locationStyle == "LOCATION_DEPENDENT_COMPAT" ||
		locationStyle == "LOCATION_OPTIONALLY_DEPENDENT"

	sysIncludes := []string{"memory"}
	if isLocationDependent {
		sysIncludes = append(sysIncludes, "string")
	}
	slices.Sort(sysIncludes)

	locationDoc := ""
	if isLocationDependent {
		locationDoc = "\n * @param location Sets the prefix for the default `EndpointOption` value."
	}

	data := map[string]any{
		"header_include_guard":             guard,
		"copyright_year":                   serviceVars["copyright_year"],
		"proto_file_name":                  serviceVars["proto_file_name"],
		"product_namespace":                serviceVars["product_namespace"],
		"connection_class_name":            serviceVars["connection_class_name"],
		"client_class_name":                serviceVars["client_class_name"],
		"service_name":                     serviceVars["service_name"],
		"is_location_dependent":            isLocationDependent,
		"is_location_dependent_compat":     locationStyle == "LOCATION_DEPENDENT_COMPAT",
		"is_location_optionally_dependent": locationStyle == "LOCATION_OPTIONALLY_DEPENDENT",
		"location_doc":                     locationDoc,
		"local_includes":                   localIncludes,
		"system_includes":                  sysIncludes,
	}

	content, err := renderTemplate("templates/connection_rest.h.mustache", data)
	if err != nil {
		panic(err)
	}

	return filepath.Clean(headerPath), content
}

func generateRestConnectionCc(serviceVars map[string]string, lib *config.Library) (string, string) {
	ccPath := serviceVars["connection_rest_cc_path"]

	localIncludes := []string{
		serviceVars["connection_rest_header_path"],
		serviceVars["options_header_path"],
		serviceVars["option_defaults_header_path"],
		serviceVars["connection_impl_rest_header_path"],
		serviceVars["stub_factory_rest_header_path"],
		serviceVars["tracing_connection_header_path"],
		"google/cloud/common_options.h",
		"google/cloud/credentials.h",
		"google/cloud/internal/rest_background_threads_impl.h",
		"google/cloud/internal/rest_options.h",
	}
	if len(localIncludes) > 1 {
		slices.Sort(localIncludes[1:])
	}

	locationStyle := ""
	if lib != nil && lib.Cpp != nil {
		locationStyle = lib.Cpp.EndpointLocationStyle
	}
	isLocationDependent := locationStyle == "LOCATION_DEPENDENT" ||
		locationStyle == "LOCATION_DEPENDENT_COMPAT" ||
		locationStyle == "LOCATION_OPTIONALLY_DEPENDENT"
	hasNonLocationOverload := locationStyle == "LOCATION_OPTIONALLY_DEPENDENT" ||
		locationStyle == "LOCATION_DEPENDENT_COMPAT"

	data := map[string]any{
		"copyright_year":                  serviceVars["copyright_year"],
		"proto_file_name":                 serviceVars["proto_file_name"],
		"product_namespace":               serviceVars["product_namespace"],
		"product_internal_namespace":      serviceVars["product_internal_namespace"],
		"connection_class_name":           serviceVars["connection_class_name"],
		"service_name":                    serviceVars["service_name"],
		"stub_rest_class_name":            serviceVars["stub_rest_class_name"],
		"connection_impl_rest_class_name": serviceVars["connection_impl_rest_class_name"],
		"tracing_connection_class_name":   serviceVars["tracing_connection_class_name"],
		"is_location_dependent":           isLocationDependent,
		"has_non_location_overload":       hasNonLocationOverload,
		"local_includes":                  localIncludes,
		"system_includes":                 []string{"memory", "utility"},
	}

	content, err := renderTemplate("templates/connection_rest.cc.mustache", data)
	if err != nil {
		panic(err)
	}

	return filepath.Clean(ccPath), content
}
