package tome

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
)

// Tome defines the configuration and runtime state used to render
// files and directories.
type Tome struct {
	Source  string         `json:"source"`
	Target  string         `json:"target"`
	Mode    os.FileMode    `json:"mode"`
	Strip   []string       `json:"strip"`
	Include []string       `json:"include"`
	Exclude []string       `json:"exclude"`
	Copy    []string       `json:"copy"`
	Temp    []string       `json:"temp"`
	Values  map[string]any `json:"values"`

	// Runtime options.
	Strict  bool `json:"strict"`
	DryRun  bool `json:"dryRun"`
	Verbose bool `json:"verbose"`
	Force   bool `json:"force"`
}

// String returns a human-readable representation of the Tome configuration.
func (t *Tome) String() string {
	return fmt.Sprintf(
		"{source: %s, target: %s, mode: %o, strip: %s, include: %v, exclude: %v, copy: %v, temp: %v, values: %v}",
		t.Source,
		t.Target,
		t.Mode,
		t.Strip,
		t.Include,
		t.Exclude,
		t.Copy,
		t.Temp,
		t.Values,
	)
}

// New creates a Tome from the supplied source, target, rendering patterns,
// file mode, and template values.
//
// Include and exclude patterns are mutually exclusive, as are copy and
// template-only patterns. The supplied mode may be expressed as octal
// permissions or as a symbolic nine-character permission string.
func New(
	source string,
	target string,
	mode string,
	strip []string,
	include []string,
	exclude []string,
	copyPatterns []string,
	tempPatterns []string,
	values map[string]any,
) (*Tome, error) {
	if len(include) > 0 && len(exclude) > 0 {
		return nil, fmt.Errorf("cannot use both include and exclude patterns")
	}

	if len(copyPatterns) > 0 && len(tempPatterns) > 0 {
		return nil, fmt.Errorf(
			"cannot use both copy-only and template-only patterns")
	}

	var fileMode os.FileMode

	if mode != "" {
		parsedMode, err := parseFileMode(mode)
		if err != nil {
			return nil, fmt.Errorf("invalid file mode: %w", err)
		}

		fileMode = parsedMode
	}

	// ENsure the values map is initialized before adding Tome metadata.
	if values == nil {
		values = make(map[string]any)
	}

	values["__tome__"] = map[string]any{
		"source":  source,
		"target":  target,
		"mode":    fileMode.String(),
		"strip":   strip,
		"include": include,
		"exclude": exclude,
		"copy":    copyPatterns,
		"temp":    tempPatterns,
	}

	return &Tome{
		Source:  source,
		Target:  target,
		Mode:    fileMode,
		Strip:   strip,
		Include: include,
		Exclude: exclude,
		Copy:    copyPatterns,
		Temp:    tempPatterns,
		Values:  values,
	}, nil
}

// parseFileMode parses a file mode expressed as either octal permissions
// or a symbolic nine-character permission string such as rwxr-xr-x.
func parseFileMode(modeStr string) (os.FileMode, error) {

	// Try parsing as octal permissions.
	if len(modeStr) >= 2 && modeStr[0] == '0' {
		if n, err := strconv.ParseUint(modeStr, 8, 32); err == nil {
			return os.FileMode(n), nil
		}
	}

	// Parse symbolic permissions.
	if len(modeStr) != 9 {
		return 0, fmt.Errorf(
			"invalid symbolic file mode: %s",
			modeStr,
		)
	}

	var mode os.FileMode

	symbols := []struct {
		char byte
		bit  os.FileMode
	}{
		// user
		{'r', 0400},
		{'w', 0200},
		{'x', 0100},
		// group
		{'r', 0040},
		{'w', 0020},
		{'x', 0010},
		// others
		{'r', 0004},
		{'w', 0002},
		{'x', 0001},
	}

	for i, symbol := range symbols {
		switch modeStr[i] {
		case symbol.char:
			mode |= symbol.bit
		case '-':
			// Permission is not set.
		default:
			return 0, fmt.Errorf(
				"unexpected character '%c' in symbolic mode",
				modeStr[i],
			)
		}
	}

	return mode, nil
}

// ShouldInclude reports whether the specified path should be processed.
//
// When include patterns are configured, only matching paths are included.
// When exclude patterns are configured, matching paths are excluded.
// If neither is configured, all pahts are included.
func (t *Tome) ShouldInclude(name string) bool {
	if len(t.Include) > 0 {
		return t.matchPatterns(t.Include, name)
	}

	if len(t.Exclude) > 0 {
		return !t.matchPatterns(t.Exclude, name)
	}

	return true
}

// shouldCopy reports whether the specified path should be copied
// instead of being processed as a template.
func (t *Tome) shouldCopy(name string) bool {
	if len(t.Copy) > 0 {
		return t.matchPatterns(t.Copy, name)
	}

	if len(t.Temp) > 0 {
		return !t.matchPatterns(t.Temp, name)
	}

	return false
}

// matchPattenrs reports whether the specified path matches any of the
// supplied glob patterns.
func (t *Tome) matchPatterns(patterns []string, name string) bool {
	for _, pattern := range patterns {
		if pattern == "" {
			continue
		}

		if pattern[0] != '/' {
			pattern = filepath.Join(t.Source, pattern)
		}

		matched, err := doublestar.PathMatch(pattern, name)
		if err != nil {
			fmt.Printf(
				"[templar] Error matching pattern %q: %v\n",
				pattern,
				err,
			)
			continue
		}

		if matched {
			return true
		}
	}

	return false
}

// formatPath converts an input path into its corresponding output path.
//
// The path is made relative to the Tome source directory, its components
// are processed as templates, and the resulting path is joined with the
// Tome target directory.
func (t *Tome) formatPath(inputPath string) (string, error) {
	relPath, err := filepath.Rel(t.Source, inputPath)
	if err != nil {
		return "", fmt.Errorf("error getting relative path: %w", err)
	}

	outputPath, err := t.templatePath(relPath)
	if err != nil {
		return "", fmt.Errorf("error templating path: %w", err)
	}

	return filepath.Join(t.Target, outputPath), nil
}

// templatePath renders a path as a template and applies configured suffix
// stripping to each path component.
func (t *Tome) templatePath(inputPath string) (string, error) {
	var templatedPath bytes.Buffer

	if err := t.Template(
		&templatedPath,
		inputPath,
		inputPath,
	); err != nil {
		return "", fmt.Errorf("error templating name: %w", err)
	}

	outputPath := templatedPath.String()

	if len(t.Strip) == 0 {
		return outputPath, nil
	}

	parts := strings.Split(
		outputPath,
		string(filepath.Separator),
	)

	for i, part := range parts {
		for _, suffix := range t.Strip {
			if strings.HasSuffix(part, suffix) {
				parts[i] = strings.TrimSuffix(part, suffix)
				break
			}
		}
	}

	if strings.HasPrefix(outputPath, string(filepath.Separator)) {
		return string(filepath.Separator) + filepath.Join(parts...), nil
	}

	return filepath.Join(parts...), nil
}
