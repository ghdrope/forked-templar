package main

import (
	"templar/internal/render"

	"github.com/spf13/cobra"
)

// newRenderCommand creates the "render" subcommand.
//
// The render command defines the command-line interface and
// delegates the rendering workflow to the internal render package.
func newRenderCommand() *cobra.Command {

	opts := &render.Options{}

	cmd := &cobra.Command{
		Use:   "render <input dir/file>",
		Short: "Render a file or directory using templates",
		Args:  cobra.ExactArgs(1),

		RunE: func(cmd *cobra.Command, args []string) error {
			return render.Render(cmd.Context(), args[0], opts)
		},
	}

	// General behaviour
	cmd.Flags().BoolVarP(
		&opts.DryRun,
		"dry-run",
		"d",
		false,
		"Simulate actions without writing files",
	)

	cmd.Flags().BoolVarP(
		&opts.Verbose,
		"verbose",
		"D",
		false,
		"Enable verbose logging",
	)

	cmd.Flags().BoolVarP(
		&opts.Strict,
		"strict",
		"S",
		false,
		"Fail on missing values",
	)

	cmd.Flags().BoolVarP(
		&opts.Force,
		"force",
		"F",
		false,
		"Overwrite files in output directory without confirmation",
	)

	cmd.Flags().StringVarP(
		&opts.Mode,
		"mode",
		"m",
		"",
		"Set file mode (permissions) for created files (octal or symbolic)",
	)

	cmd.Flags().StringVarP(
		&opts.Out,
		"out",
		"o",
		"",
		"Output directory for generated files (default: standard output)",
	)

	cmd.Flags().StringSliceVarP(
		&opts.Values,
		"values",
		"v",
		[]string{},
		"Path to values YAML file (can be repeated)",
	)

	cmd.Flags().StringSliceVarP(
		&opts.SetValues,
		"set",
		"s",
		[]string{},
		"Set a value (key=value) (can be repeated)",
	)

	cmd.Flags().StringSliceVarP(
		&opts.IncludePatterns,
		"include",
		"i",
		[]string{},
		"Glob pattern of files to include (can be repeated)",
	)

	cmd.Flags().StringSliceVarP(
		&opts.ExcludePatterns,
		"exclude",
		"e",
		[]string{},
		"Glob pattern of files to exclude (can be repeated)",
	)

	cmd.Flags().StringSliceVarP(
		&opts.CopyPatterns,
		"copy",
		"c",
		[]string{},
		"Glob pattern for files to copy without templating (can be repeated)",
	)

	cmd.Flags().StringSliceVarP(
		&opts.TempPatterns,
		"temp",
		"t",
		[]string{},
		"Glob pattern for files to template; others are copied as-is (mutually exclusive with --copy)",
	)

	cmd.Flags().StringSliceVarP(
		&opts.StripSuffix,
		"strip",
		"r",
		[]string{},
		"Suffix to strip from output filenames if templated (can be repeated)",
	)

	return cmd
}
