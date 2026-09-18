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

func generateRestStubFactoryHeader(ann *serviceAnnotations) (string, string) {
	headerPath := ann.StubFactoryRestHeaderPath()
	guard := ann.StubFactoryRestHeaderIncludeGuard()

	localIncludes := []string{
		ann.StubRestHeaderPath(),
		"google/cloud/options.h",
		"google/cloud/version.h",
	}
	slices.Sort(localIncludes)

	data := map[string]any{
		"header_include_guard":       guard,
		"copyright_year":             ann.CopyrightYear,
		"proto_file_name":            ann.ProtoFileName,
		"product_internal_namespace": ann.InternalNamespace(),
		"stub_rest_class_name":       ann.StubRestClassName(),
		"local_includes":             localIncludes,
	}

	content, err := renderTemplate("templates/internal/rest_stub_factory.h.mustache", data)
	if err != nil {
		panic(err)
	}

	return filepath.Clean(headerPath), content
}

func generateRestStubFactoryCc(ann *serviceAnnotations) (string, string) {
	ccPath := ann.StubFactoryRestCcPath()

	localIncludes := []string{
		ann.StubFactoryRestHeaderPath(),
		"absl/strings/match.h",
		ann.LoggingRestHeaderPath(),
		ann.MetadataRestHeaderPath(),
		ann.StubRestHeaderPath(),
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
		"copyright_year":             ann.CopyrightYear,
		"proto_file_name":            ann.ProtoFileName,
		"product_internal_namespace": ann.InternalNamespace(),
		"stub_rest_class_name":       ann.StubRestClassName(),
		"metadata_rest_class_name":   ann.MetadataRestClassName(),
		"logging_rest_class_name":    ann.LoggingRestClassName(),
		"local_includes":             localIncludes,
	}

	content, err := renderTemplate("templates/internal/rest_stub_factory.cc.mustache", data)
	if err != nil {
		panic(err)
	}

	return filepath.Clean(ccPath), content
}
