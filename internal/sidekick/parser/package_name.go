// Copyright 2025 Google LLC
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

package parser

import (
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/googleapis/librarian/internal/serviceconfig"
	"github.com/googleapis/librarian/internal/sidekick/api"
	"github.com/googleapis/librarian/internal/sidekick/parser/svcconfig"
	"google.golang.org/protobuf/types/pluginpb"
)

// updatePackageNames sets the PackageName field.
//
// This happens
// often with protobuf libraries that lack a service config YAML file, typically
// type-only libraries.
func updatePackageNames(model *api.API, serviceConfig *serviceconfig.Service, req *pluginpb.CodeGeneratorRequest) error {
	packageName := ""
	packageNamePascalCase := ""
	// If Ruby, PHP, and C# agree on how the package name should be capitalized,
	// we use that capitalization for the "PascalCase" version of the package
	// name.
	csharpNamespaces := make(map[string]struct{})
	phpNamespaces := make(map[string]struct{})
	rubyPackages := make(map[string]struct{})

	for _, file := range req.GetSourceFileDescriptors() {
		pkg := file.GetPackage()
		if packageName != "" && pkg != packageName {
			return fmt.Errorf("inconsistent package names, file %s has %s, expected %s", file.GetName(), pkg, packageName)
		}
		packageName = pkg
		// Normalize the namespaces / packages.
		csharpNamespaces[file.Options.GetCsharpNamespace()] = struct{}{}
		phpNamespaces[file.Options.GetPhpNamespace()] = struct{}{}
		rubyPackages[file.Options.GetRubyPackage()] = struct{}{}
	}
	normalizedNamespaces := make(map[string]struct{})
	maps.Copy(normalizedNamespaces, csharpNamespaces)
	for name, v := range phpNamespaces {
		normalizedNamespaces[strings.ReplaceAll(name, "\\", ".")] = v
	}
	for name, v := range rubyPackages {
		normalizedNamespaces[strings.ReplaceAll(name, "::", ".")] = v
	}
	if names := slices.Collect(maps.Keys(normalizedNamespaces)); len(names) == 1 {
		packageNamePascalCase = names[0]
	}
	if overrides := svcconfig.ExtractPackageName(serviceConfig); overrides != nil {
		packageName = overrides.PackageName
	}
	model.WithPackageNames(packageName, packageNamePascalCase)
	return nil
}
