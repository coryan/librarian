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
	"fmt"
	"slices"
	"strings"
	"unicode"

	"github.com/googleapis/librarian/internal/sidekick/api"
)

type namespaceType int

const (
	namespaceNormal namespaceType = iota
	namespaceInternal
	namespaceMocks
)

func formatProductPath(path string) string {
	path = strings.TrimPrefix(path, "/")
	if !strings.HasSuffix(path, "/") {
		path += "/"
	}
	return path
}

func parseProductPath(productPath string) (prefix, libraryName, serviceSubdirectory string) {
	parts := strings.Split(strings.Trim(productPath, "/"), "/")
	if len(parts) == 0 {
		return "", "", ""
	}
	if len(parts) > 2 && parts[0] == "google" && parts[1] == "cloud" {
		prefix = "google/cloud"
		libraryName = parts[2]
		serviceSubdirectory = strings.Join(parts[3:], "/")
		return
	}
	for i, p := range parts {
		if p == "golden" {
			prefix = strings.Join(parts[:i], "/")
			libraryName = "golden"
			serviceSubdirectory = strings.Join(parts[i+1:], "/")
			return
		}
	}
	libraryName = parts[len(parts)-1]
	prefix = strings.Join(parts[:len(parts)-1], "/")
	return
}

func libraryPath(productPath string) string {
	p, l, _ := parseProductPath(productPath)
	if p != "" {
		return p + "/" + l + "/"
	}
	return l + "/"
}

func optionsGroup(productPath string) string {
	lp := libraryPath(productPath)
	lp = strings.ReplaceAll(lp, "/", "-")
	return lp + "options"
}

func namespace(productPath string, nsType namespaceType) string {
	_, l, s := parseProductPath(productPath)
	ns := l
	if s != "" {
		ns += "_" + strings.ReplaceAll(s, "/", "_")
	}
	switch nsType {
	case namespaceInternal:
		ns += "_internal"
	case namespaceMocks:
		ns += "_mocks"
	}
	return ns
}

type computeOperationInfo struct {
	HeaderInclude           string
	GetRequestType          string
	CancelRequestType       string
	SetOperationFields      string
	AwaitSetOperationFields string
	GetOperationPath        string
	CancelOperationPath     string
}

func getComputeOperationInfo(operationService string) computeOperationInfo {
	switch operationService {
	case "GlobalOperations":
		path := `absl::StrCat("/compute/", rest_internal::DetermineApiVersion("v1", *options), "/projects/", request.project(), "/global/operations/", request.operation())`
		return computeOperationInfo{
			HeaderInclude:           "google/cloud/compute/global_operations/v1/global_operations.pb.h",
			GetRequestType:          "google::cloud::cpp::compute::global_operations::v1::GetOperationRequest",
			CancelRequestType:       "google::cloud::cpp::compute::global_operations::v1::DeleteOperationRequest",
			SetOperationFields:      "      r.set_project(request.project());\n      r.set_operation(op);",
			AwaitSetOperationFields: "      r.set_project(info.project);\n      r.set_operation(info.operation);",
			GetOperationPath:        path,
			CancelOperationPath:     path,
		}
	case "GlobalOrganizationOperations":
		path := `absl::StrCat("/compute/", rest_internal::DetermineApiVersion("v1", *options), "/locations/global/operations/", request.operation())`
		return computeOperationInfo{
			HeaderInclude:           "google/cloud/compute/global_organization_operations/v1/global_organization_operations.pb.h",
			GetRequestType:          "google::cloud::cpp::compute::global_organization_operations::v1::GetOperationRequest",
			CancelRequestType:       "google::cloud::cpp::compute::global_organization_operations::v1::DeleteOperationRequest",
			SetOperationFields:      "      r.set_operation(op);",
			AwaitSetOperationFields: "      r.set_operation(info.operation);",
			GetOperationPath:        path,
			CancelOperationPath:     path,
		}
	case "RegionOperations":
		path := `absl::StrCat("/compute/", rest_internal::DetermineApiVersion("v1", *options), "/projects/", request.project(), "/regions/", request.region(), "/operations/", request.operation())`
		return computeOperationInfo{
			HeaderInclude:           "google/cloud/compute/region_operations/v1/region_operations.pb.h",
			GetRequestType:          "google::cloud::cpp::compute::region_operations::v1::GetOperationRequest",
			CancelRequestType:       "google::cloud::cpp::compute::region_operations::v1::DeleteOperationRequest",
			SetOperationFields:      "      r.set_project(request.project());\n      r.set_region(request.region());\n      r.set_operation(op);",
			AwaitSetOperationFields: "      r.set_project(info.project);\n      r.set_region(info.region);\n      r.set_operation(info.operation);",
			GetOperationPath:        path,
			CancelOperationPath:     path,
		}
	case "ZoneOperations":
		path := `absl::StrCat("/compute/", rest_internal::DetermineApiVersion("v1", *options), "/projects/", request.project(), "/zones/", request.zone(), "/operations/", request.operation())`
		return computeOperationInfo{
			HeaderInclude:           "google/cloud/compute/zone_operations/v1/zone_operations.pb.h",
			GetRequestType:          "google::cloud::cpp::compute::zone_operations::v1::GetOperationRequest",
			CancelRequestType:       "google::cloud::cpp::compute::zone_operations::v1::DeleteOperationRequest",
			SetOperationFields:      "      r.set_project(request.project());\n      r.set_zone(request.zone());\n      r.set_operation(op);",
			AwaitSetOperationFields: "      r.set_project(info.project);\n      r.set_zone(info.zone);\n      r.set_operation(info.operation);",
			GetOperationPath:        path,
			CancelOperationPath:     path,
		}
	default:
		return computeOperationInfo{}
	}
}

func formatHeaderIncludeGuard(headerPath string) string {
	g := "GOOGLE_CLOUD_CPP_" + headerPath
	g = strings.ReplaceAll(g, "/", "_")
	g = strings.ReplaceAll(g, ".", "_")
	return strings.ToUpper(g)
}

func camelCaseToSnakeCase(input string) string {
	var output strings.Builder
	for i := 0; i < len(input); i++ {
		c := input[i]
		lower := byte(unicode.ToLower(rune(c)))
		if c != '_' && i+2 < len(input) {
			if unicode.IsUpper(rune(input[i+1])) && unicode.IsLower(rune(input[i+2])) {
				output.WriteByte(lower)
				output.WriteByte('_')
				continue
			}
		}
		if c != '_' && i+1 < len(input) {
			if (unicode.IsLower(rune(c)) || unicode.IsDigit(rune(c))) && unicode.IsUpper(rune(input[i+1])) {
				output.WriteByte(lower)
				output.WriteByte('_')
				continue
			}
		}
		output.WriteByte(lower)
	}
	res := output.String()
	res = strings.ReplaceAll(res, "big_query", "bigquery")
	return res
}

func serviceNameToFilePath(serviceName string) string {
	components := strings.Split(serviceName, ".")
	last := components[len(components)-1]
	last = strings.TrimSuffix(last, "Service")
	components[len(components)-1] = last
	var formatted []string
	for _, c := range components {
		formatted = append(formatted, camelCaseToSnakeCase(c))
	}
	return strings.Join(formatted, "/")
}

func protoNameToCppName(protoName string) string {
	protoName = strings.TrimPrefix(protoName, ".")
	return strings.ReplaceAll(protoName, ".", "::")
}

func cppTypeToString(field *api.Field) string {
	if field == nil {
		return ""
	}
	switch field.Typez {
	case api.TypezInt32, api.TypezSint32, api.TypezSfixed32:
		return "std::int32_t"
	case api.TypezInt64, api.TypezSint64, api.TypezSfixed64:
		return "std::int64_t"
	case api.TypezUint32, api.TypezFixed32:
		return "std::uint32_t"
	case api.TypezUint64, api.TypezFixed64:
		return "std::uint64_t"
	case api.TypezDouble:
		return "double"
	case api.TypezFloat:
		return "float"
	case api.TypezBool:
		return "bool"
	case api.TypezString:
		return "std::string"
	case api.TypezBytes:
		return "std::string"
	case api.TypezEnum:
		return protoNameToCppName(field.TypezID)
	case api.TypezMessage:
		return protoNameToCppName(field.TypezID)
	default:
		return protoNameToCppName(field.TypezID)
	}
}

func retryStatusCodeExpression(serviceName string, codes []string) string {
	if len(codes) == 0 {
		return ""
	}
	sortedCodes := slices.Clone(codes)
	slices.Sort(sortedCodes)

	var expr strings.Builder
	expr.WriteString("status.code() != StatusCode::kOk")
	for _, code := range sortedCodes {
		parts := strings.Split(code, ".")
		if len(parts) == 1 {
			expr.WriteString(" && status.code() != StatusCode::" + parts[0])
		} else if parts[0] == serviceName {
			expr.WriteString(" && status.code() != StatusCode::" + parts[1])
		}
	}
	return expr.String()
}

func transientErrorsComment(serviceName string, codes []string) string {
	if len(codes) == 0 {
		return ""
	}
	sortedCodes := slices.Clone(codes)
	slices.Sort(sortedCodes)

	var transientComments []string
	for _, code := range sortedCodes {
		parts := strings.Split(code, ".")
		if len(parts) == 1 {
			transientComments = append(transientComments, parts[0])
		} else if parts[0] == serviceName {
			transientComments = append(transientComments, parts[1])
		}
	}
	if len(transientComments) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("\n * In this class the following status codes are treated as transient errors:")
	for _, c := range transientComments {
		fmt.Fprintf(&b, "\n * - [`%s`](@ref google::cloud::StatusCode)", c)
	}
	return b.String()
}
