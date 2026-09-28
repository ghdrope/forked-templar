package tome

import (
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"text/template"
	"text/template/parse"
)

// Template parses and executes a template using the Tome configuration and
// values. It reports missing top-level template keys and, when Strict is
// enabled, returns an error instead of executing the template.
func (t *Tome) Template(writer io.Writer, text string, name string) error {
	tmpl, err := template.New(name).
		Funcs(t.funcMap(filepath.Dir(name))).
		Parse(text)
	if err != nil {
		return err
	}

	missingTemplateKeys, err := findMissingTemplateKeys(
		tmpl,
		text,
		t.Values,
	)
	if err != nil {
		return fmt.Errorf(
			"error finding missing template keys for %s: %w",
			name,
			err,
		)
	}

	if len(missingTemplateKeys) > 0 {
		for _, missingKey := range missingTemplateKeys {
			fmt.Printf(
				"[templar] ⚠️  %s:%d:%d missing key '%s'\n",
				name,
				missingKey.Line,
				missingKey.Column,
				missingKey.Name,
			)
		}

		if t.Strict {
			return errors.New(
				"missing template keys not allowed in strict mode",
			)
		}
	}

	return tmpl.Execute(writer, t.Values)
}

// MissingKey describes a template key that is missing from the values map,
// including its position in the template source.
type MissingKey struct {
	Name   string
	Line   int
	Column int
}

// findMissingTemplateKeys walks the template AST and returns all top-level
// field keys that are missing from the provided values map.
func findMissingTemplateKeys(
	tmpl *template.Template,
	tmplStr string,
	values map[string]any,
) ([]MissingKey, error) {
	var missing []MissingKey

	lines := strings.Split(tmplStr, "\n")

	var walk func(node parse.Node)
	walk = func(node parse.Node) {
		switch n := node.(type) {
		case *parse.ListNode:
			for _, child := range n.Nodes {
				walk(child)
			}

		case *parse.ActionNode:
			walk(n.Pipe)

		case *parse.PipeNode:
			for _, command := range n.Cmds {
				walk(command)
			}

		case *parse.CommandNode:
			for _, argument := range n.Args {
				walk(argument)
			}

		case *parse.FieldNode:
			if len(n.Ident) == 0 {
				return
			}

			key := n.Ident[0]

			if _, ok := values[key]; ok {
				return
			}

			line, column := positionFromOffset(n.Pos, lines)

			missing = append(missing, MissingKey{
				Name:   key,
				Line:   line,
				Column: column,
			})

		case *parse.TemplateNode:
			// Nested templates are intentionally not processed here.
		}
	}

	walk(tmpl.Root)

	return missing, nil
}

// positionFromOffset converts a template parse position into a one-based
// line and column position.
func positionFromOffset(pos parse.Pos, lines []string) (int, int) {
	offset := int(pos) - 1

	currentOffset := 0

	for lineNumber, line := range lines {
		lineLength := len(line) + 1

		if currentOffset+lineLength > offset {
			column := offset - currentOffset + 1
			return lineNumber + 1, column
		}

		currentOffset += lineLength
	}

	return 0, 0
}
