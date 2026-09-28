package values

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

var envVarRegexp = regexp.MustCompile(`\$\{([^}]+)\}`)

// LoadAndMerge loads values from YAML files, substitutes environment variables,
// merges the files in order, and applies values provided through --set.
func LoadAndMerge(valueFiles []string, setVals []string) (map[string]any, error) {
	final := map[string]any{}

	// Load values from files
	for _, file := range valueFiles {
		data, err := os.ReadFile(file)
		if err != nil {
			return nil, fmt.Errorf("failed to read values file %s: %w", file, err)
		}

		// Substitute environment variables before parsing
		yamlText := SubstituteEnvVars(string(data))

		var parsed map[string]any
		if err := yaml.Unmarshal([]byte(yamlText), &parsed); err != nil {
			return nil, fmt.Errorf("invalid YAML in file %s: %w :: \n%s", file, err, yamlText)
		}

		MergeMaps(final, parsed)
	}

	// Merge --set values
	for _, setVal := range setVals {
		parts := strings.SplitN(setVal, "=", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("--set must be in key=value format")
		}
		setNestedValue(final, parts[0], parts[1])
	}

	return final, nil
}

// setNestedValue sets a value in a nested map using a dot-separated key.
//
// For example, "app.name=templar" produces:
//
//	app:
//		name: templar
func setNestedValue(m map[string]any, key string, value string) {
	keys := strings.Split(key, ".")
	last := len(keys) - 1

	curr := m
	for i, k := range keys {
		if i == last {
			curr[k] = parseYAMLValue(value)
			return
		}

		if next, ok := curr[k].(map[string]any); ok {
			curr = next
		} else {
			next := make(map[string]any)
			curr[k] = next
			curr = next
		}
	}
}

// MergeMaps recursively merges src into dst.
//
// Nested maps are merged recursively, while scalar values and other types
// from src replace existing values in dst.
func MergeMaps(dst, src map[string]any) {
	for k, v := range src {
		if vMap, ok := v.(map[string]any); ok {
			if dstMap, ok := dst[k].(map[string]any); ok {
				MergeMaps(dstMap, vMap)
			} else {
				dst[k] = vMap
			}
		} else {
			dst[k] = v
		}
	}
}

// SubstituteEnvVars replaces ${VAR} expressions with the corresponding
// environment variable values.
//
// The escaped form ${{VAR}} is preserved as the literal ${VAR}.
func SubstituteEnvVars(yamlContent string) string {

	// Escape ${{VAR}} to a temporary placeholder
	yamlContent = strings.ReplaceAll(yamlContent, "${{", "__ESCAPED_VAR__START__")
	yamlContent = strings.ReplaceAll(yamlContent, "}}", "__ESCAPED_VAR__END__")

	// Substitute all ${VAR}
	yamlContent = envVarRegexp.ReplaceAllStringFunc(yamlContent, func(m string) string {
		key := envVarRegexp.FindStringSubmatch(m)[1]
		value := os.Getenv(key)

		// Check for integers
		if _, err := strconv.Atoi(value); err == nil && strings.HasPrefix(value, "0") {
			return fmt.Sprintf("\"%s\"", value)
		}

		return value
	})

	// Restore escaped ${VAR}
	yamlContent = strings.ReplaceAll(yamlContent, "__ESCAPED_VAR__START__", "${")
	yamlContent = strings.ReplaceAll(yamlContent, "__ESCAPED_VAR__END__", "}")

	return yamlContent
}

// parseYAMLValue converts a string value into an appropriate Go value.
//
// Boolean, null, integer, float, and list values are converted to their
// corresponding Go types. Values that do not match a supported type remain
// strings.
func parseYAMLValue(value string) any {
	// Check for booleans
	if value == "true" || value == "True" {
		return true
	}

	if value == "false" || value == "False" {
		return false
	}

	// Check for null
	if value == "null" || value == "~" {
		return nil
	}

	// Check for leading zeroes
	if strings.HasPrefix(value, "0") {
		return value
	}

	// Check for integers
	if intVal, err := strconv.Atoi(value); err == nil {
		return intVal
	}

	// Check for floats
	if floatVal, err := strconv.ParseFloat(value, 64); err == nil {
		return floatVal
	}

	// Check for lists
	if strings.HasPrefix(value, "{") && strings.HasSuffix(value, "}") {
		trimmed := strings.Trim(value, "{}")
		parts := strings.Split(trimmed, ",")

		for i := range parts {
			parts[i] = strings.TrimSpace(parts[i])
		}

		return parts
	}

	// Default to string
	return value
}
