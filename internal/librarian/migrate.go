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

package librarian

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/googleapis/librarian/internal/librarian/cpp"
	"github.com/googleapis/librarian/internal/yaml"
	"github.com/urfave/cli/v3"
)

var errInputRequired = errors.New("input textproto file path is required")

// migrateCommand returns the CLI command for migrating legacy generator configs.
func migrateCommand() *cli.Command {
	return &cli.Command{
		Name:      "migrate",
		Usage:     "migrate legacy generator configurations to librarian.yaml",
		UsageText: "librarian migrate [command]",
		Commands: []*cli.Command{
			migrateCppConfigCommand(),
		},
	}
}

func migrateCppConfigCommand() *cli.Command {
	return &cli.Command{
		Name:      "cpp-config",
		Usage:     "convert google-cloud-cpp generator_config.textproto to librarian.yaml",
		UsageText: "librarian migrate cpp-config <path/to/generator_config.textproto> [flags]",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "output",
				Aliases: []string{"o"},
				Usage:   "path to write the converted librarian.yaml (defaults to stdout if not specified)",
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			inputPath := cmd.Args().First()
			if inputPath == "" {
				return errInputRequired
			}
			outputPath := cmd.String("output")
			return runMigrateCppConfig(cmd.Root().Writer, inputPath, outputPath)
		},
	}
}

func runMigrateCppConfig(w io.Writer, inputPath, outputPath string) error {
	cfg, err := cpp.ConvertConfigFile(inputPath)
	if err != nil {
		return fmt.Errorf("converting %s: %w", inputPath, err)
	}

	if outputPath != "" {
		return yaml.Write(outputPath, cfg)
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshaling YAML: %w", err)
	}
	_, err = w.Write(data)
	return err
}
