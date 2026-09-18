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
	"strings"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/sidekick/api"
)

func getProtoGrpcHeaders(svc *api.Service, serviceVars map[string]string, methods []*api.Method, lib *config.Library, model *api.API) []string {
	var headers []string
	seen := make(map[string]bool)
	hasGrpc := lib == nil || lib.Cpp == nil || lib.Cpp.HasGrpcTransport()
	if hasGrpc {
		if h := serviceVars["proto_grpc_header_path"]; h != "" {
			seen[h] = true
			headers = append(headers, h)
		}
	} else {
		if h := serviceVars["proto_header_path"]; h != "" {
			seen[h] = true
			headers = append(headers, h)
		}
	}
	for _, m := range methods {
		if m.SourceService != nil && m.SourceService.ID != svc.ID {
			h := ""
			suffix := ".pb.h"
			if hasGrpc {
				suffix = ".grpc.pb.h"
			}
			if loc, ok := model.DefinitionLocation(m.SourceService.ID[1:]); ok {
				h = strings.TrimSuffix(loc.Filename, ".proto") + suffix
			} else if m.SourceService.ID == ".google.cloud.location.Locations" {
				if hasGrpc {
					h = "google/cloud/location/locations.grpc.pb.h"
				} else {
					h = "google/cloud/location/locations.pb.h"
				}
			}
			if h != "" && !seen[h] {
				seen[h] = true
				headers = append(headers, h)
			}
		}
	}
	slices.Sort(headers)
	return headers
}

func buildIdempotencyMethodList(svc *api.Service, methods []*api.Method, serviceVars map[string]string, lib *config.Library, model *api.API) []map[string]any {
	var list []map[string]any
	for _, m := range methods {
		if isStreaming(m) {
			continue
		}
		mVars := buildMethodVars(svc, m, serviceVars, lib, model)
		isSetIam := m.OutputTypeID == ".google.iam.v1.Policy" && m.InputTypeID == ".google.iam.v1.SetIamPolicyRequest"
		entry := map[string]any{
			"method_name":  mVars["method_name"],
			"request_type": mVars["request_type"],
			"idempotency":  mVars["idempotency"],
		}
		if isSetIam {
			entry["is_set_iam"] = true
		} else if isPaginated(m) {
			entry["is_paginated"] = true
			if hasRequestID(m) {
				entry["has_request_id"] = true
				entry["request_id_field_name"] = mVars["request_id_field_name"]
			}
		} else {
			entry["is_other"] = true
			if hasRequestID(m) {
				entry["has_request_id"] = true
				entry["request_id_field_name"] = mVars["request_id_field_name"]
			}
		}
		list = append(list, entry)
	}
	return list
}

func generateIdempotencyPolicyHeader(svc *api.Service, serviceVars map[string]string, methods []*api.Method, lib *config.Library, model *api.API) (string, string) {
	headerPath := serviceVars["idempotency_policy_header_path"]
	guard := formatHeaderIncludeGuard(headerPath)

	localIncludes := []string{
		"google/cloud/idempotency.h",
		"google/cloud/version.h",
	}
	slices.Sort(localIncludes)

	pbHeaders := getProtoGrpcHeaders(svc, serviceVars, methods, lib, model)

	data := map[string]any{
		"header_include_guard":   guard,
		"copyright_year":         serviceVars["copyright_year"],
		"proto_file_name":        serviceVars["proto_file_name"],
		"product_namespace":      serviceVars["product_namespace"],
		"idempotency_class_name": serviceVars["idempotency_class_name"],
		"local_includes":         localIncludes,
		"proto_includes":         pbHeaders,
		"system_includes":        []string{"memory"},
		"methods":                buildIdempotencyMethodList(svc, methods, serviceVars, lib, model),
	}

	content, err := renderTemplate("templates/connection_idempotency_policy.h.mustache", data)
	if err != nil {
		panic(err)
	}

	return filepath.Clean(headerPath), content
}

func generateIdempotencyPolicyCc(svc *api.Service, serviceVars map[string]string, methods []*api.Method, lib *config.Library, model *api.API) (string, string) {
	ccPath := serviceVars["idempotency_policy_cc_path"]

	data := map[string]any{
		"copyright_year":         serviceVars["copyright_year"],
		"proto_file_name":        serviceVars["proto_file_name"],
		"product_namespace":      serviceVars["product_namespace"],
		"idempotency_class_name": serviceVars["idempotency_class_name"],
		"local_includes":         []string{serviceVars["idempotency_policy_header_path"]},
		"system_includes":        []string{"memory"},
		"methods":                buildIdempotencyMethodList(svc, methods, serviceVars, lib, model),
	}

	content, err := renderTemplate("templates/connection_idempotency_policy.cc.mustache", data)
	if err != nil {
		panic(err)
	}

	return filepath.Clean(ccPath), content
}
