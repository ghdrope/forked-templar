package tome

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestRender_File verifies that a regular template file is rendered
// to the configured output directory.
func TestRender_File(t *testing.T) {
	inputDir := t.TempDir()
	outputDir := t.TempDir()

	inputPath := filepath.Join(inputDir, "hello.txt")

	if err := os.WriteFile(
		inputPath,
		[]byte("Hello, {{.Name}}!"),
		0644,
	); err != nil {
		t.Fatalf("failed to create input file: %v", err)
	}

	tome := &Tome{
		Source: inputDir,
		Target: outputDir,
		Values: map[string]any{
			"Name": "World",
		},
	}

	if err := tome.Render(inputPath); err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	outputPath := filepath.Join(outputDir, "hello.txt")

	content, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("failed to read output file: %v", err)
	}

	if got, want := string(content), "Hello, World!"; got != want {
		t.Errorf("unexpected output: got %q, want %q", got, want)
	}
}

// TestRender_FileCopy verifies that files matching the copy configuration
// are copied without template processing.
func TestRender_FileCopy(t *testing.T) {
	inputDir := t.TempDir()
	outputDir := t.TempDir()

	inputPath := filepath.Join(inputDir, "config.txt")

	content := []byte("Hello, {{.Name}}!")

	if err := os.WriteFile(inputPath, content, 0644); err != nil {
		t.Fatalf("failed to create input file: %v", err)
	}

	tome := &Tome{
		Source: inputDir,
		Target: outputDir,
		Copy: []string{
			"*.txt",
		},
		Values: map[string]any{
			"Name": "World",
		},
	}

	if err := tome.Render(inputPath); err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	outputPath := filepath.Join(outputDir, "config.txt")

	got, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("failed to read output file: %v", err)
	}

	if string(got) != string(content) {
		t.Errorf(
			"file was templated instead of copied: got %q, want %q",
			string(got),
			string(content),
		)
	}
}

// TestRender_Directory verifies that Render recursively processes
// files contained in a directory.
func TestRender_Directory(t *testing.T) {
	inputDir := t.TempDir()
	outputDir := t.TempDir()

	nestedDir := filepath.Join(inputDir, "nested")

	if err := os.MkdirAll(nestedDir, 0755); err != nil {
		t.Fatalf("failed to create nested directory: %v", err)
	}

	files := map[string]string{
		filepath.Join(inputDir, "hello.txt"):    "Hello, {{.Name}}!",
		filepath.Join(nestedDir, "message.txt"): "Message for {{.Name}}.",
		filepath.Join(nestedDir, "static.txt"):  "Static content",
	}

	for path, content := range files {
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatalf("failed to create %s: %v", path, err)
		}
	}

	tome := &Tome{
		Source: inputDir,
		Target: outputDir,
		Values: map[string]any{
			"Name": "World",
		},
	}

	if err := tome.Render(inputDir); err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	tests := map[string]string{
		filepath.Join(outputDir, "hello.txt"):             "Hello, World!",
		filepath.Join(outputDir, "nested", "message.txt"): "Message for World.",
		filepath.Join(outputDir, "nested", "static.txt"):  "Static content",
	}

	for path, want := range tests {
		content, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("failed to read %s: %v", path, err)
		}

		if got := string(content); got != want {
			t.Errorf(
				"unexpected content in %s: got %q, want %q",
				path,
				got,
				want,
			)
		}
	}
}

// TestRender_MissingInput verifies that Render returns an error when
// the input path does not exist.
func TestRender_MissingInput(t *testing.T) {
	inputPath := filepath.Join(t.TempDir(), "does-not-exist")
	outputDir := t.TempDir()

	tome := &Tome{
		Source: inputPath,
		Target: outputDir,
	}

	err := tome.Render(inputPath)
	if err == nil {
		t.Fatal("expected error for missing input, got nil")
	}

	if !strings.Contains(err.Error(), "failed to stat") {
		t.Errorf("unexpected error: %v", err)
	}
}

// TestRender_DryRun verifies that dry-run mode does not create
// output files or directories.
func TestRender_DryRun(t *testing.T) {
	inputDir := t.TempDir()
	outputDir := filepath.Join(t.TempDir(), "output")

	inputPath := filepath.Join(inputDir, "hello.txt")

	if err := os.WriteFile(
		inputPath,
		[]byte("Hello, {{.Name}}!"),
		0644,
	); err != nil {
		t.Fatalf("failed to create input file: %v", err)
	}

	tome := &Tome{
		Source: inputDir,
		Target: outputDir,
		Values: map[string]any{
			"Name": "World",
		},
		DryRun: true,
	}

	if err := tome.Render(inputPath); err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	if _, err := os.Stat(outputDir); !os.IsNotExist(err) {
		t.Errorf("expected output directory not to exist")
	}
}

// TestRender_StrictMissingValue verifies that strict mode returns an
// error when a template contains a missing value.
func TestRender_StrictMissingValue(t *testing.T) {
	inputDir := t.TempDir()
	outputDir := t.TempDir()

	inputPath := filepath.Join(inputDir, "hello.txt")

	if err := os.WriteFile(
		inputPath,
		[]byte("Hello, {{.Missing}}!"),
		0644,
	); err != nil {
		t.Fatalf("failed to create input file: %v", err)
	}

	tome := &Tome{
		Source: inputDir,
		Target: outputDir,
		Values: map[string]any{},
		Strict: true,
	}

	err := tome.Render(inputPath)
	if err == nil {
		t.Fatal("expected strict mode error, got nil")
	}

	if !strings.Contains(
		err.Error(),
		"missing template keys not allowed in strict mode",
	) {
		t.Errorf("unexpected error: %v", err)
	}
}

// TestRender_NonStrictMissingValue verifies that a missing template value
// does not cause an error when strict mode is disabled.
func TestRender_NonStrictMissingValue(t *testing.T) {
	inputDir := t.TempDir()
	outputDir := t.TempDir()

	inputPath := filepath.Join(inputDir, "hello.txt")

	if err := os.WriteFile(
		inputPath,
		[]byte("Hello, {{.Missing}}!"),
		0644,
	); err != nil {
		t.Fatalf("failed to create input file: %v", err)
	}

	tome := &Tome{
		Source: inputDir,
		Target: outputDir,
		Values: map[string]any{},
		Strict: false,
	}

	if err := tome.Render(inputPath); err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	outputPath := filepath.Join(outputDir, "hello.txt")

	if _, err := os.Stat(outputPath); err != nil {
		t.Fatalf("expected output file to exist: %v", err)
	}
}

// TestRender_Force verifies that force mode overwrites an existing
// output file without prompting for confirmation.
func TestRender_Force(t *testing.T) {
	inputDir := t.TempDir()
	outputDir := t.TempDir()

	inputPath := filepath.Join(inputDir, "hello.txt")
	outputPath := filepath.Join(outputDir, "hello.txt")

	if err := os.MkdirAll(outputDir, 0755); err != nil {
		t.Fatalf("failed to create output directory: %v", err)
	}

	if err := os.WriteFile(
		inputPath,
		[]byte("New content"),
		0644,
	); err != nil {
		t.Fatalf("failed to create input file: %v", err)
	}

	if err := os.WriteFile(
		outputPath,
		[]byte("Old content"),
		0644,
	); err != nil {
		t.Fatalf("failed to create existing output file: %v", err)
	}

	tome := &Tome{
		Source: inputDir,
		Target: outputDir,
		Force:  true,
	}

	if err := tome.Render(inputPath); err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	content, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("failed to read output file: %v", err)
	}

	if got, want := string(content), "New content"; got != want {
		t.Errorf("unexpected output: got %q, want %q", got, want)
	}
}

// TestRender_SkipsTomeFile verifies that .tome.yaml files are ignored
// when Render is called directly on them.
func TestRender_SkipsTomeFile(t *testing.T) {
	inputDir := t.TempDir()
	outputDir := t.TempDir()

	inputPath := filepath.Join(inputDir, ".tome.yaml")

	if err := os.WriteFile(
		inputPath,
		[]byte("target: output"),
		0644,
	); err != nil {
		t.Fatalf("failed to create tome file: %v", err)
	}

	tome := &Tome{
		Source: inputDir,
		Target: outputDir,
	}

	if err := tome.Render(inputPath); err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	if _, err := os.Stat(filepath.Join(outputDir, ".tome.yaml")); !os.IsNotExist(err) {
		t.Errorf("expected .tome.yaml not to be rendered")
	}
}
