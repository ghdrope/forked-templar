package tome

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSeq(t *testing.T) {
	until := 5
	expected := []int{1, 2, 3, 4, 5}

	result := seq(until)
	for i, v := range result {
		if v != expected[i] {
			t.Errorf("Expected %d at index %d, got %d", expected[i], i, v)
		}
	}
}

func TestImportContent(t *testing.T) {
	// Mock Tome struct
	tempDir := t.TempDir()
	tempFile := filepath.Join(tempDir, "test.txt")
	expectedContent := "Hello, World!"
	err := os.WriteFile(tempFile, []byte("{{ .msg }}"), 0644)
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	rd := RenderDir{
		Dir: tempDir,
		Tome: &Tome{
			Values: map[string]interface{}{
				"msg": expectedContent,
			},
		},
	}

	// Test importContent from file
	result, err := rd.importContent(tempFile)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	assert.Equal(t, expectedContent, result)
}

func TestImportContentFromURL(t *testing.T) {
	expectedContent := "Hello, World!"
	// Start a local HTTP server
	ts := os.TempDir()
	server := httpTestServer([]byte("{{ .msg }}"))
	defer server.Close()

	rd := RenderDir{
		Dir: ts,
		Tome: &Tome{
			Values: map[string]interface{}{
				"msg": expectedContent,
			},
		},
	}

	result, err := rd.importContent(server.URL)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	assert.Equal(t, expectedContent, result)
}

// httpTestServer is a helper to start a test HTTP server returning a template
func httpTestServer(content []byte) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, err := w.Write(content); err != nil {
			panic(err)
		}
	}))
}

func TestToYaml(t *testing.T) {
	type testStruct struct {
		Name  string
		Value int
	}
	input := testStruct{Name: "Test", Value: 42}
	expected := "name: Test\nvalue: 42\n"

	result, err := toYaml(input)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if result != expected {
		t.Errorf("Expected %q, got %q", expected, result)
	}
}

func TestFromYamlMap(t *testing.T) {
	yamlStr := `
key1: value1
key2: value2
`
	expected := map[any]any{
		"key1": "value1",
		"key2": "value2",
	}

	result, err := fromYaml(yamlStr)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	assert.Equal(t, expected, result)
}

func TestFromYamlArray(t *testing.T) {
	yamlStr := `
- value1
- value2
`
	expected := map[any]any{
		0: "value1",
		1: "value2",
	}

	result, err := fromYaml(yamlStr)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	assert.Equal(t, expected, result)
}

func TestToToml(t *testing.T) {
	input := map[string]any{
		"key1": "value1",
		"key2": 42,
	}
	expected := "key1 = \"value1\"\nkey2 = 42\n"

	result := toToml(input)
	assert.Equal(t, expected, result)
}

func TestFromToml(t *testing.T) {
	tomlStr := `
key1 = "value1"
key2 = 42
`
	expected := map[string]any{
		"key1": "value1",
		"key2": int64(42),
	}

	result, err := fromToml(tomlStr)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	assert.Equal(t, expected, result)
}

func TestToJson(t *testing.T) {
	input := map[string]any{
		"key1": "value1",
		"key2": 42,
	}
	expected := `{"key1":"value1","key2":42}`

	result, err := toJson(input)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	assert.JSONEq(t, expected, result)
}

func TestFromJsonMap(t *testing.T) {
	jsonStr := `{"key1":"value1","key2":42}`
	expected := map[any]any{
		"key1": "value1",
		"key2": float64(42),
	}

	result, err := fromJson(jsonStr)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	assert.Equal(t, expected, result)
}

func TestFromJsonArray(t *testing.T) {
	jsonStr := `["value1","value2"]`
	expected := map[any]any{
		0: "value1",
		1: "value2",
	}

	result, err := fromJson(jsonStr)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	assert.Equal(t, expected, result)
}

func TestRequired(t *testing.T) {
	value := "test"
	result, err := required(value)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	assert.Equal(t, value, result)

	_, err = required(nil)
	assert.Error(t, err)
	assert.Equal(t, "no value given for required parameter", err.Error())
}
