package tome

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestLoadTomeFile verifies that Tome configurations are correctly loaded
// from single and multiple Tome YAML configurations.
func TestLoadTomeFile(t *testing.T) {
	tests := []struct {
		name        string
		fileContent string
		base        Tome
		expected    []Tome
		expectError bool
	}{
		{
			name: "valid multiple tomes",
			fileContent: `
- target: "custom-target1"
  strip:
    - "custom-strip"
  include: ["custom-include"]
  values:
    key: "custom-value"
- target: "custom-target2"
  mode: 0777
  exclude: ["custom-exclude"]
  values:
    key: "custom-value2"
`,
			base: Tome{
				Target: "/tmp",
			},
			expected: []Tome{
				{
					Target:  "/tmp/custom-target1",
					Strip:   []string{"custom-strip"},
					Include: []string{"custom-include"},
					Values:  map[string]any{"key": "custom-value"},
				},
				{
					Target:  "/tmp/custom-target2",
					Mode:    0777,
					Exclude: []string{"custom-exclude"},
					Values:  map[string]any{"key": "custom-value2"},
				},
			},
		},
		{
			name: "valid single tome",
			fileContent: `
target: "custom-target"
strip:
  - "custom-strip"
include: ["custom-include"]
values:
  key: "custom-value"
`,
			base: Tome{
				Target: "/tmp",
			},
			expected: []Tome{
				{
					Target:  "/tmp/custom-target",
					Strip:   []string{"custom-strip"},
					Include: []string{"custom-include"},
					Values:  map[string]any{"key": "custom-value"},
				},
			},
		},
		{
			name: "empty tome inherits base",
			fileContent: `
- {}
`,
			base: Tome{
				Target:  "/tmp/target",
				Strip:   []string{"default-strip"},
				Exclude: []string{"default-exclude"},
				Temp:    []string{"default-temp"},
				Values:  map[string]any{"key": "value"},
			},
			expected: []Tome{
				{
					Target:  "/tmp/target/{{ .tempdir }}",
					Strip:   []string{"default-strip"},
					Exclude: []string{"default-exclude"},
					Temp:    []string{"default-temp"},
					Values:  map[string]any{"key": "value"},
				},
			},
		},
		{
			name: "invalid YAML",
			fileContent: `
- target: "custom-target
`,
			expectError: true,
		},
		{
			name: "include and exclude conflict",
			fileContent: `
- include: ["include-pattern"]
  exclude: ["exclude-pattern"]
`,
			expectError: true,
		},
		{
			name: "copy and temp conflict",
			fileContent: `
- copy: ["copy-pattern"]
  temp: ["temp-pattern"]
`,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()
			tomeFile := createTomeTestFile(t, tempDir, tt.fileContent)

			base := tt.base

			// Keep the same Source semantics as the original test.
			base.Source = filepath.Dir(tempDir)

			tomes, err := LoadTomeFile(tomeFile, &base)

			if tt.expectError {
				if err == nil {
					t.Fatal("expected an error, got nil")
				}

				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(tomes) != len(tt.expected) {
				t.Fatalf(
					"expected %d tomes, got %d",
					len(tt.expected),
					len(tomes),
				)
			}

			for i, expected := range tt.expected {
				assertTomeEqual(
					t,
					expected,
					tomes[i],
					filepath.Base(tempDir),
				)
			}
		})
	}
}

// createTomeTestFile creates a temporary Tome YAML file in the provided
// directory and returns its path.
func createTomeTestFile(
	t *testing.T,
	dir string,
	content string,
) string {
	t.Helper()

	file, err := os.CreateTemp(dir, ".tome.yaml")
	if err != nil {
		t.Fatalf("failed to create temp Tome file: %v", err)
	}

	t.Cleanup(func() {
		_ = os.Remove(file.Name())
	})

	if _, err := file.WriteString(content); err != nil {
		t.Fatalf("failed to write Tome file: %v", err)
	}

	if err := file.Close(); err != nil {
		t.Fatalf("failed to close Tome file: %v", err)
	}

	return file.Name()
}

// assertTomeEqual compares a loaded Tome against the expected configuration,
// ignoring the internal __tome__ value and resolving the tempdir placeholder.
func assertTomeEqual(
	t *testing.T,
	expected Tome,
	actual *Tome,
	tempDirName string,
) {
	t.Helper()

	expectedTarget := strings.ReplaceAll(
		expected.Target,
		"{{ .tempdir }}",
		tempDirName,
	)

	assert.Equal(t, expectedTarget, actual.Target, "Target mismatch")
	assert.Equal(t, expected.Mode, actual.Mode, "Mode mismatch")
	assert.Equal(t, expected.Strip, actual.Strip, "Strip mismatch")
	assert.Equal(t, expected.Include, actual.Include, "Include mismatch")
	assert.Equal(t, expected.Exclude, actual.Exclude, "Exclude mismatch")
	assert.Equal(t, expected.Copy, actual.Copy, "Copy mismatch")
	assert.Equal(t, expected.Temp, actual.Temp, "Temp mismatch")

	actualValues := cloneTomeValues(actual.Values)
	delete(actualValues, "__tome__")

	assert.Equal(t, expected.Values, actualValues, "Values mismatch")
}

// cloneTomeValues creates a shallow copy of a Tome values map so tests can
// remove internal values without modifying the loaded Tome.
func cloneTomeValues(values map[string]any) map[string]any {
	cloned := make(map[string]any, len(values))

	for key, value := range values {
		cloned[key] = value
	}

	return cloned
}
