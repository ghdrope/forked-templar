package tome

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Render processes the input path according to the Tome configuration.
//
// Files are either copied or templated depending on the configured patterns.
// Directories are traversed recursively, and .tome.yaml files are used to
// create and apply sub-Tomes when present.
//
// Dry-run mode reports the actions without modifying the file system.
// Force mode allows existing output files to be overwritten without prompting.
func (t *Tome) Render(inputPath string) error {
	if filepath.Base(inputPath) == ".tome.yaml" {
		return nil
	}

	if !t.ShouldInclude(inputPath) {
		if t.Verbose {
			fmt.Println("[templar] Skipping:", filepath.Base(inputPath))
		}
		return nil
	}

	info, err := os.Lstat(inputPath)
	if err != nil {
		return fmt.Errorf("failed to stat %s: %w", inputPath, err)
	}

	outputPath, err := t.formatPath(inputPath)
	if err != nil {
		return fmt.Errorf("error formatting path: %w", err)
	}

	mode := info.Mode().Perm()
	if t.Mode != 0 {
		mode = t.Mode
	}

	if info.IsDir() {
		return t.renderDirectory(inputPath, outputPath, mode)
	}

	return t.renderFile(inputPath, outputPath, info, mode)
}

// renderDirectory processes a directory and recursively renders its contents.
//
// If the directory contains a .tome.yaml file, the file is used to create
// sub-Tomes that control how the directory contents are rendered.
func (t *Tome) renderDirectory(inputPath, outputPath string, mode os.FileMode) error {
	entries, err := os.ReadDir(inputPath)
	if err != nil {
		return fmt.Errorf("failed to read directory: %w", err)
	}

	tomesFile := filepath.Join(inputPath, ".tome.yaml")

	if _, err := os.Stat(tomesFile); errors.Is(err, os.ErrNotExist) {
		return t.renderDirectoryWithCurrentTome(
			inputPath,
			outputPath,
			mode,
			entries,
		)
	}

	subTomes, err := LoadTomeFile(tomesFile, t)
	if err != nil {
		return fmt.Errorf("failed to load tomes from %s: %w", tomesFile, err)
	}

	return t.renderDirectoryWithSubTomes(inputPath, entries, subTomes)
}

// renderDirectoryWithCurrentTome creates the output directory and renders
// all entries using the current Tome configuration.
func (t *Tome) renderDirectoryWithCurrentTome(
	inputPath string,
	outputPath string,
	mode os.FileMode,
	entries []os.DirEntry,
) error {
	if t.Verbose {
		fmt.Printf("[templar] Creating directory %v %s\n", mode, outputPath)
	}

	if !t.DryRun {
		if err := os.MkdirAll(outputPath, mode); err != nil {
			return fmt.Errorf("error creating output directory: %w", err)
		}
	}

	for _, entry := range entries {
		entryPath := filepath.Join(inputPath, entry.Name())

		if err := t.Render(entryPath); err != nil {
			return err
		}
	}

	return nil
}

// renderDirectoryWithSubTomes renders directory entries using the configured
// sub-Tomes loaded from the directory's .tome.yaml file.
func (t *Tome) renderDirectoryWithSubTomes(
	inputPath string,
	entries []os.DirEntry,
	subTomes []*Tome,
) error {
	for _, subTome := range subTomes {
		if t.Verbose {
			b, err := json.MarshalIndent(subTome, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to marshal sub-tome: %w", err)
			}

			fmt.Printf("[templar] Tome %s\n", string(b))
			fmt.Printf(
				"[templar] Creating directory %v %s\n",
				subTome.Mode,
				subTome.Target,
			)
		}

		if !t.DryRun {
			if err := os.MkdirAll(subTome.Target, subTome.Mode); err != nil {
				return fmt.Errorf("error creating output directory: %w", err)
			}
		}

		for _, entry := range entries {
			entryPath := filepath.Join(inputPath, entry.Name())

			if err := subTome.Render(entryPath); err != nil {
				return err
			}
		}
	}

	return nil
}

// renderFile processes a single file or symbolic link.
//
// Symbolic links are recreated at the destination, while regular files are
// either copied directly or rendered as templates.
func (t *Tome) renderFile(
	inputPath string,
	outputPath string,
	info os.FileInfo,
	mode os.FileMode,
) error {
	symlink := info.Mode()&os.ModeSymlink != 0
	copyFile := t.shouldCopy(inputPath)

	t.logFileAction(inputPath, outputPath, mode, symlink, copyFile)

	if t.DryRun {
		return nil
	}

	if err := os.MkdirAll(filepath.Dir(outputPath), mode); err != nil {
		return fmt.Errorf("error creating output directory: %w", err)
	}

	if _, err := os.Stat(outputPath); !errors.Is(err, os.ErrNotExist) &&
		!t.Force &&
		!confirmOverwrite(outputPath) {
		return nil
	}

	if symlink {
		return t.renderSymlink(inputPath, outputPath)
	}

	return t.renderRegularFile(inputPath, outputPath, mode, copyFile)
}

// logFileAction logs the operation that will be performed on a file.
func (t *Tome) logFileAction(
	inputPath string,
	outputPath string,
	mode os.FileMode,
	symlink bool,
	copyFile bool,
) {
	if !t.Verbose {
		return
	}

	switch {
	case symlink:
		fmt.Printf(
			"[templar] Recreating symlink %s -> %s\n",
			inputPath,
			outputPath,
		)
	case copyFile:
		fmt.Printf(
			"[templar] Copying %s -> %v %s\n",
			inputPath,
			mode,
			outputPath,
		)
	default:
		fmt.Printf(
			"[templar] Templating %s -> %v %s\n",
			inputPath,
			mode,
			outputPath,
		)
	}
}

// renderSymlink recreates a symbolic link at the output path.
func (t *Tome) renderSymlink(inputPath, outputPath string) error {
	target, err := os.Readlink(inputPath)
	if err != nil {
		return fmt.Errorf("error reading symlink %q: %w", inputPath, err)
	}

	target, err = t.templatePath(target)
	if err != nil {
		return fmt.Errorf(
			"error formatting target for symlink %q: %w",
			inputPath,
			err,
		)
	}

	if _, err := os.Lstat(outputPath); err == nil && t.Force {
		if t.Verbose {
			fmt.Printf("[templar] Removing existing symlink %s\n", outputPath)
		}

		if err := os.Remove(outputPath); err != nil {
			return fmt.Errorf("failed to remove existing symlink: %w", err)
		}
	}

	if err := os.Symlink(target, outputPath); err != nil {
		return fmt.Errorf(
			"symlink %q -> %q at %q: %w",
			inputPath,
			target,
			outputPath,
			err,
		)
	}

	return nil
}

// renderRegularFile reads a regular file and either copies or templates
// its contents to the output path.
func (t *Tome) renderRegularFile(
	inputPath string,
	outputPath string,
	mode os.FileMode,
	copyFile bool,
) error {
	content, err := os.ReadFile(inputPath)
	if err != nil {
		return fmt.Errorf("error reading input file: %w", err)
	}

	if copyFile {
		if err := os.WriteFile(outputPath, content, mode); err != nil {
			return fmt.Errorf("error writing output file: %w", err)
		}
	} else {
		if err := t.renderTemplateFile(
			outputPath,
			inputPath,
			content,
		); err != nil {
			return err
		}
	}

	if err := os.Chmod(outputPath, mode); err != nil {
		return fmt.Errorf("error setting file permissions: %w", err)
	}

	return nil
}

// renderTemplateFile renders a file to a temporary output file and atomically
// replaces the destination when rendering succeeds.
func (t *Tome) renderTemplateFile(
	outputPath string,
	inputPath string,
	content []byte,
) error {
	tempPath := outputPath + ".tmp"

	outFile, err := os.Create(tempPath)
	if err != nil {
		return fmt.Errorf("error creating output file: %w", err)
	}

	err = t.Template(outFile, string(content), inputPath)

	closeErr := outFile.Close()
	if closeErr != nil && err == nil {
		err = fmt.Errorf("error closing output file: %w", closeErr)
	}

	if err != nil {
		_ = os.Remove(tempPath)
		return fmt.Errorf("error templating contents: %w", err)
	}

	if err := os.Rename(tempPath, outputPath); err != nil {
		_ = os.Remove(tempPath)
		return fmt.Errorf("error replacing output file: %w", err)
	}

	return nil
}

// confirmOverwrite asks the user whether an existing output file should
// be overwritten.
func confirmOverwrite(path string) bool {
	fmt.Printf(
		"[templar] ⚠️  '%s' already exists. Overwrite? [y/N]: ",
		path,
	)

	reader := bufio.NewReader(os.Stdin)

	answer, _ := reader.ReadString('\n')
	answer = strings.ToLower(strings.TrimSpace(answer))

	return answer == "y" || answer == "yes"
}
