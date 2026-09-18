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

func generateStubFactoryHeader(_ *api.Service, serviceVars map[string]string, _ *config.Library) (string, string) {
	headerPath := serviceVars["stub_factory_header_path"]
	guard := formatHeaderIncludeGuard(headerPath)

	localIncludes := []string{
		serviceVars["stub_header_path"],
		"google/cloud/internal/unified_grpc_credentials.h",
		"google/cloud/options.h",
		"google/cloud/version.h",
	}
	slices.Sort(localIncludes)

	data := map[string]any{
		"header_include_guard":       guard,
		"copyright_year":             serviceVars["copyright_year"],
		"proto_file_name":            serviceVars["proto_file_name"],
		"product_internal_namespace": serviceVars["product_internal_namespace"],
		"stub_class_name":            serviceVars["stub_class_name"],
		"local_includes":             localIncludes,
	}

	content, err := renderTemplate("templates/internal/stub_factory.h.mustache", data)
	if err != nil {
		panic(err)
	}

	return filepath.Clean(headerPath), content
}

func generateStubFactoryCc(svc *api.Service, serviceVars map[string]string, methods []*api.Method, _ *config.Library) (string, string) {
	ccPath := serviceVars["stub_factory_cc_path"]

	localIncludes := []string{
		serviceVars["stub_factory_header_path"],
		serviceVars["auth_header_path"],
		serviceVars["logging_header_path"],
		serviceVars["metadata_header_path"],
		serviceVars["stub_header_path"],
		serviceVars["tracing_stub_header_path"],
		"google/cloud/common_options.h",
		"google/cloud/grpc_options.h",
		"google/cloud/internal/algorithm.h",
		"google/cloud/internal/opentelemetry.h",
		"google/cloud/log.h",
		"google/cloud/options.h",
	}
	if len(localIncludes) > 1 {
		slices.Sort(localIncludes[1:])
	}

	var pbIncludes []string
	if serviceVars["proto_grpc_header_path"] != "" {
		pbIncludes = append(pbIncludes, serviceVars["proto_grpc_header_path"])
	}
	allMixins := getMixinStubs(svc, methods)
	for _, mixin := range allMixins {
		if mixin.header != "" {
			pbIncludes = append(pbIncludes, mixin.header)
		}
	}
	slices.Sort(pbIncludes)

	hasLro := hasLongrunningMethod(methods)
	var filteredMixins []mixinStubInfo
	for _, mixin := range allMixins {
		if hasLro && mixin.stubName == "operations_stub" {
			continue
		}
		filteredMixins = append(filteredMixins, mixin)
	}

	var mixinInits []map[string]string
	var mixinMoves []string
	for _, mixin := range filteredMixins {
		mixinInits = append(mixinInits, map[string]string{
			"stub_name": mixin.stubName,
			"stub_fqn":  mixin.stubFQN,
		})
		mixinMoves = append(mixinMoves, mixin.stubName)
	}

	data := map[string]any{
		"copyright_year":             serviceVars["copyright_year"],
		"proto_file_name":            serviceVars["proto_file_name"],
		"product_internal_namespace": serviceVars["product_internal_namespace"],
		"stub_class_name":            serviceVars["stub_class_name"],
		"grpc_stub_fqn":              serviceVars["grpc_stub_fqn"],
		"auth_class_name":            serviceVars["auth_class_name"],
		"metadata_class_name":        serviceVars["metadata_class_name"],
		"logging_class_name":         serviceVars["logging_class_name"],
		"tracing_stub_class_name":    serviceVars["tracing_stub_class_name"],
		"has_lro":                    hasLro,
		"local_includes":             localIncludes,
		"proto_includes":             pbIncludes,
		"mixin_inits":                mixinInits,
		"mixin_moves":                mixinMoves,
	}

	content, err := renderTemplate("templates/internal/stub_factory.cc.mustache", data)
	if err != nil {
		panic(err)
	}

	return filepath.Clean(ccPath), content
}
