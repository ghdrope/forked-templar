package tome

import (
	"bytes"
	"strings"
	"testing"
	"text/template"
	"text/template/parse"
)

func TestTemplate_AllKeysPresent(t *testing.T) {
	tome := Tome{
		Values: map[string]interface{}{
			"Name": "World",
		},
	}
	var buf bytes.Buffer
	templateText := "Hello, {{.Name}}!"
	err := tome.Template(&buf, templateText, "test.tmpl")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := buf.String()
	want := "Hello, World!"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestTemplate_MissingKey_NonStrict(t *testing.T) {
	tome := Tome{
		Values: map[string]interface{}{
			"Name": "World",
		},
		Strict: false,
	}

	var buf bytes.Buffer
	templateText := "Hello, {{.Name}}!"
	err := tome.Template(&buf, templateText, "test.tmpl")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// The output will be "Hello, <no value>!"
	got := buf.String()
	if !strings.Contains(got, "Hello,") {
		t.Errorf("expected output to contain 'Hello,', got %q", got)
	}
}

func TestTemplate_MissingKey_Strict(t *testing.T) {
	tome := Tome{
		Values: map[string]interface{}{
			"Name": "World",
		},
		Strict: true,
	}

	var buf bytes.Buffer
	templateText := "Hello, {{.MissingKey}}!"

	err := tome.Template(&buf, templateText, "test.tmpl")

	if err == nil {
		t.Fatal("expected error for missing key in strict mode, got nil")
	}

	if !strings.Contains(err.Error(), "missing template keys not allowed in strict mode") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestFindMissingTemplateKeys(t *testing.T) {
	tmplText := "Hello, {{.Name}}! Your age is {{.Age}}."
	tmpl, err := parseTemplateForTest("test", tmplText)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	values := map[string]interface{}{
		"Name": "Alice",
	}
	missing, err := findMissingTemplateKeys(tmpl, tmplText, values)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(missing) != 1 {
		t.Fatalf("expected 1 missing key, got %d", len(missing))
	}
	if missing[0].Name != "Age" {
		t.Errorf("expected missing key 'Age', got %q", missing[0].Name)
	}
}

func TestPositionFromOffset(t *testing.T) {
	lines := []string{
		"Hello, {{.Name}}!",
		"Your age is {{.Age}}.",
	}
	// Find offset for ".Age"
	offset := strings.Index(lines[1], ".Age") + 1 // parse.Pos is 1-indexed
	line, col := positionFromOffset(parse.Pos(len(lines[0])+1+offset), lines)
	if line != 2 {
		t.Errorf("expected line 2, got %d", line)
	}
	if col != strings.Index(lines[1], ".Age")+1 {
		t.Errorf("expected col %d, got %d", strings.Index(lines[1], ".Age")+1, col)
	}
}

// Helper to parse template for test
func parseTemplateForTest(name, text string) (*template.Template, error) {
	return template.New(name).Parse(text)
}

// Optionally, test that findMissingTemplateKeys returns no error for all keys present
func TestFindMissingTemplateKeys_AllPresent(t *testing.T) {
	tmplText := "Hello, {{.Name}}!"
	tmpl, err := parseTemplateForTest("test", tmplText)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	values := map[string]interface{}{
		"Name": "Bob",
	}
	missing, err := findMissingTemplateKeys(tmpl, tmplText, values)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(missing) != 0 {
		t.Errorf("expected no missing keys, got %d", len(missing))
	}
}

// Optionally, test multiple missing keys
func TestFindMissingTemplateKeys_MultipleMissing(t *testing.T) {
	tmplText := "Hello, {{.Name}}! {{.Foo}} {{.Bar}}"
	tmpl, err := parseTemplateForTest("test", tmplText)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	values := map[string]interface{}{}
	missing, err := findMissingTemplateKeys(tmpl, tmplText, values)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(missing) != 3 {
		t.Errorf("expected 3 missing keys, got %d", len(missing))
	}
	names := []string{missing[0].Name, missing[1].Name, missing[2].Name}
	expected := []string{"Name", "Foo", "Bar"}
	for i, want := range expected {
		if names[i] != want {
			t.Errorf("expected missing key %q, got %q", want, names[i])
		}
	}
}

// Optionally, test that nested fields only check top-level
func TestFindMissingTemplateKeys_NestedFields(t *testing.T) {
	tmplText := "Hello, {{.User.Name}}!"
	tmpl, err := parseTemplateForTest("test", tmplText)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	values := map[string]interface{}{}
	missing, err := findMissingTemplateKeys(tmpl, tmplText, values)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(missing) != 1 || missing[0].Name != "User" {
		t.Errorf("expected missing key 'User', got %+v", missing)
	}
}

// Optionally, test that no panic occurs for empty template
func TestTemplate_EmptyTemplate(t *testing.T) {
	tome := Tome{
		Values: map[string]interface{}{
			"Name": "World",
		},
	}
	var buf bytes.Buffer
	templateText := ""
	err := tome.Template(&buf, templateText, "empty.tmpl")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if buf.String() != "" {
		t.Errorf("expected empty output, got %q", buf.String())
	}
}

// Optionally, test error on invalid template syntax
func TestTemplate_InvalidSyntax(t *testing.T) {
	tome := Tome{}
	var buf bytes.Buffer
	templateText := "Hello, {{.Name"
	err := tome.Template(&buf, templateText, "bad.tmpl")
	if err == nil {
		t.Fatal("expected error for invalid template syntax, got nil")
	}
}
