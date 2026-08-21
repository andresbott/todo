# todo

A minimalistic, keyboard-driven TODO manager for the terminal. It opens a plain
markdown file as a task list — headers are categories, `- [ ]` items are tasks,
and nested items are subtasks. Everything you do is written straight back to the
file, so your todos stay readable, greppable, and git-friendly.

Built with the [Charm](https://charm.sh) stack: [Bubble Tea](https://github.com/charmbracelet/bubbletea),
[Bubbles](https://github.com/charmbracelet/bubbles), and [Lipgloss](https://github.com/charmbracelet/lipgloss),
with [Cobra](https://github.com/spf13/cobra) for the CLI.

## Usage

```sh
todo                 # opens TODO.md in the current directory
todo path/to/file.md # opens a specific file
```

With no argument it defaults to `TODO.md`. If the file doesn't exist yet it
starts empty and is created on the first change.

The screen is a split pane: a collapsible **category/task tree** on the left and
the selected item's **details** (status, subtask progress, description) on the
right.

## Keys

| Key | Action |
|-----|--------|
| `↑` / `↓` (or `w`/`s`, `k`/`j`) | Move the selection |
| `←` / `→` (or `h`/`l`) | Collapse / expand (Left on a leaf jumps to the parent) |
| `enter` | Fold / unfold the selected item |
| `space` / `x` | Toggle a task done |
| `n` | New task — a task in the selected category, or a subtask of the selected task |
| `c` | Add a category (a subcategory when a category is selected) |
| `e` | Edit the selected item's title (and a task's description) |
| `d` | Delete the selected item (and its subtree) — asks to confirm |
| `q` / `esc` | Quit |

Tasks always live under a category — there are no root-level tasks. The tree
always ends with a **`+ new category`** row: select it and press `c` (or
`enter`) to add a top-level category. On an existing category, `c` adds a
subcategory instead.

In the add/edit dialog: `tab` moves between fields, `enter` saves (it inserts a
newline while you're in the description box), `esc` cancels.

## Completing a parent

Marking a parent task done completes **all** of its subtasks at once. If you did
it by accident, just unmark the parent again and the previous subtask states are
restored — that undo memory is kept in RAM for the session, so the file only
ever records the current checkbox states.

## File format

The app owns a small, standard subset of markdown:

```markdown
# Work

- [ ] Ship v1.0 release
  The description sits indented under the task and shows in the right pane.
  - [x] Write changelog
  - [ ] Cut the git tag
- [x] Fix login bug

## Backend

- [ ] Migrate the database
```

- **Headers** (`#`..`######`) are categories and nest by level.
- **`- [ ]` / `- [x]`** lines are tasks; indentation nests subtasks.
- **Indented text** under a task (that isn't a checkbox) is its description.
- Text before the first header/task is preserved as-is; other free-form prose
  between items is not part of the format.

## Development

```sh
make test    # run tests with coverage
make run     # run against example.md
make build   # build the ./todo binary
```
