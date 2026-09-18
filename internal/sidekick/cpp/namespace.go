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
	"strings"
)

type productPathInfo struct {
	prefix              string
	libraryName         string
	serviceSubdirectory string
}

func parseProductPath(productPath string) productPathInfo {
	normalized := strings.ReplaceAll(productPath, "\\", "/")
	parts := strings.Split(normalized, "/")
	var components []string
	for _, p := range parts {
		if p != "" {
			components = append(components, p)
		}
	}
	if len(components) == 0 {
		return productPathInfo{}
	}
	if len(components) > 2 && components[0] == "google" && components[1] == "cloud" {
		return productPathInfo{
			prefix:              strings.Join(components[:2], "/"),
			libraryName:         components[2],
			serviceSubdirectory: strings.Join(components[3:], "/"),
		}
	}
	if idx := slices.Index(components, "golden"); idx != -1 {
		return productPathInfo{
			prefix:              strings.Join(components[:idx], "/"),
			libraryName:         components[idx],
			serviceSubdirectory: strings.Join(components[idx+1:], "/"),
		}
	}
	return productPathInfo{
		prefix:              strings.Join(components[:len(components)-1], "/"),
		libraryName:         components[len(components)-1],
		serviceSubdirectory: "",
	}
}

func deriveNamespace(productPath string) string {
	info := parseProductPath(productPath)
	if info.libraryName == "" {
		return ""
	}
	if info.serviceSubdirectory == "" {
		return info.libraryName
	}
	sub := strings.ReplaceAll(info.serviceSubdirectory, "/", "_")
	return info.libraryName + "_" + sub
}

func (ann *serviceAnnotations) PackageNamespace() string {
	return deriveNamespace(ann.ProductPath)
}

func (ann *serviceAnnotations) InternalNamespace() string {
	ns := ann.PackageNamespace()
	if ns == "" {
		return ""
	}
	return ns + "_internal"
}

func (ann *serviceAnnotations) MocksNamespace() string {
	ns := ann.PackageNamespace()
	if ns == "" {
		return ""
	}
	return ns + "_mocks"
}

func (ann *serviceAnnotations) ForwardingNamespace() string {
	if ann.ForwardingPath == "" {
		return ""
	}
	return deriveNamespace(ann.ForwardingPath)
}

func (ann *serviceAnnotations) ForwardingMocksNamespace() string {
	fwd := ann.ForwardingNamespace()
	if fwd == "" {
		return ""
	}
	return fwd + "_mocks"
}
