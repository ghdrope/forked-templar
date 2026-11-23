package values

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
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

// TestLoadAndMerge_InvalidYAMLHidesContent verifies that the error for an
// invalid values file does not include its content, which may hold secrets
// after the env var substitution.
func TestLoadAndMerge_InvalidYAMLHidesContent(t *testing.T) {
	t.Setenv("TEMPLAR_TEST_SECRET", "s3cr3t")

	file := filepath.Join(t.TempDir(), "values.yaml")
	if err := os.WriteFile(file, []byte("token: ${TEMPLAR_TEST_SECRET}\nlist: [unclosed\n"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	_, err := LoadAndMerge([]string{file}, nil)
	if err == nil {
		t.Fatal("expected an error for invalid YAML")
	}
	if strings.Contains(err.Error(), "s3cr3t") {
		t.Errorf("error leaks the file content: %v", err)
	}
}

// TestParseYAMLValue verifies how --set values are converted.
func TestParseYAMLValue(t *testing.T) {
	tests := []struct {
		in   string
		want any
	}{
		{"true", true},
		{"True", true},
		{"false", false},
		{"False", false},
		{"null", nil},
		{"~", nil},
		{"0", 0},
		{"42", 42},
		{"-3", -3},
		{"007", "007"},
		{"0.5", 0.5},
		{"1.5", 1.5},
		{"inf", "inf"},
		{"NaN", "NaN"},
		{"{a, b}", []string{"a", "b"}},
		{"hello", "hello"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			if got := parseYAMLValue(tt.in); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseYAMLValue(%q) = %#v; want %#v", tt.in, got, tt.want)
			}
		})
	}
}

// TestSubstituteEnvVars verifies the env var substitution, the ${{VAR}}
// escape, and that other "}}" (e.g. Go templates) are left alone.
func TestSubstituteEnvVars(t *testing.T) {
	t.Setenv("TEMPLAR_TEST_FOO", "bar")
	t.Setenv("TEMPLAR_TEST_ZERO", "0")
	t.Setenv("TEMPLAR_TEST_PAD", "007")

	tests := []struct {
		name string
		in   string
		want string
	}{
		{"variable", "a: ${TEMPLAR_TEST_FOO}", "a: bar"},
		{"escaped", "a: ${{TEMPLAR_TEST_FOO}}", "a: ${TEMPLAR_TEST_FOO}"},
		{"go template", `a: "{{ .x }}"`, `a: "{{ .x }}"`},
		{"go template and variable", `a: "{{ .x }} ${TEMPLAR_TEST_FOO}"`, `a: "{{ .x }} bar"`},
		{"zero", "a: ${TEMPLAR_TEST_ZERO}", "a: 0"},
		{"leading zero is quoted", "a: ${TEMPLAR_TEST_PAD}", `a: "007"`},
		{"unset", "a: ${TEMPLAR_TEST_UNSET}", "a: "},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := SubstituteEnvVars(tt.in); got != tt.want {
				t.Errorf("SubstituteEnvVars(%q) = %q; want %q", tt.in, got, tt.want)
			}
		})
	}
}

// TestCloneMap verifies that the clone is deep: changing nested maps in the
// clone does not change the original.
func TestCloneMap(t *testing.T) {
	if CloneMap(nil) != nil {
		t.Error("CloneMap(nil) should be nil")
	}

	orig := map[string]any{"app": map[string]any{"env": "prod"}, "n": 1}
	clone := CloneMap(orig)
	clone["app"].(map[string]any)["env"] = "dev"
	clone["n"] = 2

	want := map[string]any{"app": map[string]any{"env": "prod"}, "n": 1}
	if !reflect.DeepEqual(orig, want) {
		t.Errorf("original changed: %v", orig)
	}
}

// TestSetNestedValue verifies that a dotted key creates nested maps,
// replacing a scalar on the way.
func TestSetNestedValue(t *testing.T) {
	m := map[string]any{"app": "scalar"}
	setNestedValue(m, "app.name", "templar")

	want := map[string]any{"app": map[string]any{"name": "templar"}}
	if !reflect.DeepEqual(m, want) {
		t.Errorf("got %v; want %v", m, want)
	}
}
