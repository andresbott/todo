package tui

import (
	"strings"
	"testing"

	"github.com/andresbott/todo/app/metainfo"
	"github.com/andresbott/todo/internal/todo"
)

func TestNavigateAndSelect(t *testing.T) {
	m, _ := newTestModel(t, "# Work\n\n- [ ] a\n- [ ] b\n")
	if got := m.tree.selected().Title; got != "Work" {
		t.Fatalf("start selection = %q, want Work", got)
	}
	m = press(m, "down")
	if got := m.tree.selected().Title; got != "a" {
		t.Fatalf("after down = %q, want a", got)
	}
	m = press(m, "down", "up")
	if got := m.tree.selected().Title; got != "a" {
		t.Fatalf("after down/up = %q, want a", got)
	}
}

func TestFoldHidesChildren(t *testing.T) {
	m, _ := newTestModel(t, "# Work\n\n- [ ] a\n")
	before := len(m.tree.rows)
	m = press(m, "enter") // collapse Work (selected)
	if len(m.tree.rows) != before-1 {
		t.Fatalf("collapse should hide the one child: rows %d -> %d", before, len(m.tree.rows))
	}
	m = press(m, "enter") // expand again
	if len(m.tree.rows) != before {
		t.Fatalf("expand should restore rows to %d, got %d", before, len(m.tree.rows))
	}
}

func TestToggleLeafPersists(t *testing.T) {
	m, path := newTestModel(t, "# Work\n\n- [ ] a\n")
	m = press(m, "down", "space")
	if !find(m.doc, "a").Done {
		t.Errorf("task should be done in memory")
	}
	if !strings.Contains(readFile(t, path), "- [x] a") {
		t.Errorf("done state not persisted:\n%s", readFile(t, path))
	}
}

func TestCascadeCompleteAndRevert(t *testing.T) {
	m, _ := newTestModel(t, "# Work\n\n- [ ] parent\n  - [ ] c1\n  - [x] c2\n")
	m = press(m, "down") // select parent

	// Mark the parent done: everything cascades to done.
	m = press(m, "space")
	for _, tt := range []string{"parent", "c1", "c2"} {
		if !find(m.doc, tt).Done {
			t.Fatalf("%s should be done after cascade", tt)
		}
	}

	// Unmark the parent: children restore to the prior mixed state.
	m = press(m, "space")
	if find(m.doc, "parent").Done {
		t.Errorf("parent should be undone")
	}
	if find(m.doc, "c1").Done {
		t.Errorf("c1 should be restored to not-done")
	}
	if !find(m.doc, "c2").Done {
		t.Errorf("c2 should be restored to done (its prior state)")
	}
}

func TestManualChildToggleInvalidatesSnapshot(t *testing.T) {
	m, _ := newTestModel(t, "# Work\n\n- [ ] parent\n  - [ ] c1\n  - [ ] c2\n")
	m = press(m, "down")  // parent
	m = press(m, "space") // cascade all done (snapshot: all were false)

	m = press(m, "down")  // c1
	m = press(m, "space") // manual toggle c1 off -> invalidates parent's snapshot

	m = press(m, "up")    // parent
	m = press(m, "space") // unmark parent (no snapshot now)

	if find(m.doc, "parent").Done {
		t.Errorf("parent should be undone")
	}
	if find(m.doc, "c1").Done {
		t.Errorf("c1 was manually turned off, should stay off")
	}
	if !find(m.doc, "c2").Done {
		t.Errorf("c2 should still be done — the stale snapshot must not have restored it to false")
	}
}

func TestToggleCategoryIsNoOp(t *testing.T) {
	m, _ := newTestModel(t, "# Work\n\n- [ ] a\n")
	m = press(m, "space") // Work category is selected
	if m.status == "" {
		t.Errorf("expected a status message explaining categories can't be completed")
	}
}

func TestNewTaskUnderCategory(t *testing.T) {
	m, path := newTestModel(t, "# Work\n\n- [ ] a\n")
	// Work is selected; 'n' adds a task under it.
	m = press(m, "n")
	if m.mode != modeForm {
		t.Fatal("new should open the form")
	}
	m = typeText(m, "b")
	m = press(m, "enter")
	if m.mode != modeMain {
		t.Fatal("submit should return to the main view")
	}
	b := find(m.doc, "b")
	if b == nil || b.Parent == nil || b.Parent.Title != "Work" {
		t.Fatalf("task b should be a child of Work")
	}
	if m.tree.selected() != b {
		t.Errorf("the new task should be selected")
	}
	if !strings.Contains(readFile(t, path), "- [ ] b") {
		t.Errorf("new task not persisted")
	}
}

func TestNewOnTaskMakesSubtask(t *testing.T) {
	m, path := newTestModel(t, "# Work\n\n- [ ] a\n")
	m = press(m, "down") // task a
	m = press(m, "N")    // 'N' on a task adds a child subtask
	m = typeText(m, "sub")
	m = press(m, "enter")
	a := find(m.doc, "a")
	if len(a.Children) != 1 || a.Children[0].Title != "sub" {
		t.Fatalf("sub should be a child of a")
	}
	if !strings.Contains(readFile(t, path), "  - [ ] sub") {
		t.Errorf("subtask indentation not persisted:\n%s", readFile(t, path))
	}
}

func TestNewSiblingOnTask(t *testing.T) {
	m, path := newTestModel(t, "# Work\n\n- [ ] a\n- [ ] b\n")
	m = press(m, "down") // task a — not the last sibling
	m = press(m, "n")    // 'n' on a task adds a sibling at the end of the level
	if m.mode != modeForm {
		t.Fatal("n should open the add-task form")
	}
	m = typeText(m, "c")
	m = press(m, "enter")
	c := find(m.doc, "c")
	if c == nil || c.Parent == nil || c.Parent.Title != "Work" {
		t.Fatalf("c should be a task directly under Work, a sibling of a")
	}
	// The sibling must land at the END of the level (after b), not right after a.
	var order []string
	for _, ch := range find(m.doc, "Work").Children {
		order = append(order, ch.Title)
	}
	if got := strings.Join(order, ","); got != "a,b,c" {
		t.Errorf("sibling order = %q, want \"a,b,c\"", got)
	}
	if m.tree.selected() != c {
		t.Errorf("the new sibling should be selected")
	}
	if !strings.Contains(readFile(t, path), "- [ ] c") {
		t.Errorf("new sibling not persisted")
	}
}

func TestNewChildOnCategory(t *testing.T) {
	// A task can't be a category's sibling, so on a category 'N' (like 'n') adds
	// the task inside the category.
	m, _ := newTestModel(t, "# Work\n\n- [ ] a\n")
	m = press(m, "N") // Work is selected
	m = typeText(m, "b")
	m = press(m, "enter")
	b := find(m.doc, "b")
	if b == nil || b.Parent == nil || b.Parent.Title != "Work" {
		t.Fatalf("b should be a task inside Work")
	}
}

func TestNewOnPlaceholderAddsRootCategory(t *testing.T) {
	// An empty document starts on the placeholder row; 'n' there adds a
	// top-level category (there are no root-level tasks).
	m, _ := newTestModel(t, "")
	if !m.tree.onPlaceholder() {
		t.Fatal("an empty document should start on the placeholder row")
	}
	m = press(m, "n")
	if m.mode != modeForm {
		t.Fatal("n on the placeholder should open the category form")
	}
	m = typeText(m, "Work")
	m = press(m, "enter")
	if w := find(m.doc, "Work"); w == nil || w.Parent != nil || w.Level != 1 {
		t.Fatalf("Work should be a top-level category, got %+v", w)
	}
}

func TestAddSecondRootCategory(t *testing.T) {
	// The reported bug: with one root category you couldn't add a second.
	m, _ := newTestModel(t, "# Work\n\n- [ ] a\n")
	for !m.tree.onPlaceholder() { // walk down to the placeholder
		m = press(m, "down")
	}
	m = press(m, "c")
	if m.mode != modeForm {
		t.Fatal("c on the placeholder should open the category form")
	}
	m = typeText(m, "Personal")
	m = press(m, "enter")
	p := find(m.doc, "Personal")
	if p == nil || p.Parent != nil || p.Level != 1 {
		t.Fatalf("Personal should be a second top-level (H1) category, got %+v", p)
	}
	if len(m.doc.Roots) != 2 {
		t.Errorf("expected 2 root categories, got %d", len(m.doc.Roots))
	}
	if m.tree.selected() != p {
		t.Errorf("the new category should be selected")
	}
}

func TestNewTaskInsertedBeforeSubcategory(t *testing.T) {
	// Adding a task to a category that already has a subcategory must place the
	// task before the subheader, so it stays under that category on reload.
	m, path := newTestModel(t, "# Work\n\n## Backend\n\n- [ ] deep\n")
	m = press(m, "n") // Work is selected
	m = typeText(m, "shallow")
	m = press(m, "enter")
	if s := find(m.doc, "shallow"); s == nil || s.Parent == nil || s.Parent.Title != "Work" {
		t.Fatalf("shallow should be a task directly under Work")
	}
	reloaded := todo.Parse(readFile(t, path))
	s := find(reloaded, "shallow")
	if s == nil || s.Parent == nil || s.Parent.Title != "Work" {
		t.Errorf("after reload shallow must still be under Work, not Backend:\n%s", readFile(t, path))
	}
}

func TestAddSubcategory(t *testing.T) {
	m, _ := newTestModel(t, "# Work\n\n- [ ] a\n")
	m = press(m, "c") // Work selected -> subcategory
	m = typeText(m, "Backend")
	m = press(m, "enter")
	be := find(m.doc, "Backend")
	if be == nil || be.Level != 2 || be.Parent == nil || be.Parent.Title != "Work" {
		t.Fatalf("Backend should be an H2 under Work, got %+v", be)
	}
}

func TestEditTaskTitleAndDescription(t *testing.T) {
	m, path := newTestModel(t, "# Work\n\n- [ ] a\n")
	m = press(m, "down") // task a
	m = press(m, "e")
	if m.mode != modeForm {
		t.Fatal("edit should open the form")
	}
	m = typeText(m, "-done") // title becomes "a-done"
	m = press(m, "tab")      // focus description
	m = typeText(m, "some notes")
	m = press(m, "tab")   // focus Save
	m = press(m, "enter") // submit
	it := find(m.doc, "a-done")
	if it == nil {
		t.Fatalf("edited title not found:\n%s", readFile(t, path))
	}
	if it.Description != "some notes" {
		t.Errorf("description = %q, want %q", it.Description, "some notes")
	}
	if !strings.Contains(readFile(t, path), "some notes") {
		t.Errorf("description not persisted")
	}
}

func TestEmptyTitleIgnored(t *testing.T) {
	m, _ := newTestModel(t, "# Work\n\n- [ ] a\n")
	rootsBefore := len(m.doc.Roots)
	m = press(m, "n")
	m = press(m, "enter") // submit with an empty title
	if m.mode != modeForm {
		t.Errorf("empty submit should keep the form open")
	}
	if len(m.doc.Roots) != rootsBefore {
		t.Errorf("empty submit should not add anything")
	}
}

func TestEscCancelsForm(t *testing.T) {
	m, _ := newTestModel(t, "# Work\n\n- [ ] a\n")
	m = press(m, "n")
	m = typeText(m, "b")
	m = press(m, "esc")
	if m.mode != modeMain {
		t.Errorf("esc should close the form")
	}
	if find(m.doc, "b") != nil {
		t.Errorf("cancelled task should not be added")
	}
}

func TestDeleteTaskConfirmed(t *testing.T) {
	m, path := newTestModel(t, "# Work\n\n- [ ] a\n- [ ] b\n")
	m = press(m, "down") // task a
	m = press(m, "d")
	if m.mode != modeConfirm {
		t.Fatal("d should open the confirm dialog")
	}
	m = press(m, "y")
	if m.mode != modeMain {
		t.Fatal("confirming should return to the main view")
	}
	if find(m.doc, "a") != nil {
		t.Errorf("task a should be gone")
	}
	if find(m.doc, "b") == nil {
		t.Errorf("task b should remain")
	}
	if strings.Contains(readFile(t, path), "- [ ] a") {
		t.Errorf("deleted task should not be in the file:\n%s", readFile(t, path))
	}
}

func TestDeleteDefaultsToCancel(t *testing.T) {
	m, _ := newTestModel(t, "# Work\n\n- [ ] a\n")
	m = press(m, "down") // task a
	m = press(m, "d")
	if m.confirmOnDelete {
		t.Errorf("focus should default to Cancel")
	}
	if !strings.Contains(m.View(), "Confirm delete") {
		t.Errorf("confirm dialog should be visible")
	}
	m = press(m, "enter") // enter on the default (Cancel)
	if m.mode != modeMain {
		t.Errorf("enter on Cancel should close the dialog")
	}
	if find(m.doc, "a") == nil {
		t.Errorf("task a should survive a cancelled delete")
	}
}

func TestDeleteCancelledByEsc(t *testing.T) {
	m, _ := newTestModel(t, "# Work\n\n- [ ] a\n")
	m = press(m, "down", "d", "esc")
	if m.mode != modeMain {
		t.Errorf("esc should close the dialog")
	}
	if find(m.doc, "a") == nil {
		t.Errorf("task should survive esc")
	}
}

func TestDeleteCategorySubtree(t *testing.T) {
	m, path := newTestModel(t, "# Work\n\n- [ ] a\n  - [ ] a1\n")
	// Work is selected; deleting it removes the whole subtree.
	m = press(m, "d", "y")
	if len(m.doc.Roots) != 0 {
		t.Errorf("deleting the only category should empty the document")
	}
	// The file keeps the managed guide but holds no task/category content.
	on := readFile(t, path)
	if !strings.Contains(on, "<!-- todo:guide") {
		t.Errorf("an emptied file should still carry the guide, got %q", on)
	}
	if reloaded := todo.Parse(on); len(reloaded.Roots) != 0 || reloaded.Preamble != "" {
		t.Errorf("an emptied file should reload to an empty document, got %d roots / preamble %q", len(reloaded.Roots), reloaded.Preamble)
	}
}

func TestDeleteNoOpWhenEmpty(t *testing.T) {
	m, _ := newTestModel(t, "")
	m = press(m, "d")
	if m.mode == modeConfirm {
		t.Errorf("delete with nothing selected should be a no-op")
	}
}

func TestClearDoneConfirmed(t *testing.T) {
	m, path := newTestModel(t, "# Work\n\n- [x] a\n- [ ] b\n- [x] c\n")
	m = press(m, "D")
	if m.mode != modeConfirm {
		t.Fatal("D should open the confirm dialog")
	}
	m = press(m, "y")
	if m.mode != modeMain {
		t.Fatal("confirming should return to the main view")
	}
	if find(m.doc, "a") != nil || find(m.doc, "c") != nil {
		t.Errorf("completed tasks a and c should be gone")
	}
	if find(m.doc, "b") == nil {
		t.Errorf("open task b should remain")
	}
	on := readFile(t, path)
	if strings.Contains(on, "- [x] a") || strings.Contains(on, "- [x] c") {
		t.Errorf("cleared tasks should not be in the file:\n%s", on)
	}
	if !strings.Contains(on, "- [ ] b") {
		t.Errorf("open task should be persisted:\n%s", on)
	}
	_ = m.View() // the post-clear view must render without panicking
}

func TestClearDoneDefaultsToCancel(t *testing.T) {
	m, _ := newTestModel(t, "# Work\n\n- [x] a\n- [x] c\n- [ ] b\n")
	m = press(m, "D")
	if m.confirmOnDelete {
		t.Errorf("focus should default to Cancel")
	}
	if out := m.View(); !strings.Contains(out, "Remove 2 completed task") {
		t.Errorf("confirm should show how many tasks will be removed:\n%s", out)
	}
	m = press(m, "enter") // enter on the default (Cancel)
	if m.mode != modeMain {
		t.Errorf("enter on Cancel should close the dialog")
	}
	if find(m.doc, "a") == nil {
		t.Errorf("completed task should survive a cancelled clear")
	}
}

func TestClearDoneCancelledByEsc(t *testing.T) {
	m, _ := newTestModel(t, "# Work\n\n- [x] a\n- [ ] b\n")
	m = press(m, "D", "esc")
	if m.mode != modeMain {
		t.Errorf("esc should close the dialog")
	}
	if find(m.doc, "a") == nil {
		t.Errorf("completed task should survive esc")
	}
}

func TestClearDoneNoOpWhenNothingDone(t *testing.T) {
	m, _ := newTestModel(t, "# Work\n\n- [ ] a\n- [ ] b\n")
	m = press(m, "D")
	if m.mode == modeConfirm {
		t.Errorf("D with no completed tasks should not open a confirm")
	}
	if m.status == "" {
		t.Errorf("expected a status message explaining there is nothing to remove")
	}
}

func TestDeleteAfterCancelledClear(t *testing.T) {
	// A cancelled clear must reset the confirm action, otherwise the next single
	// delete would wrongly clear every completed task.
	m, _ := newTestModel(t, "# Work\n\n- [x] a\n- [x] c\n- [ ] b\n")
	m = press(m, "D", "esc") // open the clear-done confirm, then cancel it
	m = press(m, "down")     // select task a
	m = press(m, "d", "y")   // delete only a
	if find(m.doc, "a") != nil {
		t.Errorf("task a should be deleted")
	}
	if find(m.doc, "c") == nil {
		t.Errorf("c must survive — a single delete must not clear all completed tasks")
	}
}

func TestFooterShowsClearDoneHint(t *testing.T) {
	m, _ := newTestModel(t, "# Work\n\n- [x] a\n")
	if got := m.footer(120); !strings.Contains(got, "Clear") {
		t.Errorf("footer should advertise the clear-done shortcut:\n%s", got)
	}
}

func TestHeaderShowsVersion(t *testing.T) {
	m, _ := newTestModel(t, "# Work\n")
	if !strings.Contains(m.header(80), metainfo.Version) {
		t.Errorf("header should show the version %q", metainfo.Version)
	}
}

func TestViewSmoke(t *testing.T) {
	m, _ := newTestModel(t, "# Work\n\n- [ ] alpha\n")
	out := m.View()
	for _, want := range []string{"todo", "Tasks", "Details", "Work", "alpha"} {
		if !strings.Contains(out, want) {
			t.Errorf("main view missing %q", want)
		}
	}
	m = press(m, "n")
	if !strings.Contains(m.View(), "Add task") {
		t.Errorf("form view should show the Add task title")
	}
}
