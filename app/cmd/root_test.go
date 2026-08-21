package cmd

import "testing"

func TestRootArgs(t *testing.T) {
	c := newRootCommand()
	if err := c.Args(c, nil); err != nil {
		t.Errorf("no argument should be valid (defaults to TODO.md): %v", err)
	}
	if err := c.Args(c, []string{"a.md"}); err != nil {
		t.Errorf("one argument should be valid: %v", err)
	}
	if err := c.Args(c, []string{"a.md", "b.md"}); err == nil {
		t.Error("expected an error with two arguments")
	}
}

func TestFilePathDefault(t *testing.T) {
	if got := filePath(nil); got != defaultFile {
		t.Errorf("filePath(nil) = %q, want %q", got, defaultFile)
	}
	if got := filePath([]string{"custom.md"}); got != "custom.md" {
		t.Errorf("filePath([custom.md]) = %q, want custom.md", got)
	}
}

func TestRootMeta(t *testing.T) {
	c := newRootCommand()
	if c.Use == "" || c.Short == "" {
		t.Error("the root command should set Use and Short")
	}
}
