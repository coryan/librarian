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
	"path/filepath"
	"slices"
	"strings"

	"github.com/googleapis/librarian/internal/sidekick/api"
)

// ProtoAnnotations decorates a protobuf file with Python-specific metadata.
type ProtoAnnotations struct {
	Model               *ModelAnnotations
	ModuleName          string
	FileName            string
	CopyrightYear       string
	ProtoPackage        string
	Package             string
	Imports             []string
	Manifest            []string
	TopLevelMessages    []*MessageAnnotations
	TopLevelEnums       []*EnumAnnotations
	SortedMessages      []*MessageAnnotations
	SortedEnums         []*EnumAnnotations
	HasMessagesOrEnums  bool
	HasImports          bool
	HasTopLevelEnums    bool
	HasTopLevelMessages bool
	HasNext             bool
}

func (c *codec) annotateProtos(ann *ModelAnnotations) error {
	protoFiles := c.collectPackageProtoFiles()

	for _, pFile := range protoFiles {
		pAnn := c.buildProtoAnnotations(pFile, ann)
		ann.Protos = append(ann.Protos, pAnn)
		if pAnn.HasMessagesOrEnums {
			ann.TypesProtos = append(ann.TypesProtos, pAnn)
		}
	}

	slices.SortFunc(ann.Protos, func(a, b *ProtoAnnotations) int {
		return cmp.Compare(a.ModuleName, b.ModuleName)
	})

	slices.SortFunc(ann.TypesProtos, func(a, b *ProtoAnnotations) int {
		return cmp.Compare(a.ModuleName, b.ModuleName)
	})

	for i, p := range ann.TypesProtos {
		p.HasNext = (i < len(ann.TypesProtos)-1)
		for _, m := range p.SortedMessages {
			ann.AllTypes = append(ann.AllTypes, m.Name)
		}
		for _, e := range p.SortedEnums {
			ann.AllTypes = append(ann.AllTypes, e.Name)
		}
	}

	return nil
}

func (c *codec) collectPackageProtoFiles() []string {
	seen := make(map[string]bool)
	var files []string

	addFile := func(id string) {
		loc, ok := c.Model.DefinitionLocation(id)
		if !ok || loc.Filename == "" {
			return
		}
		filename := loc.Filename
		if !seen[filename] && c.isPackageProtoFile(filename) {
			seen[filename] = true
			files = append(files, filename)
		}
	}

	for _, svc := range c.Model.Services {
		addFile(svc.ID)
	}
	for _, msg := range c.Model.Messages {
		addFile(msg.ID)
	}
	for _, enum := range c.Model.Enums {
		addFile(enum.ID)
	}

	// If no definition locations were found, fallback to model package name
	if len(files) == 0 && c.Model.PackageName != "" {
		pkgPath := strings.ReplaceAll(c.Model.PackageName, ".", "/")
		parts := strings.Split(c.Model.PackageName, ".")
		mod := parts[len(parts)-1]
		files = append(files, filepath.Join(pkgPath, mod+".proto"))
	}

	return files
}

func (c *codec) isPackageProtoFile(filename string) bool {
	if c.Model.PackageName == "" {
		return true
	}
	expectedDir := strings.ReplaceAll(c.Model.PackageName, ".", "/")
	return strings.HasPrefix(filename, expectedDir+"/") || filename == expectedDir+".proto" || strings.Contains(filename, expectedDir)
}

func (c *codec) buildProtoAnnotations(filename string, ann *ModelAnnotations) *ProtoAnnotations {
	base := filepath.Base(filename)
	moduleName := strings.TrimSuffix(base, ".proto")
	versionedPkg := strings.ReplaceAll(c.packageDir(), "/", ".")

	pAnn := &ProtoAnnotations{
		Model:         ann,
		ModuleName:    moduleName,
		FileName:      filename,
		CopyrightYear: c.GenerationYear,
		ProtoPackage:  c.Model.PackageName,
		Package:       versionedPkg,
	}

	hasLocations := c.Model.DefinitionLocationCount() > 0

	// Collect top-level enums and messages in proto definition order
	for _, enum := range c.Model.Enums {
		if enum.Parent != nil {
			continue
		}
		loc, ok := c.Model.DefinitionLocation(enum.ID)
		if (hasLocations && ok && loc.Filename == filename) || !hasLocations {
			if eAnn, ok := enum.Codec.(*EnumAnnotations); ok && eAnn != nil {
				pAnn.TopLevelEnums = append(pAnn.TopLevelEnums, eAnn)
				pAnn.Manifest = append(pAnn.Manifest, eAnn.Name)
			}
		}
	}

	for _, msg := range c.Model.Messages {
		if msg.Parent != nil || msg.IsMap {
			continue
		}
		loc, ok := c.Model.DefinitionLocation(msg.ID)
		if (hasLocations && ok && loc.Filename == filename) || !hasLocations {
			if mAnn, ok := msg.Codec.(*MessageAnnotations); ok && mAnn != nil {
				pAnn.TopLevelMessages = append(pAnn.TopLevelMessages, mAnn)
				pAnn.Manifest = append(pAnn.Manifest, mAnn.Name)
			}
		}
	}

	pAnn.HasMessagesOrEnums = len(pAnn.TopLevelMessages) > 0 || len(pAnn.TopLevelEnums) > 0

	pAnn.SortedMessages = append([]*MessageAnnotations(nil), pAnn.TopLevelMessages...)
	slices.SortFunc(pAnn.SortedMessages, func(a, b *MessageAnnotations) int {
		return cmp.Compare(a.Name, b.Name)
	})

	pAnn.SortedEnums = append([]*EnumAnnotations(nil), pAnn.TopLevelEnums...)
	slices.SortFunc(pAnn.SortedEnums, func(a, b *EnumAnnotations) int {
		return cmp.Compare(a.Name, b.Name)
	})

	// Collect imports
	pAnn.Imports = c.collectProtoImports(pAnn)
	pAnn.HasImports = len(pAnn.Imports) > 0
	pAnn.HasTopLevelEnums = len(pAnn.TopLevelEnums) > 0
	pAnn.HasTopLevelMessages = len(pAnn.TopLevelMessages) > 0

	return pAnn
}

func (c *codec) collectProtoImports(pAnn *ProtoAnnotations) []string {
	importSet := make(map[string]bool)
	versionedPkg := strings.ReplaceAll(c.packageDir(), "/", ".")

	// Build set of field and message names defined in this proto file to detect collisions
	identifiersInProto := make(map[string]bool)
	for _, msgAnn := range pAnn.TopLevelMessages {
		identifiersInProto[msgAnn.Name] = true
		collectMessageIdentifiers(msgAnn, identifiersInProto)
	}

	for _, msgAnn := range pAnn.TopLevelMessages {
		c.collectFieldImports(msgAnn, pAnn, versionedPkg, identifiersInProto, importSet)
	}

	var imports []string
	for imp := range importSet {
		imports = append(imports, imp)
	}
	slices.Sort(imports)
	return imports
}

func collectMessageIdentifiers(msg *MessageAnnotations, idSet map[string]bool) {
	for _, f := range msg.Fields {
		idSet[f.Name] = true
	}
	for _, nested := range msg.NestedMessages {
		idSet[nested.Name] = true
		collectMessageIdentifiers(nested, idSet)
	}
}

func (c *codec) collectFieldImports(
	msg *MessageAnnotations,
	pAnn *ProtoAnnotations,
	versionedPkg string,
	identifiersInProto map[string]bool,
	importSet map[string]bool,
) {
	for _, f := range msg.Fields {
		c.resolveFieldImport(f.Field, pAnn, versionedPkg, identifiersInProto, importSet)
	}
	for _, nested := range msg.NestedMessages {
		c.collectFieldImports(nested, pAnn, versionedPkg, identifiersInProto, importSet)
	}
}

func (c *codec) resolveFieldImport(
	field *api.Field,
	pAnn *ProtoAnnotations,
	versionedPkg string,
	identifiersInProto map[string]bool,
	importSet map[string]bool,
) {
	if field == nil {
		return
	}

	if field.Map {
		if mapMsg := c.Model.Message(field.TypezID); mapMsg != nil && len(mapMsg.Fields) >= 2 {
			c.resolveFieldImport(mapMsg.Fields[0], pAnn, versionedPkg, identifiersInProto, importSet)
			c.resolveFieldImport(mapMsg.Fields[1], pAnn, versionedPkg, identifiersInProto, importSet)
		}
		return
	}

	if field.Typez != api.TypezMessage && field.Typez != api.TypezEnum {
		return
	}

	typeID := field.TypezID
	if typeID == "" {
		return
	}

	targetMsg := c.Model.Message(typeID)
	var targetEnum *api.Enum
	if targetMsg == nil {
		targetEnum = c.Model.Enum(typeID)
	}

	loc, hasLoc := c.Model.DefinitionLocation(typeID)
	targetFile := ""
	if hasLoc {
		targetFile = loc.Filename
	}

	// If in the same proto file, no import needed
	if targetFile == pAnn.FileName && targetFile != "" {
		return
	}

	targetPkg := ""
	if targetMsg != nil {
		targetPkg = targetMsg.Package
	} else if targetEnum != nil {
		targetPkg = targetEnum.Package
	} else {
		// Parse package from TypezID (.google.protobuf.Timestamp -> google.protobuf)
		trimmed := strings.TrimPrefix(typeID, ".")
		if lastDot := strings.LastIndex(trimmed, "."); lastDot != -1 {
			targetPkg = trimmed[:lastDot]
		}
	}

	targetModule := ""
	if targetFile != "" {
		targetModule = strings.TrimSuffix(filepath.Base(targetFile), ".proto")
	} else {
		targetModule = deriveModuleFromTypeID(typeID)
	}

	// Check if this type belongs to the current package being generated
	if targetPkg == c.Model.PackageName {
		if targetModule != pAnn.ModuleName {
			alias := ""
			if identifiersInProto[targetModule] {
				alias = computeModuleAlias(strings.Split(targetPkg, "."), c.CurrentVersion, targetModule)
			}
			if alias != "" {
				importSet["from "+versionedPkg+".types import "+targetModule+" as "+alias] = true
			} else {
				importSet["from "+versionedPkg+".types import "+targetModule] = true
			}
		}
		return
	}

	// External protobuf type
	importStmt := "import " + targetPkg + "." + targetModule + "_pb2 as " + targetModule + "_pb2  # type: ignore"
	importSet[importStmt] = true
}

func deriveModuleFromTypeID(typeID string) string {
	trimmed := strings.TrimPrefix(typeID, ".")
	parts := strings.Split(trimmed, ".")
	typeName := parts[len(parts)-1]
	switch typeName {
	case "Duration":
		return "duration"
	case "Timestamp":
		return "timestamp"
	case "FieldMask":
		return "field_mask"
	case "Any":
		return "any"
	case "Empty":
		return "empty"
	case "Struct", "Value", "ListValue":
		return "struct"
	default:
		return snakeCase(typeName)
	}
}

func computeModuleAlias(pkgParts []string, version string, module string) string {
	var b strings.Builder
	for _, p := range pkgParts {
		if p == version {
			continue
		}
		for part := range strings.SplitSeq(p, "_") {
			if len(part) > 0 {
				b.WriteByte(part[0])
			}
		}
	}
	b.WriteByte('_')
	b.WriteString(module)
	return b.String()
}
