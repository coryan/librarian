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
	"regexp"
	"strings"
	"unicode"
)

const nonBreakingSpace = "\x01"

var (
	markdownLinkRegex       = regexp.MustCompile(`\[([^\]]+)\]\(([^)]+)\)`)
	numberedListRegex       = regexp.MustCompile(`^\d+\.\s+`)
	colonNewlineRegex       = regexp.MustCompile(`:\n([^\n])`)
	hasMarkdownRegex        = regexp.MustCompile(`[|*` + "`" + `_[\]]`)
	htmlTagRegex            = regexp.MustCompile(`</?[a-zA-Z]+(?:\s+[^>]*)?>`)
	loneAsteriskRegex       = regexp.MustCompile(`([\s(])\*([\s),.?])`)
	parensUnderscoreRegex   = regexp.MustCompile(`\(_\)`)
	quotedDotStarRegex      = regexp.MustCompile(`"\.\*([a-zA-Z]+)\.\*"`)
	domainDotStarQuoteRegex = regexp.MustCompile(`\.([a-zA-Z0-9]+)\.\*"`)
	bracedSnakeRegex        = regexp.MustCompile(`(\{[^}]+\})_(\{[A-Za-z_]+\})_(\{[^}]+\})`)
)

// formatDocLines splits documentation into lines, trimming trailing whitespace.
func formatDocLines(doc string) []string {
	if doc == "" {
		return nil
	}
	lines := strings.Split(doc, "\n")
	result := make([]string, 0, len(lines))
	for _, line := range lines {
		result = append(result, strings.TrimRight(line, " \t\r"))
	}
	return result
}

// formatRSTDocLines converts documentation into a sequence of wrapped lines for Python docstrings.
// The returned lines are not indented with leading spaces, allowing Mustache templates to control
// horizontal indentation.
func formatRSTDocLines(doc string, width, indent int) []string {
	trimmedDoc := strings.TrimSpace(doc)
	if trimmedDoc == "" {
		return nil
	}

	availWidth := width - indent
	// docstringPrefixLen accounts for the r""" prefix on the first line of the Python docstring.
	const docstringPrefixLen = 3
	offset := indent + docstringPrefixLen

	var result []string
	if !hasMarkdownRegex.MatchString(doc) {
		result = wrapPythonPlain(doc, availWidth, offset, indent)
	} else {
		converted := convertMarkdownToRST(doc)
		result = wrapPythonMarkdown(converted, availWidth)
	}

	for len(result) > 0 && result[len(result)-1] == "" {
		result = result[:len(result)-1]
	}

	return result
}

func wrapPythonPlain(text string, width, offset, indent int) []string {
	if text == "" {
		return nil
	}
	text = strings.ReplaceAll(text, "\n ", "\n")

	lines := strings.Split(text, "\n")
	first := lines[0] + "\n"
	if strings.HasSuffix(first, ":\n") {
		first += "\n"
	}

	if len(first) > width-offset {
		initial := wrapPlainWords(first, width-offset)
		if strings.Contains(text, "\n") {
			remaining := strings.Join(lines[1:], "")
			if !isListItem(strings.TrimSpace(remaining)) {
				text = strings.Replace(text, "\n", " ", 1)
			}
		}
		if len(initial) > 0 {
			first = initial[0] + "\n"
		}
	}

	text = colonNewlineRegex.ReplaceAllString(text, ":\n\n$1")

	// Upstream GAPIC generator appends a period if docstring text ends in a double quote ("),
	// avoiding Python raw docstring syntax confusion before closing triple quotes.
	if len(text) < len(first) {
		trimmed := strings.TrimRight(first, "\n")
		if strings.HasSuffix(trimmed, "\"") {
			trimmed += "."
		}
		return []string{trimmed}
	}
	text = text[len(first):]
	if strings.TrimSpace(text) == "" {
		trimmed := strings.TrimRight(first, "\n")
		if strings.HasSuffix(trimmed, "\"") {
			trimmed += "."
		}
		return []string{trimmed}
	}

	newLine := ""
	if strings.HasPrefix(text, "\n") {
		newLine = "\n"
	}
	text = newLine + strings.TrimSpace(text)

	var tokens []string
	var token strings.Builder
	for line := range strings.SplitSeq(text, "\n") {
		trimmed := strings.TrimSpace(line)
		if (isListItem(trimmed) || len(line) == 0) && token.Len() > 0 {
			tokens = append(tokens, token.String())
			token.Reset()
		}
		token.WriteString(line)
		token.WriteByte('\n')
		if len(line)*4 < width*3 || strings.HasSuffix(line, ":") {
			tokens = append(tokens, token.String())
			token.Reset()
		}
	}
	if token.Len() > 0 {
		tokens = append(tokens, token.String())
	}

	var result []string
	result = append(result, strings.TrimRight(first, "\n"))

	for _, tok := range tokens {
		trimmedTok := strings.TrimRightFunc(tok, unicode.IsSpace)
		if strings.TrimSpace(trimmedTok) == "" {
			result = append(result, "")
			continue
		}
		extraIndent := 0
		if isListItem(strings.TrimSpace(trimmedTok)) {
			extraIndent = getSubsequentLineIndentationLevel(strings.TrimSpace(trimmedTok))
		}
		wrapped := fillToken(trimmedTok, width-indent, width-indent-extraIndent, extraIndent)
		result = append(result, wrapped...)
	}

	if len(result) > 0 {
		last := result[len(result)-1]
		// Append period if ending in double quote to avoid docstring syntax ambiguity before closing triple quotes.
		if strings.HasSuffix(last, "\"") {
			result[len(result)-1] = last + "."
		}
	}

	return result
}

func wrapPythonMarkdown(text string, width int) []string {
	lines := strings.Split(text, "\n")
	var result []string
	var currentBlock []string
	blockType := 0 // 0: none/paragraph, 1: top-list, 2: sub-list

	flushBlock := func() {
		if len(currentBlock) == 0 {
			return
		}
		switch blockType {
		case 1:
			result = append(result, wrapListItem(currentBlock, width)...)
		case 2:
			result = append(result, wrapSubListItem(currentBlock, width)...)
		default:
			result = append(result, wrapBlockWords(currentBlock, width)...)
		}
		currentBlock = nil
	}

	inCodeBlock := false
	for _, line := range lines {
		trimmed := strings.TrimRightFunc(line, unicode.IsSpace)
		if strings.TrimSpace(trimmed) == "```" {
			if !inCodeBlock {
				flushBlock()
				result = append(result, "", "::", "")
				inCodeBlock = true
			} else {
				inCodeBlock = false
			}
			continue
		}
		if inCodeBlock {
			result = append(result, "   "+line)
			continue
		}

		if strings.TrimSpace(trimmed) == "" {
			flushBlock()
			blockType = 0
			if len(result) > 0 && result[len(result)-1] != "" {
				result = append(result, "")
			}
			continue
		}

		if isSubListItem(trimmed) {
			if blockType != 2 {
				flushBlock()
				result = append(result, "") // blank line before sub-list
				blockType = 2
			} else {
				flushBlock()
			}
			currentBlock = append(currentBlock, trimmed)
			continue
		}

		if isTopListItem(trimmed) {
			switch blockType {
			case 2:
				flushBlock()
				result = append(result, "") // blank line after sub-list
				blockType = 1
			case 0:
				if len(currentBlock) > 0 {
					flushBlock()
					result = append(result, "") // blank line before list after paragraph
				}
				blockType = 1
			default:
				flushBlock()
			}
			currentBlock = append(currentBlock, trimmed)
			continue
		}

		if blockType == 1 || blockType == 2 {
			// Continuation line of list item
			currentBlock = append(currentBlock, trimmed)
			continue
		}

		// Paragraph
		blockType = 0
		currentBlock = append(currentBlock, trimmed)
	}
	flushBlock()

	if len(result) > 0 {
		last := result[len(result)-1]
		// Append period if ending in double quote to avoid docstring syntax ambiguity before closing triple quotes.
		if strings.HasSuffix(last, "\"") {
			result[len(result)-1] = last + "."
		}
	}

	return result
}

func convertMarkdownToRST(text string) string {
	// Strip HTML tags (e.g. <asset type>, <type>) that pandoc strips
	// during markdown parsing because it interprets them as HTML elements.
	text = htmlTagRegex.ReplaceAllString(text, "")

	// Convert list bullets * and + to -
	lines := strings.Split(text, "\n")
	for i, l := range lines {
		trimmed := strings.TrimSpace(l)
		if strings.HasPrefix(trimmed, "* ") || strings.HasPrefix(trimmed, "+ ") {
			idx := strings.Index(l, "* ")
			if idx == -1 {
				idx = strings.Index(l, "+ ")
			}
			lines[i] = l[:idx] + "- " + l[idx+2:]
		}
	}
	text = strings.Join(lines, "\n")

	// Convert markdown backtick code spans `code` to ``code`` with non-breaking spaces
	text = convertCodeSpans(text)

	// Convert markdown links [text](url) to `text <url>`__ with atomic url binding
	res := markdownLinkRegex.ReplaceAllStringFunc(text, func(m string) string {
		submatches := markdownLinkRegex.FindStringSubmatch(m)
		if len(submatches) < 3 {
			return m
		}
		label := submatches[1]
		url := submatches[2]
		words := strings.Fields(label)
		if len(words) == 0 {
			return "`<" + url + ">`__"
		}
		if len(words) == 1 {
			return "`" + words[0] + nonBreakingSpace + "<" + url + ">`__"
		}
		return "`" + strings.Join(words[:len(words)-1], " ") + " " + words[len(words)-1] + nonBreakingSpace + "<" + url + ">`__"
	})

	// Pandoc RST conversion artifacts:
	// Replace lone quoted underscore "_" with "*"
	res = strings.ReplaceAll(res, ` "_"`, ` "*"`)
	res = strings.ReplaceAll(res, `"_".`, `"*".`)

	// Escape lone * in text (e.g. wildcard lists "(such as * and ?)")
	res = loneAsteriskRegex.ReplaceAllString(res, "${1}\\*${2}")

	// Escape underscore in parentheses (_)
	res = parensUnderscoreRegex.ReplaceAllString(res, `(\_)`)

	// In pandoc RST output, inline regexes like ".*Word.*" require backslash before asterisk:
	res = quotedDotStarRegex.ReplaceAllString(res, `".\ *$1.*"`)

	// Escape .* at end of domain/path quotes (e.g. "googleapis.com.*")
	res = domainDotStarQuoteRegex.ReplaceAllString(res, `.${1}.\*"`)

	// Pandoc treats snake-case uppercase placeholders in braces as emphasis
	res = bracedSnakeRegex.ReplaceAllString(res, `${1}\ *${2}*\ ${3}`)

	return res
}

func convertCodeSpans(text string) string {
	runes := []rune(text)
	n := len(runes)
	var b strings.Builder
	i := 0
	for i < n {
		if runes[i] == '`' {
			start := i
			for i < n && runes[i] == '`' {
				i++
			}
			count := i - start
			if count == 1 {
				closeIdx := -1
				for j := i; j < n; j++ {
					if runes[j] == '\n' && j+1 < n && runes[j+1] == '\n' {
						break
					}
					if runes[j] == '`' {
						if (j+1 >= n || runes[j+1] != '`') && (j == 0 || runes[j-1] != '`') {
							closeIdx = j
							break
						}
					}
				}
				if closeIdx != -1 {
					b.WriteString("``")
					for k := i; k < closeIdx; k++ {
						if runes[k] == ' ' || runes[k] == '\n' || runes[k] == '\t' {
							if b.Len() > 0 && !strings.HasSuffix(b.String(), nonBreakingSpace) && !strings.HasSuffix(b.String(), "``") {
								b.WriteString(nonBreakingSpace)
							}
						} else {
							b.WriteRune(runes[k])
						}
					}
					b.WriteString("``")
					i = closeIdx + 1
					continue
				}
			}
			for k := start; k < i; k++ {
				b.WriteRune(runes[k])
			}
		} else {
			b.WriteRune(runes[i])
			i++
		}
	}
	return b.String()
}

func isListItem(s string) bool {
	trimmed := strings.TrimSpace(s)
	return strings.HasPrefix(trimmed, "- ") ||
		strings.HasPrefix(trimmed, "* ") ||
		strings.HasPrefix(trimmed, "+ ") ||
		numberedListRegex.MatchString(trimmed)
}

func isSubListItem(line string) bool {
	return strings.HasPrefix(line, "    - ") ||
		strings.HasPrefix(line, "    * ") ||
		strings.HasPrefix(line, "    + ") ||
		strings.HasPrefix(line, "\t- ")
}

func isTopListItem(line string) bool {
	if isSubListItem(line) {
		return false
	}
	trimmed := strings.TrimLeft(line, " \t")
	return strings.HasPrefix(trimmed, "- ") ||
		strings.HasPrefix(trimmed, "* ") ||
		strings.HasPrefix(trimmed, "+ ") ||
		numberedListRegex.MatchString(trimmed)
}

func isListMarkerOnly(s string) bool {
	trimmed := strings.TrimSpace(s)
	return trimmed == "-" ||
		trimmed == "*" ||
		trimmed == "+" ||
		numberedListRegex.MatchString(trimmed+".") ||
		numberedListRegex.MatchString(trimmed)
}

func getSubsequentLineIndentationLevel(listItem string) int {
	if len(listItem) >= 2 && (strings.HasPrefix(listItem, "- ") || strings.HasPrefix(listItem, "+ ") || strings.HasPrefix(listItem, "* ")) {
		return 2
	}
	if len(listItem) >= 4 && numberedListRegex.MatchString(listItem) {
		return 4
	}
	return 0
}

type docChunk struct {
	Text    string
	IsSpace bool
}

func tokenizeDocChunks(text string) []docChunk {
	var chunks []docChunk
	var cur strings.Builder
	inSpace := false
	for _, r := range text {
		isSpace := unicode.IsSpace(r) && r != '\x01'
		if cur.Len() > 0 && isSpace != inSpace {
			chunks = append(chunks, docChunk{Text: cur.String(), IsSpace: inSpace})
			cur.Reset()
		}
		inSpace = isSpace
		if isSpace {
			cur.WriteByte(' ')
		} else {
			cur.WriteRune(r)
		}
	}
	if cur.Len() > 0 {
		chunks = append(chunks, docChunk{Text: cur.String(), IsSpace: inSpace})
	}
	return chunks
}

func wrapDocChunks(chunks []docChunk, widthFirst, widthSubsequent, extraIndent int) []string {
	if len(chunks) == 0 {
		return nil
	}
	var lines []string
	var current strings.Builder
	indentStr := strings.Repeat(" ", extraIndent)
	targetWidth := widthFirst
	pendingSpace := ""

	for _, c := range chunks {
		if c.IsSpace {
			if current.Len() > 0 {
				pendingSpace = c.Text
			}
			continue
		}

		wClean := strings.ReplaceAll(c.Text, nonBreakingSpace, " ")
		wLen := len(wClean)

		if current.Len() == 0 {
			if len(lines) > 0 {
				current.WriteString(indentStr)
			}
			current.WriteString(wClean)
			pendingSpace = ""
		} else if current.Len()+len(pendingSpace)+wLen <= targetWidth {
			current.WriteString(pendingSpace)
			current.WriteString(wClean)
			pendingSpace = ""
		} else {
			lines = append(lines, current.String())
			current.Reset()
			targetWidth = widthSubsequent
			current.WriteString(indentStr)
			current.WriteString(wClean)
			pendingSpace = ""
		}
	}

	if current.Len() > 0 {
		lines = append(lines, current.String())
	}
	return lines
}

func wrapPlainWords(text string, width int) []string {
	chunks := tokenizeDocChunks(text)
	return wrapDocChunks(chunks, width, width, 0)
}

// methodDocSummary contains the lead and optional rest of a method docstring summary.
// For short method names, Lead contains the entire humanized name and Wrap is false.
// For longer method names that exceed the first line budget (30 chars = 70 total - 40 prefix),
// Lead contains the first line and Rest contains the subsequent wrapped line(s).
type methodDocSummary struct {
	Lead string
	Rest string
	Wrap bool
}

func formatMethodDocSummary(methodName string) methodDocSummary {
	humanized := strings.ReplaceAll(snakeCase(methodName), "_", " ")
	const (
		totalWidth         = 70
		firstLinePrefixLen = 40 // 8 spaces indent + len(`r"""Return a callable for the `)
		suffixLen          = 18 // len(" method over gRPC.")
	)
	firstLineAvail := totalWidth - firstLinePrefixLen // 30 chars
	if len(humanized) <= firstLineAvail {
		return methodDocSummary{
			Lead: humanized,
			Wrap: false,
		}
	}

	words := strings.Fields(humanized)
	var line1Words []string
	currLen := 0
	splitIdx := 0
	for i, w := range words {
		addedLen := len(w)
		if len(line1Words) > 0 {
			addedLen++ // space separator
		}
		if len(line1Words) > 0 && currLen+addedLen > firstLineAvail {
			splitIdx = i
			break
		}
		line1Words = append(line1Words, w)
		currLen += addedLen
	}
	if splitIdx == 0 && len(words) > 0 {
		line1Words = []string{words[0]}
		splitIdx = 1
	}

	lead := strings.Join(line1Words, " ")
	restWords := words[splitIdx:]
	if len(restWords) == 0 {
		return methodDocSummary{
			Lead: lead,
			Wrap: false,
		}
	}

	restLines := wrapWords(strings.Join(restWords, " "), totalWidth-8-suffixLen)
	rest := strings.Join(restLines, "\n        ")

	return methodDocSummary{
		Lead: lead,
		Rest: rest,
		Wrap: true,
	}
}

func wrapWords(text string, width int) []string {
	words := strings.Fields(text)
	if len(words) == 0 {
		return nil
	}
	var lines []string
	var current strings.Builder
	for _, w := range words {
		wClean := strings.ReplaceAll(w, nonBreakingSpace, " ")
		wLen := len(wClean)
		if current.Len() == 0 {
			current.WriteString(wClean)
		} else if current.Len()+1+wLen <= width {
			current.WriteByte(' ')
			current.WriteString(wClean)
		} else {
			lines = append(lines, current.String())
			current.Reset()
			current.WriteString(wClean)
		}
	}
	if current.Len() > 0 {
		lines = append(lines, current.String())
	}
	return lines
}

func fillToken(token string, widthFirst, widthSubsequent, extraIndent int) []string {
	words := strings.Fields(token)
	if len(words) == 0 {
		return nil
	}
	var lines []string
	var current strings.Builder
	indentStr := strings.Repeat(" ", extraIndent)

	targetWidth := widthFirst
	for _, w := range words {
		wClean := strings.ReplaceAll(w, nonBreakingSpace, " ")
		wLen := len(wClean)
		if current.Len() == 0 {
			if len(lines) > 0 {
				current.WriteString(indentStr)
			}
			current.WriteString(wClean)
		} else if current.Len()+1+wLen <= targetWidth {
			current.WriteByte(' ')
			current.WriteString(wClean)
		} else {
			lines = append(lines, current.String())
			current.Reset()
			targetWidth = widthSubsequent
			current.WriteString(indentStr)
			current.WriteString(wClean)
		}
	}
	if current.Len() > 0 {
		lines = append(lines, current.String())
	}
	return lines
}

func wrapBlockWords(lines []string, width int) []string {
	text := strings.Join(lines, " ")
	return wrapWords(text, width)
}

func wrapListItem(lines []string, width int) []string {
	text := strings.Join(lines, " ")
	markerLen := 2
	trimmed := strings.TrimSpace(text)
	if m := numberedListRegex.FindString(trimmed); m != "" {
		markerLen = len(m)
	}
	indentStr := strings.Repeat(" ", markerLen)

	words := strings.Fields(text)
	if len(words) == 0 {
		return nil
	}

	var result []string
	var current strings.Builder

	for _, w := range words {
		wClean := strings.ReplaceAll(w, nonBreakingSpace, " ")
		wLen := len(wClean)
		if current.Len() == 0 {
			if len(result) > 0 {
				current.WriteString(indentStr)
			}
			current.WriteString(wClean)
		} else if isListMarkerOnly(current.String()) {
			current.WriteByte(' ')
			current.WriteString(wClean)
		} else if current.Len()+1+wLen <= width {
			current.WriteByte(' ')
			current.WriteString(wClean)
		} else {
			result = append(result, current.String())
			current.Reset()
			current.WriteString(indentStr)
			current.WriteString(wClean)
		}
	}
	if current.Len() > 0 {
		result = append(result, current.String())
	}
	return result
}

func wrapSubListItem(lines []string, width int) []string {
	text := strings.Join(lines, " ")
	trimmed := strings.TrimSpace(text)
	marker := "- "
	if after, ok := strings.CutPrefix(trimmed, "- "); ok {
		trimmed = after
	} else if after, ok := strings.CutPrefix(trimmed, "* "); ok {
		trimmed = after
	} else if after, ok := strings.CutPrefix(trimmed, "+ "); ok {
		trimmed = after
	} else if m := numberedListRegex.FindString(trimmed); m != "" {
		marker = m
		trimmed = strings.TrimPrefix(trimmed, m)
	}

	words := strings.Fields(trimmed)
	if len(words) == 0 {
		return nil
	}

	prefix := "  " + marker
	continuationIndent := "    "

	var result []string
	var current strings.Builder
	current.WriteString(prefix)

	for i, w := range words {
		wClean := strings.ReplaceAll(w, nonBreakingSpace, " ")
		wLen := len(wClean)
		if i == 0 {
			current.WriteString(wClean)
		} else if current.Len()+1+wLen <= width {
			current.WriteByte(' ')
			current.WriteString(wClean)
		} else {
			result = append(result, current.String())
			current.Reset()
			current.WriteString(continuationIndent)
			current.WriteString(wClean)
		}
	}
	if current.Len() > 0 {
		result = append(result, current.String())
	}
	return result
}
