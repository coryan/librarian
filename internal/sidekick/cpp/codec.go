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

// codec represents the configuration and state for a C++ sidekick Codec.
type codec struct {
	// Model is the universal API IR model.
	Model *api.API

	// Library is the librarian library configuration.
	Library *config.Library

	// Cpp is the C++-specific library configuration.
	Cpp *config.CppLibrary

	// Output is the output directory path.
	Output string
}

func newCodec(model *api.API, outdir string, library *config.Library) *codec {
	var cppCfg *config.CppLibrary
	if library != nil && library.Cpp != nil {
		cppCfg = library.Cpp
	} else {
		cppCfg = &config.CppLibrary{}
	}
	if cppCfg.ProductPath == "" {
		if library != nil && library.Output != "" {
			cppCfg.ProductPath = library.Output
		} else {
			cppCfg.ProductPath = outdir
		}
	}
	return &codec{
		Model:   model,
		Library: library,
		Cpp:     cppCfg,
		Output:  outdir,
	}
}

func (c *codec) hasGrpc() bool {
	if c.Cpp == nil || c.Cpp.GenerateGrpcTransport == nil {
		return true
	}
	return *c.Cpp.GenerateGrpcTransport
}

func (c *codec) hasRest() bool {
	if c.Cpp == nil {
		return false
	}
	return c.Cpp.GenerateRestTransport
}

func (c *codec) hasRoundRobin() bool {
	if c.Cpp == nil {
		return false
	}
	return c.Cpp.GenerateRoundRobinDecorator
}

func (c *codec) hasRetryTraits() bool {
	if c.Cpp == nil {
		return false
	}
	return len(c.Cpp.RetryableStatusCodes) > 0
}

func (c *codec) isOmittedService(service *api.Service) bool {
	if c.Cpp == nil {
		return false
	}
	return slices.Contains(c.Cpp.OmittedServices, service.Name)
}

func (c *codec) forwardingRelDir() string {
	if c.Cpp == nil || c.Cpp.ForwardingProductPath == "" {
		return ""
	}
	productPath := c.Cpp.ProductPath
	if productPath == "" && c.Library != nil {
		productPath = c.Library.Output
	}
	if productPath == "" {
		productPath = c.Output
	}
	if filepath.IsAbs(productPath) != filepath.IsAbs(c.Cpp.ForwardingProductPath) && c.Library != nil && c.Library.Output != "" {
		productPath = c.Library.Output
	}
	rel, err := filepath.Rel(productPath, c.Cpp.ForwardingProductPath)
	if err != nil {
		return ""
	}
	return rel
}
