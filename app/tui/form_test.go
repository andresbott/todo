package tui

import (
	"strings"
	"testing"
)

func TestFormRing(t *testing.T) {
	task := newForm("", "", true, false)
	if len(task.ring()) != 4 {
		t.Errorf("task form should have 4 focus stops, got %d", len(task.ring()))
	}
	cat := newForm("", "", false, false)
	if len(cat.ring()) != 3 {
		t.Errorf("category form should have 3 focus stops (no description), got %d", len(cat.ring()))
	}
}

func TestFormFocusStepWraps(t *testing.T) {
	f := newForm("", "", true, false)
	if f.focus != focusTitle {
		t.Fatal("form should open focused on the title")
	}
	f.focusStep(-1) // wrap backwards from title -> cancel
	if f.focus != focusCancel {
		t.Errorf("stepping back from title should wrap to cancel, got %v", f.focus)
	}
}

func TestFormValuesTrim(t *testing.T) {
	f := newForm("  hi  ", "  body\n", true, true)
	title, desc := f.values()
	if title != "hi" {
		t.Errorf("title = %q, want trimmed 'hi'", title)
	}
	if desc != "  body" {
		t.Errorf("desc = %q, want trailing newline trimmed", desc)
	}
}

func TestFormViewTitles(t *testing.T) {
	f := newForm("", "", true, false)
	f.setWidth(100)
	if !strings.Contains(f.view(), "Add task") {
		t.Errorf("task add form should be titled 'Add task'")
	}
	if !strings.Contains(f.view(), "Description") {
		t.Errorf("task form should show the Description field")
	}
	cat := newForm("", "", false, true)
	cat.setWidth(100)
	if !strings.Contains(cat.view(), "Rename category") {
		t.Errorf("category edit form should be titled 'Rename category'")
	}
	if strings.Contains(cat.view(), "Description") {
		t.Errorf("category form should not show a Description field")
	}
}
