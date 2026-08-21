package tui

import (
	"strings"
	"testing"
)

func TestSlashOpensSearch(t *testing.T) {
	m, _ := newTestModel(t, "# Work\n\n- [ ] a\n")
	m = press(m, "/")
	if !m.searching {
		t.Fatal("/ should open the search input")
	}
}

func TestSearchFiltersLive(t *testing.T) {
	m, _ := newTestModel(t, "# Work\n\n- [ ] Task A\n  - [ ] Explore catacombs\n  - [ ] Buy supplies\n- [ ] Task B\n")
	m = press(m, "/")
	m = typeText(m, "cat")
	got := rowTitles(m.tree)
	want := []string{"Work", "Task A", "Explore catacombs"}
	if !equalStrings(got, want) {
		t.Errorf("live-filtered rows = %v, want %v", got, want)
	}
}

func TestSearchEscClearsFilter(t *testing.T) {
	m, _ := newTestModel(t, "# Work\n\n- [ ] Task A\n  - [ ] Explore catacombs\n")
	before := len(m.tree.rows)
	m = press(m, "/")
	m = typeText(m, "cat")
	m = press(m, "esc")
	if m.searching {
		t.Error("esc should close the search input")
	}
	if m.tree.filter != "" {
		t.Errorf("esc should clear the filter, got %q", m.tree.filter)
	}
	if len(m.tree.rows) != before {
		t.Errorf("esc should restore the tree: got %d rows, want %d", len(m.tree.rows), before)
	}
}

func TestSearchEnterKeepsFilter(t *testing.T) {
	m, _ := newTestModel(t, "# Work\n\n- [ ] Task A\n  - [ ] Explore catacombs\n  - [ ] Buy supplies\n")
	m = press(m, "/")
	m = typeText(m, "cat")
	m = press(m, "enter")
	if m.searching {
		t.Error("enter should stop editing the query")
	}
	if m.tree.filter != "cat" {
		t.Errorf("enter should keep the filter active, got %q", m.tree.filter)
	}
	got := rowTitles(m.tree)
	want := []string{"Work", "Task A", "Explore catacombs"}
	if !equalStrings(got, want) {
		t.Errorf("filtered rows after enter = %v, want %v", got, want)
	}
}

func TestEscInMainClearsActiveFilter(t *testing.T) {
	m, _ := newTestModel(t, "# Work\n\n- [ ] Task A\n  - [ ] Explore catacombs\n")
	before := len(m.tree.rows)
	m = press(m, "/")
	m = typeText(m, "cat")
	m = press(m, "enter") // filter active, no longer editing
	m = press(m, "esc")   // esc in the main view clears the active filter (not quit)
	if m.tree.filter != "" {
		t.Errorf("esc in the main view should clear the active filter, got %q", m.tree.filter)
	}
	if len(m.tree.rows) != before {
		t.Errorf("clearing the filter should restore the tree: got %d rows, want %d", len(m.tree.rows), before)
	}
}

func TestSearchMatchesDescription(t *testing.T) {
	m, _ := newTestModel(t, "# Work\n\n- [ ] groceries\n    remember cat food\n- [ ] laundry\n")
	m = press(m, "/")
	m = typeText(m, "cat food")
	got := rowTitles(m.tree)
	if !contains(got, "groceries") {
		t.Errorf("a description match should keep the item visible, got %v", got)
	}
	if contains(got, "laundry") {
		t.Errorf("a non-matching item should be hidden, got %v", got)
	}
}

func TestSearchBarVisibleWhileTyping(t *testing.T) {
	m, _ := newTestModel(t, "# Work\n\n- [ ] a\n")
	m = press(m, "/")
	m = typeText(m, "xy")
	if out := m.View(); !strings.Contains(out, "xy") {
		t.Errorf("the search query should be visible in the view while typing")
	}
}

func TestFooterAdvertisesSearch(t *testing.T) {
	m, _ := newTestModel(t, "# Work\n")
	if got := m.footer(200); !strings.Contains(got, "Search") {
		t.Errorf("the footer should advertise the / search shortcut:\n%s", got)
	}
}
