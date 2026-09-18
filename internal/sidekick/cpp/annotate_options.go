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
	"strings"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/sidekick/api"
)

// optionsAnnotations contains C++-specific metadata for service options and defaults.
// Annotations are private to the package to encapsulate implementation details and enforce
// accessor method usage for derived properties.
type optionsAnnotations struct {
	Service                *api.Service
	ProductPath            string
	ServiceEndpointEnvVar  string
	EmulatorEndpointEnvVar string
	DefaultEndpoint        string
	DefaultPort            string
	EndpointLocationStyle  string
}

func (o *optionsAnnotations) OptionsClassName() string {
	if o == nil || o.Service == nil {
		return ""
	}
	return o.Service.Name + "ConnectionOptions"
}

func (o *optionsAnnotations) ProductOptionsPage() string {
	if o == nil {
		return ""
	}
	return optionsGroup(o.ProductPath)
}

func (o *optionsAnnotations) HasEndpointEnvVar() bool {
	return o != nil && o.ServiceEndpointEnvVar != ""
}

func (o *optionsAnnotations) HasEmulatorEnvVar() bool {
	return o != nil && o.EmulatorEndpointEnvVar != ""
}

// annotateOptions computes C++ options metadata and environment variables for a service.
func annotateOptions(svc *api.Service, lib *config.Library) *optionsAnnotations {
	if svc == nil {
		return nil
	}

	productPath := ""
	if lib != nil && lib.Cpp != nil {
		productPath = formatProductPath(lib.Cpp.ProductPath)
	}

	endpointEnvVar := ""
	emulatorEnvVar := ""
	locationStyle := ""
	if lib != nil && lib.Cpp != nil {
		endpointEnvVar = lib.Cpp.ServiceEndpointEnvVar
		emulatorEnvVar = lib.Cpp.EmulatorEndpointEnvVar
		locationStyle = lib.Cpp.EndpointLocationStyle
	}

	defaultEndpoint := svc.DefaultHost
	defaultPort := "443"
	if parts := strings.Split(defaultEndpoint, ":"); len(parts) == 2 {
		defaultEndpoint = parts[0]
		defaultPort = parts[1]
	}

	ann := &optionsAnnotations{
		Service:                svc,
		ProductPath:            productPath,
		ServiceEndpointEnvVar:  endpointEnvVar,
		EmulatorEndpointEnvVar: emulatorEnvVar,
		DefaultEndpoint:        defaultEndpoint,
		DefaultPort:            defaultPort,
		EndpointLocationStyle:  locationStyle,
	}

	return ann
}
