package values

import (
	"fmt"
	"math"
	"os"
	"regexp"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

var (
	envVarRegexp     = regexp.MustCompile(`\$\{([^}]+)\}`)
	escapedVarRegexp = regexp.MustCompile(`\$\{\{([^{}]+)\}\}`)
)

const (
	escapedVarStart = "__ESCAPED_VAR__START__"
	escapedVarEnd   = "__ESCAPED_VAR__END__"
)

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
			// The content is not included in the error: after the env var
			// substitution it may contain secrets.
			return nil, fmt.Errorf("invalid YAML in file %s: %w", file, err)
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

// CloneMap returns a deep copy of m. Nested maps are copied, so merging
// into the copy never changes m.
func CloneMap(m map[string]any) map[string]any {
	if m == nil {
		return nil
	}

	clone := make(map[string]any, len(m))
	for k, v := range m {
		if nested, ok := v.(map[string]any); ok {
			clone[k] = CloneMap(nested)
		} else {
			clone[k] = v
		}
	}

	return clone
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

	// Escape ${{VAR}} to a temporary placeholder. Only the escape itself is
	// replaced, so other "}}" (e.g. Go templates in values) are left alone.
	yamlContent = escapedVarRegexp.ReplaceAllString(yamlContent, escapedVarStart+"${1}"+escapedVarEnd)

	// Substitute all ${VAR}
	yamlContent = envVarRegexp.ReplaceAllStringFunc(yamlContent, func(m string) string {
		key := envVarRegexp.FindStringSubmatch(m)[1]
		value := os.Getenv(key)

		// Keep numbers with leading zeroes (e.g. 007) as strings
		if _, err := strconv.Atoi(value); err == nil && hasLeadingZero(value) {
			return fmt.Sprintf("\"%s\"", value)
		}

		return value
	})

	// Restore escaped ${VAR}
	yamlContent = strings.ReplaceAll(yamlContent, escapedVarStart, "${")
	yamlContent = strings.ReplaceAll(yamlContent, escapedVarEnd, "}")

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

	// Keep numbers with leading zeroes (e.g. 007) as strings
	if hasLeadingZero(value) {
		return value
	}

	// Check for integers
	if intVal, err := strconv.Atoi(value); err == nil {
		return intVal
	}

	// Check for floats (ParseFloat also accepts "inf" and "nan", which
	// should stay strings)
	if floatVal, err := strconv.ParseFloat(value, 64); err == nil &&
		!math.IsInf(floatVal, 0) && !math.IsNaN(floatVal) {
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

// hasLeadingZero reports whether value starts with a zero followed by
// another digit, like 007. A plain "0" or "0.5" does not count.
func hasLeadingZero(value string) bool {
	return len(value) > 1 && value[0] == '0' && value[1] >= '0' && value[1] <= '9'
}
