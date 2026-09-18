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

func generateStubFactoryHeader(_ *api.Service, ann *serviceAnnotations, _ *config.Library) (string, string) {
	headerPath := ann.StubFactoryHeaderPath()
	guard := ann.StubFactoryHeaderIncludeGuard()

	localIncludes := []string{
		ann.StubHeaderPath(),
		"google/cloud/internal/unified_grpc_credentials.h",
		"google/cloud/options.h",
		"google/cloud/version.h",
	}
	slices.Sort(localIncludes)

	data := map[string]any{
		"header_include_guard":       guard,
		"copyright_year":             ann.CopyrightYear,
		"proto_file_name":            ann.ProtoFileName,
		"product_internal_namespace": ann.InternalNamespace(),
		"stub_class_name":            ann.StubClassName(),
		"local_includes":             localIncludes,
	}

	content, err := renderTemplate("templates/internal/stub_factory.h.mustache", data)
	if err != nil {
		panic(err)
	}

	return filepath.Clean(headerPath), content
}

func generateStubFactoryCc(svc *api.Service, ann *serviceAnnotations, methods []*api.Method, _ *config.Library) (string, string) {
	ccPath := ann.StubFactoryCcPath()

	localIncludes := []string{
		ann.StubFactoryHeaderPath(),
		ann.AuthHeaderPath(),
		ann.LoggingHeaderPath(),
		ann.MetadataHeaderPath(),
		ann.StubHeaderPath(),
		ann.TracingStubHeaderPath(),
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
	if h := ann.ProtoGrpcHeaderPath(); h != "" {
		pbIncludes = append(pbIncludes, h)
	}
	allMixins := getMixinStubs(svc, methods)
	for _, mixin := range allMixins {
		if mixin.header != "" {
			pbIncludes = append(pbIncludes, mixin.header)
		}
	}
	slices.Sort(pbIncludes)

	hasLro := hasLongrunningMethod(methods)
	var filteredMixins []mixinStub
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
		"copyright_year":             ann.CopyrightYear,
		"proto_file_name":            ann.ProtoFileName,
		"product_internal_namespace": ann.InternalNamespace(),
		"stub_class_name":            ann.StubClassName(),
		"grpc_stub_fqn":              ann.GrpcStubFQN(),
		"auth_class_name":            ann.AuthClassName(),
		"metadata_class_name":        ann.MetadataClassName(),
		"logging_class_name":         ann.LoggingClassName(),
		"tracing_stub_class_name":    ann.TracingStubClassName(),
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
