package tome

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"

	"templar/internal/values"

	"gopkg.in/yaml.v3"
)

// Config represents the YAML configuration of a Tome file.
type Config struct {
	Mode    string         `yaml:"mode"`
	Target  string         `yaml:"target"`
	Strip   []string       `yaml:"strip"`
	Include []string       `yaml:"include"`
	Exclude []string       `yaml:"exclude"`
	Copy    []string       `yaml:"copy"`
	Temp    []string       `yaml:"temp"`
	Values  map[string]any `yaml:"values"`
}

// LoadTomeFile reads, templates, parses, and builds the Tome configurations
// defined by a Tome YAML file.
func LoadTomeFile(file string, base *Tome) ([]*Tome, error) {
	data, err := os.ReadFile(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read tome file: %w", err)
	}

	var templatedData bytes.Buffer

	if err := base.Template(&templatedData, string(data), file); err != nil {
		return nil, fmt.Errorf("failed to template tome file: %w", err)
	}

	configs, err := parseTomeConfigs(templatedData.Bytes())
	if err != nil {
		return nil, fmt.Errorf("invalid YAML in tome file: %w", err)
	}

	if len(configs) == 0 {
		return nil, fmt.Errorf("no tomes found in tome file")
	}

	targetDir, err := formatTomeDirectory(base, file)
	if err != nil {
		return nil, err
	}

	tomes := make([]*Tome, len(configs))

	for i, config := range configs {
		tome, err := buildTomeFromConfig(
			filepath.Dir(file),
			targetDir,
			config,
			base,
		)
		if err != nil {
			return nil, fmt.Errorf(
				"failed to create tome %d: %w",
				i+1,
				err,
			)
		}

		tomes[i] = tome
	}

	return tomes, nil
}

// parseTomeConfigs parses YAML data as either a list of Tome configurations
// or a single Tome configuration.
func parseTomeConfigs(data []byte) ([]Config, error) {
	var configs []Config

	if err := yaml.Unmarshal(data, &configs); err == nil {
		return configs, nil
	}

	var config Config

	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	return []Config{config}, nil
}

// formatTomeDirectory resolves and formats the directory containing the Tome
// configuration file using the base Tome path configuration.
func formatTomeDirectory(base *Tome, file string) (string, error) {
	dir := filepath.Dir(file)

	formattedDir, err := base.formatPath(dir)
	if err != nil {
		return "", fmt.Errorf("failed to format target path: %w", err)
	}

	return formattedDir, nil
}

// buildTomeFromConfig creates a Tome from a parsed configuration and applies
// inherited values and options from the base Tome.
func buildTomeFromConfig(
	dir string,
	formattedDir string,
	config Config,
	base *Tome,
) (*Tome, error) {
	target, err := resolveTomeTarget(formattedDir, config.Target)
	if err != nil {
		return nil, err
	}

	applyInheritedConfig(&config, base)

	mergedValues := mergeTomeValues(base.Values, config.Values)

	tome, err := New(
		dir,
		target,
		config.Mode,
		config.Strip,
		config.Include,
		config.Exclude,
		config.Copy,
		config.Temp,
		mergedValues,
	)
	if err != nil {
		return nil, err
	}

	return tome, nil
}

// resolveTomeTarget resolves a Tome target relative to the formatted
// directory of the Tome configuration file.
func resolveTomeTarget(formattedDir string, target string) (string, error) {
	if target == "" {
		return formattedDir, nil
	}

	if filepath.IsAbs(target) {
		return "", fmt.Errorf(
			"target path must be relative to the tome file directory: %s",
			target,
		)
	}

	return filepath.Join(filepath.Dir(formattedDir), target), nil
}

// applyInheritedConfig fills missing configuration fields with values inherited
// from the base Tome.
func applyInheritedConfig(config *Config, base *Tome) {
	if len(config.Strip) == 0 {
		config.Strip = base.Strip
	}

	if len(config.Include) == 0 && len(config.Exclude) == 0 {
		config.Include = base.Include
		config.Exclude = base.Exclude
	}

	if len(config.Copy) == 0 && len(config.Temp) == 0 {
		config.Copy = base.Copy
		config.Temp = base.Temp
	}
}

// mergeTomeValues creates a new values map containing base values overridden
// by values explicitly defined in the Tome configuration.
func mergeTomeValues(
	baseValues map[string]any,
	configValues map[string]any,
) map[string]any {
	mergedValues := make(map[string]any, len(baseValues))

	for key, value := range baseValues {
		mergedValues[key] = value
	}

	values.MergeMaps(mergedValues, configValues)

	return mergedValues
}
