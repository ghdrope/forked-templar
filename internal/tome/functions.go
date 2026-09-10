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
	case 2:
		start := params[0]
		end := params[1]
		step := 1
		if end < start {
			step = -1
		}
		return untilStep(start, end+step, step)
	default:
		return []int{}
	}
}

type RenderDir struct {
	Dir  string
	Tome *Tome
}

func (rd *RenderDir) importContent(path string) (result string, err error) {
	var content []byte

	if u, parseErr := url.Parse(path); parseErr == nil && (u.Scheme == "http" || u.Scheme == "https") {
		// URL path
		resp, httpErr := http.Get(path)
		if httpErr != nil {
			return "", fmt.Errorf("error fetching URL %s: %w", path, httpErr)
		}
		defer func() {
			if closeErr := resp.Body.Close(); closeErr != nil && err == nil {
				result = ""
				err = fmt.Errorf("error closing response body from %s: %w", path, closeErr)
			}
		}()
		if resp.StatusCode != 200 {
			return "", fmt.Errorf("error fetching URL %s: status %s", path, resp.Status)
		}
		content, err = io.ReadAll(resp.Body)
		if err != nil {
			return "", fmt.Errorf("error reading response body from %s: %w", path, err)
		}
	} else {
		// Local file path
		if path[0] != '/' {
			path = filepath.Join(rd.Dir, path)
		}
		content, err = os.ReadFile(path)
		if err != nil {
			return "", fmt.Errorf("error reading file %s: %w", path, err)
		}
	}

	var templatedContent bytes.Buffer
	err = rd.Tome.Template(&templatedContent, string(content), path)
	if err != nil {
		return "", fmt.Errorf("error templating import: %w", err)
	}
	return templatedContent.String(), nil
}

func toYaml(v any) (string, error) {
	// Marshal the value to YAML
	data, err := yaml.Marshal(v)
	if err != nil {
		return "", fmt.Errorf("error converting to YAML: %w", err)
	}
	// Convert the byte slice to a string
	return string(data), nil
}

func fromYaml(str string) (map[any]any, error) {
	ret := map[any]any{}

	if err := yaml.Unmarshal([]byte(str), &ret); err != nil {
		a := []any{}
		if err := yaml.Unmarshal([]byte(str), &a); err != nil {
			return nil, fmt.Errorf("error converting from YAML: %w", err)
		}
		for i, v := range a {
			ret[i] = v
		}
	}
	return ret, nil
}

func toToml(v any) string {
	b := bytes.NewBuffer(nil)
	e := toml.NewEncoder(b)
	err := e.Encode(v)
	if err != nil {
		return err.Error()
	}
	return b.String()
}

func fromToml(str string) (map[string]any, error) {
	ret := make(map[string]any)
	if err := toml.Unmarshal([]byte(str), &ret); err != nil {
		return nil, fmt.Errorf("error converting from TOML: %w", err)
	}
	return ret, nil
}

func toJson(v any) (string, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return "", fmt.Errorf("error converting to JSON: %w", err)
	}
	return string(data), nil
}

func fromJson(str string) (map[any]any, error) {
	ret := map[any]any{}
	m := map[string]any{}
	if err := json.Unmarshal([]byte(str), &m); err != nil {
		a := []any{}
		if err := json.Unmarshal([]byte(str), &a); err != nil {
			return nil, fmt.Errorf("error converting from JSON: %w", err)
		}
		for i, v := range a {
			ret[i] = v
		}
	}
	for i, v := range m {
		ret[i] = v
	}
	return ret, nil
}

func required(v any) (any, error) {
	if v == nil {
		return nil, fmt.Errorf("no value given for required parameter")
	}
	return v, nil
}

func (t *Tome) funcMap(dir string) template.FuncMap {
	funcMap := sprig.TxtFuncMap()
	rd := &RenderDir{
		Dir:  dir,
		Tome: t,
	}
	funcMap["include"] = rd.importContent

	funcMap["seq"] = seq
	funcMap["toToml"] = toToml
	funcMap["fromToml"] = fromToml
	funcMap["toYaml"] = toYaml
	funcMap["fromYaml"] = fromYaml
	funcMap["toJson"] = toJson
	funcMap["fromJson"] = fromJson
	funcMap["required"] = required

	return funcMap
}
