package tome

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// writeFile creates path, and its parent directories, with the given content.
func writeFile(t *testing.T, path, content string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
}

// readFile returns the content of path, failing the test if it can't be read.
func readFile(t *testing.T, path string) string {
	t.Helper()
	content, err := os.ReadFile(path)
	require.NoError(t, err)
	return string(content)
}

// captureLog redirects the diagnostic output to a buffer for the duration of
// the test. Tests using it must not run in parallel.
func captureLog(t *testing.T) *bytes.Buffer {
	t.Helper()
	orig := logOut
	var buf bytes.Buffer
	logOut = &buf
	t.Cleanup(func() { logOut = orig })
	return &buf
}

// setStdin replaces the input read by the overwrite prompt for the duration
// of the test. Tests using it must not run in parallel.
func setStdin(t *testing.T, input string) {
	t.Helper()
	orig := stdin
	stdin = strings.NewReader(input)
	t.Cleanup(func() { stdin = orig })
}

// newSubTomeTree creates a template tree with a .tome.yaml in d1 that renders
// d1 twice (targets "one" and "two"), and a sibling d2 without a tome.
func newSubTomeTree(t *testing.T) string {
	t.Helper()
	in := t.TempDir()
	writeFile(t, filepath.Join(in, "d1", ".tome.yaml"), `
- target: one
  values:
    app:
      name: one
- target: two
`)
	writeFile(t, filepath.Join(in, "d1", "f.txt"), "{{ .app.name }}-{{ .app.env }}")
	writeFile(t, filepath.Join(in, "d2", "f.txt"), "{{ .app.name }}-{{ .app.env }}")
	return in
}

// TestRender_SubTomes verifies that a .tome.yaml renders the directory once
// per tome, that the target directories are usable, and that values set in
// one tome don't leak to the other tomes, to sibling directories or to the
// base tome.
func TestRender_SubTomes(t *testing.T) {
	in := newSubTomeTree(t)
	out := t.TempDir()

	base, err := New(in, out, "", nil, nil, nil, nil, nil, map[string]any{
		"app": map[string]any{"env": "prod"},
	})
	require.NoError(t, err)

	require.NoError(t, base.Render(in))

	assert.Equal(t, "one-prod", readFile(t, filepath.Join(out, "one", "f.txt")))
	assert.Equal(t, "<no value>-prod", readFile(t, filepath.Join(out, "two", "f.txt")))
	assert.Equal(t, "<no value>-prod", readFile(t, filepath.Join(out, "d2", "f.txt")))

	assert.NotContains(t, base.Values["app"], "name", "base values must not be changed by a sub-tome")

	srcInfo, err := os.Stat(filepath.Join(in, "d1"))
	require.NoError(t, err)
	for _, dir := range []string{"one", "two"} {
		info, err := os.Stat(filepath.Join(out, dir))
		require.NoError(t, err)
		assert.Equal(t, srcInfo.Mode().Perm(), info.Mode().Perm(), "target %s should keep the template directory mode", dir)
	}
}

// TestRender_SubTomesInheritRuntimeOptions verifies that sub-tomes use the
// runtime options (dry-run, strict, ...) of the base tome.
func TestRender_SubTomesInheritRuntimeOptions(t *testing.T) {
	t.Run("dry-run", func(t *testing.T) {
		in := newSubTomeTree(t)
		out := filepath.Join(t.TempDir(), "out")
		log := captureLog(t)

		base, err := New(in, out, "", nil, nil, nil, nil, nil, nil)
		require.NoError(t, err)
		base.DryRun = true
		base.Verbose = true

		require.NoError(t, base.Render(in))

		assert.NoDirExists(t, out)
		assert.Contains(t, log.String(), "[templar] Tome {")
		assert.Contains(t, log.String(), "Templating")
	})

	t.Run("strict", func(t *testing.T) {
		in := newSubTomeTree(t)
		_ = captureLog(t)

		base, err := New(in, t.TempDir(), "", nil, nil, nil, nil, nil, nil)
		require.NoError(t, err)
		base.Strict = true

		err = base.Render(in)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "strict mode")
	})
}

// TestRender_InvalidTomeFile verifies that an invalid .tome.yaml is reported.
func TestRender_InvalidTomeFile(t *testing.T) {
	in := t.TempDir()
	writeFile(t, filepath.Join(in, "d", ".tome.yaml"), "target: [unclosed")

	base, err := New(in, t.TempDir(), "", nil, nil, nil, nil, nil, nil)
	require.NoError(t, err)

	err = base.Render(in)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to load tomes")
}

// TestRender_ModeAppliesOnlyToFiles verifies that the configured mode is used
// for files, while directories keep the template directory mode (a file mode
// like 0600 has no execute bit and would make directories unusable).
func TestRender_ModeAppliesOnlyToFiles(t *testing.T) {
	in := t.TempDir()
	out := t.TempDir()
	writeFile(t, filepath.Join(in, "sub", "f.txt"), "hello")

	base, err := New(in, out, "0600", nil, nil, nil, nil, nil, nil)
	require.NoError(t, err)

	require.NoError(t, base.Render(in))

	srcInfo, err := os.Stat(filepath.Join(in, "sub"))
	require.NoError(t, err)
	dirInfo, err := os.Stat(filepath.Join(out, "sub"))
	require.NoError(t, err)
	assert.Equal(t, srcInfo.Mode().Perm(), dirInfo.Mode().Perm())

	fileInfo, err := os.Stat(filepath.Join(out, "sub", "f.txt"))
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o600), fileInfo.Mode().Perm())
}

// TestRender_CreatesMissingParentDirectories verifies that a templated path
// with new directories (e.g. "{{ .dir }}/f.txt") gets usable parent
// directories.
func TestRender_CreatesMissingParentDirectories(t *testing.T) {
	in := t.TempDir()
	out := t.TempDir()
	input := filepath.Join(in, "f.txt")
	writeFile(t, input, "hello")

	tome := &Tome{Source: in, Target: filepath.Join(out, "a", "b"), Mode: 0o600}

	require.NoError(t, tome.Render(input))
	assert.Equal(t, "hello", readFile(t, filepath.Join(out, "a", "b", "f.txt")))
}

// TestRender_Symlink verifies that symlinks are recreated with a templated
// target, and that an existing link is only replaced with Force.
func TestRender_Symlink(t *testing.T) {
	in := t.TempDir()
	out := t.TempDir()
	writeFile(t, filepath.Join(in, "real.txt"), "hello")
	link := filepath.Join(in, "link")
	require.NoError(t, os.Symlink("{{ .name }}.txt", link))

	tome := &Tome{Source: in, Target: out, Values: map[string]any{"name": "real"}}

	require.NoError(t, tome.Render(link))

	target, err := os.Readlink(filepath.Join(out, "link"))
	require.NoError(t, err)
	assert.Equal(t, "real.txt", target)

	t.Run("existing link is kept without force", func(t *testing.T) {
		_ = captureLog(t)
		setStdin(t, "n\n")
		tome.Values = map[string]any{"name": "other"}

		require.NoError(t, tome.Render(link))

		target, err := os.Readlink(filepath.Join(out, "link"))
		require.NoError(t, err)
		assert.Equal(t, "real.txt", target)
	})

	t.Run("existing link is replaced after confirmation", func(t *testing.T) {
		_ = captureLog(t)
		setStdin(t, "y\n")
		tome.Values = map[string]any{"name": "confirmed"}

		require.NoError(t, tome.Render(link))

		target, err := os.Readlink(filepath.Join(out, "link"))
		require.NoError(t, err)
		assert.Equal(t, "confirmed.txt", target)
	})

	t.Run("existing link is replaced with force", func(t *testing.T) {
		log := captureLog(t)
		tome.Values = map[string]any{"name": "other"}
		tome.Force = true
		tome.Verbose = true

		require.NoError(t, tome.Render(link))

		target, err := os.Readlink(filepath.Join(out, "link"))
		require.NoError(t, err)
		assert.Equal(t, "other.txt", target)
		assert.Contains(t, log.String(), "Removing existing symlink")
	})

	t.Run("invalid templated target", func(t *testing.T) {
		badLink := filepath.Join(in, "bad")
		require.NoError(t, os.Symlink("{{ .name", badLink))

		err := tome.Render(badLink)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "error formatting target for symlink")
	})
}

// TestRender_OverwritePrompt verifies that an existing output file is only
// overwritten when the user confirms it.
func TestRender_OverwritePrompt(t *testing.T) {
	tests := []struct {
		name   string
		answer string
		want   string
	}{
		{name: "yes", answer: "y\n", want: "new"},
		{name: "full yes", answer: " YES \n", want: "new"},
		{name: "no", answer: "n\n", want: "old"},
		{name: "empty answer", answer: "", want: "old"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			in := t.TempDir()
			out := t.TempDir()
			input := filepath.Join(in, "f.txt")
			writeFile(t, input, "new")
			writeFile(t, filepath.Join(out, "f.txt"), "old")

			log := captureLog(t)
			setStdin(t, tt.answer)

			tome := &Tome{Source: in, Target: out}
			require.NoError(t, tome.Render(input))

			assert.Equal(t, tt.want, readFile(t, filepath.Join(out, "f.txt")))
			assert.Contains(t, log.String(), "already exists. Overwrite?")
		})
	}
}

// TestRender_VerboseLogs verifies that every action is logged in verbose mode,
// and that the logs go to the diagnostic output, not to stdout.
func TestRender_VerboseLogs(t *testing.T) {
	in := t.TempDir()
	out := t.TempDir()
	writeFile(t, filepath.Join(in, "a.txt"), "a")
	writeFile(t, filepath.Join(in, "data.bin"), "bin")
	writeFile(t, filepath.Join(in, "skip.txt"), "skip")
	writeFile(t, filepath.Join(in, "sub", "b.txt"), "b")
	require.NoError(t, os.Symlink("a.txt", filepath.Join(in, "link")))

	log := captureLog(t)

	base, err := New(in, out, "", nil, nil, []string{"skip.txt"}, []string{"*.bin"}, nil, nil)
	require.NoError(t, err)
	base.Verbose = true

	require.NoError(t, base.Render(in))

	for _, want := range []string{
		"[templar] Templating",
		"[templar] Copying",
		"[templar] Skipping: skip.txt",
		"[templar] Creating directory",
		"[templar] Recreating symlink",
	} {
		assert.Contains(t, log.String(), want)
	}
	assert.NoFileExists(t, filepath.Join(out, "skip.txt"))
	assert.Equal(t, "bin", readFile(t, filepath.Join(out, "data.bin")))
}

// TestRender_TemplateErrorCleansUp verifies that a failed render leaves no
// output or temporary file behind.
func TestRender_TemplateErrorCleansUp(t *testing.T) {
	in := t.TempDir()
	out := t.TempDir()
	input := filepath.Join(in, "f.txt")
	writeFile(t, input, "{{ fail \"boom\" }}")

	tome := &Tome{Source: in, Target: out}

	err := tome.Render(input)
	require.Error(t, err)

	entries, err := os.ReadDir(out)
	require.NoError(t, err)
	assert.Empty(t, entries)
}

// TestTemplate_WarningsGoToLog verifies that missing key warnings are written
// to the diagnostic output and never mixed with the rendered content.
func TestTemplate_WarningsGoToLog(t *testing.T) {
	log := captureLog(t)

	var rendered bytes.Buffer
	tome := &Tome{Values: map[string]any{}}
	require.NoError(t, tome.Template(&rendered, "{{ .missing }}ok", "w.tmpl"))

	assert.Equal(t, "<no value>ok", rendered.String())
	assert.Contains(t, log.String(), "w.tmpl:1:3 missing key 'missing'")
}
