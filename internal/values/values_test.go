package values

import (
	"os"
	"reflect"
	"testing"
)

func TestLoadAndMerge(t *testing.T) {
	t.Run("Load values from files and merge with --set values", func(t *testing.T) {
		// Set environment variable for substitution
		if err := os.Setenv("ENV_VAR", "01914634"); err != nil {
			t.Fatalf("failed to set environment variable: %v", err)
		}
		defer func() {
			if err := os.Unsetenv("ENV_VAR"); err != nil {
				t.Errorf("failed to unset environment variable: %v", err)
			}
		}()

		// Create temporary YAML files
		tempDir := t.TempDir()
		if err := os.Chdir(tempDir); err != nil {
			t.Fatalf("failed to change to temporary directory: %v", err)
		}

		file1 := "test_values1.yaml"
		file2 := "test_values2.yaml"
		file3 := "test_values_env.yaml"

		err := os.WriteFile(file1, []byte(`
app:
  name: test-app
  version: "1.0"
`), 0644)
		if err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		err = os.WriteFile(file2, []byte(`
app:
  version: "2.0"
  description: "A test application"
`), 0644)
		if err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		err = os.WriteFile(file3, []byte(`
config:
  env: ${ENV_VAR}
`), 0644)
		if err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		// Call LoadAndMerge
		valueFiles := []string{file1, file2, file3}
		setVals := []string{"app.name=overridden-app", "app.newKey=newValue", "app.tag=01914634"}
		result, err := LoadAndMerge(valueFiles, setVals)
		if err != nil {
			t.Fatalf("LoadAndMerge failed: %v", err)
		}

		// Expected result
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

		// Compare results
		if !reflect.DeepEqual(result, expected) {
			t.Errorf("Expected %v, got %v", expected, result)
		}
	})

	t.Run("Handle invalid YAML file", func(t *testing.T) {
		tempDir := t.TempDir()
		if err := os.Chdir(tempDir); err != nil {
			t.Fatalf("failed to change to temporary directory: %v", err)
		}
		invalidFile := "invalid.yaml"
		err := os.WriteFile(invalidFile, []byte(`
app:
  name: test-app
  version: "1.0"
  invalid: [unclosed
`), 0644)
		if err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		_, err = LoadAndMerge([]string{invalidFile}, nil)
		if err == nil {
			t.Fatal("Expected error for invalid YAML, got nil")
		}
	})

	t.Run("Handle missing file", func(t *testing.T) {
		_, err := LoadAndMerge([]string{"nonexistent.yaml"}, nil)
		if err == nil {
			t.Fatal("Expected error for missing file, got nil")
		}
	})

	t.Run("Handle invalid --set format", func(t *testing.T) {
		_, err := LoadAndMerge(nil, []string{"invalidSetFormat"})
		if err == nil {
			t.Fatal("Expected error for invalid --set format, got nil")
		}
	})
}
