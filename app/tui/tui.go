// Package tui is the Bubble Tea frontend of the TODO app: a split-pane view
// with a category/task tree on the left and the selected item's details on the
// right, an add/edit modal, and the keybindings that drive them. All domain
// logic lives in internal/todo; this package only renders it and routes keys.
package tui

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/andresbott/todo/app/metainfo"
	"github.com/andresbott/todo/internal/todo"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

type mode int

const (
	modeMain mode = iota
	modeForm
	modeConfirm
)

// formTarget records what submitting the open form should do.
type formTarget int

const (
	// targetAddTask adds a task under the selected item: a task in the selected
	// category, or a subtask of the selected task. Tasks always live under a
	// category, so this needs a selection (there are no root-level tasks).
	targetAddTask     formTarget = iota
	targetAddCategory            // a category (a subcategory when a category is selected)
	targetEdit                   // rename/re-describe the selected item
)

type model struct {
	path    string
	version string
	doc     *todo.Document
	tree    tree
	width   int
	height  int

	mode     mode
	form     form
	target   formTarget
	editing  *todo.Item // the item being edited, for targetEdit
	deleting *todo.Item // the item the confirm dialog is guarding, for modeConfirm
	// confirmOnDelete is which confirm-dialog button has focus: false = Cancel
	// (the safe default so a stray key can't delete), true = Delete.
	confirmOnDelete bool

	// snapshots holds, per cascade-completed parent task, the descendant Done
	// states captured at completion time, so unchecking that parent can restore
	// them. Session-only: it lives here in memory and is never written to disk.
	snapshots map[*todo.Item]todo.DoneStates

	status string // transient one-line footer message
	err    error  // last save error, shown in the footer
}

// Run loads the TODO file at path and starts the interactive TUI.
func Run(path string) error {
	m, err := newModel(path)
	if err != nil {
		return err
	}
	_, err = tea.NewProgram(m, tea.WithAltScreen()).Run()
	return err
}

// newModel loads the file at path into a fresh model (main view, empty undo
// snapshots). Split out from Run so tests can drive the model without starting
// a real terminal program.
func newModel(path string) (model, error) {
	doc, err := todo.Load(path)
	if err != nil {
		return model{}, err
	}
	return model{
		path:      path,
		version:   metainfo.Version,
		doc:       doc,
		tree:      newTree(doc),
		mode:      modeMain,
		snapshots: map[*todo.Item]todo.DoneStates{},
	}, nil
}

func (m model) Init() tea.Cmd { return nil }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		if m.mode == modeForm {
			m.form.setWidth(msg.Width)
		}
		return m, nil
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
		switch m.mode {
		case modeForm:
			return m.updateForm(msg)
		case modeConfirm:
			return m.updateConfirm(msg)
		default:
			return m.updateMain(msg)
		}
	}
	// Non-key messages (cursor blink, etc.) drive the form's inputs.
	if m.mode == modeForm {
		var cmd tea.Cmd
		m.form, cmd = m.form.update(msg)
		return m, cmd
	}
	return m, nil
}

func (m model) updateMain(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	m.status = ""
	// On the "+ new category" placeholder row the only meaningful action is
	// adding a top-level category; the item-scoped keys are no-ops there.
	onPH := m.tree.onPlaceholder()
	switch msg.String() {
	case "q", "esc":
		return m, tea.Quit
	case "up", "w", "k":
		m.tree.moveUp()
	case "down", "s", "j":
		m.tree.moveDown()
	case "left", "h":
		m.tree.collapse()
	case "right", "l":
		m.tree.expand()
	case "enter":
		if onPH {
			return m.openForm(targetAddCategory)
		}
		m.tree.toggleFold()
	case " ", "x":
		if onPH {
			return m, nil
		}
		return m.toggleDone()
	case "n":
		if onPH {
			return m.openForm(targetAddCategory)
		}
		return m.openForm(targetAddTask)
	case "c":
		return m.openForm(targetAddCategory)
	case "e":
		if onPH {
			return m, nil
		}
		return m.openForm(targetEdit)
	case "d":
		if onPH {
			return m, nil
		}
		return m.openDelete()
	}
	return m, nil
}

// openDelete opens the confirm dialog for the selected item (focused on the
// safe Cancel button). A no-op when nothing is selected.
func (m model) openDelete() (tea.Model, tea.Cmd) {
	sel := m.tree.selected()
	if sel == nil || m.tree.onPlaceholder() {
		return m, nil
	}
	m.deleting = sel
	m.confirmOnDelete = false
	m.mode = modeConfirm
	return m, nil
}

// updateConfirm handles the delete confirmation: y/enter-on-Delete confirms,
// n/esc cancels, tab/arrows toggle the focused button.
func (m model) updateConfirm(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "Y":
		return m.doDelete()
	case "n", "N", "esc":
		m.mode = modeMain
		return m, nil
	case "left", "right", "tab", "shift+tab":
		m.confirmOnDelete = !m.confirmOnDelete
		return m, nil
	case "enter", " ":
		if m.confirmOnDelete {
			return m.doDelete()
		}
		m.mode = modeMain
		return m, nil
	}
	return m, nil
}

// doDelete removes the confirmed item (and its subtree) from the document,
// persists, and moves the selection to a sensible neighbour — the row above the
// deleted item, which is never one of its descendants.
func (m model) doDelete() (tea.Model, tea.Cmd) {
	it := m.deleting
	m.deleting = nil
	m.mode = modeMain
	if it == nil {
		return m, nil
	}
	var prev *todo.Item
	if m.tree.cursor > 0 {
		prev = m.tree.rows[m.tree.cursor-1].item
	}
	m.doc.Remove(it)
	// A structural change ends the accidental-complete undo window.
	m.snapshots = map[*todo.Item]todo.DoneStates{}
	m.save()
	m.tree.rebuild()
	if prev != nil {
		m.tree.selectItem(prev)
	} else {
		m.tree.cursor = 0
	}
	return m, nil
}

// toggleDone flips the selected task's completion. Marking a parent done
// snapshots its subtree and cascades completion to all descendants; unmarking a
// parent restores that snapshot if one exists (the accidental-complete undo).
// A manual toggle invalidates any ancestor's snapshot, since reverting it later
// would fight this explicit change.
func (m model) toggleDone() (tea.Model, tea.Cmd) {
	it := m.tree.selected()
	if it == nil || !it.IsTask() {
		m.status = "Only tasks can be completed."
		return m, nil
	}
	for p := it.Parent; p != nil; p = p.Parent {
		delete(m.snapshots, p)
	}
	switch {
	case it.Done:
		if snap, ok := m.snapshots[it]; ok {
			todo.RestoreDone(snap)
			delete(m.snapshots, it)
		} else {
			it.Done = false
		}
	case len(it.Children) > 0:
		m.snapshots[it] = todo.SnapshotDone(it)
		todo.CascadeSetDone(it, true)
	default:
		it.Done = true
	}
	m.save()
	m.tree.rebuild()
	return m, nil
}

// openForm prepares and opens the add/edit modal for the given target.
func (m model) openForm(t formTarget) (tea.Model, tea.Cmd) {
	sel := m.tree.selected()
	switch t {
	case targetAddTask:
		if sel == nil || m.tree.onPlaceholder() {
			m.status = "Add a category first (press c) — tasks live under a category."
			return m, nil
		}
		m.form = newForm("", "", true, false)
	case targetEdit:
		if sel == nil || m.tree.onPlaceholder() {
			return m, nil
		}
		m.editing = sel
		m.form = newForm(sel.Title, sel.Description, sel.IsTask(), true)
	case targetAddCategory:
		m.form = newForm("", "", false, false)
	}
	m.target = t
	m.form.setWidth(m.width)
	m.mode = modeForm
	return m, textinput.Blink
}

func (m model) updateForm(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.mode = modeMain
		return m, nil
	case "tab":
		return m, m.form.focusStep(1)
	case "shift+tab":
		return m, m.form.focusStep(-1)
	case "left":
		if m.form.focus == focusCancel {
			m.form.focus = focusSave
			return m, m.form.applyFocus()
		}
	case "right":
		if m.form.focus == focusSave {
			m.form.focus = focusCancel
			return m, m.form.applyFocus()
		}
	case "enter":
		// Enter in the description inserts a newline; on the title or Save it
		// submits, on Cancel it closes.
		if m.form.focus == focusDesc {
			break
		}
		if m.form.focus == focusCancel {
			m.mode = modeMain
			return m, nil
		}
		return m.submitForm()
	}
	var cmd tea.Cmd
	m.form, cmd = m.form.update(msg)
	return m, cmd
}

// submitForm applies the open form: editing the selected item, or inserting a
// new task/subtask/category at the right spot. An empty title is ignored (the
// dialog stays open). Any structural or content change clears the undo
// snapshots — the accidental-complete window is meant to be immediate.
func (m model) submitForm() (tea.Model, tea.Cmd) {
	title, desc := m.form.values()
	if title == "" {
		return m, nil
	}
	sel := m.tree.selected()
	var focus *todo.Item

	switch m.target {
	case targetEdit:
		m.editing.Title = title
		if m.editing.IsTask() {
			m.editing.Description = desc
		}
		focus = m.editing
	case targetAddCategory:
		cat := &todo.Item{Kind: todo.Category, Title: title, Level: 1}
		// A category selected → subcategory; the root placeholder (or a task
		// selected) → a new top-level category.
		if !m.tree.onPlaceholder() && sel != nil && sel.Kind == todo.Category {
			if cat.Level = sel.Level + 1; cat.Level > 6 {
				cat.Level = 6
			}
			sel.AppendChild(cat)
			delete(m.tree.collapsed, sel)
		} else {
			m.doc.AppendRoot(cat)
		}
		focus = cat
	default: // targetAddTask: a task under the selected category, or a subtask
		task := todo.NewTask(title, desc, false)
		sel.AppendTask(task)
		delete(m.tree.collapsed, sel)
		focus = task
	}

	m.snapshots = map[*todo.Item]todo.DoneStates{}
	m.mode = modeMain
	m.save()
	m.tree.rebuild()
	if focus != nil {
		m.tree.selectItem(focus)
	}
	return m, nil
}

// save writes the document to disk, recording any error for the footer.
func (m *model) save() {
	if err := m.doc.Save(m.path); err != nil {
		m.err = err
		return
	}
	m.err = nil
}

func (m model) View() string {
	switch m.mode {
	case modeForm:
		return placeCenter(m.mainView(true), m.form.view())
	case modeConfirm:
		return placeCenter(m.mainView(true), confirmModal(deletionQuestion(m.deleting), m.confirmOnDelete, m.width))
	default:
		return m.mainView(false)
	}
}

// mainView composes the header, the two panels, and the footer to the terminal
// size. When dim is true the panels render unfocused (the dimmed backdrop
// behind the modal).
func (m model) mainView(dim bool) string {
	w, h := m.width, m.height
	if w == 0 {
		w, h = 80, 24
	}
	bodyH := h - 2
	if bodyH < 3 {
		bodyH = 3
	}
	leftW := w / 3
	if leftW < 20 {
		leftW = 20
	}
	if leftW > 48 {
		leftW = 48
	}
	rightW := w - leftW

	left := titledBox("Tasks", m.tree.view(leftW-2, bodyH-2), leftW, bodyH, !dim)
	right := titledBox("Details", m.detailsView(rightW-2, bodyH-2), rightW, bodyH, false)
	panels := lipgloss.JoinHorizontal(lipgloss.Top, left, right)

	return m.header(w) + "\n" + panels + "\n" + m.footer(w)
}

func (m model) header(width int) string {
	done, total := overallCounts(m.doc)
	left := headerAppStyle.Render("todo") + " " + headerHintSty.Render(m.version) + "  " + headerHintSty.Render(filepath.Base(m.path))
	right := headerHintSty.Render(fmt.Sprintf("%d/%d done", done, total))
	gap := width - 1 - lipgloss.Width(left) - lipgloss.Width(right)
	if gap < 1 {
		gap = 1
	}
	return ansi.Truncate(" "+left+strings.Repeat(" ", gap)+right, width, "")
}

func (m model) footer(width int) string {
	switch {
	case m.err != nil:
		return ansi.Truncate(" "+errStyle.Render("save failed: "+m.err.Error()), width, "")
	case m.status != "":
		return ansi.Truncate(" "+helpTextStyle.Render(m.status), width, "")
	}
	parts := []string{
		hint("↑↓", "Move"), hint("←→", "Fold"), hint("space", "Done"),
		hint("n", "New"), hint("c", "Cat"), hint("e", "Edit"), hint("d", "Del"), hint("q", "Quit"),
	}
	return ansi.Truncate(" "+strings.Join(parts, "  "), width, "")
}

// detailsView renders the right pane for the selected item: a category's
// progress, or a task's status, subtask progress, and description.
func (m model) detailsView(width, height int) string {
	if m.tree.onPlaceholder() {
		return " " + helpTextStyle.Render("Add a top-level category here — press c or enter.")
	}
	it := m.tree.selected()
	if it == nil {
		return " " + helpTextStyle.Render("Nothing selected — press c to add a category.")
	}
	var b strings.Builder
	if it.Kind == todo.Category {
		done, total := it.TaskCounts()
		b.WriteString(" " + categoryStyle.Render(it.Title))
		b.WriteString("\n\n " + labelStyle.Render("Category"))
		_, _ = fmt.Fprintf(&b, "\n %d of %d tasks done", done, total)
		return b.String()
	}

	status := helpTextStyle.Render("○ open")
	if it.Done {
		status = doneStyle.Render("● done")
	}
	b.WriteString(" " + status)
	b.WriteString("\n\n " + categoryStyle.Render(it.Title))
	if done, total := it.TaskCounts(); total > 0 {
		b.WriteString("\n " + labelStyle.Render("Subtasks") + fmt.Sprintf(" %d/%d done", done, total))
	}
	b.WriteString("\n\n")
	if it.Description != "" {
		b.WriteString(wrapText(it.Description, width-1))
	} else {
		b.WriteString(" " + helpTextStyle.Render("No description. Press e to add one."))
	}
	return b.String()
}

// wrapText word-wraps s to width and gives every line a one-column left margin.
func wrapText(s string, width int) string {
	if width < 4 {
		width = 4
	}
	wrapped := lipgloss.NewStyle().Width(width).Render(s)
	lines := strings.Split(wrapped, "\n")
	for i := range lines {
		lines[i] = " " + lines[i]
	}
	return strings.Join(lines, "\n")
}

// overallCounts totals done/total tasks across the whole document.
func overallCounts(d *todo.Document) (done, total int) {
	for _, r := range d.Roots {
		if r.Kind == todo.Task {
			total++
			if r.Done {
				done++
			}
		}
		dn, tt := r.TaskCounts()
		done += dn
		total += tt
	}
	return done, total
}
