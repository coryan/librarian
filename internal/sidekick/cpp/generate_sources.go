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
)

// generateSourcesCc generates the internal/<service>_sources.cc unity build file.
func generateSourcesCc(ann *serviceAnnotations) (string, string) {
	outPath := ann.SourcesCcPath()

	// This generator was added in 2024. We do not want to create new files with a copyright date in the past.
	copyrightYear := max("2024", ann.CopyrightYear)

	var sources []string
	sources = append(sources,
		ann.ClientCcPath(),
		ann.ConnectionCcPath(),
		ann.IdempotencyCcPath(),
		ann.OptionDefaultsCcPath(),
		ann.TracingConnectionCcPath(),
	)

	if ann.HasGrpcTransport {
		sources = append(sources,
			ann.ConnectionImplCcPath(),
			ann.StubFactoryCcPath(),
			ann.AuthCcPath(),
			ann.LoggingCcPath(),
			ann.MetadataCcPath(),
			ann.StubCcPath(),
			ann.TracingStubCcPath(),
		)
		if ann.HasRoundRobinDecorator {
			sources = append(sources, ann.RoundRobinCcPath())
		}
	}

	if ann.HasRestTransport {
		sources = append(sources,
			ann.ConnectionRestCcPath(),
			ann.ConnectionImplRestCcPath(),
			ann.LoggingRestCcPath(),
			ann.MetadataRestCcPath(),
			ann.StubFactoryRestCcPath(),
			ann.StubRestCcPath(),
		)
	}

	slices.Sort(sources)

	data := map[string]any{
		"copyright_year":  copyrightYear,
		"proto_file_name": ann.ProtoFileName,
		"sources":         sources,
	}

	content, err := renderTemplate("templates/internal/sources.cc.mustache", data)
	if err != nil {
		panic(err)
	}

	return outPath, content
}
