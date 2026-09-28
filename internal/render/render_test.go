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
