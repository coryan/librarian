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
	"context"
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/sidekick/api"
)

//go:embed all:templates
var templates embed.FS

// Generate orchestrates C++ client library code generation for the provided API model.
func Generate(ctx context.Context, model *api.API, outdir string, library *config.Library) error {
	codec, err := newCodec(model, library, outdir)
	if err != nil {
		return fmt.Errorf("creating C++ codec: %w", err)
	}
	codec.annotateModel()

	if library == nil || library.Cpp == nil || !library.Cpp.HasGrpcTransport() {
		return nil
	}

	base := library.Cpp.ProductPath
	if library.Cpp.ForwardingProductPath != "" {
		base = library.Cpp.ForwardingProductPath
	}
	basePrefix := formatProductPath(base)

	type fileEntry struct {
		path    string
		content string
	}

	for _, svc := range model.Services {
		serviceVars := buildServiceVars(svc, library, model)
		methods, asyncMethods := getEffectiveMethods(svc, library)

		var files []fileEntry
		add := func(p, c string) {
			files = append(files, fileEntry{path: p, content: c})
		}

		add(generateOptionsHeader(svc, serviceVars, methods, library))
		if p, c, ok := generateRetryTraitsHeader(svc, serviceVars, library); ok {
			add(p, c)
		}
		add(generateIdempotencyPolicyHeader(svc, serviceVars, methods, library, model))
		add(generateIdempotencyPolicyCc(svc, serviceVars, methods, library, model))
		add(generateMockConnectionHeader(svc, serviceVars, methods, asyncMethods, library, model))
		add(generateOptionDefaultsHeader(svc, serviceVars, library))
		add(generateOptionDefaultsCc(svc, serviceVars, methods, library))
		add(generateStubHeader(svc, serviceVars, methods, asyncMethods, library, model))
		add(generateStubCc(svc, serviceVars, methods, asyncMethods, library, model))
		add(generateStubFactoryHeader(svc, serviceVars, library))
		add(generateStubFactoryCc(svc, serviceVars, methods, library))
		add(generateAuthDecoratorHeader(svc, serviceVars, methods, asyncMethods, library, model))
		add(generateAuthDecoratorCc(svc, serviceVars, methods, asyncMethods, library, model))
		add(generateLoggingDecoratorHeader(svc, serviceVars, methods, asyncMethods, library, model))
		add(generateLoggingDecoratorCc(svc, serviceVars, methods, asyncMethods, library, model))
		add(generateMetadataDecoratorHeader(svc, serviceVars, methods, asyncMethods, library, model))
		add(generateMetadataDecoratorCc(svc, serviceVars, methods, asyncMethods, library, model))
		add(generateTracingStubHeader(svc, serviceVars, methods, asyncMethods, library, model))
		add(generateTracingStubCc(svc, serviceVars, methods, asyncMethods, library, model))
		add(generateTracingConnectionHeader(svc, serviceVars, methods, asyncMethods, library, model))
		add(generateTracingConnectionCc(svc, serviceVars, methods, asyncMethods, library, model))

		if library.Cpp.GenerateRoundRobinDecorator {
			add(generateRoundRobinDecoratorHeader(svc, serviceVars, methods, asyncMethods, library, model))
			add(generateRoundRobinDecoratorCc(svc, serviceVars, methods, asyncMethods, library, model))
		}

		add(generateConnectionImplHeader(svc, serviceVars, methods, asyncMethods, library, model))
		add(generateConnectionImplCc(svc, serviceVars, methods, asyncMethods, library, model))
		add(generateConnectionHeader(svc, serviceVars, methods, asyncMethods, library, model))
		add(generateConnectionCc(svc, serviceVars, methods, asyncMethods, library, model))
		add(generateClientHeader(svc, serviceVars, methods, asyncMethods, library, model))
		add(generateClientCc(svc, serviceVars, methods, asyncMethods, library, model))

		if library.Cpp.ForwardingProductPath != "" {
			add(generateForwardingClientHeader(svc, serviceVars, library))
			add(generateForwardingConnectionHeader(svc, serviceVars, library))
			add(generateForwardingIdempotencyPolicyHeader(svc, serviceVars, library))
			add(generateForwardingMockConnectionHeader(svc, serviceVars, library))
			add(generateForwardingOptionsHeader(svc, serviceVars, methods, library))
		}

		for _, f := range files {
			rel := strings.TrimPrefix(filepath.ToSlash(f.path), basePrefix)
			rel = strings.TrimPrefix(rel, "/")
			destPath := filepath.Join(outdir, filepath.FromSlash(rel))

			if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
				return fmt.Errorf("creating directory for %s: %w", destPath, err)
			}
			if err := os.WriteFile(destPath, []byte(f.content), 0644); err != nil {
				return fmt.Errorf("writing file %s: %w", destPath, err)
			}
		}
	}

	return nil
}
