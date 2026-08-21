package tui

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/andresbott/todo/internal/todo"
)

// --- reloadCheck: the pure file-change probe ---

func TestReloadCheckUnchangedIsNil(t *testing.T) {
	path := filepath.Join(t.TempDir(), "todo.md")
	content := "# Work\n\n- [ ] a\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	if msg := reloadCheck(path, content); msg != nil {
		t.Errorf("an unchanged file must yield no reload, got %#v", msg)
	}
}

func TestReloadCheckDetectsChange(t *testing.T) {
	path := filepath.Join(t.TempDir(), "todo.md")
	changed := "# Work\n\n- [ ] a\n- [ ] b\n"
	if err := os.WriteFile(path, []byte(changed), 0o644); err != nil {
		t.Fatal(err)
	}
	msg := reloadCheck(path, "# Work\n\n- [ ] a\n")
	r, ok := msg.(fileReloadedMsg)
	if !ok {
		t.Fatalf("a changed file must yield a fileReloadedMsg, got %#v", msg)
	}
	if r.content != changed {
		t.Errorf("reload must carry the new content, got %q", r.content)
	}
	if find(r.doc, "b") == nil {
		t.Errorf("reloaded document must include the new task b")
	}
}

func TestReloadCheckMissingFileIsNil(t *testing.T) {
	if msg := reloadCheck(filepath.Join(t.TempDir(), "nope.md"), "x"); msg != nil {
		t.Errorf("a missing/unreadable file must not trigger a reload, got %#v", msg)
	}
}

// --- applying a reload to the model ---

func TestReloadAppliesExternalChange(t *testing.T) {
	m, _ := newTestModel(t, "# Work\n\n- [ ] a\n")
	m = send(m, fileReloadedMsg{doc: todo.Parse("# Work\n\n- [ ] a\n- [ ] b\n"), content: "# Work\n\n- [ ] a\n- [ ] b\n"})
	if find(m.doc, "b") == nil {
		t.Fatalf("an external change must be applied to the model")
	}
	if !hasRow(m, "b") {
		t.Errorf("the new task must appear in the rebuilt tree")
	}
}

func TestReloadPreservesSelection(t *testing.T) {
	m, _ := newTestModel(t, "# Work\n\n- [ ] a\n- [ ] c\n")
	m = press(m, "down", "down") // Work, a, c -> select c
	if got := m.tree.selected().Title; got != "c" {
		t.Fatalf("precondition: expected c selected, got %q", got)
	}
	m = send(m, fileReloadedMsg{doc: todo.Parse("# Work\n\n- [ ] a\n- [ ] b\n- [ ] c\n"), content: "x"})
	if got := m.tree.selected().Title; got != "c" {
		t.Errorf("selection must stay on c across a reload, got %q", got)
	}
}

func TestReloadPreservesFold(t *testing.T) {
	m, _ := newTestModel(t, "# Work\n\n- [ ] a\n")
	m = press(m, "enter") // Work is selected at start; collapse it
	m = send(m, fileReloadedMsg{doc: todo.Parse("# Work\n\n- [ ] a\n- [ ] b\n"), content: "x"})
	work := m.tree.selected()
	if work.Title != "Work" || !m.tree.collapsed[work] {
		t.Errorf("Work must remain collapsed after a reload")
	}
	if hasRow(m, "a") || hasRow(m, "b") {
		t.Errorf("a collapsed category's children must stay hidden after a reload")
	}
}

func TestReloadDeferredDuringModal(t *testing.T) {
	m, _ := newTestModel(t, "# Work\n\n- [ ] a\n")
	m = press(m, "n") // open the add-task form
	if m.mode != modeForm {
		t.Fatal("precondition: form should be open")
	}
	m = send(m, fileReloadedMsg{doc: todo.Parse("# Work\n\n- [ ] a\n- [ ] b\n"), content: "x"})
	if m.mode != modeForm {
		t.Errorf("a reload must not close an open modal")
	}
	if find(m.doc, "b") != nil {
		t.Errorf("a reload must be deferred while a modal is open")
	}
}

func TestReloadIgnoresOwnWrite(t *testing.T) {
	m, _ := newTestModel(t, "# Work\n\n- [ ] a\n")
	m.lastContent = "the exact bytes we last wrote"
	m = send(m, fileReloadedMsg{doc: todo.Parse("# Work\n\n- [ ] zzz\n"), content: "the exact bytes we last wrote"})
	if find(m.doc, "zzz") != nil {
		t.Errorf("a reload matching our own last write must be ignored")
	}
}

func TestSaveUpdatesLastContent(t *testing.T) {
	m, path := newTestModel(t, "# Work\n\n- [ ] a\n")
	m = press(m, "down", "space") // toggle a done -> triggers a save
	if msg := reloadCheck(path, m.lastContent); msg != nil {
		t.Errorf("after our own save, a poll must see no change, got %#v", msg)
	}
}

func TestInitStartsPolling(t *testing.T) {
	m, _ := newTestModel(t, "# Work\n")
	if m.Init() == nil {
		t.Errorf("Init must start the file-poll loop")
	}
}

// hasRow reports whether a visible tree row has the given title.
func hasRow(m model, title string) bool {
	for _, r := range m.tree.rows {
		if r.item != nil && r.item.Title == title {
			return true
		}
	}
	return false
}
