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
	"context"
	"io/fs"
	"os/exec"
	"path/filepath"

	"github.com/googleapis/librarian/internal/command"
	"github.com/googleapis/librarian/internal/config"
)

// Format formats emitted C++ header and source files in place using clang-format.
func Format(ctx context.Context, cfg *config.Config, library *config.Library) error {
	if library.Output == "" {
		return nil
	}
	var files []string
	_ = filepath.WalkDir(library.Output, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d == nil || d.IsDir() {
			return nil
		}
		ext := filepath.Ext(path)
		if ext == ".h" || ext == ".cc" {
			files = append(files, path)
		}
		return nil
	})
	if len(files) == 0 {
		return nil
	}

	formatter := "clang-format"
	if cfg != nil && cfg.Tools != nil && cfg.Tools.ClangFormat != nil && cfg.Tools.ClangFormat.Path != "" {
		formatter = cfg.Tools.ClangFormat.Path
	}
	if _, err := exec.LookPath(formatter); err != nil {
		return nil
	}
	args := append([]string{"-i"}, files...)
	return command.Run(ctx, formatter, args...)
}
