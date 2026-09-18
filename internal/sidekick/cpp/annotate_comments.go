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

// CommentAnnotations contains formatted Doxygen comments for a service or method.
type CommentAnnotations struct {
	// ClassComment is the formatted Doxygen comment for a generated class.
	ClassComment string

	// MethodComment is the formatted Doxygen comment for a default Protobuf request method overload.
	MethodComment string

	// SignatureComments maps method signature indices to formatted Doxygen comments.
	SignatureComments map[int]string
}

// annotateComments formats Doxygen documentation comments for a service class.
func annotateComments(svc *api.Service, serviceName string, model *api.API) *CommentAnnotations {
	if svc == nil {
		return nil
	}

	ann := &CommentAnnotations{
		ClassComment:      formatClassComments(svc, serviceName, model),
		SignatureComments: make(map[int]string),
	}

	return ann
}

// annotateMethodComments formats Doxygen documentation comments for an RPC method and its signature overloads.
func annotateMethodComments(method *api.Method, model *api.API) *CommentAnnotations {
	if method == nil {
		return nil
	}

	ann := &CommentAnnotations{
		MethodComment:     formatMethodCommentsProtobufRequest(method, model),
		SignatureComments: make(map[int]string),
	}

	for i, sig := range method.Signatures {
		ann.SignatureComments[i] = formatMethodCommentsMethodSignature(method, sig, model)
	}

	return ann
}
