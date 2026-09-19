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
	"bufio"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
)

type protoIndex struct {
	locations map[string]protoDefinitionLocation
}

func newProtoIndex() *protoIndex {
	return &protoIndex{
		locations: make(map[string]protoDefinitionLocation),
	}
}

func (idx *protoIndex) find(symbol string) (protoDefinitionLocation, bool) {
	if idx != nil {
		if loc, ok := idx.locations[symbol]; ok {
			return loc, true
		}
	}
	return findProtoLocation(symbol)
}

type protoScopeItem struct {
	name       string
	braceDepth int
	isEnum     bool
	isService  bool
}

var (
	reImport    = regexp.MustCompile(`^\s*import\s+(?:public\s+|weak\s+)?"([^"]+)";`)
	rePackage   = regexp.MustCompile(`^\s*package\s+([a-zA-Z0-9_.]+)\s*;`)
	reMessage   = regexp.MustCompile(`\bmessage\s+([A-Za-z0-9_]+)`)
	reEnum      = regexp.MustCompile(`\benum\s+([A-Za-z0-9_]+)`)
	reService   = regexp.MustCompile(`\bservice\s+([A-Za-z0-9_]+)`)
	reRpc       = regexp.MustCompile(`^\s*rpc\s+([A-Za-z0-9_]+)`)
	reEnumValue = regexp.MustCompile(`^\s*([A-Za-z0-9_]+)\s*=\s*[0-9-]+\s*;`)
	reField     = regexp.MustCompile(`^\s*(?:repeated\s+|optional\s+)?([A-Za-z0-9_.]+)\s+([a-z0-9_]+)\s*=\s*[0-9]+`)
)

func (idx *protoIndex) scanFile(absPath string, relPath string) error {
	_, err := idx.scanFileWithImports(absPath, relPath)
	return err
}

func (idx *protoIndex) scanFileWithImports(absPath string, relPath string) ([]string, error) {
	f, err := os.Open(absPath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	relPath = filepath.ToSlash(relPath)
	scanner := bufio.NewScanner(f)
	var pkg string
	var scope []protoScopeItem
	var imports []string
	currentBraceDepth := 0
	inBlockComment := false
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		trimmed := stripProtoComments(line, &inBlockComment)
		if trimmed == "" {
			continue
		}

		if m := reImport.FindStringSubmatch(trimmed); m != nil {
			imports = append(imports, m[1])
		}

		if pkg == "" {
			if m := rePackage.FindStringSubmatch(trimmed); m != nil {
				pkg = m[1]
			}
		}

		if m := reMessage.FindStringSubmatch(trimmed); m != nil {
			name := m[1]
			scope = append(scope, protoScopeItem{name: name, braceDepth: currentBraceDepth + 1, isEnum: false})
			fqn := qualifyProtoSymbol(pkg, scope)
			idx.locations[fqn] = protoDefinitionLocation{Filename: relPath, Line: lineNum}
		} else if m := reEnum.FindStringSubmatch(trimmed); m != nil {
			name := m[1]
			scope = append(scope, protoScopeItem{name: name, braceDepth: currentBraceDepth + 1, isEnum: true})
			fqn := qualifyProtoSymbol(pkg, scope)
			idx.locations[fqn] = protoDefinitionLocation{Filename: relPath, Line: lineNum}
		} else if m := reService.FindStringSubmatch(trimmed); m != nil {
			name := m[1]
			scope = append(scope, protoScopeItem{name: name, braceDepth: currentBraceDepth + 1, isEnum: false, isService: true})
			fqn := qualifyProtoSymbol(pkg, scope)
			idx.locations[fqn] = protoDefinitionLocation{Filename: relPath, Line: lineNum}
		} else if len(scope) > 0 && scope[len(scope)-1].isEnum {
			if m := reEnumValue.FindStringSubmatch(trimmed); m != nil {
				valName := m[1]
				fqn := qualifyProtoSymbol(pkg, scope) + "." + valName
				idx.locations[fqn] = protoDefinitionLocation{Filename: relPath, Line: lineNum}
				if len(scope) >= 2 {
					altFqn := qualifyProtoSymbol(pkg, scope[:len(scope)-1]) + "." + valName
					idx.locations[altFqn] = protoDefinitionLocation{Filename: relPath, Line: lineNum}
				}
			}
		} else if len(scope) > 0 && scope[len(scope)-1].isService {
			if m := reRpc.FindStringSubmatch(trimmed); m != nil {
				rpcName := m[1]
				fqn := qualifyProtoSymbol(pkg, scope) + "." + rpcName
				idx.locations[fqn] = protoDefinitionLocation{Filename: relPath, Line: lineNum}
			}
		} else if len(scope) > 0 && !scope[len(scope)-1].isEnum {
			if m := reField.FindStringSubmatch(trimmed); m != nil {
				fieldName := m[2]
				fqn := qualifyProtoSymbol(pkg, scope) + "." + fieldName
				idx.locations[fqn] = protoDefinitionLocation{Filename: relPath, Line: lineNum}
			}
		}

		for _, ch := range trimmed {
			switch ch {
			case '{':
				currentBraceDepth++
			case '}':
				currentBraceDepth--
				if len(scope) > 0 && currentBraceDepth < scope[len(scope)-1].braceDepth {
					scope = scope[:len(scope)-1]
				}
			}
		}
	}

	return imports, scanner.Err()
}

func qualifyProtoSymbol(pkg string, scope []protoScopeItem) string {
	var parts []string
	if pkg != "" {
		parts = append(parts, pkg)
	}
	for _, s := range scope {
		parts = append(parts, s.name)
	}
	return strings.Join(parts, ".")
}

func stripProtoComments(line string, inBlockComment *bool) string {
	var sb strings.Builder
	inString := false
	escaped := false
	i := 0
	for i < len(line) {
		if *inBlockComment {
			if i+1 < len(line) && line[i] == '*' && line[i+1] == '/' {
				*inBlockComment = false
				i += 2
				continue
			}
			i++
			continue
		}

		ch := line[i]
		if inString {
			sb.WriteByte(ch)
			if escaped {
				escaped = false
				i++
				continue
			}
			if ch == '\\' {
				escaped = true
				i++
				continue
			}
			if ch == '"' {
				inString = false
			}
			i++
			continue
		}

		if ch == '"' {
			inString = true
			sb.WriteByte('"')
			i++
			continue
		}

		if i+1 < len(line) && line[i] == '/' && line[i+1] == '/' {
			break
		}

		if i+1 < len(line) && line[i] == '/' && line[i+1] == '*' {
			*inBlockComment = true
			i += 2
			continue
		}

		sb.WriteByte(ch)
		i++
	}
	return strings.TrimSpace(sb.String())
}

func (c *codec) buildProtoIndex() *protoIndex {
	idx := newProtoIndex()

	var roots []string
	if c.Library != nil {
		for _, r := range c.Library.Roots {
			if r == "googleapis" {
				if c.Cpp != nil && c.Cpp.SourceRoot != "" {
					roots = append(roots, c.Cpp.SourceRoot)
				}
			} else if stat, err := os.Stat(r); err == nil && stat.IsDir() {
				roots = append(roots, r)
			}
		}
	}
	if c.Cpp != nil && c.Cpp.SourceRoot != "" && !slices.Contains(roots, c.Cpp.SourceRoot) {
		roots = append(roots, c.Cpp.SourceRoot)
	}
	if len(roots) == 0 {
		return idx
	}

	visited := make(map[string]bool)
	var scanWithImports func(relPath string)
	scanWithImports = func(relPath string) {
		relPath = filepath.ToSlash(relPath)
		if visited[relPath] {
			return
		}
		visited[relPath] = true

		var absPath string
		for _, root := range roots {
			cand := filepath.Join(root, filepath.FromSlash(relPath))
			if stat, err := os.Stat(cand); err == nil && !stat.IsDir() {
				absPath = cand
				break
			}
		}
		if absPath == "" {
			return
		}

		imports, err := idx.scanFileWithImports(absPath, relPath)
		if err != nil {
			return
		}
		for _, imp := range imports {
			scanWithImports(imp)
		}
	}

	// 1. Scan the library's API proto directory / files
	if c.Library != nil && len(c.Library.APIs) > 0 {
		for _, api := range c.Library.APIs {
			apiPath := api.Path
			if filepath.Ext(apiPath) == ".proto" {
				scanWithImports(apiPath)
			}
			dir := filepath.Dir(apiPath)
			for _, root := range roots {
				absDir := filepath.Join(root, filepath.FromSlash(dir))
				if stat, err := os.Stat(absDir); err == nil && stat.IsDir() {
					_ = filepath.WalkDir(absDir, func(path string, d fs.DirEntry, err error) error {
						if err != nil || d == nil || d.IsDir() {
							return nil
						}
						if filepath.Ext(path) == ".proto" {
							rel, err := filepath.Rel(root, path)
							if err == nil {
								scanWithImports(rel)
							}
						}
						return nil
					})
				}
			}
		}
	} else if c.Cpp != nil && c.Cpp.ProductPath != "" {
		for _, root := range roots {
			absDir := filepath.Join(root, filepath.FromSlash(c.Cpp.ProductPath))
			if stat, err := os.Stat(absDir); err == nil && stat.IsDir() {
				_ = filepath.WalkDir(absDir, func(path string, d fs.DirEntry, err error) error {
					if err != nil || d == nil || d.IsDir() {
						return nil
					}
					if filepath.Ext(path) == ".proto" {
						rel, err := filepath.Rel(root, path)
						if err == nil {
							scanWithImports(rel)
						}
					}
					return nil
				})
			}
		}
	}

	// 2. Scan additional proto files if configured
	if c.Cpp != nil {
		for _, addProto := range c.Cpp.AdditionalProtoFiles {
			scanWithImports(addProto)
		}
	}

	// 3. Scan common well-known Google Cloud protos if present
	commonProtos := []string{
		"google/cloud/location/locations.proto",
		"google/iam/v1/iam_policy.proto",
		"google/iam/v1/policy.proto",
		"google/longrunning/operations.proto",
		"google/protobuf/empty.proto",
	}
	for _, cp := range commonProtos {
		scanWithImports(cp)
	}

	return idx
}
