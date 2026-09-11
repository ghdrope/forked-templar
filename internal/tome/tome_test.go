package tome

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNew verifies that New creates a Tome with the expected configuration.
func TestNew(t *testing.T) {
	values := map[string]any{
		"app": map[string]any{
			"name": "templar",
		},
	}

	tome, err := New(
		"input",
		"output",
		"0755",
		[]string{".tmpl"},
		[]string{"**/*.yaml"},
		nil,
		nil,
		nil,
		values,
	)

	require.NoError(t, err)
	require.NotNil(t, tome)

	assert.Equal(t, "input", tome.Source)
	assert.Equal(t, "output", tome.Target)
	assert.Equal(t, os.FileMode(0755), tome.Mode)
	assert.Equal(t, []string{".tmpl"}, tome.Strip)
	assert.Equal(t, []string{"**/*.yaml"}, tome.Include)
	assert.Nil(t, tome.Exclude)
	assert.Nil(t, tome.Copy)
	assert.Nil(t, tome.Temp)

	assert.Equal(t, "templar", tome.Values["app"].(map[string]any)["name"])

	_, ok := tome.Values["__tome__"]
	assert.True(t, ok, "expected __tome__ metadata to be added")
}

// TestNew_NilValues verifies that New initializes a nil values map.
func TestNew_NilValues(t *testing.T) {
	tome, err := New(
		"input",
		"output",
		"",
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
	)

	require.NoError(t, err)
	require.NotNil(t, tome)
	require.NotNil(t, tome.Values)

	_, ok := tome.Values["__tome__"]
	assert.True(t, ok, "expected __tome__ metadata to be added")
}

// TestNew_IncludeExcludeConflict verifies that include and exclude patterns
// cannot be configured at the same time.
func TestNew_IncludeExcludeConflict(t *testing.T) {
	_, err := New(
		"input",
		"output",
		"",
		nil,
		[]string{"*.txt"},
		[]string{"*.tmp"},
		nil,
		nil,
		nil,
	)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot use both include and exclude patterns")
}

// TestNew_CopyTempConflict verifies that copy-only and template-only patterns
// cannot be configured at the same time.
func TestNew_CopyTempConflict(t *testing.T) {
	_, err := New(
		"input",
		"output",
		"",
		nil,
		nil,
		nil,
		[]string{"*.txt"},
		[]string{"*.tmpl"},
		nil,
	)

	require.Error(t, err)
	assert.Contains(
		t,
		err.Error(),
		"cannot use both copy-only and template-only patterns",
	)
}

// TestNew_InvalidMode verifies that New returns an error for an invalid
// file mode.
func TestNew_InvalidMode(t *testing.T) {
	_, err := New(
		"input",
		"output",
		"invalid",
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
	)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid file mode")
}

// TestShouldInclude verifies include and exclude pattern behavior.
func TestShouldInclude(t *testing.T) {
	tests := []struct {
		name     string
		tome     Tome
		input    string
		expected bool
	}{
		{
			name: "match in include",
			tome: Tome{
				Include: []string{"*.txt"},
			},
			input:    "file.txt",
			expected: true,
		},
		{
			name: "no match in include",
			tome: Tome{
				Include: []string{"*.txt"},
			},
			input:    "file.jpg",
			expected: false,
		},
		{
			name: "match in exclude",
			tome: Tome{
				Exclude: []string{"*.txt"},
			},
			input:    "file.txt",
			expected: false,
		},
		{
			name: "no match in exclude",
			tome: Tome{
				Exclude: []string{"*.txt"},
			},
			input:    "file.jpg",
			expected: true,
		},
		{
			name:     "no include or exclude",
			tome:     Tome{},
			input:    "file.txt",
			expected: true,
		},
		{
			name: "match in nested path",
			tome: Tome{
				Include: []string{"**/*.txt"},
			},
			input:    "test/file.txt",
			expected: true,
		},
		{
			name: "no match in nested path",
			tome: Tome{
				Include: []string{"**/*.txt"},
			},
			input:    "test/file.jpg",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.tome.ShouldInclude(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestShouldCopy verifies copy and template-only pattern behavior.
func TestShouldCopy(t *testing.T) {
	tests := []struct {
		name     string
		tome     Tome
		input    string
		expected bool
	}{
		{
			name: "match in copy",
			tome: Tome{
				Copy: []string{"**/*.txt"},
			},
			input:    "test/file.txt",
			expected: true,
		},
		{
			name: "no match in copy",
			tome: Tome{
				Copy: []string{"*.txt"},
			},
			input:    "file.jpg",
			expected: false,
		},
		{
			name: "match in temp",
			tome: Tome{
				Temp: []string{"*.txt"},
			},
			input:    "file.txt",
			expected: false,
		},
		{
			name: "no match in temp",
			tome: Tome{
				Temp: []string{"*.txt"},
			},
			input:    "file.jpg",
			expected: true,
		},
		{
			name:     "no copy or temp",
			tome:     Tome{},
			input:    "file.txt",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.tome.shouldCopy(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestParseFileMode verifies that octal and symbolic file modes are parsed
// into the expected os.FileMode values.
func TestParseFileMode(t *testing.T) {
	tests := []struct {
		name string
		mode string
		want os.FileMode
	}{
		{
			name: "octal 0755",
			mode: "0755",
			want: 0755,
		},
		{
			name: "octal 0644",
			mode: "0644",
			want: 0644,
		},
		{
			name: "symbolic 755",
			mode: "rwxr-xr-x",
			want: 0755,
		},
		{
			name: "symbolic 644",
			mode: "rw-r--r--",
			want: 0644,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseFileMode(tt.mode)

			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

// TestParseFileMode_Invalid verifies that invalid file modes return errors.
func TestParseFileMode_Invalid(t *testing.T) {
	tests := []string{
		"",
		"invalid",
		"755",
		"rwxr-x",
		"rwxr-x-z",
	}

	for _, mode := range tests {
		t.Run(mode, func(t *testing.T) {
			_, err := parseFileMode(mode)
			assert.Error(t, err)
		})
	}
}

// TestFormatPath verifies that an input path is converted to the
// corresponding output path relative to the Tome source and target.
func TestFormatPath(t *testing.T) {
	inputDir := t.TempDir()
	outputDir := t.TempDir()

	inputPath := filepath.Join(inputDir, "hello.txt")

	tome := &Tome{
		Source: inputDir,
		Target: outputDir,
	}

	got, err := tome.formatPath(inputPath)

	require.NoError(t, err)

	want := filepath.Join(outputDir, "hello.txt")

	assert.Equal(t, want, got)
}

// TestFormatPath_NestedPath verifies that nested input paths preserve
// their relative directory structure in the output.
func TestFormatPath_NestedPath(t *testing.T) {
	inputDir := t.TempDir()
	outputDir := t.TempDir()

	inputPath := filepath.Join(
		inputDir,
		"templates",
		"app",
		"config.yaml",
	)

	tome := &Tome{
		Source: inputDir,
		Target: outputDir,
	}

	got, err := tome.formatPath(inputPath)

	require.NoError(t, err)

	want := filepath.Join(
		outputDir,
		"templates",
		"app",
		"config.yaml",
	)

	assert.Equal(t, want, got)
}

// TestFormatPath_Strip verifies that configured suffixes are removed
// from the generated output path.
func TestFormatPath_Strip(t *testing.T) {
	inputDir := t.TempDir()
	outputDir := t.TempDir()

	inputPath := filepath.Join(inputDir, "hello.txt.tmpl")

	tome := &Tome{
		Source: inputDir,
		Target: outputDir,
		Strip:  []string{".tmpl"},
	}

	got, err := tome.formatPath(inputPath)

	require.NoError(t, err)

	want := filepath.Join(outputDir, "hello.txt")

	assert.Equal(t, want, got)
}

// TestFormatPath_StripNestedPath verifies that suffix stripping is applied
// to path components throughout a nested path.
func TestFormatPath_StripNestedPath(t *testing.T) {
	inputDir := t.TempDir()
	outputDir := t.TempDir()

	inputPath := filepath.Join(
		inputDir,
		"config.tmpl",
		"app.tmpl",
		"settings.yaml.tmpl",
	)

	tome := &Tome{
		Source: inputDir,
		Target: outputDir,
		Strip:  []string{".tmpl"},
	}

	got, err := tome.formatPath(inputPath)

	require.NoError(t, err)

	want := filepath.Join(
		outputDir,
		"config",
		"app",
		"settings.yaml",
	)

	assert.Equal(t, want, got)
}

// TestTomeString verifies that String returns a human-readable
// representation of the Tome configuration.
func TestTomeString(t *testing.T) {
	tome := Tome{
		Source: "input",
		Target: "output",
		Mode:   0755,
		Strip:  []string{".tmpl"},
		Include: []string{
			"*.yaml",
		},
		Values: map[string]any{
			"name": "templar",
		},
	}

	result := tome.String()

	assert.Contains(t, result, "source: input")
	assert.Contains(t, result, "target: output")
	assert.Contains(t, result, "mode: 755")
	assert.Contains(t, result, "templar")
}
