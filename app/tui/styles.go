package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

// Palette (256-colour), mirroring the dibs TUI so the two apps feel of a piece.
var (
	colAccent = lipgloss.Color("205") // hot pink: focus border + key hints + selection
	colDim    = lipgloss.Color("240") // gray: unfocused borders, labels, hints
	colTitle  = lipgloss.Color("141") // purple: app name in the header
	colDone   = lipgloss.Color("10")  // green: a completed task's checkbox
)

var (
	titleActiveStyle = lipgloss.NewStyle().Foreground(colAccent).Bold(true)
	titleDimStyle    = lipgloss.NewStyle().Foreground(colDim)
	selectedRowStyle = lipgloss.NewStyle().Foreground(colAccent).Bold(true)
	categoryStyle    = lipgloss.NewStyle().Foreground(colTitle).Bold(true)
	doneStyle        = lipgloss.NewStyle().Foreground(colDone)
	doneTitleStyle   = lipgloss.NewStyle().Foreground(colDim).Strikethrough(true)
	labelStyle       = lipgloss.NewStyle().Faint(true)
	errStyle         = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))

	helpKeyStyle   = lipgloss.NewStyle().Foreground(colAccent).Bold(true)
	helpTextStyle  = lipgloss.NewStyle().Foreground(colDim)
	headerAppStyle = lipgloss.NewStyle().Foreground(colTitle).Bold(true)
	headerHintSty  = lipgloss.NewStyle().Foreground(colDim)

	focusLabelStyle = lipgloss.NewStyle().Foreground(colAccent)
)

// hint renders one "key: label" help entry — key accented, ": label" dim.
func hint(k, label string) string {
	return helpKeyStyle.Render(k) + helpTextStyle.Render(": "+label)
}

// titledBox renders a rounded-border box of exactly width x height cells with
// title embedded in the top border and body clipped/padded to fit. Border and
// title use the accent colour when focused, dim otherwise. Adapted verbatim in
// spirit from the dibs TUI.
func titledBox(title, body string, width, height int, focused bool) string {
	if width < 4 {
		width = 4
	}
	if height < 3 {
		height = 3
	}
	iw := width - 2
	ih := height - 2

	border := lipgloss.NewStyle().Foreground(colDim)
	ts := titleDimStyle
	if focused {
		border = lipgloss.NewStyle().Foreground(colAccent)
		ts = titleActiveStyle
	}

	label := " " + title + " "
	maxLabel := iw - 3
	if maxLabel < 0 {
		maxLabel = 0
	}
	if lipgloss.Width(label) > maxLabel {
		label = ansi.Truncate(label, maxLabel, "")
	}
	fill := iw - lipgloss.Width(label) - 1
	if fill < 0 {
		fill = 0
	}
	top := border.Render("╭─") + ts.Render(label) + border.Render(strings.Repeat("─", fill)+"╮")

	lines := strings.Split(body, "\n")
	rows := make([]string, 0, ih)
	for i := 0; i < ih; i++ {
		content := ""
		if i < len(lines) {
			content = lines[i]
		}
		content = ansi.Truncate(content, iw, "")
		if pad := iw - lipgloss.Width(content); pad > 0 {
			content += strings.Repeat(" ", pad)
		}
		rows = append(rows, border.Render("│")+content+border.Render("│"))
	}

	bottom := border.Render("╰" + strings.Repeat("─", iw) + "╯")
	return strings.Join(append(append([]string{top}, rows...), bottom), "\n")
}
