package tome

import (
	"bytes"
	"strings"
	"testing"
	"text/template"
	"text/template/parse"
)

// TestTemplate_AllKeysPresent verifies that a template is rendered correctly
// when all referenced keys are present in the Tome values.
func TestTemplate_AllKeysPresent(t *testing.T) {
	tome := Tome{
		Values: map[string]any{
			"Name": "World",
		},
	}

	var buf bytes.Buffer

	err := tome.Template(&buf, "Hello, {{.Name}}!", "test.tmpl")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got, want := buf.String(), "Hello, World!"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

// TestTemplate_MissingKey_NonStrict verifies that a missing template key does
// not return an error when strict mode is disabled.
func TestTemplate_MissingKey_NonStrict(t *testing.T) {
	tome := Tome{
		Values: map[string]any{
			"Name": "World",
		},
		Strict: false,
	}

	var buf bytes.Buffer

	err := tome.Template(
		&buf,
		"Hello, {{.Name}}! {{.MissingKey}}",
		"test.tmpl",
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := buf.String()

	if !strings.Contains(got, "Hello, World!") {
		t.Errorf("expected output to contain %q, got %q", "Hello, World!", got)
	}

	if !strings.Contains(got, "<no value>") {
		t.Errorf("expected output to contain %q, got %q", "<no value>", got)
	}
}

// TestTemplate_MissingKey_Strict verifies that a missing template key returns
// an error when strict mode is enabled.
func TestTemplate_MissingKey_Strict(t *testing.T) {
	tome := Tome{
		Values: map[string]any{
			"Name": "World",
		},
		Strict: true,
	}

	var buf bytes.Buffer

	err := tome.Template(
		&buf,
		"Hello, {{.Name}}! {{.MissingKey}}",
		"test.tmpl",
	)

	if err == nil {
		t.Fatal("expected error for missing key in strict mode")
	}

	const want = "missing template keys not allowed in strict mode"

	if !strings.Contains(err.Error(), want) {
		t.Errorf("expected error to contain %q, got %q", want, err)
	}
}

// TestTemplate_EmptyTemplate verifies that an empty template is executed
// successfully and produces no output.
func TestTemplate_EmptyTemplate(t *testing.T) {
	tome := Tome{
		Values: map[string]any{
			"Name": "World",
		},
	}

	var buf bytes.Buffer

	err := tome.Template(&buf, "", "empty.tmpl")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got := buf.String(); got != "" {
		t.Errorf("expected empty output, got %q", got)
	}
}

// TestTemplate_InvalidSyntax verifies that invalid template syntax returns
// an error during template parsing.
func TestTemplate_InvalidSyntax(t *testing.T) {
	tome := Tome{}

	var buf bytes.Buffer

	err := tome.Template(
		&buf,
		"Hello, {{.Name",
		"bad.tmpl",
	)
	if err == nil {
		t.Fatal("expected error for invalid template syntax")
	}
}

// TestFindMissingTemplateKeys verifies that missing top-level template keys
// are correctly identified in different template scenarios.
func TestFindMissingTemplateKeys(t *testing.T) {
	tests := []struct {
		name      string
		template  string
		values    map[string]any
		wantNames []string
	}{
		{
			name:     "all keys present",
			template: "Hello, {{.Name}}!",
			values: map[string]any{
				"Name": "Bob",
			},
			wantNames: nil,
		},
		{
			name:      "single missing key",
			template:  "Hello, {{.Name}}! Your age is {{.Age}}.",
			values:    map[string]any{"Name": "Alice"},
			wantNames: []string{"Age"},
		},
		{
			name:      "multiple missing keys",
			template:  "Hello, {{.Name}}! {{.Foo}} {{.Bar}}",
			values:    map[string]any{},
			wantNames: []string{"Name", "Foo", "Bar"},
		},
		{
			name:      "nested field",
			template:  "Hello, {{.User.Name}}!",
			values:    map[string]any{},
			wantNames: []string{"User"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpl := parseTemplateForTest(t, tt.template)

			missing, err := findMissingTemplateKeys(
				tmpl,
				tt.template,
				tt.values,
			)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(missing) != len(tt.wantNames) {
				t.Fatalf(
					"expected %d missing keys, got %d",
					len(tt.wantNames),
					len(missing),
				)
			}

			for i, want := range tt.wantNames {
				if got := missing[i].Name; got != want {
					t.Errorf(
						"missing[%d].Name = %q, want %q",
						i,
						got,
						want,
					)
				}
			}
		})
	}
}

// TestPositionFromOffset verifies that a template parse position is converted
// to the expected one-based line and column.
func TestPositionFromOffset(t *testing.T) {
	lines := []string{
		"Hello, {{.Name}}!",
		"Your age is {{.Age}}.",
	}

	offset := strings.Index(lines[1], ".Age") + 1
	pos := parse.Pos(len(lines[0]) + 1 + offset)

	line, column := positionFromOffset(pos, lines)

	if line != 2 {
		t.Errorf("expected line 2, got %d", line)
	}

	wantColumn := strings.Index(lines[1], ".Age") + 1

	if column != wantColumn {
		t.Errorf("expected column %d, got %d", wantColumn, column)
	}
}

// parseTemplateForTest parses a template for use in tests and fails the test
// immediately when parsing fails.
func parseTemplateForTest(t *testing.T, text string) *template.Template {
	t.Helper()

	tmpl, err := template.New("test").Parse(text)
	if err != nil {
		t.Fatalf("failed to parse template: %v", err)
	}

	return tmpl
}
