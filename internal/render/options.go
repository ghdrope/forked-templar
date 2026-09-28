package render

// Options contains the configuration used by the Templar renderer.
type Options struct {
	DryRun  bool
	Verbose bool
	Force   bool
	Strict  bool

	Mode string
	Out  string

	StripSuffix []string
	Values      []string
	SetValues   []string

	IncludePatterns []string
	ExcludePatterns []string
	CopyPatterns    []string
	TempPatterns    []string
}
