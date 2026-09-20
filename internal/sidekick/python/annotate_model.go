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
	"cmp"
	"slices"
	"strings"

	"github.com/googleapis/librarian/internal/license"
)

// ModelAnnotations decorates api.API with Python-specific metadata.
type ModelAnnotations struct {
	CopyrightYear    string
	BoilerPlate      []string
	PackageName      string
	PackageVersion   string
	DefaultVersion   string
	LibraryPackage   string
	ProtoPackage     string
	GAPICNamespace   string
	GAPICName        string
	PackageDirectory string
	Services         []*ServiceAnnotations
	Messages         []*MessageAnnotations
	Enums            []*EnumAnnotations
	Protos           []*ProtoAnnotations
	TypesProtos      []*ProtoAnnotations
	AllTypes         []string
}

func (c *codec) annotateModel() error {
	libPkg := strings.ReplaceAll(c.packageDir(), "/", ".")
	protoPkg := c.Model.PackageName
	ann := &ModelAnnotations{
		CopyrightYear:    c.GenerationYear,
		BoilerPlate:      license.HeaderBulk(),
		PackageName:      c.PackageName,
		PackageVersion:   c.PackageVersion,
		DefaultVersion:   c.DefaultVersion,
		LibraryPackage:   libPkg,
		ProtoPackage:     protoPkg,
		GAPICNamespace:   c.GAPICNamespace,
		GAPICName:        c.GAPICName,
		PackageDirectory: c.packageDir(),
	}
	c.Model.Codec = ann

	// Annotate messages first so services can reference message types.
	for _, message := range c.Model.Messages {
		if err := c.annotateMessage(message, ann); err != nil {
			return err
		}
		if mAnn, ok := message.Codec.(*MessageAnnotations); ok {
			ann.Messages = append(ann.Messages, mAnn)
		}
	}

	// Annotate enums.
	for _, enum := range c.Model.Enums {
		if err := c.annotateEnum(enum, ann); err != nil {
			return err
		}
		if eAnn, ok := enum.Codec.(*EnumAnnotations); ok {
			ann.Enums = append(ann.Enums, eAnn)
		}
	}

	// Annotate protos after messages and enums are populated.
	if err := c.annotateProtos(ann); err != nil {
		return err
	}

	// Annotate services after messages and enums.
	for _, service := range c.Model.Services {
		if err := c.annotateService(service, ann); err != nil {
			return err
		}
		if sAnn, ok := service.Codec.(*ServiceAnnotations); ok {
			ann.Services = append(ann.Services, sAnn)
		}
	}

	slices.SortFunc(ann.Services, func(a, b *ServiceAnnotations) int {
		return cmp.Compare(a.ProtoName, b.ProtoName)
	})
	for i, s := range ann.Services {
		s.HasNext = (i < len(ann.Services)-1)
	}

	return nil
}
