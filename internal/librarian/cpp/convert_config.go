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
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/googleapis/librarian/internal/config"
)

type tokenType int

const (
	tokEOF tokenType = iota
	tokIdent
	tokString
	tokOpenBrace    // {
	tokCloseBrace   // }
	tokOpenBracket  // [
	tokCloseBracket // ]
	tokColon        // :
	tokComma        // ,
)

type token struct {
	typ  tokenType
	val  string
	line int
}

type textprotoLexer struct {
	input []rune
	pos   int
	line  int
}

func newTextprotoLexer(s string) *textprotoLexer {
	return &textprotoLexer{
		input: []rune(s),
		pos:   0,
		line:  1,
	}
}

func (l *textprotoLexer) nextToken() (token, error) {
	for l.pos < len(l.input) {
		r := l.input[l.pos]
		if r == '\n' {
			l.line++
			l.pos++
			continue
		}
		if unicode.IsSpace(r) {
			l.pos++
			continue
		}
		if r == '#' {
			for l.pos < len(l.input) && l.input[l.pos] != '\n' {
				l.pos++
			}
			continue
		}
		switch r {
		case '{':
			l.pos++
			return token{typ: tokOpenBrace, val: "{", line: l.line}, nil
		case '}':
			l.pos++
			return token{typ: tokCloseBrace, val: "}", line: l.line}, nil
		case '[':
			l.pos++
			return token{typ: tokOpenBracket, val: "[", line: l.line}, nil
		case ']':
			l.pos++
			return token{typ: tokCloseBracket, val: "]", line: l.line}, nil
		case ':':
			l.pos++
			return token{typ: tokColon, val: ":", line: l.line}, nil
		case ',':
			l.pos++
			return token{typ: tokComma, val: ",", line: l.line}, nil
		case '"':
			return l.readString()
		default:
			return l.readIdent()
		}
	}
	return token{typ: tokEOF, val: "", line: l.line}, nil
}

func (l *textprotoLexer) readString() (token, error) {
	startLine := l.line
	l.pos++ // skip opening '"'
	var sb strings.Builder
	for l.pos < len(l.input) {
		r := l.input[l.pos]
		if r == '"' {
			l.pos++
			return token{typ: tokString, val: sb.String(), line: startLine}, nil
		}
		if r == '\\' {
			l.pos++
			if l.pos >= len(l.input) {
				return token{}, fmt.Errorf("line %d: unexpected EOF in escape sequence", l.line)
			}
			esc := l.input[l.pos]
			l.pos++
			switch esc {
			case 'n':
				sb.WriteRune('\n')
			case 'r':
				sb.WriteRune('\r')
			case 't':
				sb.WriteRune('\t')
			case '\\':
				sb.WriteRune('\\')
			case '"':
				sb.WriteRune('"')
			default:
				sb.WriteRune(esc)
			}
			continue
		}
		if r == '\n' {
			l.line++
		}
		sb.WriteRune(r)
		l.pos++
	}
	return token{}, fmt.Errorf("line %d: unterminated string literal", startLine)
}

func (l *textprotoLexer) readIdent() (token, error) {
	startLine := l.line
	start := l.pos
	for l.pos < len(l.input) {
		r := l.input[l.pos]
		if unicode.IsSpace(r) || r == '#' || r == '{' || r == '}' || r == '[' || r == ']' || r == ':' || r == ',' || r == '"' {
			break
		}
		l.pos++
	}
	return token{typ: tokIdent, val: string(l.input[start:l.pos]), line: startLine}, nil
}

type textprotoParser struct {
	tokens []token
	idx    int
}

func newTextprotoParser(lexer *textprotoLexer) (*textprotoParser, error) {
	var tokens []token
	for {
		tok, err := lexer.nextToken()
		if err != nil {
			return nil, err
		}
		tokens = append(tokens, tok)
		if tok.typ == tokEOF {
			break
		}
	}
	return &textprotoParser{tokens: tokens}, nil
}

func (p *textprotoParser) peek() token {
	if p.idx >= len(p.tokens) {
		return token{typ: tokEOF}
	}
	return p.tokens[p.idx]
}

func (p *textprotoParser) next() token {
	tok := p.peek()
	if tok.typ != tokEOF {
		p.idx++
	}
	return tok
}

func (p *textprotoParser) match(typ tokenType) bool {
	if p.peek().typ == typ {
		p.next()
		return true
	}
	return false
}

func (p *textprotoParser) skipBlock() {
	depth := 1
	for p.peek().typ != tokEOF && depth > 0 {
		tok := p.next()
		switch tok.typ {
		case tokOpenBrace:
			depth++
		case tokCloseBrace:
			depth--
		}
	}
}

// ConvertConfig parses textproto content containing service definitions and converts
// it into a Librarian C++ configuration.
func ConvertConfig(content string) (*config.Config, error) {
	lexer := newTextprotoLexer(content)
	parser, err := newTextprotoParser(lexer)
	if err != nil {
		return nil, fmt.Errorf("tokenizing textproto: %w", err)
	}

	usedNames := make(map[string]int)
	var libraries []*config.Library

	for parser.peek().typ != tokEOF {
		tok := parser.next()
		if tok.typ != tokIdent {
			continue
		}
		switch tok.val {
		case "service":
			parser.match(tokColon)
			if !parser.match(tokOpenBrace) {
				return nil, fmt.Errorf("line %d: expected '{' after service", tok.line)
			}
			lib, err := parseServiceBlock(parser, usedNames)
			if err != nil {
				return nil, err
			}
			if lib != nil {
				libraries = append(libraries, lib)
			}
		case "discovery_products":
			parser.match(tokColon)
			if parser.match(tokOpenBrace) {
				discLibs, err := parseDiscoveryProductsBlock(parser, usedNames)
				if err != nil {
					return nil, err
				}
				libraries = append(libraries, discLibs...)
			}
		}
	}

	return &config.Config{
		Language:  config.LanguageCpp,
		Libraries: libraries,
	}, nil
}

// ConvertConfigFile reads a generator_config.textproto file and converts it to a Config.
func ConvertConfigFile(path string) (*config.Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading textproto file %s: %w", path, err)
	}
	return ConvertConfig(string(data))
}

func parseServiceBlock(p *textprotoParser, usedNames map[string]int) (*config.Library, error) {
	cppLib := &config.CppLibrary{}
	var serviceProtoPath string

	for p.peek().typ != tokEOF && p.peek().typ != tokCloseBrace {
		fieldTok := p.next()
		if fieldTok.typ != tokIdent {
			return nil, fmt.Errorf("line %d: expected field identifier, got %q", fieldTok.line, fieldTok.val)
		}
		fieldName := fieldTok.val
		p.match(tokColon)

		if p.peek().typ == tokOpenBrace {
			// Nested block
			p.next()
			p.skipBlock()
			continue
		}

		if p.peek().typ == tokOpenBracket {
			p.next()
			if err := parseListField(p, fieldName, cppLib); err != nil {
				return nil, err
			}
			continue
		}

		valTok := p.next()
		val := valTok.val

		switch fieldName {
		case "service_proto_path":
			serviceProtoPath = val
		case "product_path":
			cppLib.ProductPath = val
		case "forwarding_product_path":
			cppLib.ForwardingProductPath = val
		case "initial_copyright_year":
			cppLib.InitialCopyrightYear = val
		case "service_endpoint_env_var":
			cppLib.ServiceEndpointEnvVar = val
		case "emulator_endpoint_env_var":
			cppLib.EmulatorEndpointEnvVar = val
		case "endpoint_location_style":
			cppLib.EndpointLocationStyle = val
		case "override_service_config_yaml_name":
			cppLib.OverrideServiceConfigYamlName = val
			cppLib.ServiceConfig = val
		case "generate_rest_transport":
			cppLib.GenerateRestTransport = val == "true"
		case "generate_grpc_transport":
			b := val == "true"
			cppLib.GenerateGrpcTransport = &b
		case "generate_round_robin_decorator":
			cppLib.GenerateRoundRobinDecorator = val == "true"
		case "backwards_compatibility_namespace_alias":
			cppLib.BackwardsCompatibilityNamespace = val == "true"
		case "omit_client":
			cppLib.OmitClient = val == "true"
		case "omit_connection":
			cppLib.OmitConnection = val == "true"
		case "omit_stub_factory":
			cppLib.OmitStubFactory = val == "true"
		case "omit_streaming_updater":
			cppLib.OmitStreamingUpdater = val == "true"
		case "omit_repo_metadata":
			cppLib.OmitRepoMetadata = val == "true"
		case "experimental":
			cppLib.Experimental = val == "true"
		case "preserve_proto_field_names_in_json":
			cppLib.PreserveProtoFieldNamesInJson = val == "true"
		case "additional_proto_files":
			cppLib.AdditionalProtoFiles = append(cppLib.AdditionalProtoFiles, val)
		case "omitted_rpcs":
			cppLib.OmittedRPCs = append(cppLib.OmittedRPCs, val)
		case "gen_async_rpcs":
			cppLib.GenAsyncRPCs = append(cppLib.GenAsyncRPCs, val)
		case "omitted_services":
			cppLib.OmittedServices = append(cppLib.OmittedServices, val)
		case "retryable_status_codes":
			cppLib.RetryableStatusCodes = append(cppLib.RetryableStatusCodes, val)
		}
	}

	if !p.match(tokCloseBrace) {
		return nil, fmt.Errorf("line %d: unclosed service block", p.peek().line)
	}

	if serviceProtoPath == "" && cppLib.ProductPath == "" {
		return nil, nil
	}

	name := deriveLibraryName(cppLib.ProductPath, serviceProtoPath, usedNames)

	return &config.Library{
		Name:          name,
		CopyrightYear: cppLib.InitialCopyrightYear,
		Output:        cppLib.ProductPath,
		APIs: []*config.API{
			{Path: serviceProtoPath},
		},
		Cpp: cppLib,
	}, nil
}

func parseListField(p *textprotoParser, fieldName string, cppLib *config.CppLibrary) error {
	for p.peek().typ != tokEOF && p.peek().typ != tokCloseBracket {
		if p.peek().typ == tokComma {
			p.next()
			continue
		}
		if p.peek().typ == tokOpenBrace {
			p.next()
			override, err := parseIdempotencyOverride(p)
			if err != nil {
				return err
			}
			cppLib.IdempotencyOverrides = append(cppLib.IdempotencyOverrides, override)
			continue
		}

		tok := p.next()
		val := tok.val
		switch fieldName {
		case "omitted_rpcs":
			cppLib.OmittedRPCs = append(cppLib.OmittedRPCs, val)
		case "gen_async_rpcs":
			cppLib.GenAsyncRPCs = append(cppLib.GenAsyncRPCs, val)
		case "omitted_services":
			cppLib.OmittedServices = append(cppLib.OmittedServices, val)
		case "retryable_status_codes":
			cppLib.RetryableStatusCodes = append(cppLib.RetryableStatusCodes, val)
		case "additional_proto_files":
			cppLib.AdditionalProtoFiles = append(cppLib.AdditionalProtoFiles, val)
		}
	}

	if !p.match(tokCloseBracket) {
		return fmt.Errorf("line %d: unclosed bracket list", p.peek().line)
	}
	return nil
}

func parseIdempotencyOverride(p *textprotoParser) (config.IdempotencyRule, error) {
	var rule config.IdempotencyRule
	for p.peek().typ != tokEOF && p.peek().typ != tokCloseBrace {
		if p.peek().typ == tokComma {
			p.next()
			continue
		}
		tok := p.next()
		if tok.typ != tokIdent {
			continue
		}
		fieldName := tok.val
		p.match(tokColon)
		valTok := p.next()
		switch fieldName {
		case "rpc_name":
			rule.RPCName = valTok.val
		case "idempotency":
			rule.Idempotency = valTok.val
		}
	}
	if !p.match(tokCloseBrace) {
		return rule, fmt.Errorf("line %d: unclosed idempotency override brace", p.peek().line)
	}
	return rule, nil
}

func deriveLibraryName(productPath, protoPath string, usedNames map[string]int) string {
	base := strings.TrimSuffix(filepath.Base(protoPath), ".proto")
	baseClean := strings.ReplaceAll(base, "_", "-")
	pkg := strings.ReplaceAll(strings.Trim(productPath, "/"), "/", "-")
	if pkg == "" {
		pkg = baseClean
	}

	candidate := pkg
	if base != "service" && !strings.Contains(pkg, baseClean) {
		candidate = pkg + "-" + baseClean
	}

	count := usedNames[candidate]
	usedNames[candidate] = count + 1
	if count == 0 {
		return candidate
	}
	return fmt.Sprintf("%s-%d", candidate, count+1)
}

func parseDiscoveryProductsBlock(p *textprotoParser, usedNames map[string]int) ([]*config.Library, error) {
	var libraries []*config.Library
	for p.peek().typ != tokEOF && p.peek().typ != tokCloseBrace {
		tok := p.next()
		if tok.typ != tokIdent {
			continue
		}
		p.match(tokColon)
		if tok.val == "rest_services" {
			if !p.match(tokOpenBrace) {
				return nil, fmt.Errorf("line %d: expected '{' after rest_services", tok.line)
			}
			lib, err := parseServiceBlock(p, usedNames)
			if err != nil {
				return nil, err
			}
			if lib != nil {
				libraries = append(libraries, lib)
			}
			continue
		}
		if p.peek().typ == tokOpenBrace {
			p.next()
			p.skipBlock()
		} else if p.peek().typ == tokOpenBracket {
			p.next()
			depth := 1
			for p.peek().typ != tokEOF && depth > 0 {
				t := p.next()
				switch t.typ {
				case tokOpenBracket:
					depth++
				case tokCloseBracket:
					depth--
				}
			}
		} else {
			p.next()
		}
	}
	p.match(tokCloseBrace)
	return libraries, nil
}
