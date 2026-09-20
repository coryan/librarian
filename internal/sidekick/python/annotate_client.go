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
	"fmt"
	"path/filepath"
	"slices"
	"strings"

	"github.com/googleapis/librarian/internal/sidekick/api"
)

// emptyMessageDocFallback provides the standard docstring for google.protobuf.Empty
// when an LRO returns Empty and the proto source lacks an explicit message description.
const emptyMessageDocFallback = `A generic empty message that you can re-use to avoid defining duplicated
empty messages in your APIs. A typical example is to
use it as the request or the response type of an API
method. For instance:

   service Foo {
      rpc Bar(google.protobuf.Empty) returns
      (google.protobuf.Empty);

   }`

// ClientAnnotations holds all data needed to render client.py for a service.
type ClientAnnotations struct {
	Service *api.Service

	ClientClassName       string
	TransportClassName    string
	ServiceName           string
	ServiceFQN            string
	ServiceTitleSnakeCase string
	ClientTitleSnakeCase  string
	DefaultHost           string
	DefaultHostTemplate   string
	VersionPackage        string
	ParentNamespace       string
	VersionPackageName    string

	DocLines          []string
	FirstDocLine      string
	RemainingDocLines []string
	HasDocLines       bool
	HasRemainingDoc   bool

	RestAsyncIOEnabled bool
	HasLocationMixin   bool
	HasOperationsMixin bool

	HasListOperations  bool
	HasGetOperation    bool
	HasDeleteOperation bool
	HasCancelOperation bool
	HasWaitOperation   bool
	HasGetLocation     bool
	HasListLocations   bool

	IntermediateImports []string
	ResourcePaths       []*ClientResourcePath
	Methods             []*ClientMethodAnnotations
}

// ClientResourcePath represents a service-specific resource path helper.
type ClientResourcePath struct {
	MethodName   string
	PathPattern  string
	FormatArgs   []string
	HasArgs      bool
	Args         []string
	RegexPattern string
}

// ClientMethodAnnotations holds data needed to render a service method in client.py.
type ClientMethodAnnotations struct {
	Name              string
	TransportSafeName string
	RequestType       string
	ReturnType        string
	IsVoid            bool
	IsLRO             bool
	IsPaged           bool
	IsUnary           bool
	HasResponse       bool

	FlattenedParams     []*ClientFlattenedParam
	FlattenedParamsList string
	HasFlattenedParams  bool

	RoutingHeaders []*ClientRoutingHeader
	HasRouting     bool

	LROResponseType string
	LROMetadataType string
	PagerClassName  string

	DocLines                   []string
	FirstDocLine               string
	RemainingDocLines          []string
	HasDocLines                bool
	HasRemainingDoc            bool
	RequestDocLines            []string
	HasRequestDocLines         bool
	ResultDocLines             []string
	FirstResultDocLine         string
	RemainingResultDocLines    []string
	HasResultDocLines          bool
	ResultHasTrailingBlankLine bool
	LROOnSameLine              bool

	ResultSphinx string
	PagerSphinx  string

	HasSnippet             bool
	SampleSubmessages      []*ClientSampleSubmessage
	SampleRequestFields    []*ClientSampleRequestField
	HasSampleRequestFields bool
	RequestTypeName        string
}

// ClientFlattenedParam represents a keyword-only flattened parameter in a client method.
type ClientFlattenedParam struct {
	Name       string
	Key        string
	TypeIdent  string
	SphinxType string
	DocLines   []string
}

// ClientRoutingHeader represents implicit routing headers extracted from AIP-4222 annotations.
type ClientRoutingHeader struct {
	HeaderKey string
	FieldPath string
}

// ClientSampleSubmessage represents a submessage instantiation in the sample code block.
type ClientSampleSubmessage struct {
	VarName     string
	TypeName    string
	Assignments []*ClientSampleAssignment
}

// ClientSampleAssignment represents a field assignment on a submessage in sample code.
type ClientSampleAssignment struct {
	FieldPath string
	Value     string
}

// ClientSampleRequestField represents a request constructor argument in sample code.
type ClientSampleRequestField struct {
	FieldName string
	Value     string
}

func (c *codec) annotateClient(service *api.Service, svcAnn *ServiceAnnotations) *ClientAnnotations {
	if service == nil || svcAnn == nil || svcAnn.Transport == nil {
		return nil
	}

	model := c.Model
	if model == nil && service.Model != nil {
		model = service.Model
	}

	tAnn := svcAnn.Transport
	versionPackage := tAnn.VersionPackage

	parentNamespace := versionPackage
	versionPackageName := versionPackage
	if lastDot := strings.LastIndex(versionPackage, "."); lastDot != -1 {
		parentNamespace = versionPackage[:lastDot]
		versionPackageName = versionPackage[lastDot+1:]
	}

	serviceTitleSnakeCase := strings.ReplaceAll(snakeCase(service.Name), "_", " ")
	clientTitleSnakeCase := strings.ReplaceAll(snakeCase(svcAnn.ClientName), "_", " ")

	docLines := formatRSTDocLines(service.Documentation, 72, 4)
	var firstDoc string
	var remDocs []string
	if len(docLines) > 0 {
		firstDoc = docLines[0]
		remDocs = docLines[1:]
	}

	defaultHost := tAnn.DefaultHost
	defaultHostTemplate := defaultHost
	if prefix, ok := strings.CutSuffix(defaultHost, ".googleapis.com"); ok {
		defaultHostTemplate = prefix + ".{UNIVERSE_DOMAIN}"
	}

	cAnn := &ClientAnnotations{
		Service:               service,
		ClientClassName:       svcAnn.ClientName,
		TransportClassName:    tAnn.TransportClassName,
		ServiceName:           service.Name,
		ServiceFQN:            tAnn.ServiceFQN,
		ServiceTitleSnakeCase: serviceTitleSnakeCase,
		ClientTitleSnakeCase:  clientTitleSnakeCase,
		DefaultHost:           defaultHost,
		DefaultHostTemplate:   defaultHostTemplate,
		VersionPackage:        versionPackage,
		ParentNamespace:       parentNamespace,
		VersionPackageName:    versionPackageName,
		DocLines:              docLines,
		FirstDocLine:          firstDoc,
		RemainingDocLines:     remDocs,
		HasDocLines:           len(docLines) > 0,
		HasRemainingDoc:       len(remDocs) > 0,
		RestAsyncIOEnabled:    tAnn.RestAsyncIOEnabled,
		HasLocationMixin:      tAnn.HasLocationMixin,
		HasOperationsMixin:    tAnn.HasOperationsMixin,
		HasListOperations:     tAnn.HasListOperations,
		HasGetOperation:       tAnn.HasGetOperation,
		HasDeleteOperation:    tAnn.HasDeleteOperation,
		HasCancelOperation:    tAnn.HasCancelOperation,
		HasWaitOperation:      tAnn.HasWaitOperation,
		HasGetLocation:        tAnn.HasGetLocation,
		HasListLocations:      tAnn.HasListLocations,
	}

	cAnn.ResourcePaths = c.buildResourcePaths(service, model)
	cAnn.IntermediateImports = c.buildIntermediateImports(service, svcAnn)

	for _, m := range service.Methods {
		if isMixinMethod(m, service) {
			continue
		}
		mAnn := c.buildClientMethod(m, svcAnn, cAnn, service, model)
		cAnn.Methods = append(cAnn.Methods, mAnn)
	}

	return cAnn
}

var commonResources = map[string]bool{
	"cloudresourcemanager.googleapis.com/Project":      true,
	"cloudresourcemanager.googleapis.com/Organization": true,
	"cloudresourcemanager.googleapis.com/Folder":       true,
	"cloudbilling.googleapis.com/BillingAccount":       true,
	"locations.googleapis.com/Location":                true,
}

func (c *codec) buildResourcePaths(service *api.Service, model *api.API) []*ClientResourcePath {
	if model == nil {
		return nil
	}

	referencedResources := make(map[string]*api.Resource)
	visitedMessages := make(map[string]bool)

	var visitMessage func(msg *api.Message)
	visitMessage = func(msg *api.Message) {
		if msg == nil || visitedMessages[msg.ID] {
			return
		}
		visitedMessages[msg.ID] = true

		if msg.Resource != nil && !commonResources[msg.Resource.Type] {
			referencedResources[msg.Resource.Type] = msg.Resource
		}

		for _, f := range msg.Fields {
			if f.ResourceReference != nil {
				resType := f.ResourceReference.Type
				if resType == "" {
					resType = f.ResourceReference.ChildType
				}
				if resType != "" && !commonResources[resType] && resType != "*" {
					if res := findResource(model, resType); res != nil {
						referencedResources[res.Type] = res
					}
				}
			}
			if f.Typez == api.TypezMessage {
				subMsg := model.Message(f.TypezID)
				visitMessage(subMsg)
			}
		}
	}

	for _, m := range service.Methods {
		if isMixinMethod(m, service) {
			continue
		}
		reqMsg := model.Message(m.InputTypeID)
		visitMessage(reqMsg)

		respMsg := model.Message(m.OutputTypeID)
		if m.OperationInfo != nil && m.OperationInfo.ResponseTypeID != "" {
			respMsg = model.Message(m.OperationInfo.ResponseTypeID)
		}
		visitMessage(respMsg)
	}

	type resEntry struct {
		name string
		res  *api.Resource
	}
	var entries []resEntry
	for _, res := range referencedResources {
		if res == nil || commonResources[res.Type] {
			continue
		}
		entries = append(entries, resEntry{
			name: extractResourceEntityName(res.Type),
			res:  res,
		})
	}
	slices.SortFunc(entries, func(a, b resEntry) int {
		if c := strings.Compare(a.name, b.name); c != 0 {
			return c
		}
		return strings.Compare(a.res.Type, b.res.Type)
	})

	var paths []*ClientResourcePath
	for _, entry := range entries {
		name := entry.name
		res := entry.res

		if len(res.Patterns) == 0 {
			continue
		}
		pattern := res.Patterns[0]

		if len(pattern) == 0 || (len(pattern) == 1 && (pattern[0].Literal == "*" || (pattern[0].Variable != nil && len(pattern[0].Variable.FieldPath) > 0 && pattern[0].Variable.FieldPath[0] == "*"))) {
			paths = append(paths, &ClientResourcePath{
				MethodName:   name,
				PathPattern:  "*",
				RegexPattern: "^.*$",
			})
			continue
		}

		pathPattern, formatArgs, args, regexPattern := parseResourcePattern(pattern)
		paths = append(paths, &ClientResourcePath{
			MethodName:   name,
			PathPattern:  pathPattern,
			FormatArgs:   formatArgs,
			HasArgs:      len(args) > 0,
			Args:         args,
			RegexPattern: regexPattern,
		})
	}

	return paths
}

func findResource(model *api.API, resType string) *api.Resource {
	if model == nil {
		return nil
	}
	if res := model.Resource(resType); res != nil {
		return res
	}
	if idx := slices.IndexFunc(model.ResourceDefinitions, func(r *api.Resource) bool {
		return r.Type == resType
	}); idx != -1 {
		return model.ResourceDefinitions[idx]
	}
	return nil
}

func extractResourceEntityName(resType string) string {
	parts := strings.Split(resType, "/")
	raw := parts[len(parts)-1]
	return snakeCase(raw)
}

func parseResourcePattern(pattern api.ResourcePattern) (string, []string, []string, string) {
	var args []string
	var regexParts []string
	var formatParts []string

	for _, s := range pattern {
		if strings.HasPrefix(s.Literal, "//") {
			continue
		}
		if s.Literal != "" {
			lit := strings.Trim(s.Literal, "/")
			if lit != "" {
				formatParts = append(formatParts, lit)
				regexParts = append(regexParts, lit)
			}
		}
		if s.Variable != nil && len(s.Variable.FieldPath) > 0 {
			varName := s.Variable.FieldPath[0]
			args = append(args, varName)
			formatParts = append(formatParts, fmt.Sprintf("{%s}", varName))
			regexParts = append(regexParts, fmt.Sprintf("(?P<%s>.+?)", varName))
		}
	}

	var formatArgs []string
	for _, a := range args {
		formatArgs = append(formatArgs, fmt.Sprintf("%s=%s", a, a))
	}

	pathPattern := strings.Join(formatParts, "/")
	regexPattern := fmt.Sprintf("^%s$", strings.Join(regexParts, "/"))
	return pathPattern, formatArgs, args, regexPattern
}

func resolveProtoModule(typeID string, model *api.API) string {
	if model != nil {
		for curID := typeID; curID != ""; {
			if loc, ok := model.DefinitionLocation(curID); ok && loc.Filename != "" {
				return strings.TrimSuffix(filepath.Base(loc.Filename), ".proto")
			}
			idx := strings.LastIndex(curID, ".")
			if idx <= 0 {
				break
			}
			curID = curID[:idx]
		}
	}
	return deriveModuleFromTypeID(typeID)
}

func resolveProtoPackage(typeID string, model *api.API) string {
	if model != nil {
		for curID := typeID; curID != ""; {
			if msg := model.Message(curID); msg != nil {
				return msg.Package
			}
			if en := model.Enum(curID); en != nil {
				return en.Package
			}
			idx := strings.LastIndex(curID, ".")
			if idx <= 0 {
				break
			}
			curID = curID[:idx]
		}
	}
	trimmed := strings.TrimPrefix(typeID, ".")
	if lastDot := strings.LastIndex(trimmed, "."); lastDot != -1 {
		return trimmed[:lastDot]
	}
	return ""
}

func (c *codec) addTypeIDImport(typeID string, service *api.Service, versionPackage string, importsSet map[string]bool) {
	if typeID == "" {
		return
	}
	trimmed := strings.TrimPrefix(typeID, ".")
	if trimmed == "google.protobuf.Any" || strings.HasPrefix(trimmed, "google.longrunning.") {
		return
	}

	pkgPrefix := service.Package
	modelPkgPrefix := ""
	if c.Model != nil {
		modelPkgPrefix = c.Model.PackageName
	}

	if (pkgPrefix != "" && (trimmed == pkgPrefix || strings.HasPrefix(trimmed, pkgPrefix+"."))) ||
		(modelPkgPrefix != "" && (trimmed == modelPkgPrefix || strings.HasPrefix(trimmed, modelPkgPrefix+"."))) {
		modName := c.resolveTypeModule(typeID, service)
		if modName != "" {
			importsSet[fmt.Sprintf("from %s.types import %s", versionPackage, modName)] = true
		}
		return
	}

	targetPkg := resolveProtoPackage(typeID, c.Model)
	targetModule := resolveProtoModule(typeID, c.Model)
	if targetPkg != "" && targetModule != "" {
		importsSet[fmt.Sprintf("import %s.%s_pb2 as %s_pb2  # type: ignore", targetPkg, targetModule, targetModule)] = true
	}
}

func (c *codec) buildIntermediateImports(service *api.Service, svcAnn *ServiceAnnotations) []string {
	model := service.Model
	if model == nil && c.Model != nil {
		model = c.Model
	}
	versionPackage := svcAnn.Transport.VersionPackage
	importsSet := make(map[string]bool)

	for _, m := range service.Methods {
		if isMixinMethod(m, service) {
			continue
		}

		c.addTypeIDImport(m.InputTypeID, service, versionPackage, importsSet)

		for _, sig := range m.Signatures {
			for _, f := range sig.Fields {
				if f.Map {
					continue
				}
				if f.Typez == api.TypezMessage || f.Typez == api.TypezEnum {
					c.addTypeIDImport(f.TypezID, service, versionPackage, importsSet)
				}
			}
		}

		isLRO := m.OperationInfo != nil || m.IsLRO || m.OutputTypeID == ".google.longrunning.Operation"
		isVoid := m.ReturnsEmpty || m.OutputTypeID == "" || m.OutputTypeID == ".google.protobuf.Empty" || m.OutputTypeID == "google.protobuf.Empty"

		if !isVoid && !isLRO {
			c.addTypeIDImport(m.OutputTypeID, service, versionPackage, importsSet)
			var respMsg *api.Message
			if model != nil {
				respMsg = model.Message(m.OutputTypeID)
			}
			if respMsg == nil && m.OutputType != nil {
				respMsg = m.OutputType
			}
			if respMsg != nil {
				for _, f := range respMsg.Fields {
					if f.Map {
						continue
					}
					if f.Typez == api.TypezMessage || f.Typez == api.TypezEnum {
						c.addTypeIDImport(f.TypezID, service, versionPackage, importsSet)
					}
				}
			}
		}

		if isLRO && m.OperationInfo != nil {
			if m.OperationInfo.ResponseTypeID != "" {
				c.addTypeIDImport(m.OperationInfo.ResponseTypeID, service, versionPackage, importsSet)
				if model != nil {
					respMsg := model.Message(m.OperationInfo.ResponseTypeID)
					if respMsg != nil {
						for _, f := range respMsg.Fields {
							if f.Map {
								continue
							}
							if f.Typez == api.TypezMessage || f.Typez == api.TypezEnum {
								c.addTypeIDImport(f.TypezID, service, versionPackage, importsSet)
							}
						}
					}
				}
			}
			if m.OperationInfo.MetadataTypeID != "" {
				c.addTypeIDImport(m.OperationInfo.MetadataTypeID, service, versionPackage, importsSet)
			}
		}
	}

	if svcAnn.Pagers != nil && len(svcAnn.Pagers.Pagers) > 0 {
		importsSet[fmt.Sprintf("from %s.services.%s import pagers", versionPackage, svcAnn.DirectoryName)] = true
	}
	// Note: The single space before '# type: ignore' matches upstream gapic-generator-python
	// and golden fixture formatting for location and operations mixin imports.
	if svcAnn.Transport.HasLocationMixin {
		importsSet["from google.cloud.location import locations_pb2 # type: ignore"] = true
	}
	if svcAnn.Transport.HasOperationsMixin {
		importsSet["from google.longrunning import operations_pb2 # type: ignore"] = true
	}
	if svcAnn.Transport.HasLRO {
		importsSet["import google.api_core.operation as operation  # type: ignore"] = true
		importsSet["import google.api_core.operation_async as operation_async  # type: ignore"] = true
	}

	var result []string
	for imp := range importsSet {
		result = append(result, imp)
	}
	slices.Sort(result)
	return result
}

func (c *codec) buildClientMethod(m *api.Method, svcAnn *ServiceAnnotations, cAnn *ClientAnnotations, service *api.Service, model *api.API) *ClientMethodAnnotations {
	versionPackage := cAnn.VersionPackage
	methodName := snakeCase(m.Name)
	transportSafeName := methodName

	var reqMsg *api.Message
	if model != nil {
		reqMsg = model.Message(m.InputTypeID)
	}
	if reqMsg == nil && m.InputType != nil {
		reqMsg = m.InputType
	}
	reqModule := c.resolveTypeModule(typeIDOrMsgID(m.InputTypeID, reqMsg), service)
	reqTypeName := resolveMessageRelativeName(reqMsg)
	if reqTypeName == "" {
		reqTypeName = typeNameWithFallback(m.InputTypeID, m.Name+"Request")
	}
	requestType := fmt.Sprintf("%s.%s", reqModule, reqTypeName)

	isVoid := m.ReturnsEmpty || m.OutputTypeID == "" || m.OutputTypeID == ".google.protobuf.Empty" || m.OutputTypeID == "google.protobuf.Empty"
	isLRO := m.OperationInfo != nil || m.IsLRO
	isPaged := m.Pagination != nil
	isUnary := !isVoid && !isLRO && !isPaged

	returnType := "None"
	var lroRespType, lroMetaType string
	var pagerClassName, pagerSphinx, resultSphinx string
	var lroOnSameLine bool
	var lroRespMsg *api.Message
	var resultDocLines []string
	var firstResultDoc string
	var remResultDocs []string
	if isLRO {
		returnType = "operation.Operation"
		if m.OperationInfo != nil {
			respTypeID := m.OperationInfo.ResponseTypeID
			if model != nil {
				lroRespMsg = model.Message(respTypeID)
			}
			if lroRespMsg != nil {
				if lroRespMsg.Package == service.Package || (model != nil && lroRespMsg.Package == model.PackageName) {
					respMod := c.resolveTypeModule(lroRespMsg.ID, service)
					relName := resolveMessageRelativeName(lroRespMsg)
					lroRespType = fmt.Sprintf("%s.%s", respMod, relName)
					resultSphinx = fmt.Sprintf("%s.types.%s", versionPackage, relName)
				} else {
					targetModule := resolveProtoModule(lroRespMsg.ID, model)
					lroRespType = fmt.Sprintf("%s_pb2.%s", targetModule, lroRespMsg.Name)
					resultSphinx = fmt.Sprintf("%s.%s_pb2.%s", lroRespMsg.Package, targetModule, lroRespMsg.Name)
				}
			} else if respTypeID != "" {
				trimmed := strings.TrimPrefix(respTypeID, ".")
				parts := strings.Split(trimmed, ".")
				typeName := parts[len(parts)-1]
				targetPkg := resolveProtoPackage(respTypeID, model)
				targetModule := resolveProtoModule(respTypeID, model)
				lroRespType = fmt.Sprintf("%s_pb2.%s", targetModule, typeName)
				resultSphinx = fmt.Sprintf("%s.%s_pb2.%s", targetPkg, targetModule, typeName)
			}

			metaTypeID := m.OperationInfo.MetadataTypeID
			var metaMsg *api.Message
			if model != nil {
				metaMsg = model.Message(metaTypeID)
			}
			if metaMsg != nil {
				if metaMsg.Package == service.Package || (model != nil && metaMsg.Package == model.PackageName) {
					metaMod := c.resolveTypeModule(metaMsg.ID, service)
					relName := resolveMessageRelativeName(metaMsg)
					lroMetaType = fmt.Sprintf("%s.%s", metaMod, relName)
				} else {
					targetModule := resolveProtoModule(metaMsg.ID, model)
					lroMetaType = fmt.Sprintf("%s_pb2.%s", targetModule, metaMsg.Name)
				}
			} else if metaTypeID != "" {
				trimmed := strings.TrimPrefix(metaTypeID, ".")
				parts := strings.Split(trimmed, ".")
				typeName := parts[len(parts)-1]
				targetModule := resolveProtoModule(metaTypeID, model)
				lroMetaType = fmt.Sprintf("%s_pb2.%s", targetModule, typeName)
			}
		}

		var respDoc string
		if lroRespMsg != nil {
			respDoc = strings.TrimSpace(lroRespMsg.Documentation)
		}
		if respDoc == "" && m.OperationInfo != nil && (m.OperationInfo.ResponseTypeID == ".google.protobuf.Empty" || m.OperationInfo.ResponseTypeID == "google.protobuf.Empty") {
			respDoc = emptyMessageDocFallback
		}
		if strings.Contains(respDoc, "\n") || strings.Contains(respDoc, "[") {
			lroOnSameLine = true
			lines := strings.Split(respDoc, "\n")
			if len(lines) > 0 {
				firstResultDoc = lines[0]
				for _, line := range lines[1:] {
					if strings.TrimSpace(line) == "" {
						remResultDocs = append(remResultDocs, "")
					} else {
						remResultDocs = append(remResultDocs, "   "+line)
					}
				}
			}
		} else {
			lroOnSameLine = false
			if respDoc != "" {
				words := strings.Fields(respDoc)
				// In legacy gapic-generator-python, if the first word of the response docstring
				// fits on the opening Returns line (<= 10 chars), it is placed inline after
				// the opening return type annotation; longer words start on the subsequent line.
				const maxInlineFirstWordLen = 10
				if len(words) > 0 && len(words[0]) <= maxInlineFirstWordLen {
					firstResultDoc = words[0]
					remText := strings.TrimSpace(strings.TrimPrefix(respDoc, words[0]))
					remResultDocs = formatRSTDocLines(remText, 72, 16)
				} else {
					remResultDocs = formatRSTDocLines(respDoc, 72, 16)
				}
			}
		}
	} else if isPaged {
		pagerClassName = fmt.Sprintf("%sPager", m.Name)
		returnType = fmt.Sprintf("pagers.%s", pagerClassName)
		pagerSphinx = fmt.Sprintf("%s.services.%s.pagers.%s", versionPackage, svcAnn.DirectoryName, pagerClassName)
		var respMsg *api.Message
		if model != nil {
			respMsg = model.Message(m.OutputTypeID)
		}
		if respMsg == nil && m.OutputType != nil {
			respMsg = m.OutputType
		}
		var pagedDoc string
		if respMsg != nil && strings.TrimSpace(respMsg.Documentation) != "" {
			pagedDoc = strings.TrimSpace(respMsg.Documentation) + "\n\nIterating over this object will yield results and resolve additional pages automatically."
		} else {
			pagedDoc = "Iterating over this object will yield results and resolve additional pages automatically."
		}
		hasExplicitBracketNewline := respMsg != nil && strings.Contains(respMsg.Documentation, "\n[")
		resultDocLines = formatRSTDocLines(pagedDoc, 72, 16)
		if hasExplicitBracketNewline {
			for i, line := range resultDocLines {
				if strings.HasPrefix(line, "[") {
					resultDocLines[i] = "   " + line
				}
			}
		}
		if len(resultDocLines) > 0 {
			firstResultDoc = resultDocLines[0]
			remResultDocs = resultDocLines[1:]
		}
	} else if !isVoid {
		var respMsg *api.Message
		if model != nil {
			respMsg = model.Message(m.OutputTypeID)
		}
		if respMsg == nil && m.OutputType != nil {
			respMsg = m.OutputType
		}
		if respMsg != nil {
			respMod := c.resolveTypeModule(typeIDOrMsgID(m.OutputTypeID, respMsg), service)
			relName := resolveMessageRelativeName(respMsg)
			if relName == "" {
				relName = respMsg.Name
			}
			returnType = fmt.Sprintf("%s.%s", respMod, relName)
			resultSphinx = fmt.Sprintf("%s.types.%s", versionPackage, relName)
			hasExplicitBracketNewline := strings.Contains(respMsg.Documentation, "\n[")
			resultDocLines = formatRSTDocLines(respMsg.Documentation, 72, 16)
			if hasExplicitBracketNewline {
				for i, line := range resultDocLines {
					if strings.HasPrefix(line, "[") {
						resultDocLines[i] = "   " + line
					}
				}
			}
			if len(resultDocLines) > 0 {
				firstResultDoc = resultDocLines[0]
				remResultDocs = resultDocLines[1:]
			}
		}
	}

	docLines := formatRSTDocLines(m.Documentation, 72, 8)
	var firstDoc string
	var remDocs []string
	if len(docLines) > 0 {
		firstDoc = docLines[0]
		remDocs = docLines[1:]
	}

	var reqDocLines []string
	if reqMsg != nil && strings.TrimSpace(reqMsg.Documentation) != "" {
		lines := formatRSTDocLines(strings.TrimSpace(reqMsg.Documentation), 72, 16)
		if len(lines) > 0 {
			lines[0] = "The request object. " + lines[0]
			reqDocLines = lines
		} else {
			reqDocLines = []string{"The request object."}
		}
	} else {
		reqDocLines = []string{"The request object."}
	}

	flattenedParams, flattenedList := c.buildFlattenedParams(m, reqMsg, svcAnn, model, service)
	routingHeaders := c.buildRoutingHeaders(m)
	sampleSubmessages, sampleReqFields := c.buildSampleSnippet(reqMsg, model)

	return &ClientMethodAnnotations{
		Name:                       methodName,
		TransportSafeName:          transportSafeName,
		RequestType:                requestType,
		ReturnType:                 returnType,
		IsVoid:                     isVoid,
		IsLRO:                      isLRO,
		IsPaged:                    isPaged,
		IsUnary:                    isUnary,
		HasResponse:                !isVoid,
		FlattenedParams:            flattenedParams,
		FlattenedParamsList:        flattenedList,
		HasFlattenedParams:         len(flattenedParams) > 0,
		RoutingHeaders:             routingHeaders,
		HasRouting:                 len(routingHeaders) > 0,
		LROResponseType:            lroRespType,
		LROMetadataType:            lroMetaType,
		PagerClassName:             pagerClassName,
		DocLines:                   docLines,
		FirstDocLine:               firstDoc,
		RemainingDocLines:          remDocs,
		HasDocLines:                len(docLines) > 0,
		HasRemainingDoc:            len(remDocs) > 0,
		RequestDocLines:            reqDocLines,
		HasRequestDocLines:         len(reqDocLines) > 0,
		ResultDocLines:             resultDocLines,
		FirstResultDocLine:         firstResultDoc,
		RemainingResultDocLines:    remResultDocs,
		HasResultDocLines:          len(resultDocLines) > 0 || firstResultDoc != "",
		ResultHasTrailingBlankLine: isUnary && (len(resultDocLines) > 1 || (len(resultDocLines) == 0 && firstResultDoc == "")),
		LROOnSameLine:              lroOnSameLine,
		ResultSphinx:               resultSphinx,
		PagerSphinx:                pagerSphinx,
		HasSnippet:                 true,
		SampleSubmessages:          sampleSubmessages,
		SampleRequestFields:        sampleReqFields,
		HasSampleRequestFields:     len(sampleReqFields) > 0,
		RequestTypeName:            reqTypeName,
	}
}

func (c *codec) buildFlattenedParams(m *api.Method, reqMsg *api.Message, svcAnn *ServiceAnnotations, model *api.API, service *api.Service) ([]*ClientFlattenedParam, string) {
	if reqMsg == nil || len(m.Signatures) == 0 {
		return nil, ""
	}

	seen := make(map[string]bool)
	var params []*ClientFlattenedParam
	var paramNames []string

	for _, sig := range m.Signatures {
		for _, f := range sig.Fields {
			paramName := pythonIdentifier(snakeCase(f.Name))
			if seen[paramName] {
				continue
			}
			seen[paramName] = true
			paramNames = append(paramNames, paramName)

			typeIdent := c.formatFieldTypeIdent(f, service.Package, model, service)
			sphinxType := c.formatFieldSphinxType(f, service.Package, svcAnn.Transport.VersionPackage, model)
			docLines := formatRSTDocLines(f.Documentation, 72, 16)
			if len(docLines) > 1 {
				docLines = append(docLines, "")
			}

			params = append(params, &ClientFlattenedParam{
				Name:       paramName,
				Key:        f.Name,
				TypeIdent:  typeIdent,
				SphinxType: sphinxType,
				DocLines:   docLines,
			})
		}
	}

	return params, strings.Join(paramNames, ", ")
}

type resolvedFieldType struct {
	found      bool
	isInternal bool
	moduleName string
	typeName   string
	pkg        string
}

func (c *codec) resolveFieldType(f *api.Field, currentPkg string, model *api.API, service *api.Service) resolvedFieldType {
	switch f.Typez {
	case api.TypezMessage:
		var subMsg *api.Message
		if model != nil {
			subMsg = model.Message(f.TypezID)
		}
		if subMsg != nil {
			if subMsg.Package == currentPkg || (model != nil && subMsg.Package == model.PackageName) {
				return resolvedFieldType{
					found:      true,
					isInternal: true,
					moduleName: c.resolveTypeModule(subMsg.ID, service),
					typeName:   resolveMessageRelativeName(subMsg),
					pkg:        subMsg.Package,
				}
			}
			return resolvedFieldType{
				found:      true,
				isInternal: false,
				moduleName: resolveProtoModule(subMsg.ID, model) + "_pb2",
				typeName:   subMsg.Name,
				pkg:        subMsg.Package,
			}
		}
	case api.TypezEnum:
		var subEnum *api.Enum
		if model != nil {
			subEnum = model.Enum(f.TypezID)
		}
		if subEnum != nil {
			if subEnum.Package == currentPkg || (model != nil && subEnum.Package == model.PackageName) {
				return resolvedFieldType{
					found:      true,
					isInternal: true,
					moduleName: c.resolveTypeModule(subEnum.ID, service),
					typeName:   resolveEnumRelativeName(subEnum),
					pkg:        subEnum.Package,
				}
			}
			return resolvedFieldType{
				found:      true,
				isInternal: false,
				moduleName: resolveProtoModule(subEnum.ID, model) + "_pb2",
				typeName:   subEnum.Name,
				pkg:        subEnum.Package,
			}
		}
	}

	trimmed := strings.TrimPrefix(f.TypezID, ".")
	parts := strings.Split(trimmed, ".")
	typeName := parts[len(parts)-1]
	pkg := ""
	if lastDot := strings.LastIndex(trimmed, "."); lastDot != -1 {
		pkg = trimmed[:lastDot]
	}
	isInternal := pkg == currentPkg || (model != nil && pkg == model.PackageName)
	return resolvedFieldType{
		found:      false,
		isInternal: isInternal,
		moduleName: resolveProtoModule(f.TypezID, model) + "_pb2",
		typeName:   typeName,
		pkg:        pkg,
	}
}

func (c *codec) formatFieldTypeIdent(f *api.Field, currentPkg string, model *api.API, service *api.Service) string {
	var base string
	switch f.Typez {
	case api.TypezString:
		base = "str"
	case api.TypezBool:
		base = "bool"
	case api.TypezBytes:
		base = "bytes"
	case api.TypezInt32, api.TypezInt64, api.TypezUint32, api.TypezUint64,
		api.TypezSint32, api.TypezSint64, api.TypezFixed32, api.TypezFixed64,
		api.TypezSfixed32, api.TypezSfixed64:
		base = "int"
	case api.TypezFloat, api.TypezDouble:
		base = "float"
	case api.TypezMessage, api.TypezEnum:
		rt := c.resolveFieldType(f, currentPkg, model, service)
		if rt.isInternal && rt.found {
			if rt.moduleName != "" {
				base = fmt.Sprintf("%s.%s", rt.moduleName, rt.typeName)
			} else {
				base = rt.typeName
			}
		} else {
			base = fmt.Sprintf("%s.%s", rt.moduleName, rt.typeName)
		}
	default:
		base = "Any"
	}

	if f.Repeated {
		return fmt.Sprintf("MutableSequence[%s]", base)
	}
	return base
}

func (c *codec) formatFieldSphinxType(f *api.Field, currentPkg, versionPackage string, model *api.API) string {
	var base string
	switch f.Typez {
	case api.TypezString:
		base = "str"
	case api.TypezBool:
		base = "bool"
	case api.TypezBytes:
		base = "bytes"
	case api.TypezInt32, api.TypezInt64, api.TypezUint32, api.TypezUint64,
		api.TypezSint32, api.TypezSint64, api.TypezFixed32, api.TypezFixed64,
		api.TypezSfixed32, api.TypezSfixed64:
		base = "int"
	case api.TypezFloat, api.TypezDouble:
		base = "float"
	case api.TypezMessage, api.TypezEnum:
		rt := c.resolveFieldType(f, currentPkg, model, nil)
		if rt.isInternal {
			base = fmt.Sprintf("%s.types.%s", versionPackage, rt.typeName)
		} else {
			base = fmt.Sprintf("%s.%s.%s", rt.pkg, rt.moduleName, rt.typeName)
		}
	default:
		base = "Any"
	}

	if f.Repeated {
		return fmt.Sprintf("MutableSequence[%s]", base)
	}
	return base
}

func (c *codec) buildRoutingHeaders(m *api.Method) []*ClientRoutingHeader {
	var headers []*ClientRoutingHeader

	if len(m.Routing) > 0 {
		seen := make(map[string]bool)
		for _, r := range m.Routing {
			if r.Name == "" || len(r.Variants) == 0 {
				continue
			}
			if seen[r.Name] {
				continue
			}
			seen[r.Name] = true

			v := r.Variants[0]
			var pathParts []string
			for _, fp := range v.FieldPath {
				pathParts = append(pathParts, snakeCase(fp))
			}
			fieldPath := "request." + strings.Join(pathParts, ".")
			headers = append(headers, &ClientRoutingHeader{
				HeaderKey: r.Name,
				FieldPath: fieldPath,
			})
		}
		return headers
	}

	if m.PathInfo != nil && len(m.PathInfo.Bindings) > 0 {
		b := m.PathInfo.Bindings[0]
		if b.PathTemplate != nil {
			seen := make(map[string]bool)
			for _, seg := range b.PathTemplate.Segments {
				if seg.Variable != nil && len(seg.Variable.FieldPath) > 0 {
					headerKey := strings.Join(seg.Variable.FieldPath, ".")
					if seen[headerKey] {
						continue
					}
					seen[headerKey] = true

					var pathParts []string
					for _, fp := range seg.Variable.FieldPath {
						pathParts = append(pathParts, snakeCase(fp))
					}
					fieldPath := "request." + strings.Join(pathParts, ".")
					headers = append(headers, &ClientRoutingHeader{
						HeaderKey: headerKey,
						FieldPath: fieldPath,
					})
				}
			}
		}
	}

	return headers
}

func (c *codec) buildSampleSnippet(reqMsg *api.Message, model *api.API) ([]*ClientSampleSubmessage, []*ClientSampleRequestField) {
	if reqMsg == nil {
		return nil, nil
	}

	var candidateFields []*api.Field
	for _, oneof := range reqMsg.OneOfs {
		if len(oneof.Fields) > 0 {
			candidateFields = append(candidateFields, oneof.Fields[0])
		}
	}
	for _, f := range reqMsg.Fields {
		if f.DocumentAsRequired() && !f.IsOneOf {
			candidateFields = append(candidateFields, f)
		}
	}

	var submessages []*ClientSampleSubmessage
	var reqFields []*ClientSampleRequestField

	for _, f := range candidateFields {
		if f.Typez == api.TypezMessage && model != nil {
			subMsg := model.Message(f.TypezID)
			if subMsg == nil {
				continue
			}

			var subCandidates []*api.Field
			for _, oneof := range subMsg.OneOfs {
				if len(oneof.Fields) > 0 {
					subCandidates = append(subCandidates, oneof.Fields[0])
				}
			}
			for _, sf := range subMsg.Fields {
				if sf.DocumentAsRequired() && !sf.IsOneOf {
					subCandidates = append(subCandidates, sf)
				}
			}

			if len(subCandidates) == 0 {
				continue
			}

			var assignments []*ClientSampleAssignment
			for _, sc := range subCandidates {
				if sc.Typez == api.TypezMessage && model != nil {
					nestedMsg := model.Message(sc.TypezID)
					if nestedMsg != nil {
						var nestedCandidates []*api.Field
						for _, oneof := range nestedMsg.OneOfs {
							if len(oneof.Fields) > 0 {
								nestedCandidates = append(nestedCandidates, oneof.Fields[0])
							}
						}
						for _, nf := range nestedMsg.Fields {
							if nf.DocumentAsRequired() && !nf.IsOneOf {
								nestedCandidates = append(nestedCandidates, nf)
							}
						}
						for _, nf := range nestedCandidates {
							if nf.Typez == api.TypezMessage {
								continue
							}
							val := c.mockFieldValue(nf, model)
							assignments = append(assignments, &ClientSampleAssignment{
								FieldPath: fmt.Sprintf("%s.%s", sc.Name, nf.Name),
								Value:     val,
							})
						}
					}
				} else {
					val := c.mockFieldValue(sc, model)
					assignments = append(assignments, &ClientSampleAssignment{
						FieldPath: sc.Name,
						Value:     val,
					})
				}
			}

			submessages = append(submessages, &ClientSampleSubmessage{
				VarName:     f.Name,
				TypeName:    subMsg.Name,
				Assignments: assignments,
			})
			reqFields = append(reqFields, &ClientSampleRequestField{
				FieldName: f.Name,
				Value:     f.Name,
			})
		} else {
			val := c.mockFieldValue(f, model)
			reqFields = append(reqFields, &ClientSampleRequestField{
				FieldName: f.Name,
				Value:     val,
			})
		}
	}

	return submessages, reqFields
}

func (c *codec) mockFieldValue(f *api.Field, model *api.API) string {
	if f.Repeated {
		if f.Typez == api.TypezString {
			return fmt.Sprintf("['%s_value1', '%s_value2']", f.Name, f.Name)
		}
		if f.Typez == api.TypezEnum && model != nil {
			en := model.Enum(f.TypezID)
			if en != nil && len(en.Values) > 0 {
				last := en.Values[len(en.Values)-1].Name
				return fmt.Sprintf("['%s']", last)
			}
		}
		return "[]"
	}

	switch f.Typez {
	case api.TypezString:
		return fmt.Sprintf("\"%s_value\"", f.Name)
	case api.TypezBytes:
		return fmt.Sprintf("b'%s_blob'", f.Name)
	case api.TypezBool:
		return "True"
	case api.TypezInt32, api.TypezInt64, api.TypezUint32, api.TypezUint64,
		api.TypezSint32, api.TypezSint64, api.TypezFixed32, api.TypezFixed64,
		api.TypezSfixed32, api.TypezSfixed64:
		// Replicate legacy gapic-generator-python's mock integer generation algorithm,
		// which sums the ASCII character codes of the field name to produce deterministic
		// integer mock values in sample snippets.
		sum := 0
		for _, ch := range f.Name {
			sum += int(ch)
		}
		return fmt.Sprintf("%d", sum)
	case api.TypezEnum:
		if model != nil {
			en := model.Enum(f.TypezID)
			if en != nil && len(en.Values) > 0 {
				last := en.Values[len(en.Values)-1].Name
				return fmt.Sprintf("\"%s\"", last)
			}
		}
		return "\"\""
	default:
		return "\"\""
	}
}
