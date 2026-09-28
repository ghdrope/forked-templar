package values

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

// TestLoadAndMerge_LoadFilesAndSetValues verifies that values are loaded
// from multiple YAML files, merged in order, and overridden by --set values.
func TestLoadAndMerge_LoadFilesAndSetValues(t *testing.T) {
	t.Setenv("ENV_VAR", "01914634")

	tempDir := t.TempDir()

	file1 := filepath.Join(tempDir, "test_values1.yaml")
	file2 := filepath.Join(tempDir, "test_values2.yaml")
	file3 := filepath.Join(tempDir, "test_values_env.yaml")

	if err := os.WriteFile(file1, []byte(`
app:
  name: test-app
  version: "1.0"
`), 0644); err != nil {
		t.Fatalf("failed to create test file %s: %v", file1, err)
	}

	if err := os.WriteFile(file2, []byte(`
app:
  version: "2.0"
  description: "A test application"
`), 0644); err != nil {
		t.Fatalf("failed to create test file %s: %v", file2, err)
	}

	if err := os.WriteFile(file3, []byte(`
config:
  env: ${ENV_VAR}
`), 0644); err != nil {
		t.Fatalf("failed to create test file %s: %v", file3, err)
	}

	valueFiles := []string{
		file1,
		file2,
		file3,
	}

	setValues := []string{
		"app.name=overridden-app",
		"app.newKey=newValue",
		"app.tag=01914634",
	}

	result, err := LoadAndMerge(valueFiles, setValues)
	if err != nil {
		t.Fatalf("LoadAndMerge failed: %v", err)
	}

	expected := map[string]interface{}{
		"app": map[string]interface{}{
			"name":        "overridden-app",
			"version":     "2.0",
			"description": "A test application",
			"newKey":      "newValue",
			"tag":         "01914634",
		},
		"config": map[string]interface{}{
			"env": "01914634",
		},
	}

	if !reflect.DeepEqual(result, expected) {
		t.Errorf("expected %v, got %v", expected, result)
	}
}

// TestLoadAndMerge_InvalidYAML verifies that LoadAndMerge returns an error
// when one of the YAML files contains invalid syntax.
func TestLoadAndMerge_InvalidYAML(t *testing.T) {
	tempDir := t.TempDir()
	invalidFile := filepath.Join(tempDir, "invalid.yaml")

	if err := os.WriteFile(invalidFile, []byte(`
app:
  name: test-app
  version: "1.0"
  invalid: [unclosed
`), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	_, err := LoadAndMerge([]string{invalidFile}, nil)
	if err == nil {
		t.Fatal("expected error for invalid YAML, got nil")
	}
}

// TestLoadAndMerge_MissingFile verifies that LoadAndMerge returns an error
// when a specified values file does not exist.
func TestLoadAndMerge_MissingFile(t *testing.T) {
	_, err := LoadAndMerge([]string{
		filepath.Join(t.TempDir(), "nonexistent.yaml"),
	}, nil)

	if err == nil {
		t.Fatal("expected error for missing file, got nil")
	}
}

// TestLoadAndMerge_InvalidSetFormat verifies that LoadAndMerge returns an error
// when a --set value does not use the expected key=value format.
func TestLoadAndMerge_InvalidSetFormat(t *testing.T) {
	_, err := LoadAndMerge(nil, []string{
		"invalidSetFormat",
	})

	if err == nil {
		t.Fatal("expected error for invalid --set format, got nil")
	}
}
