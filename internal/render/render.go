package render

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"templar/internal/tome"
	"templar/internal/values"

	"go.uber.org/zap"
)

// Render renders the provided input file or directory.
func Render(ctx context.Context, input string, opts *Options) error {

	logger := zap.L()

	input = strings.TrimSpace(input)

	logger.Info(
		"Starting Templar",
		zap.String("input", input),
	)

	vals, err := values.LoadAndMerge(
		opts.Values,
		opts.SetValues,
	)
	if err != nil {
		return fmt.Errorf("failed to load values: %w", err)
	}

	// Validate input path
	info, err := os.Stat(input)
	if err != nil {
		return fmt.Errorf("failed to access input path: %w", err)
	}

	// Create base tome
	baseTome, err := tome.New(
		input,
		strings.TrimSpace(opts.Out),
		opts.Mode,
		opts.StripSuffix,
		opts.IncludePatterns,
		opts.ExcludePatterns,
		opts.CopyPatterns,
		opts.TempPatterns,
		vals,
	)
	if err != nil {
		return fmt.Errorf("failed to create base tome: %w", err)
	}

	baseTome.Strict = opts.Strict

	if opts.Verbose {
		b, err := json.MarshalIndent(baseTome, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal tome: %w", err)
		}

		logger.Info(
			"Tome",
			zap.String("tome", string(b)),
		)
	}

	if !info.IsDir() {
		return renderFile(baseTome, input, opts)
	}

	if err := baseTome.Render(input); err != nil {
		return fmt.Errorf("error walking files: %w", err)
	}

	logger.Info("Template rendering complete")

	return nil
}

// renderFile renders a single input file.
//
// If no output file is specified, the rendered content is written
// to stdout. Otherwise, it is written to the configured output file.
func renderFile(
	baseTome *tome.Tome,
	input string,
	opts *Options,
) error {

	content, err := os.ReadFile(input)
	if err != nil {
		return fmt.Errorf("failed to read input file: %w", err)
	}

	writer := os.Stdout

	if opts.Out != "" {
		writer, err = os.Create(opts.Out)
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
