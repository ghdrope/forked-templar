package tome

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"text/template"

	"github.com/BurntSushi/toml"
	"github.com/Masterminds/sprig/v3"
	"gopkg.in/yaml.v3"
)

// seq returns a sequence of integers using the same argument conventions as
// the Sprig untilStep function.
func seq(params ...int) []int {
	untilStep := sprig.FuncMap()["untilStep"].(func(int, int, int) []int)
	increment := 1

	switch len(params) {
	case 0:
		return []int{}

	case 1:
		start := 1
		end := params[0]

		if end < start {
			increment = -1
		}

		return untilStep(start, end+increment, increment)

	case 2:
		start := params[0]
		end := params[1]
		step := 1

		if end < start {
			step = -1
		}

		return untilStep(start, end+step, step)

	case 3:
		start := params[0]
		end := params[2]
		step := params[1]

		if end < start {
			increment = -1

			if step > 0 {
				return []int{}
			}
		}

		return untilStep(start, end+increment, step)

	default:
		return []int{}
	}
}

// RenderDir contains the directory and Tome configuration used to render
// imported template content.
type RenderDir struct {
	Dir  string
	Tome *Tome
}

// importContent reads content from a local file or HTTP(S) URL, renders it
// using the current Tome configuration, and returns the rendered content.
func (rd *RenderDir) importContent(path string) (string, error) {
	content, resolvedPath, err := rd.readContent(path)
	if err != nil {
		return "", err
	}

	var templatedContent bytes.Buffer

	if err := rd.Tome.Template(
		&templatedContent,
		string(content),
		resolvedPath,
	); err != nil {
		return "", fmt.Errorf("error templating import: %w", err)
	}

	return templatedContent.String(), nil
}

// readContent reads content from a local file or HTTP(S) URL and returns the
// content together with the resolved path used for template processing.
func (rd *RenderDir) readContent(path string) ([]byte, string, error) {
	if isHTTPURL(path) {
		content, err := readHTTPContent(path)
		if err != nil {
			return nil, "", err
		}

		return content, path, nil
	}

	resolvedPath := path

	if !filepath.IsAbs(resolvedPath) {
		resolvedPath = filepath.Join(rd.Dir, resolvedPath)
	}

	content, err := os.ReadFile(resolvedPath)
	if err != nil {
		return nil, "", fmt.Errorf(
			"error reading file %s: %w",
			resolvedPath,
			err,
		)
	}

	return content, resolvedPath, nil
}

// isHTTPURL reports whether the provided path is an HTTP or HTTPS URL.
func isHTTPURL(path string) bool {
	parsedURL, err := url.Parse(path)
	if err != nil {
		return false
	}

	return parsedURL.Scheme == "http" || parsedURL.Scheme == "https"
}

// readHTTPContent fetches content from an HTTP(S) URL and returns the response
// body after validating the HTTP status code.
func readHTTPContent(path string) ([]byte, error) {
	resp, err := http.Get(path)
	if err != nil {
		return nil, fmt.Errorf("error fetching URL %s: %w", path, err)
	}

	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf(
			"error fetching URL %s: status %s",
			path,
			resp.Status,
		)
	}

	content, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf(
			"error reading response body from %s: %w",
			path,
			err,
		)
	}

	return content, nil
}

// toYaml converts a Go value to a YAML string.
func toYaml(value any) (string, error) {
	data, err := yaml.Marshal(value)
	if err != nil {
		return "", fmt.Errorf("error converting to YAML: %w", err)
	}

	return string(data), nil
}

// fromYaml converts a YAML string into a map, using numeric indexes for
// top-level arrays.
func fromYaml(text string) (map[any]any, error) {
	result := map[any]any{}

	if err := yaml.Unmarshal([]byte(text), &result); err == nil {
		return result, nil
	}

	var values []any

	if err := yaml.Unmarshal([]byte(text), &values); err != nil {
		return nil, fmt.Errorf("error converting from YAML: %w", err)
	}

	for index, value := range values {
		result[index] = value
	}

	return result, nil
}

// toToml converts a Go value to a TOML string.
func toToml(value any) string {
	buffer := bytes.NewBuffer(nil)
	encoder := toml.NewEncoder(buffer)

	if err := encoder.Encode(value); err != nil {
		return err.Error()
	}

	return buffer.String()
}

// fromToml converts a TOML string into a string-keyed map.
func fromToml(text string) (map[string]any, error) {
	result := make(map[string]any)

	if err := toml.Unmarshal([]byte(text), &result); err != nil {
		return nil, fmt.Errorf("error converting from TOML: %w", err)
	}

	return result, nil
}

// toJson converts a Go value to a JSON string.
func toJson(value any) (string, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return "", fmt.Errorf("error converting to JSON: %w", err)
	}

	return string(data), nil
}

// fromJson converts a JSON string into a map, using numeric indexes for
// top-level arrays.
func fromJson(text string) (map[any]any, error) {
	result := map[any]any{}

	var object map[string]any

	if err := json.Unmarshal([]byte(text), &object); err == nil {
		for key, value := range object {
			result[key] = value
		}

		return result, nil
	}

	var values []any

	if err := json.Unmarshal([]byte(text), &values); err != nil {
		return nil, fmt.Errorf("error converting from JSON: %w", err)
	}

	for index, value := range values {
		result[index] = value
	}

	return result, nil
}

// required returns the provided value or an error when the value is nil.
func required(value any) (any, error) {
	if value == nil {
		return nil, fmt.Errorf(
			"no value given for required parameter",
		)
	}

	return value, nil
}

// funcMap returns the template functions available to a Tome template.
func (t *Tome) funcMap(dir string) template.FuncMap {
	functions := sprig.TxtFuncMap()

	renderDir := &RenderDir{
		Dir:  dir,
		Tome: t,
	}

	functions["include"] = renderDir.importContent
	functions["seq"] = seq
	functions["toToml"] = toToml
	functions["fromToml"] = fromToml
	functions["toYaml"] = toYaml
	functions["fromYaml"] = fromYaml
	functions["toJson"] = toJson
	functions["fromJson"] = fromJson
	functions["required"] = required

	return functions
}
