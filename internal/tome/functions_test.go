package tome

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSeq verifies that seq generates the expected integer sequences for
// supported argument combinations.
func TestSeq(t *testing.T) {
	tests := []struct {
		name   string
		params []int
		want   []int
	}{
		{
			name:   "empty",
			params: nil,
			want:   []int{},
		},
		{
			name:   "positive end",
			params: []int{5},
			want:   []int{1, 2, 3, 4, 5},
		},
		{
			name:   "descending end",
			params: []int{-2},
			want:   []int{1, 0, -1, -2},
		},
		{
			name:   "ascending range",
			params: []int{2, 5},
			want:   []int{2, 3, 4, 5},
		},
		{
			name:   "descending range",
			params: []int{5, 2},
			want:   []int{5, 4, 3, 2},
		},
		{
			name:   "ascending range with step",
			params: []int{1, 2, 5},
			want:   []int{1, 3, 5},
		},
		{
			name:   "descending range with negative step",
			params: []int{5, -2, 1},
			want:   []int{5, 3, 1},
		},
		{
			name:   "descending range with positive step",
			params: []int{5, 2, 1},
			want:   []int{},
		},
		{
			name:   "too many arguments",
			params: []int{1, 2, 3, 4},
			want:   []int{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, seq(tt.params...))
		})
	}
}

// TestImportContent verifies that local template files are read and rendered
// using the Tome values.
func TestImportContent(t *testing.T) {
	tempDir := t.TempDir()
	tempFile := filepath.Join(tempDir, "test.txt")

	err := os.WriteFile(
		tempFile,
		[]byte("{{ .msg }}"),
		0644,
	)
	require.NoError(t, err)

	rd := RenderDir{
		Dir: tempDir,
		Tome: &Tome{
			Values: map[string]any{
				"msg": "Hello, World!",
			},
		},
	}

	result, err := rd.importContent(tempFile)

	require.NoError(t, err)
	assert.Equal(t, "Hello, World!", result)
}

// TestImportContentFromRelativeFile verifies that relative import paths are
// resolved against the RenderDir directory.
func TestImportContentFromRelativeFile(t *testing.T) {
	tempDir := t.TempDir()
	tempFile := filepath.Join(tempDir, "test.txt")

	err := os.WriteFile(
		tempFile,
		[]byte("{{ .msg }}"),
		0644,
	)
	require.NoError(t, err)

	rd := RenderDir{
		Dir: tempDir,
		Tome: &Tome{
			Values: map[string]any{
				"msg": "Hello, World!",
			},
		},
	}

	result, err := rd.importContent("test.txt")

	require.NoError(t, err)
	assert.Equal(t, "Hello, World!", result)
}

// TestImportContentFromURL verifies that HTTP template content is fetched and
// rendered using the Tome values.
func TestImportContentFromURL(t *testing.T) {
	server := httpTestServer([]byte("{{ .msg }}"))
	defer server.Close()

	rd := RenderDir{
		Dir: t.TempDir(),
		Tome: &Tome{
			Values: map[string]any{
				"msg": "Hello, World!",
			},
		},
	}

	result, err := rd.importContent(server.URL)

	require.NoError(t, err)
	assert.Equal(t, "Hello, World!", result)
}

// TestImportContentFromURLFailure verifies that an HTTP error response is
// returned as an error when importing remote template content.
func TestImportContentFromURLFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(
		w http.ResponseWriter,
		_ *http.Request,
	) {
		http.Error(w, "not found", http.StatusNotFound)
	}))
	defer server.Close()

	rd := RenderDir{
		Dir:  t.TempDir(),
		Tome: &Tome{},
	}

	_, err := rd.importContent(server.URL)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "status 404")
}

// TestImportContentMissingFile verifies that importing a missing local file
// returns an error.
func TestImportContentMissingFile(t *testing.T) {
	rd := RenderDir{
		Dir:  t.TempDir(),
		Tome: &Tome{},
	}

	_, err := rd.importContent("missing.txt")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "error reading file")
}

// TestToYaml verifies that a Go value is correctly converted to YAML.
func TestToYaml(t *testing.T) {
	type testStruct struct {
		Name  string
		Value int
	}

	input := testStruct{
		Name:  "Test",
		Value: 42,
	}

	expected := "name: Test\nvalue: 42\n"

	result, err := toYaml(input)

	require.NoError(t, err)
	assert.Equal(t, expected, result)
}

// TestFromYaml verifies that a YAML object is converted into a map.
func TestFromYaml(t *testing.T) {
	input := `
key1: value1
key2: value2
`

	expected := map[any]any{
		"key1": "value1",
		"key2": "value2",
	}

	result, err := fromYaml(input)

	require.NoError(t, err)
	assert.Equal(t, expected, result)
}

// TestFromYamlArray verifies that a top-level YAML array is converted into
// a map indexed by integer positions.
func TestFromYamlArray(t *testing.T) {
	input := `
- value1
- value2
`

	expected := map[any]any{
		0: "value1",
		1: "value2",
	}

	result, err := fromYaml(input)

	require.NoError(t, err)
	assert.Equal(t, expected, result)
}

// TestFromYamlInvalid verifies that invalid YAML input returns an error.
func TestFromYamlInvalid(t *testing.T) {
	input := `
- value1
  invalid: [
`

	_, err := fromYaml(input)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "error converting from YAML")
}

// TestToToml verifies that a Go value is correctly converted to TOML.
func TestToToml(t *testing.T) {
	input := map[string]any{
		"key1": "value1",
		"key2": 42,
	}

	expected := "key1 = \"value1\"\nkey2 = 42\n"

	result := toToml(input)

	assert.Equal(t, expected, result)
}

// TestFromToml verifies that TOML input is converted into a map.
func TestFromToml(t *testing.T) {
	input := `
key1 = "value1"
key2 = 42
`

	expected := map[string]any{
		"key1": "value1",
		"key2": int64(42),
	}

	result, err := fromToml(input)

	require.NoError(t, err)
	assert.Equal(t, expected, result)
}

// TestFromTomlInvalid verifies that invalid TOML input returns an error.
func TestFromTomlInvalid(t *testing.T) {
	input := `
key1 = [
`

	_, err := fromToml(input)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "error converting from TOML")
}

// TestToJson verifies that a Go value is correctly converted to JSON.
func TestToJson(t *testing.T) {
	input := map[string]any{
		"key1": "value1",
		"key2": 42,
	}

	expected := `{"key1":"value1","key2":42}`

	result, err := toJson(input)

	require.NoError(t, err)
	assert.JSONEq(t, expected, result)
}

// TestFromJson verifies that a JSON object is converted into a map.
func TestFromJson(t *testing.T) {
	input := `{"key1":"value1","key2":42}`

	expected := map[any]any{
		"key1": "value1",
		"key2": float64(42),
	}

	result, err := fromJson(input)

	require.NoError(t, err)
	assert.Equal(t, expected, result)
}

// TestFromJsonArray verifies that a top-level JSON array is converted into
// a map indexed by integer positions.
func TestFromJsonArray(t *testing.T) {
	input := `["value1","value2"]`

	expected := map[any]any{
		0: "value1",
		1: "value2",
	}

	result, err := fromJson(input)

	require.NoError(t, err)
	assert.Equal(t, expected, result)
}

// TestFromJsonInvalid verifies that invalid JSON input returns an error.
func TestFromJsonInvalid(t *testing.T) {
	input := `{"key":`

	_, err := fromJson(input)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "error converting from JSON")
}

// TestRequired verifies that required returns a value when it is present and
// returns an error when the value is nil.
func TestRequired(t *testing.T) {
	value := "test"

	result, err := required(value)

	require.NoError(t, err)
	assert.Equal(t, value, result)

	result, err = required(nil)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(
		t,
		"no value given for required parameter",
		err.Error(),
	)
}

// httpTestServer creates a local HTTP test server that returns the provided
// content for every request.
func httpTestServer(content []byte) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(
		writer http.ResponseWriter,
		_ *http.Request,
	) {
		if _, err := writer.Write(content); err != nil {
			panic(err)
		}
	}))
}
