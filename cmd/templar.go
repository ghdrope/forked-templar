package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"templar/internal/options"
	"templar/internal/tome"
	"templar/internal/values"

	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

// newTemplarCommand creates the "render" subcommand.
//
// The render command is responsible for loading configuration,
// creating the base tome and rendering either a single file
// or an entire directory.
func newTemplarCommand() *cobra.Command {

	cmd := &cobra.Command{
		Use:   "render <input dir/file>",
		Short: "Render a file or directory using templates",
		Args:  cobra.ExactArgs(1),

		RunE: func(cmd *cobra.Command, args []string) error {

			logger := zap.L()

			input := strings.TrimSpace(args[0])

			logger.Info(
				"Starting Templar",
				zap.String("input", input),
			)

			// Load values.
			//
			// These options are currently still provided by the
			// existing options package.
			vals, err := values.LoadAndMerge(
				options.Values,
				options.SetValues,
			)
			if err != nil {
				return fmt.Errorf("failed to load values: %w", err)
			}

			// Validate input path.
			info, err := os.Stat(input)
			if err != nil {
				return fmt.Errorf("failed to access input path: %w", err)
			}

			// Create base tome.
			baseTome, err := tome.New(
				input,
				strings.TrimSpace(options.Out),
				options.Mode,
				options.StripSuffix,
				options.IncludePatterns,
				options.ExcludePatterns,
				options.CopyPatterns,
				options.TempPatterns,
				vals,
			)
			if err != nil {
				return fmt.Errorf("failed to create base tome: %w", err)
			}

			// Single file
			if !info.IsDir() {
				return renderFile(cmd, baseTome, input)
			}

			// Directory
			if options.Verbose {
				b, err := json.MarshalIndent(baseTome, "", " ")
				if err != nil {
					return fmt.Errorf("failed to marshal tome: %w", err)
				}

				logger.Info(
					"Tome",
					zap.String("tome", string(b)),
				)
			}

			if err := baseTome.Render(input); err != nil {
				return fmt.Errorf("error walking files: %w", err)
			}

			logger.Info("Template rendering complete")

			return nil
		},
	}

	return cmd
}

// renderFile renders a single input file.
//
// If no output file is specified, the rendered content is written
// to stdout. Otherwise, it is written to the configured output file.
func renderFile(
	cmd *cobra.Command,
	baseTome *tome.Tome,
	input string,
) error {

	content, err := os.ReadFile(input)
	if err != nil {
		return fmt.Errorf("failed to read input file: %w", err)
	}

	writer := os.Stdout

	if options.Out != "" {
		writer, err = os.Create(options.Out)
		if err != nil {
			return fmt.Errorf("failed to create output file: %w", err)
		}

		defer func() {
			if err := writer.Close(); err != nil {
				zap.L().Error(
					"failed to close output file",
					zap.Error(err),
				)
			}
		}()
	}

	if err := baseTome.Template(writer, string(content), input); err != nil {
		return fmt.Errorf("error templating file: %w", err)
	}

	return nil
}
