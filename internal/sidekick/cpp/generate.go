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
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/sidekick/api"
)

// Generate orchestrates C++ client library code generation for the provided API model.
func Generate(ctx context.Context, model *api.API, outdir string, library *config.Library) error {
	codec, err := newCodec(model, library, outdir)
	if err != nil {
		return fmt.Errorf("creating C++ codec: %w", err)
	}
	if err := codec.annotateModel(); err != nil {
		return fmt.Errorf("annotating model: %w", err)
	}

	hasGrpc := library != nil && library.Cpp != nil && library.Cpp.HasGrpcTransport()
	hasRest := library != nil && library.Cpp != nil && library.Cpp.GenerateRestTransport
	if !hasGrpc && !hasRest {
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
		ann := svc.Codec.(*serviceAnnotations)
		methods := ann.Methods
		asyncMethods := ann.AsyncMethods

		var files []fileEntry
		add := func(p, c string) {
			files = append(files, fileEntry{path: p, content: c})
		}

		add(generateOptionsHeader(svc, ann, methods, library))
		if p, c, ok := generateRetryTraitsHeader(svc, ann, library); ok {
			add(p, c)
		}
		add(generateIdempotencyPolicyHeader(svc, ann, methods, library, model))
		add(generateIdempotencyPolicyCc(svc, ann, methods, library, model))
		add(generateMockConnectionHeader(svc, ann, methods, asyncMethods, library, model))
		add(generateOptionDefaultsHeader(svc, ann, library))
		add(generateOptionDefaultsCc(svc, ann, methods, library))

		if hasGrpc {
			add(generateStubHeader(svc, ann, methods, asyncMethods, library, model))
			add(generateStubCc(svc, ann, methods, asyncMethods, library, model))
			add(generateStubFactoryHeader(svc, ann, library))
			add(generateStubFactoryCc(svc, ann, methods, library))
			add(generateAuthDecoratorHeader(svc, ann, methods, asyncMethods, library, model))
			add(generateAuthDecoratorCc(svc, ann, methods, asyncMethods, library, model))
			add(generateLoggingDecoratorHeader(svc, ann, methods, asyncMethods, library, model))
			add(generateLoggingDecoratorCc(svc, ann, methods, asyncMethods, library, model))
			add(generateMetadataDecoratorHeader(svc, ann, methods, asyncMethods, library, model))
			add(generateMetadataDecoratorCc(svc, ann, methods, asyncMethods, library, model))
			add(generateTracingStubHeader(svc, ann, methods, asyncMethods, library, model))
			add(generateTracingStubCc(svc, ann, methods, asyncMethods, library, model))

			if library.Cpp.GenerateRoundRobinDecorator {
				add(generateRoundRobinDecoratorHeader(svc, ann, methods, asyncMethods, library, model))
				add(generateRoundRobinDecoratorCc(svc, ann, methods, asyncMethods, library, model))
			}

			add(generateConnectionImplHeader(svc, ann, methods, asyncMethods, library, model))
			add(generateConnectionImplCc(svc, ann, methods, asyncMethods, library, model))
		}

		if hasRest {
			add(generateRestConnectionHeader(ann))
			add(generateRestConnectionCc(ann))
			add(generateRestStubHeader(svc, ann, methods, asyncMethods, library, model))
			add(generateRestStubCc(svc, ann, methods, asyncMethods, library, model))
			add(generateRestStubFactoryHeader(ann))
			add(generateRestStubFactoryCc(ann))
			add(generateRestLoggingDecoratorHeader(svc, ann, methods, asyncMethods, library, model))
			add(generateRestLoggingDecoratorCc(svc, ann, methods, asyncMethods, library, model))
			add(generateRestMetadataDecoratorHeader(svc, ann, methods, asyncMethods, library, model))
			add(generateRestMetadataDecoratorCc(svc, ann, methods, asyncMethods, library, model))
			add(generateRestConnectionImplHeader(svc, ann, methods, asyncMethods, library, model))
			add(generateRestConnectionImplCc(svc, ann, methods, asyncMethods, library, model))
		}

		add(generateTracingConnectionHeader(svc, ann, methods, asyncMethods, library, model))
		add(generateTracingConnectionCc(svc, ann, methods, asyncMethods, library, model))
		add(generateConnectionHeader(svc, ann, methods, asyncMethods, library, model))
		add(generateConnectionCc(svc, ann, methods, asyncMethods, library, model))
		add(generateClientHeader(svc, ann, methods, asyncMethods, library, model))
		add(generateClientCc(svc, ann, methods, asyncMethods, library, model))
		add(generateSourcesCc(ann))

		if library.Cpp.ForwardingProductPath != "" {
			add(generateForwardingClientHeader(svc, ann, library))
			add(generateForwardingConnectionHeader(svc, ann, library))
			add(generateForwardingIdempotencyPolicyHeader(svc, ann, library))
			add(generateForwardingMockConnectionHeader(svc, ann, library))
			add(generateForwardingOptionsHeader(svc, ann, methods, library))
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
