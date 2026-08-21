package tui

import (
	"os"
	"path/filepath"
	"testing"
)

// Launching the app on a file that does not exist bootstraps it, so `todo`
// (with no argument, or a new filename) leaves a real file on disk instead of
// only writing one on the first edit.
func TestNewModelBootstrapsMissingFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "TODO.md")
	if _, err := newModel(path); err != nil {
		t.Fatalf("newModel: %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected newModel to create the missing file, stat: %v", err)
	}
}
