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

// OptionsAnnotations contains C++-specific metadata for service options and defaults.
type OptionsAnnotations struct {
	// OptionsClassName is the service options class name (e.g. "GoldenKitchenSinkConnectionOptions").
	OptionsClassName string

	// ProductOptionsPage is the Doxygen group name for the options page.
	ProductOptionsPage string

	// ServiceEndpointEnvVar is the environment variable override for the service endpoint.
	ServiceEndpointEnvVar string

	// EmulatorEndpointEnvVar is the environment variable override for the emulator endpoint.
	EmulatorEndpointEnvVar string

	// DefaultEndpoint is the default service endpoint host.
	DefaultEndpoint string

	// DefaultPort is the default port string.
	DefaultPort string

	// HasEndpointEnvVar indicates whether ServiceEndpointEnvVar is configured.
	HasEndpointEnvVar bool

	// HasEmulatorEnvVar indicates whether EmulatorEndpointEnvVar is configured.
	HasEmulatorEnvVar bool

	// EndpointLocationStyle specifies the endpoint location resolution style.
	EndpointLocationStyle string
}

// annotateOptions computes C++ options metadata and environment variables for a service.
func annotateOptions(svc *api.Service, lib *config.Library) *OptionsAnnotations {
	if svc == nil {
		return nil
	}

	productPath := ""
	serviceEndpointEnvVar := ""
	emulatorEndpointEnvVar := ""
	endpointLocationStyle := ""
	if lib != nil && lib.Cpp != nil {
		productPath = lib.Cpp.ProductPath
		serviceEndpointEnvVar = lib.Cpp.ServiceEndpointEnvVar
		emulatorEndpointEnvVar = lib.Cpp.EmulatorEndpointEnvVar
		endpointLocationStyle = lib.Cpp.EndpointLocationStyle
	}

	defaultEndpoint := svc.DefaultHost
	defaultPort := "443"
	if defaultEndpoint != "" {
		parts := strings.Split(defaultEndpoint, ":")
		defaultEndpoint = parts[0]
		if len(parts) > 1 {
			defaultPort = parts[1]
		}
	}

	ann := &OptionsAnnotations{
		OptionsClassName:       svc.Name + "ConnectionOptions",
		ProductOptionsPage:     optionsGroup(formatProductPath(productPath)),
		ServiceEndpointEnvVar:  serviceEndpointEnvVar,
		EmulatorEndpointEnvVar: emulatorEndpointEnvVar,
		DefaultEndpoint:        defaultEndpoint,
		DefaultPort:            defaultPort,
		HasEndpointEnvVar:      serviceEndpointEnvVar != "",
		HasEmulatorEnvVar:      emulatorEndpointEnvVar != "",
		EndpointLocationStyle:  endpointLocationStyle,
	}

	return ann
}
