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
	"slices"

	"github.com/googleapis/librarian/internal/config"
)

// generateSourcesCc generates the internal/<service>_sources.cc unity build file.
func generateSourcesCc(serviceVars map[string]string, hasGrpc, hasRest bool, lib *config.Library) (string, string) {
	outPath := serviceVars["sources_cc_path"]

	// This generator was added in 2024. We do not want to create new files with a copyright date in the past.
	copyrightYear := max("2024", serviceVars["copyright_year"])

	var sources []string
	sources = append(sources,
		serviceVars["client_cc_path"],
		serviceVars["connection_cc_path"],
		serviceVars["idempotency_policy_cc_path"],
		serviceVars["option_defaults_cc_path"],
		serviceVars["tracing_connection_cc_path"],
	)

	if hasGrpc {
		sources = append(sources,
			serviceVars["connection_impl_cc_path"],
			serviceVars["stub_factory_cc_path"],
			serviceVars["auth_cc_path"],
			serviceVars["logging_cc_path"],
			serviceVars["metadata_cc_path"],
			serviceVars["stub_cc_path"],
			serviceVars["tracing_stub_cc_path"],
		)
		if lib != nil && lib.Cpp != nil && lib.Cpp.GenerateRoundRobinDecorator {
			sources = append(sources, serviceVars["round_robin_cc_path"])
		}
	}

	if hasRest {
		sources = append(sources,
			serviceVars["connection_rest_cc_path"],
			serviceVars["connection_impl_rest_cc_path"],
			serviceVars["logging_rest_cc_path"],
			serviceVars["metadata_rest_cc_path"],
			serviceVars["stub_factory_rest_cc_path"],
			serviceVars["stub_rest_cc_path"],
		)
	}

	slices.Sort(sources)

	data := map[string]any{
		"copyright_year":  copyrightYear,
		"proto_file_name": serviceVars["proto_file_name"],
		"sources":         sources,
	}

	content, err := renderTemplate("templates/internal/sources.cc.mustache", data)
	if err != nil {
		panic(err)
	}

	return outPath, content
}
