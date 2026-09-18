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

func generateRoundRobinDecoratorHeader(svc *api.Service, serviceVars map[string]string, methods, asyncMethods []*api.Method, lib *config.Library, model *api.API) (string, string) {
	headerPath := serviceVars["round_robin_header_path"]
	guard := formatHeaderIncludeGuard(headerPath)

	localIncludes := []string{
		serviceVars["stub_header_path"],
		"google/cloud/version.h",
	}
	slices.Sort(localIncludes)

	data := map[string]any{
		"header_include_guard":       guard,
		"copyright_year":             serviceVars["copyright_year"],
		"proto_file_name":            serviceVars["proto_file_name"],
		"product_internal_namespace": serviceVars["product_internal_namespace"],
		"round_robin_class_name":     serviceVars["round_robin_class_name"],
		"stub_class_name":            serviceVars["stub_class_name"],
		"local_includes":             localIncludes,
		"methods":                    buildDecoratorMethodList(svc, methods, serviceVars, lib, model),
		"async_methods":              buildDecoratorAsyncMethodList(svc, asyncMethods, serviceVars, lib, model),
		"has_lro":                    hasLongrunningMethod(methods),
	}

	content, err := renderTemplate("templates/internal/round_robin_decorator.h.mustache", data)
	if err != nil {
		panic(err)
	}

	return filepath.Clean(headerPath), content
}

func generateRoundRobinDecoratorCc(svc *api.Service, serviceVars map[string]string, methods, asyncMethods []*api.Method, lib *config.Library, model *api.API) (string, string) {
	ccPath := serviceVars["round_robin_cc_path"]

	localIncludes := []string{serviceVars["round_robin_header_path"]}

	data := map[string]any{
		"copyright_year":             serviceVars["copyright_year"],
		"proto_file_name":            serviceVars["proto_file_name"],
		"product_internal_namespace": serviceVars["product_internal_namespace"],
		"round_robin_class_name":     serviceVars["round_robin_class_name"],
		"stub_class_name":            serviceVars["stub_class_name"],
		"local_includes":             localIncludes,
		"methods":                    buildDecoratorMethodList(svc, methods, serviceVars, lib, model),
		"async_methods":              buildDecoratorAsyncMethodList(svc, asyncMethods, serviceVars, lib, model),
		"has_lro":                    hasLongrunningMethod(methods),
	}

	content, err := renderTemplate("templates/internal/round_robin_decorator.cc.mustache", data)
	if err != nil {
		panic(err)
	}

	return filepath.Clean(ccPath), content
}
