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
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestFormatDocLines(t *testing.T) {
	for _, test := range []struct {
		name  string
		input string
		want  []string
	}{
		{
			name:  "empty",
			input: "",
			want:  nil,
		},
		{
			name:  "single line with spaces",
			input: "Hello world   ",
			want:  []string{"Hello world"},
		},
		{
			name:  "multi-line with trailing carriage return",
			input: "Line 1\r\nLine 2 \r\nLine 3",
			want:  []string{"Line 1", "Line 2", "Line 3"},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := formatDocLines(test.input)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestFormatRSTDocLines_Empty(t *testing.T) {
	for _, test := range []struct {
		name string
		in   string
	}{
		{"empty", ""},
		{"spaces", "   \n\t  "},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := formatRSTDocLines(test.in, 72, 12)
			if got != nil {
				t.Errorf("formatRSTDocLines(%q) = %v, want nil", test.in, got)
			}
		})
	}
}

func TestFormatRSTDocLines_MarkdownToRST(t *testing.T) {
	for _, test := range []struct {
		name   string
		in     string
		width  int
		indent int
		want   []string
	}{
		{
			name:   "inline code span",
			in:     "This is `code` and that is ``already_rst``.",
			width:  72,
			indent: 12,
			want: []string{
				"This is ``code`` and that is ``already_rst``.",
			},
		},
		{
			name:   "markdown hyperlink",
			in:     "See [Google](https://google.com) for details.",
			width:  72,
			indent: 12,
			want: []string{
				"See `Google <https://google.com>`__ for details.",
			},
		},
		{
			name:   "trailing double quote",
			in:     `Some text ending with "quote"`,
			width:  72,
			indent: 12,
			want: []string{
				`Some text ending with "quote".`,
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := formatRSTDocLines(test.in, test.width, test.indent)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestFormatRSTDocLines_Wrapping(t *testing.T) {
	for _, test := range []struct {
		name   string
		in     string
		width  int
		indent int
		want   []string
	}{
		{
			name: "credentials name field",
			in: "Required. The resource name of the service account for which the credentials\n" +
				"are requested, in the following format:\n" +
				"`projects/-/serviceAccounts/{ACCOUNT_EMAIL_OR_UNIQUEID}`. The `-` wildcard\n" +
				"character is required; replacing it with a project ID is invalid.",
			width:  72,
			indent: 12,
			want: []string{
				"Required. The resource name of the service account for which",
				"the credentials are requested, in the following format:",
				"``projects/-/serviceAccounts/{ACCOUNT_EMAIL_OR_UNIQUEID}``.",
				"The ``-`` wildcard character is required; replacing it with",
				"a project ID is invalid.",
			},
		},
		{
			name: "list items",
			in: "Currently supported values:\n\n" +
				"- REDIS_3_2 for 3.2\n" +
				"- REDIS_4_0 for 4.0",
			width:  72,
			indent: 12,
			want: []string{
				"Currently supported values:",
				"",
				"- REDIS_3_2 for 3.2",
				"- REDIS_4_0 for 4.0",
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := formatRSTDocLines(test.in, test.width, test.indent)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestTokenizeDocChunks(t *testing.T) {
	for _, test := range []struct {
		name  string
		input string
		want  []docChunk
	}{
		{
			name:  "empty",
			input: "",
			want:  nil,
		},
		{
			name:  "single word",
			input: "hello",
			want: []docChunk{
				{Text: "hello", IsSpace: false},
			},
		},
		{
			name:  "words with multiple spaces",
			input: "hello   world",
			want: []docChunk{
				{Text: "hello", IsSpace: false},
				{Text: "   ", IsSpace: true},
				{Text: "world", IsSpace: false},
			},
		},
		{
			name:  "non-breaking space is non-space",
			input: "hello\x01world",
			want: []docChunk{
				{Text: "hello\x01world", IsSpace: false},
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := tokenizeDocChunks(test.input)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestWrapDocChunks(t *testing.T) {
	chunks := tokenizeDocChunks("one two three four five")
	got := wrapDocChunks(chunks, 10, 10, 2)
	want := []string{
		"one two",
		"  three",
		"  four",
		"  five",
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}

	if gotEmpty := wrapDocChunks(nil, 10, 10, 0); gotEmpty != nil {
		t.Errorf("wrapDocChunks(nil) = %v, want nil", gotEmpty)
	}
}

func TestWrapPlainWords_PreservesInterWordSpacing(t *testing.T) {
	text := "Deletes a specific Redis instance.  Instance stops serving"
	got := wrapPlainWords(text, 53)
	want := []string{
		"Deletes a specific Redis instance.  Instance stops",
		"serving",
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestFormatMethodDocSummary(t *testing.T) {
	for _, test := range []struct {
		name       string
		methodName string
		want       methodDocSummary
	}{
		{
			name:       "short method name",
			methodName: "CreateFoo",
			want: methodDocSummary{
				Lead: "create foo",
				Wrap: false,
			},
		},
		{
			name:       "long method name wrapping to second line",
			methodName: "AnalyzeOrgPolicyGovernedAssets",
			want: methodDocSummary{
				Lead: "analyze org policy governed",
				Rest: "assets",
				Wrap: true,
			},
		},
		{
			name:       "long method name with multiple words on rest",
			methodName: "AnalyzeOrgPolicyGovernedAssetsResponse",
			want: methodDocSummary{
				Lead: "analyze org policy governed",
				Rest: "assets response",
				Wrap: true,
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := formatMethodDocSummary(test.methodName)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
