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
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/sidekick/api"
)

// annotateModel coordinates C++ annotation passes across messages, fields, and services.
func annotateModel(model *api.API, library *config.Library) error {
	for _, m := range model.Messages {
		for _, f := range m.Fields {
			annotateField(f)
		}
	}
	for _, svc := range model.Services {
		for _, m := range svc.Methods {
			annotateMethod(m, svc, library, model)
		}
		annotateService(svc, library, model)
	}
	return nil
}
