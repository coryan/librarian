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
	"strconv"
	"testing"
	"time"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/sidekick/api"
)

func TestCodec_Defaults(t *testing.T) {
	model := api.NewTestAPI(nil, nil, nil)
	c := newCodec(model, "out", nil)

	if !c.hasGrpc() {
		t.Errorf("hasGrpc() default: want true, got false")
	}
	if c.hasRest() {
		t.Errorf("hasRest() default: want false, got true")
	}
	if c.hasRoundRobin() {
		t.Errorf("hasRoundRobin() default: want false, got true")
	}
	if c.hasRetryTraits() {
		t.Errorf("hasRetryTraits() default: want false, got true")
	}
	if fwd := c.forwardingRelDir(); fwd != "" {
		t.Errorf("forwardingRelDir() default: want '', got %q", fwd)
	}
}

func TestCodec_ForwardingRelDir(t *testing.T) {
	model := api.NewTestAPI(nil, nil, nil)

	for _, test := range []struct {
		name       string
		output     string
		product    string
		fwdProduct string
		libOutput  string
		wantRelDir string
	}{
		{
			name:       "golden relative",
			output:     "generator/integration_tests/golden/v1",
			product:    "generator/integration_tests/golden/v1",
			fwdProduct: "generator/integration_tests/golden",
			wantRelDir: "..",
		},
		{
			name:       "abs output with relative config",
			output:     "/tmp/workspace/v1",
			product:    "generator/integration_tests/golden/v1",
			fwdProduct: "generator/integration_tests/golden",
			libOutput:  "generator/integration_tests/golden/v1",
			wantRelDir: "..",
		},
		{
			name:       "empty forwarding",
			output:     "v1",
			product:    "v1",
			fwdProduct: "",
			wantRelDir: "",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			lib := &config.Library{
				Output: test.libOutput,
				Cpp: &config.CppLibrary{
					ProductPath:           test.product,
					ForwardingProductPath: test.fwdProduct,
				},
			}
			c := newCodec(model, test.output, lib)
			got := c.forwardingRelDir()
			if got != test.wantRelDir {
				t.Errorf("forwardingRelDir() mismatch: want %q, got %q", test.wantRelDir, got)
			}
		})
	}
}

func TestCodec_CopyrightYear(t *testing.T) {
	model := api.NewTestAPI(nil, nil, nil)

	// 1. InitialCopyrightYear from CppLibrary takes highest precedence.
	c1 := newCodec(model, "out", &config.Library{
		CopyrightYear: "2020",
		Cpp: &config.CppLibrary{
			InitialCopyrightYear: "2022",
		},
	})
	if got := c1.copyrightYear(); got != "2022" {
		t.Errorf("copyrightYear() with InitialCopyrightYear: want 2022, got %q", got)
	}

	// 2. Falls back to Library.CopyrightYear if InitialCopyrightYear is empty.
	c2 := newCodec(model, "out", &config.Library{
		CopyrightYear: "2023",
	})
	if got := c2.copyrightYear(); got != "2023" {
		t.Errorf("copyrightYear() with Library.CopyrightYear: want 2023, got %q", got)
	}

	// 3. Defaults to current year if neither is specified.
	c3 := newCodec(model, "out", nil)
	wantYear := strconv.Itoa(time.Now().Year())
	if got := c3.copyrightYear(); got != wantYear {
		t.Errorf("copyrightYear() default: want %q, got %q", wantYear, got)
	}
}
