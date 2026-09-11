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

type Tome struct {
	Source  string         `json:"source"`
	Target  string         `json:"target"`
	Mode    os.FileMode    `json:"mode"`
	Strict  bool           `json:"strict"`
	DryRun  bool           `json:"dryRun"`
	Verbose bool           `json:"verbose"`
	Force   bool           `json:"force"`
	Strip   []string       `json:"strip"`
	Include []string       `json:"include"`
	Exclude []string       `json:"exclude"`
	Copy    []string       `json:"copy"`
	Temp    []string       `json:"temp"`
	Values  map[string]any `json:"values"`
}

func (t *Tome) String() string {
	return fmt.Sprintf("{source: %s, target: %s, mode: %o, strip: %s, include: %v, exclude: %v, copy: %v, temp: %v, values: %v}",
		t.Source, t.Target, t.Mode, t.Strip, t.Include, t.Exclude, t.Copy, t.Temp, t.Values)
}

func New(source, target, mode string, strip, include, exclude, copy, temp []string, values map[string]any) (*Tome, error) {

	if len(include) > 0 && len(exclude) > 0 {
		return nil, fmt.Errorf("cannot use both include and exclude patterns")
	}

	if len(copy) > 0 && len(temp) > 0 {
		return nil, fmt.Errorf("cannot use both copy-only and template-only patterns")
	}

	var fileMode os.FileMode
	var err error
	if mode != "" {
		fileMode, err = parseFileMode(mode)
		if err != nil {
			return nil, fmt.Errorf("invalid file mode: %w", err)
		}
	}

	values["__tome__"] = map[string]any{
		"source":  source,
		"target":  target,
		"mode":    fileMode.String(),
		"strip":   strip,
		"include": include,
		"exclude": exclude,
		"copy":    copy,
		"temp":    temp,
	}

	return &Tome{
		Source:  source,
		Target:  target,
		Mode:    fileMode,
		Strip:   strip,
		Include: include,
		Exclude: exclude,
		Copy:    copy,
		Temp:    temp,
		Values:  values,
	}, nil
}

func parseFileMode(modeStr string) (os.FileMode, error) {
	// Try parsing as octal
	if n, err := strconv.ParseUint(modeStr, 8, 32); err == nil {
		return os.FileMode(n), nil
	}

	// If not octal, parse symbolic string
	if len(modeStr) != 9 {
		return 0, fmt.Errorf("invalid symbolic file mode: %s", modeStr)
	}

	var mode os.FileMode

	symbols := []struct {
		char byte
		bit  os.FileMode
	}{
		{'r', 0400}, {'w', 0200}, {'x', 0100}, // user
		{'r', 0040}, {'w', 0020}, {'x', 0010}, // group
		{'r', 0004}, {'w', 0002}, {'x', 0001}, // others
	}

	for i, sym := range symbols {
		if modeStr[i] == sym.char {
			mode |= sym.bit
		} else if modeStr[i] != '-' {
			return 0, fmt.Errorf("unexpected character '%c' in symbolic mode", modeStr[i])
		}
	}

	return mode, nil
}

func (t *Tome) ShouldInclude(name string) bool {
	if len(t.Include) > 0 {
		return t.matchPatterns(t.Include, name)
	}
	if len(t.Exclude) > 0 {
		return !t.matchPatterns(t.Exclude, name)
	}

	return true
}

func (t *Tome) shouldCopy(name string) bool {
	if len(t.Copy) > 0 {
		return t.matchPatterns(t.Copy, name)
	}
	if len(t.Temp) > 0 {
		return !t.matchPatterns(t.Temp, name)
	}

	return false
}

func (t *Tome) matchPatterns(patterns []string, name string) bool {
	for _, pattern := range patterns {
		if pattern[0] != '/' {
			pattern = filepath.Join(t.Source, pattern)
		}
		matched, err := doublestar.PathMatch(pattern, name)
		if err != nil {
			fmt.Println("Error:", err)
			continue
		}
		if matched {
			return true
		}
	}
	return false
}

func (t *Tome) formatPath(inputPath string) (string, error) {
	// Template the file name
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

func (t *Tome) templatePath(inputPath string) (string, error) {
	var templatedPath bytes.Buffer
	err := t.Template(&templatedPath, inputPath, inputPath)
	if err != nil {
		return "", fmt.Errorf("error templating name: %w", err)
	}

	outputPath := templatedPath.String()

	// Apply suffix stripping to the output path
	if len(t.Strip) > 0 {
		// Split output path by the os specific separator
		parts := strings.Split(outputPath, string(filepath.Separator))
		for i, part := range parts {
			// Strip the suffix from each part of the path
			for _, s := range t.Strip {
				if strings.HasSuffix(part, s) {
					parts[i] = strings.TrimSuffix(part, s)
					break
				}
			}
		}
		// Rejoin the parts to form the new output path
		if outputPath[0] == '/' {
			outputPath = "/" + filepath.Join(parts...)
		} else {
			outputPath = filepath.Join(parts...)
		}
	}
	return outputPath, nil
}
