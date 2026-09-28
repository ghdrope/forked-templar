package render

import "testing"

// TestOptions_Defaults verifies that a newly created Options value
// contains the expected zero values.
func TestOptions_Defaults(t *testing.T) {
	opts := Options{}

	if opts.DryRun {
		t.Error("expected DryRun to be false by default")
	}

	if opts.Verbose {
		t.Error("expected Verbose to be false by default")
	}

	if opts.Force {
		t.Error("expected Force to be false by default")
	}

	if opts.Strict {
		t.Error("expected Strict to be false by default")
	}

	if opts.Mode != "" {
		t.Errorf("expected Mode to be empty, got %q", opts.Mode)
	}

	if opts.Out != "" {
		t.Errorf("expected Out to be empty, got %q", opts.Out)
	}

	if len(opts.Values) != 0 {
		t.Errorf("expected Values to be empty, got %v", opts.Values)
	}

	if len(opts.SetValues) != 0 {
		t.Errorf("expected SetValues to be empty, got %v", opts.SetValues)
	}

	if len(opts.IncludePatterns) != 0 {
		t.Errorf("expected IncludePatterns to be empty, got %v", opts.IncludePatterns)
	}

	if len(opts.ExcludePatterns) != 0 {
		t.Errorf("expected ExcludePatterns to be empty, got %v", opts.ExcludePatterns)
	}

	if len(opts.CopyPatterns) != 0 {
		t.Errorf("expected CopyPatterns to be empty, got %v", opts.CopyPatterns)
	}

	if len(opts.TempPatterns) != 0 {
		t.Errorf("expected TempPatterns to be empty, got %v", opts.TempPatterns)
	}

	if len(opts.StripSuffix) != 0 {
		t.Errorf("expected StripSuffix to be empty, got %v", opts.StripSuffix)
	}
}

// TestOptions_Values verifies that all Options fields can be configured.
func TestOptions_Values(t *testing.T) {
	opts := Options{
		DryRun:  true,
		Verbose: true,
		Force:   true,
		Strict:  true,

		Mode: "0755",
		Out:  "output",

		Values:    []string{"values.yaml"},
		SetValues: []string{"name=templar"},

		IncludePatterns: []string{"*.tmpl"},
		ExcludePatterns: []string{"*.bak"},
		CopyPatterns:    []string{"*.txt"},
		TempPatterns:    []string{"*.template"},
		StripSuffix:     []string{".tmpl"},
	}

	if !opts.DryRun {
		t.Error("expected DryRun to be true")
	}

	if !opts.Verbose {
		t.Error("expected Verbose to be true")
	}

	if !opts.Force {
		t.Error("expected Force to be true")
	}

	if !opts.Strict {
		t.Error("expected Strict to be true")
	}

	if opts.Mode != "0755" {
		t.Errorf("expected Mode %q, got %q", "0755", opts.Mode)
	}

	if opts.Out != "output" {
		t.Errorf("expected Out %q, got %q", "output", opts.Out)
	}

	if len(opts.Values) != 1 || opts.Values[0] != "values.yaml" {
		t.Errorf("unexpected Values: %v", opts.Values)
	}

	if len(opts.SetValues) != 1 || opts.SetValues[0] != "name=templar" {
		t.Errorf("unexpected SetValues: %v", opts.SetValues)
	}

	if len(opts.IncludePatterns) != 1 || opts.IncludePatterns[0] != "*.tmpl" {
		t.Errorf("unexpected IncludePatterns: %v", opts.IncludePatterns)
	}

	if len(opts.ExcludePatterns) != 1 || opts.ExcludePatterns[0] != "*.bak" {
		t.Errorf("unexpected ExcludePatterns: %v", opts.ExcludePatterns)
	}

	if len(opts.CopyPatterns) != 1 || opts.CopyPatterns[0] != "*.txt" {
		t.Errorf("unexpected CopyPatterns: %v", opts.CopyPatterns)
	}

	if len(opts.TempPatterns) != 1 || opts.TempPatterns[0] != "*.template" {
		t.Errorf("unexpected TempPatterns: %v", opts.TempPatterns)
	}

	if len(opts.StripSuffix) != 1 || opts.StripSuffix[0] != ".tmpl" {
		t.Errorf("unexpected StripSuffix: %v", opts.StripSuffix)
	}
}
