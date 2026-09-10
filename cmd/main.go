package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"templar/internal/options"
	"templar/internal/tome"
	"templar/internal/values"
)

const Version = "v0.1.6"

func main() {

	options.Init()

	if options.ShowVersion {
		fmt.Printf("templar %s\n", Version)
		os.Exit(0)
	}

	args := options.Args
	if options.ShowHelp || len(args) != 1 {
		fmt.Println("Usage: templar [flags] <input dir/file>")
		options.PrintDefaults()
		if len(args) < 1 {
			os.Exit(1)
		}
		os.Exit(0)
	}

	values, err := values.LoadAndMerge(options.Values, options.SetValues)
	if err != nil {
		fmt.Printf("[templar] ❌  failed to load values: %v\n", err)
		os.Exit(1)
	}

	info, err := os.Stat(args[0])
	if err != nil {
		fmt.Printf("[templar] ❌  failed to access input path: %v\n", err)
		os.Exit(1)
	}

	baseTome, err := tome.New(
		strings.Trim(args[0], " "),
		strings.Trim(options.Out, " "),
		options.Mode,
		options.StripSuffix,
		options.IncludePatterns,
		options.ExcludePatterns,
		options.CopyPatterns,
		options.TempPatterns,
		values,
	)

	if err != nil {
		fmt.Printf("[templar] ❌  failed to create base tome: %v\n", err)
		os.Exit(1)
	}

	if !info.IsDir() {
		content, err := os.ReadFile(args[0])
		if err != nil {
			fmt.Printf("[templar] ❌  failed to read input file: %v\n", err)
			os.Exit(1)
		}

		writer := os.Stdout
		if options.Out != "" {
			writer, err = os.Create(options.Out)
			if err != nil {
				fmt.Printf("[templar] ❌  failed to create output file: %v\n", err)
				os.Exit(1)
			}
		}

		err = baseTome.Template(writer, string(content), args[0])
		if options.Out != "" {
			if closeErr := writer.Close(); closeErr != nil {
				fmt.Printf("[templar] ❌  failed to close output file: %v\n", closeErr)
				os.Exit(1)
			}
		}
		if err != nil {
			fmt.Printf("[templar] ❌  error templating file: %v\n", err)
			os.Exit(1)
		}
		os.Exit(0)
	}

	if options.Verbose {
		b, _ := json.MarshalIndent(baseTome, "", "  ")
		fmt.Printf("[templar] Tome %s\n", string(b))
	}

	err = baseTome.Render(args[0])
	if err != nil {
		fmt.Printf("[templar] ❌  error walking files: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("[templar] ✅  Template rendering complete.")
	os.Exit(0)
}
