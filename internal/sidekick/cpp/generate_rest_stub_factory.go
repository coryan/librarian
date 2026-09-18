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

func generateRestStubFactoryHeader(serviceVars map[string]string) (string, string) {
	headerPath := serviceVars["stub_factory_rest_header_path"]
	guard := formatHeaderIncludeGuard(headerPath)

	localIncludes := []string{
		serviceVars["stub_rest_header_path"],
		"google/cloud/options.h",
		"google/cloud/version.h",
	}
	slices.Sort(localIncludes)

	data := map[string]any{
		"header_include_guard":       guard,
		"copyright_year":             serviceVars["copyright_year"],
		"proto_file_name":            serviceVars["proto_file_name"],
		"product_internal_namespace": serviceVars["product_internal_namespace"],
		"stub_rest_class_name":       serviceVars["stub_rest_class_name"],
		"local_includes":             localIncludes,
	}

	content, err := renderTemplate("templates/internal/rest_stub_factory.h.mustache", data)
	if err != nil {
		panic(err)
	}

	return filepath.Clean(headerPath), content
}

func generateRestStubFactoryCc(serviceVars map[string]string) (string, string) {
	ccPath := serviceVars["stub_factory_rest_cc_path"]

	localIncludes := []string{
		serviceVars["stub_factory_rest_header_path"],
		"absl/strings/match.h",
		serviceVars["logging_rest_header_path"],
		serviceVars["metadata_rest_header_path"],
		serviceVars["stub_rest_header_path"],
		"google/cloud/common_options.h",
		"google/cloud/internal/algorithm.h",
		"google/cloud/internal/populate_rest_options.h",
		"google/cloud/log.h",
		"google/cloud/options.h",
		"google/cloud/rest_options.h",
	}
	if len(localIncludes) > 1 {
		slices.Sort(localIncludes[1:])
	}

	data := map[string]any{
		"copyright_year":             serviceVars["copyright_year"],
		"proto_file_name":            serviceVars["proto_file_name"],
		"product_internal_namespace": serviceVars["product_internal_namespace"],
		"stub_rest_class_name":       serviceVars["stub_rest_class_name"],
		"metadata_rest_class_name":   serviceVars["metadata_rest_class_name"],
		"logging_rest_class_name":    serviceVars["logging_rest_class_name"],
		"local_includes":             localIncludes,
	}

	content, err := renderTemplate("templates/internal/rest_stub_factory.cc.mustache", data)
	if err != nil {
		panic(err)
	}

	return filepath.Clean(ccPath), content
}
