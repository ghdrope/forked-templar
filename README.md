# Forked-Templar

![Templar](https://github.com/user-attachments/assets/9a145b75-082e-4474-b1bc-b1313e8d5530)

Templar is a fast, lightweight templating engine for directories and files, built with Go’s text/template and Sprig.
It lets you define how directories are generated using simple, declarative control files, enabling dynamic, repeatable, and fully configurable project structures.

Templar is ideal for generating config files, scaffolding projects, or automating deployments — all with clean, composable templates.

## ✨ Features

- Directory-aware templating: Template entire directory trees, not just individual files.
- Control files (Tomes) define generation rules locally, per directory, (similar to a Makefile for templates).
- Tomes are templated themselves, as well as file- and directory names, allowing for dynamic output directroy structures.
- Filenames and paths are also templated, as well as symbolic-link names and their targets
- Go templates + Sprig functions: Powerful templating features out of the box.

## 📦 Installation

### Download

Download a prebuilt binary from the [Releases](/romosch/templar/releases).

### Clone/Build

1. Clone the repository:

    ```bash
    git clone https://github.com/romosch/templar.git
    cd templar
    ```

2. Build the binary:

    ```bash
    go build -o templar
    ```

## 🛠️ Usage

```bash
templar [options] <input dir/file>
```

### 🧰 Options

- `-c`, `--copy` Glob pattern for files to copy without templating (can be repeated)
- `-d`, `--dry-run` Simulate actions without writing files
- `-e`, `--exclude` Glob pattern of files to exclude (can be repeated)
- `-F`, `--force` Overwrite files in output directory without confirmation
- `-h`, `--help`Show help and exit
- `-i`, `--include` Glob pattern of files to include (can be repeated)
- `-m`, `--mode` Set file mode (permissions) for created files (octal or symbolic)
- `-o`, `--out` Output directory for generated files (default: standard output)
- `-s`, `--s` Set a value (key=value) (can be repeated)
- `-S`, `--strict` Fail on missing values
- `-r`, `--strip` Suffix to strip from output filenames if templated (can be repeated)
- `-t`, `--temp` Glob pattern for files to template; others are copied as-is (mutually exclusive with `--copy`)
- `-v`, `--values` Path to values YAML file (can be repeated)
- `-D`, `--verbose` Enable verbose logging
- `-V`, `--version` Show version and exit

### 🧾 Tomes

A Tome is a special YAML file (`.tome.yaml`) placed inside any template directory.
It acts as a blueprint for rendering, telling Templar how the contents of that directory should be processed and where the generated outputs should be written.

Tomes enable local control and dynamic generation. A single template directory can produce one, many, or differently customized outputs, all from the same source.

Tome files themselves are templates. Before being evaluated, a .tome.yaml is rendered just like any other file — allowing using input variables, conditional logic, and Sprig functions to control how the directory behaves based on the provided values.

#### Properties

A .tome.yaml file can be composed of a single tome, or a list of them, each with the following properties:

| Property  | Type          | Description                                                                 | Default          |
|-----------|---------------|-----------------------------------------------------------------------------|------------------|
| `mode`    | `string`      | Octal/symbolic file-mode specifying rendered files type and permissions     | Same as template |
| `target`  | `string`      | Target directory (relative or absolute)                                     | Same as template |
| `strip`   | `string`      | Suffix to strip from output filenames                                       | None             |
| `include` | `[]string`    | Glob patterns of files to include (can be repeated)                         | All              |
| `exclude` | `[]string`    | Glob patterns of files to exclude (can be repeated)                         | None             |
| `copy`    | `[]string`    | Glob patterns for files to copy without templating (can be repeated)        | None             |
| `temp`    | `[]string`    | Glob patterns for files to template; others copied                          | All              |
| `values`  | `map[string]` | Default values for rendering, overridden by higher-level values                          | None             |

### Templates

Templar uses Go's [text/template](https://pkg.go.dev/text/template) extended with functions from [sprig](https://masterminds.github.io/sprig)
and the following custom functions:

#### `seq`

Returns a slice Overrides the sprig [seq](https://masterminds.github.io/sprig/integer_slice.html) function to return a slice instead of a string.

#### `include`

Imports the content from another file. The imported content is templated using the same values as for the current file.

#### `toYaml`

Converts a given list, slice, array, dict, or object to YAML string.

#### `fromYaml`

Converts a YAML string to an iterable map object.

#### `toJson`

Converts a list, slice, array, dict, or object to JSON string.

#### `fromJson`

Converts a JSON string to an iterable map object.

#### `toToml`

Converts a list, slice, array, dict, or object to TOML string.

#### `fromToml`

Converts a TOML string to an iterable map object.

#### `required`

Throws an error if passed variable is undefined

## 🤝 Contributions

Contributions are welcome! Please open an issue or submit a pull request.

## 📜 License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.
