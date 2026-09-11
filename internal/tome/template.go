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

func (t *Tome) Template(writer io.Writer, text string, name string) error {
	tmpl, err := template.New(name).Funcs(t.funcMap(filepath.Dir(name))).Parse(text)
	if err != nil {
		return err
	}

	missingTemplateKeys, err := findMissingTemplateKeys(tmpl, text, t.Values)
	if err != nil {
		return fmt.Errorf("error finding missing template keys for %s: %w", name, err)
	}
	if len(missingTemplateKeys) > 0 {
		for _, missingKey := range missingTemplateKeys {
			fmt.Printf("[templar] ⚠️  %s:%d:%d missing key '%s'\n", name, missingKey.Line, missingKey.Column, missingKey.Name)
		}
		if t.Strict {
			return errors.New("missing template keys not allowed in strict mode")
		}
	}
	return tmpl.Execute(writer, t.Values)
}

// MissingKey holds the name and position of a missing template key
type MissingKey struct {
	Name   string
	Line   int
	Column int
}

// FindMissingTemplateKeysWithPos parses the template and returns all keys
// that are missing in the given values map, along with their line and column.
func findMissingTemplateKeys(tmpl *template.Template, tmplStr string, values map[string]interface{}) ([]MissingKey, error) {
	var missing []MissingKey

	// Split the template into lines for positional tracking
	lines := strings.Split(tmplStr, "\n")

	// Traverse the template AST
	var walk func(node parse.Node)
	walk = func(node parse.Node) {
		switch n := node.(type) {
		case *parse.ListNode:
			for _, sub := range n.Nodes {
				walk(sub)
			}
		case *parse.ActionNode:
			walk(n.Pipe)
		case *parse.PipeNode:
			for _, cmd := range n.Cmds {
				walk(cmd)
			}
		case *parse.CommandNode:
			for _, arg := range n.Args {
				walk(arg)
			}
		case *parse.FieldNode:
			// .Field format, check the first part only (top-level)
			if len(n.Ident) > 0 {
				key := n.Ident[0]
				if _, ok := values[key]; !ok {
					line, col := positionFromOffset(n.Pos, lines)
					missing = append(missing, MissingKey{Name: key, Line: line, Column: col})
				}
			}
		case *parse.TemplateNode:
			// Not processing nested templates for now
		}
	}

	walk(tmpl.Root)

	return missing, nil
}

// positionFromOffset returns the line and column number based on Pos
func positionFromOffset(pos parse.Pos, lines []string) (int, int) {
	offset := int(pos) - 1 // Pos is 1-indexed

	count := 0
	for i, line := range lines {
		lineLen := len(line) + 1 // +1 for the newline
		if count+lineLen > offset {
			col := offset - count + 1
			return i + 1, col
		}
		count += lineLen
	}
	return 0, 0 // fallback
}
