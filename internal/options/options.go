package options

import (
	flag "github.com/spf13/pflag"
)

var (
	DryRun          bool
	Verbose         bool
	Force           bool
	ShowVersion     bool
	ShowHelp        bool
	Strict          bool
	Mode            string
	Out             string
	Args            []string
	StripSuffix     []string
	Values          []string
	SetValues       []string
	IncludePatterns []string
	ExcludePatterns []string
	CopyPatterns    []string
	TempPatterns    []string
)

func Init() {
	flag.BoolVarP(&ShowVersion, "version", "V", false, "Show version and exit")
	flag.BoolVarP(&ShowHelp, "help", "h", false, "Show help and exit")
	flag.BoolVarP(&DryRun, "dry-run", "d", false, "Simulate actions without writing files")
	flag.BoolVarP(&Verbose, "verbose", "D", false, "Enable verbose logging")
	flag.BoolVarP(&Strict, "strict", "S", false, "Fail on missing values")
	flag.BoolVarP(&Force, "force", "F", false, "Overwrite files in output directory without confirmation")
	flag.StringVarP(&Mode, "mode", "m", "", "Set file mode (permissions) for created files (octal or symbolic)")
	flag.StringVarP(&Out, "out", "o", "", "Output directory for generated files (default: standard output)")
	flag.StringSliceVarP(&Values, "values", "v", []string{}, "Path to values YAML file (can be repeated)")
	flag.StringSliceVarP(&SetValues, "set", "s", []string{}, "Set a value (key=value) (can be repeated)")
	flag.StringSliceVarP(&IncludePatterns, "include", "i", []string{}, "Glob pattern of files to include (can be repeated)")
	flag.StringSliceVarP(&ExcludePatterns, "exclude", "e", []string{}, "Glob pattern of files to exclude (can be repeated)")
	flag.StringSliceVarP(&CopyPatterns, "copy", "c", []string{}, "Glob pattern for files to copy without templating (can be repeated)")
	flag.StringSliceVarP(&TempPatterns, "temp", "t", []string{}, "Glob pattern for files to template; others are copied as-is (mutually exclusive with --copy)")
	flag.StringSliceVarP(&StripSuffix, "strip", "r", []string{}, "Suffix to strip from output filenames if templated (can be repeated)")

	flag.Parse()
	Args = flag.Args()

}

func PrintDefaults() {
	flag.PrintDefaults()
}
