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
	"bytes"
	"fmt"
	"maps"
	"slices"
	"strings"
)

// printer helps generate C++ code by formatting text and substituting variables.
type printer struct {
	buf  bytes.Buffer
	vars map[string]string
}

func newPrinter(vars map[string]string) *printer {
	p := &printer{
		vars: make(map[string]string),
	}
	maps.Copy(p.vars, vars)
	return p
}

// Print formats text by replacing `$var$` with the value from vars.
func (p *printer) Print(text string) {
	p.buf.WriteString(substituteVars(text, p.vars))
}

// PrintWith formats text using additional or overridden variables for this call.
func (p *printer) PrintWith(vars map[string]string, text string) {
	merged := make(map[string]string, len(p.vars)+len(vars))
	maps.Copy(merged, p.vars)
	maps.Copy(merged, vars)
	p.buf.WriteString(substituteVars(text, merged))
}

// HeaderLocalIncludes emits sorted #include "..." lines.
func (p *printer) HeaderLocalIncludes(includes []string) {
	var filtered []string
	for _, inc := range includes {
		if inc != "" {
			filtered = append(filtered, inc)
		}
	}
	slices.Sort(filtered)
	for _, inc := range filtered {
		fmt.Fprintf(&p.buf, "#include \"%s\"\n", inc)
	}
}

// CcLocalIncludes emits #include "..." lines for .cc files.
// The first include is kept first (the matching header), and the remaining includes are sorted.
func (p *printer) CcLocalIncludes(includes []string) {
	if len(includes) == 0 {
		return
	}
	var filtered []string
	for _, inc := range includes {
		if inc != "" {
			filtered = append(filtered, inc)
		}
	}
	if len(filtered) == 0 {
		return
	}
	if len(filtered) > 1 {
		slices.Sort(filtered[1:])
	}
	for _, inc := range filtered {
		fmt.Fprintf(&p.buf, "#include \"%s\"\n", inc)
	}
}

// ProtobufIncludes emits sorted #include <...pb.h> lines.
func (p *printer) ProtobufIncludes(includes []string) {
	var filtered []string
	for _, inc := range includes {
		if inc != "" {
			filtered = append(filtered, inc)
		}
	}
	slices.Sort(filtered)
	for _, inc := range filtered {
		fmt.Fprintf(&p.buf, "#include <%s>\n", inc)
	}
}

// SystemIncludes emits sorted #include <...> lines for system headers.
func (p *printer) SystemIncludes(includes []string) {
	var filtered []string
	for _, inc := range includes {
		if inc != "" {
			filtered = append(filtered, inc)
		}
	}
	slices.Sort(filtered)
	for _, inc := range filtered {
		fmt.Fprintf(&p.buf, "#include <%s>\n", inc)
	}
}

// HeaderOpenNamespaces prints the standard google::cloud::<namespace> opening.
func (p *printer) HeaderOpenNamespaces(ns string) {
	p.Print(fmt.Sprintf(`
namespace google {
namespace cloud {
namespace %s {
GOOGLE_CLOUD_CPP_INLINE_NAMESPACE_BEGIN
`, ns))
}

// HeaderOpenForwardingNamespaces prints opening for forwarding namespaces, optionally with a comment.
func (p *printer) HeaderOpenForwardingNamespaces(ns string, comment string) {
	if comment != "" {
		p.Print(fmt.Sprintf(`
namespace google {
namespace cloud {
%s
namespace %s {
GOOGLE_CLOUD_CPP_INLINE_NAMESPACE_BEGIN
`, comment, ns))
	} else {
		p.Print(fmt.Sprintf(`
namespace google {
namespace cloud {
namespace %s {
GOOGLE_CLOUD_CPP_INLINE_NAMESPACE_BEGIN
`, ns))
	}
}

// HeaderCloseNamespaces prints the standard closing for google::cloud::<namespace>.
func (p *printer) HeaderCloseNamespaces(ns string) {
	p.Print(fmt.Sprintf(`
GOOGLE_CLOUD_CPP_INLINE_NAMESPACE_END
}  // namespace %s
}  // namespace cloud
}  // namespace google
`, ns))
}

// CopyrightHeader returns the standard copyright and license header.
func (p *printer) CopyrightHeader(year string) string {
	return fmt.Sprintf(`// Copyright %s Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
`, year)
}

// String returns the generated contents.
func (p *printer) String() string {
	return p.buf.String()
}

// substituteVars replaces all $identifier$ tokens with their value from vars.
func substituteVars(text string, vars map[string]string) string {
	var out strings.Builder
	for {
		start := strings.Index(text, "$")
		if start == -1 {
			out.WriteString(text)
			break
		}
		out.WriteString(text[:start])
		rest := text[start+1:]
		if len(rest) > 0 && rest[0] == '$' {
			out.WriteByte('$')
			text = rest[1:]
			continue
		}
		varName, remainder, found := strings.Cut(rest, "$")
		if !found {
			out.WriteString("$")
			out.WriteString(rest)
			break
		}
		if val, ok := vars[varName]; ok {
			out.WriteString(val)
		} else {
			out.WriteString("$" + varName + "$")
		}
		text = remainder
	}
	return out.String()
}
