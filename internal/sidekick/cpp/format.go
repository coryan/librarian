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
	"errors"
	"fmt"
	"io/fs"
	"os/exec"
	"path/filepath"

	"github.com/googleapis/librarian/internal/command"
)

// Format formats all C++ files (*.h, *.cc) within dir using clang-format.
func Format(ctx context.Context, dir string) error {
	var files []string
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		ext := filepath.Ext(path)
		if ext == ".h" || ext == ".cc" {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		return err
	}
	if len(files) == 0 {
		return nil
	}

	return FormatFiles(ctx, files...)
}

// FormatFiles formats specific C++ files using clang-format -i.
func FormatFiles(ctx context.Context, files ...string) error {
	if len(files) == 0 {
		return nil
	}
	if _, err := exec.LookPath("clang-format"); err != nil {
		return fmt.Errorf("clang-format not found on PATH: %w", err)
	}

	// Process files in batches to respect command-line argument length limits.
	const batchSize = 100
	for i := 0; i < len(files); i += batchSize {
		end := min(i+batchSize, len(files))
		batch := files[i:end]
		args := append([]string{"-i"}, batch...)
		if err := command.Run(ctx, "clang-format", args...); err != nil {
			return fmt.Errorf("clang-format failed on files %v: %w", batch, err)
		}
	}
	return nil
}
