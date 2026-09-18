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
)

func generateRestConnectionHeader(ann *serviceAnnotations) (string, string) {
	headerPath := ann.ConnectionRestHeaderPath()
	guard := ann.ConnectionRestHeaderIncludeGuard()

	localIncludes := []string{
		ann.ConnectionHeaderPath(),
		"google/cloud/options.h",
		"google/cloud/version.h",
	}
	slices.Sort(localIncludes)

	locationStyle := ann.EndpointLocationStyle
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
		"copyright_year":                   ann.CopyrightYear,
		"proto_file_name":                  ann.ProtoFileName,
		"product_namespace":                ann.Namespace(),
		"connection_class_name":            ann.ConnectionClassName(),
		"client_class_name":                ann.ClientClassName(),
		"service_name":                     ann.ServiceName,
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

func generateRestConnectionCc(ann *serviceAnnotations) (string, string) {
	ccPath := ann.ConnectionRestCcPath()

	localIncludes := []string{
		ann.ConnectionRestHeaderPath(),
		ann.OptionsHeaderPath(),
		ann.OptionDefaultsHeaderPath(),
		ann.ConnectionImplRestHeaderPath(),
		ann.StubFactoryRestHeaderPath(),
		ann.TracingConnectionHeaderPath(),
		"google/cloud/common_options.h",
		"google/cloud/credentials.h",
		"google/cloud/internal/rest_background_threads_impl.h",
		"google/cloud/internal/rest_options.h",
	}
	if len(localIncludes) > 1 {
		slices.Sort(localIncludes[1:])
	}

	locationStyle := ann.EndpointLocationStyle
	isLocationDependent := locationStyle == "LOCATION_DEPENDENT" ||
		locationStyle == "LOCATION_DEPENDENT_COMPAT" ||
		locationStyle == "LOCATION_OPTIONALLY_DEPENDENT"
	hasNonLocationOverload := locationStyle == "LOCATION_OPTIONALLY_DEPENDENT" ||
		locationStyle == "LOCATION_DEPENDENT_COMPAT"

	data := map[string]any{
		"copyright_year":                  ann.CopyrightYear,
		"proto_file_name":                 ann.ProtoFileName,
		"product_namespace":               ann.Namespace(),
		"product_internal_namespace":      ann.InternalNamespace(),
		"connection_class_name":           ann.ConnectionClassName(),
		"service_name":                    ann.ServiceName,
		"stub_rest_class_name":            ann.StubRestClassName(),
		"connection_impl_rest_class_name": ann.ConnectionImplRestClassName(),
		"tracing_connection_class_name":   ann.TracingConnectionClassName(),
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
