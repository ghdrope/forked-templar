package render

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestRender_File renders a single template file to an output file.
func TestRender_File(t *testing.T) {
	dir := t.TempDir()

	input := filepath.Join(dir, "input.tmpl")
	output := filepath.Join(dir, "output.txt")

	err := os.WriteFile(
		input,
		[]byte("Hello, {{.Name}}!"),
		0644,
	)
	if err != nil {
		t.Fatalf("failed to create input file: %v", err)
	}

	opts := &Options{
		Out:    output,
		Strict: true,
		SetValues: []string{
			"Name=World",
		},
	}

	err = Render(context.Background(), input, opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	content, err := os.ReadFile(output)
	if err != nil {
		t.Fatalf("failed to read output file: %v", err)
	}

	if got, want := string(content), "Hello, World!"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

// TestRender_FileMissingValue_NonStrict verifies that missing values
// do not cause an error when strict mode is disabled.
func TestRender_FileMissingValue_NonStrict(t *testing.T) {
	dir := t.TempDir()

	input := filepath.Join(dir, "input.tmpl")
	output := filepath.Join(dir, "output.txt")

	err := os.WriteFile(
		input,
		[]byte("Hello, {{.Name}}!"),
		0644,
	)
	if err != nil {
		t.Fatalf("failed to create input file: %v", err)
	}

	opts := &Options{
		Out:    output,
		Strict: false,
	}

	err = Render(context.Background(), input, opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	content, err := os.ReadFile(output)
	if err != nil {
		t.Fatalf("failed to read output file: %v", err)
	}

	if !strings.Contains(string(content), "Hello,") {
		t.Errorf("expected output to contain %q, got %q", "Hello,", string(content))
	}
}

// TestRender_FileMissingValue_Strict verifies that missing values
// cause an error when strict mode is enabled.
func TestRender_FileMissingValue_Strict(t *testing.T) {
	dir := t.TempDir()

	input := filepath.Join(dir, "input.tmpl")
	output := filepath.Join(dir, "output.txt")

	err := os.WriteFile(
		input,
		[]byte("Hello, {{.MissingName}}!"),
		0644,
	)
	if err != nil {
		t.Fatalf("failed to create input file: %v", err)
	}

	opts := &Options{
		Out:    output,
		Strict: true,
	}

	err = Render(context.Background(), input, opts)
	if err == nil {
		t.Fatal("expected error for missing value in strict mode, got nil")
	}

	if !strings.Contains(err.Error(), "missing template keys not allowed in strict mode") {
		t.Errorf("unexpected error: %v", err)
	}
}

// TestRender_Directory renders a directory containing template files.
func TestRender_Directory(t *testing.T) {
	dir := t.TempDir()

	inputDir := filepath.Join(dir, "input")
	outputDir := filepath.Join(dir, "output")

	if err := os.MkdirAll(inputDir, 0755); err != nil {
		t.Fatalf("failed to create input directory: %v", err)
	}

	input := filepath.Join(inputDir, "hello.txt")

	if err := os.WriteFile(
		input,
		[]byte("Hello, {{.Name}}!"),
		0644,
	); err != nil {
		t.Fatalf("failed to create input file: %v", err)
	}

	opts := &Options{
		Out: outputDir,
		SetValues: []string{
			"Name=World",
		},
	}

	err := Render(context.Background(), inputDir, opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := filepath.Join(outputDir, "hello.txt")

	content, err := os.ReadFile(output)
	if err != nil {
		t.Fatalf("failed to read output file: %v", err)
	}

	if got, want := string(content), "Hello, World!"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

// TestRender_InputDoesNotExist verifies that Render returns an error
// when the input path does not exist.
func TestRender_InputDoesNotExist(t *testing.T) {
	input := filepath.Join(t.TempDir(), "does-not-exist")

	err := Render(
		context.Background(),
		input,
		&Options{},
	)

	if err == nil {
		t.Fatal("expected error for missing input path, got nil")
	}

	if !strings.Contains(err.Error(), "failed to access input path") {
		t.Errorf("unexpected error: %v", err)
	}
}

// TestRender_InputPathIsTrimmed verifies that leading and trailing
// whitespace in the input path is ignored.
func TestRender_InputPathIsTrimmed(t *testing.T) {
	dir := t.TempDir()

	input := filepath.Join(dir, "input.tmpl")
	output := filepath.Join(dir, "output.txt")

	if err := os.WriteFile(
		input,
		[]byte("Hello, {{.Name}}!"),
		0644,
	); err != nil {
		t.Fatalf("failed to create input file: %v", err)
	}

	opts := &Options{
		Out: output,
		SetValues: []string{
			"Name=World",
		},
	}

	err := Render(
		context.Background(),
		"  "+input+"  ",
		opts,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	content, err := os.ReadFile(output)
	if err != nil {
		t.Fatalf("failed to read output file: %v", err)
	}

	if got, want := string(content), "Hello, World!"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

// TestRender_InvalidTomeConfiguration verifies that invalid Tome
// configuration is returned as an error.
func TestRender_InvalidTomeConfiguration(t *testing.T) {
	dir := t.TempDir()

	input := filepath.Join(dir, "input.txt")

	if err := os.WriteFile(
		input,
		[]byte("hello"),
		0644,
	); err != nil {
		t.Fatalf("failed to create input file: %v", err)
	}

	opts := &Options{
		CopyPatterns: []string{"*.txt"},
		TempPatterns: []string{"*.tmpl"},
	}

	err := Render(context.Background(), input, opts)
	if err == nil {
		t.Fatal("expected error for invalid Tome configuration, got nil")
	}

	if !strings.Contains(err.Error(), "failed to create base tome") {
		t.Errorf("unexpected error: %v", err)
	}
}

// newTemplateDir creates a template directory with a single file using .Name.
func newTemplateDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "hello.txt"), []byte("Hello, {{.Name}}!"), 0644); err != nil {
		t.Fatalf("failed to create template: %v", err)
	}
	return dir
}

// TestRender_DryRun verifies that nothing is written in dry-run mode, for
// both a directory and a single file.
func TestRender_DryRun(t *testing.T) {
	t.Run("directory", func(t *testing.T) {
		out := filepath.Join(t.TempDir(), "out")

		err := Render(context.Background(), newTemplateDir(t), &Options{
			Out:       out,
			DryRun:    true,
			SetValues: []string{"Name=World"},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if _, err := os.Stat(out); !os.IsNotExist(err) {
			t.Errorf("dry-run created %s", out)
		}
	})

	t.Run("file", func(t *testing.T) {
		out := filepath.Join(t.TempDir(), "out.txt")
		input := filepath.Join(newTemplateDir(t), "hello.txt")

		if err := Render(context.Background(), input, &Options{Out: out, DryRun: true}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if _, err := os.Stat(out); !os.IsNotExist(err) {
			t.Errorf("dry-run created %s", out)
		}
	})

	t.Run("file still reports template errors", func(t *testing.T) {
		input := filepath.Join(t.TempDir(), "bad.tmpl")
		if err := os.WriteFile(input, []byte("{{ .Name "), 0644); err != nil {
			t.Fatalf("failed to create input file: %v", err)
		}

		if err := Render(context.Background(), input, &Options{DryRun: true}); err == nil {
			t.Fatal("expected a template error")
		}
	})
}

// TestRender_Force verifies that --force overwrites existing output files
// without asking.
func TestRender_Force(t *testing.T) {
	in := newTemplateDir(t)
	out := t.TempDir()

	for _, name := range []string{"First", "Second"} {
		err := Render(context.Background(), in, &Options{
			Out:       out,
			Force:     true,
			Verbose:   true,
			SetValues: []string{"Name=" + name},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	}

	content, err := os.ReadFile(filepath.Join(out, "hello.txt"))
	if err != nil {
		t.Fatalf("failed to read output file: %v", err)
	}
	if got, want := string(content), "Hello, Second!"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

// TestRender_InvalidValues verifies that errors loading values are returned.
func TestRender_InvalidValues(t *testing.T) {
	err := Render(context.Background(), newTemplateDir(t), &Options{
		Values: []string{filepath.Join(t.TempDir(), "missing.yaml")},
	})
	if err == nil || !strings.Contains(err.Error(), "failed to load values") {
		t.Errorf("unexpected error: %v", err)
	}
}

// TestRender_DirectoryError verifies that errors while rendering a
// directory are returned.
func TestRender_DirectoryError(t *testing.T) {
	in := t.TempDir()
	if err := os.WriteFile(filepath.Join(in, "bad.txt"), []byte("{{ .Name "), 0644); err != nil {
		t.Fatalf("failed to create template: %v", err)
	}

	err := Render(context.Background(), in, &Options{Out: t.TempDir()})
	if err == nil || !strings.Contains(err.Error(), "error walking files") {
		t.Errorf("unexpected error: %v", err)
	}
}

// TestRender_FileOutputError verifies that an output file that can't be
// created is reported.
func TestRender_FileOutputError(t *testing.T) {
	input := filepath.Join(newTemplateDir(t), "hello.txt")
	out := filepath.Join(t.TempDir(), "missing-dir", "out.txt")

	err := Render(context.Background(), input, &Options{Out: out})
	if err == nil || !strings.Contains(err.Error(), "failed to create output file") {
		t.Errorf("unexpected error: %v", err)
	}
}
