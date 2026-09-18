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
	"github.com/googleapis/librarian/internal/sidekick/api"
)

// commentAnnotations contains formatted Doxygen comments for a service or method.
// Annotations are private to the package to encapsulate implementation details.
type commentAnnotations struct {
	// ClassComment is the formatted Doxygen comment for a generated class.
	ClassComment string

	// MethodComment is the formatted Doxygen comment for a default Protobuf request method overload.
	MethodComment string

	// SignatureComments maps method signature indices to formatted Doxygen comments.
	SignatureComments map[int]string
}

// annotateComments formats Doxygen documentation comments for a service class.
func annotateComments(svc *api.Service, serviceName string, model *api.API, isDiscovery bool) *commentAnnotations {
	if svc == nil {
		return nil
	}

	ann := &commentAnnotations{
		ClassComment:      formatClassComments(svc, serviceName, model, isDiscovery),
		SignatureComments: make(map[int]string),
	}

	return ann
}

// annotateMethodComments formats Doxygen documentation comments for an RPC method and its signature overloads.
func annotateMethodComments(method *api.Method, model *api.API, isDiscovery bool) *commentAnnotations {
	if method == nil {
		return nil
	}

	ann := &commentAnnotations{
		MethodComment:     formatMethodCommentsProtobufRequest(method, model, isDiscovery),
		SignatureComments: make(map[int]string),
	}

	for i, sig := range method.Signatures {
		ann.SignatureComments[i] = formatMethodCommentsMethodSignature(method, sig, model, isDiscovery)
	}

	return ann
}
