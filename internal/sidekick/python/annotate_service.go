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

package python

import (
	"slices"
	"strings"

	"github.com/googleapis/librarian/internal/sidekick/api"
)

// ServiceAnnotations decorates api.Service with Python-specific metadata.
type ServiceAnnotations struct {
	Model           *ModelAnnotations
	Service         *api.Service
	CopyrightYear   string
	Name            string
	ProtoName       string
	ServiceName     string
	DirectoryName   string
	ClientName      string
	AsyncClientName string
	DocLines        []string
	Methods         []*MethodAnnotations
	OwnMethods      []*MethodAnnotations
	HasNext         bool
}

func (c *codec) annotateService(service *api.Service, model *ModelAnnotations) error {
	docLines := formatDocLines(service.Documentation)
	name := service.Name
	clientName := name
	if !strings.HasSuffix(name, "Client") {
		clientName = name + "Client"
	}
	asyncClientName := strings.TrimSuffix(clientName, "Client") + "AsyncClient"
	copyrightYear := ""
	if model != nil {
		copyrightYear = model.CopyrightYear
	}
	ann := &ServiceAnnotations{
		Model:           model,
		Service:         service,
		CopyrightYear:   copyrightYear,
		Name:            name,
		ProtoName:       service.Name,
		ServiceName:     service.Name,
		DirectoryName:   snakeCase(name),
		ClientName:      clientName,
		AsyncClientName: asyncClientName,
		DocLines:        docLines,
	}

	for _, method := range service.Methods {
		if err := c.annotateMethod(method, ann); err != nil {
			return err
		}
		if mAnn, ok := method.Codec.(*MethodAnnotations); ok {
			ann.Methods = append(ann.Methods, mAnn)
			if !mAnn.IsMixin {
				ann.OwnMethods = append(ann.OwnMethods, mAnn)
			}
		}
	}

	slices.SortFunc(ann.Methods, func(a, b *MethodAnnotations) int {
		return strings.Compare(a.ProtoName, b.ProtoName)
	})
	for i, m := range ann.Methods {
		m.HasNext = (i < len(ann.Methods)-1)
	}

	slices.SortFunc(ann.OwnMethods, func(a, b *MethodAnnotations) int {
		return strings.Compare(a.ProtoName, b.ProtoName)
	})
	for i, m := range ann.OwnMethods {
		m.HasNextOwn = (i < len(ann.OwnMethods)-1)
	}

	service.Codec = ann
	return nil
}
