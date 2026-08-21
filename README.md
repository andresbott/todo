# todo

A minimalistic, keyboard-driven TODO manager for the terminal. It opens a plain
markdown file as a task list — headers are categories, `- [ ]` items are tasks,
and nested items are subtasks. Everything you do is written straight back to the
file, so your todos stay readable, greppable, and git-friendly.

Built with the [Charm](https://charm.sh) stack: [Bubble Tea](https://github.com/charmbracelet/bubbletea),
[Bubbles](https://github.com/charmbracelet/bubbles), and [Lipgloss](https://github.com/charmbracelet/lipgloss),
with [Cobra](https://github.com/spf13/cobra) for the CLI.

## Install

todo is a single, dependency-free binary (pure Go, no CGO). Prebuilt archives, a
Debian package, and a macOS cask are published on every tagged release.

### macOS (Homebrew)

A macOS cask is published into this repository on every tagged release. Because
the repo isn't named `homebrew-*`, tap it with an explicit URL, then install:

```bash
brew tap andresbott/todo https://github.com/andresbott/todo
brew install --cask andresbott/todo/todo
```

`brew upgrade` tracks future releases. The binary isn't notarized, so the cask
strips the Gatekeeper quarantine flag on install — no "todo is damaged" prompt.

### Debian / Ubuntu

Download the `.deb` for your architecture from the
[releases page](https://github.com/andresbott/todo/releases) and install it:

```bash
sudo apt install ./todo_*_amd64.deb
```

### From source

With a Go 1.26+ toolchain installed:

```bash
go install github.com/andresbott/todo@latest
```

Or grab a prebuilt `tar.gz` (`.zip` on Windows) for your OS/arch from the
[releases page](https://github.com/andresbott/todo/releases).

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

An example task file:

```markdown
# Work

- [ ] Ship v1.0 release
  The release notes, tag, and announcement blog post.
  - [x] Write changelog
  - [ ] Cut the git tag
- [x] Fix login bug

## Backend

- [ ] Migrate the database

# Personal

- [ ] Renew passport
- [ ] Book dentist
```

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
| `D` | Remove every completed task (keeping any with unfinished sub-tasks) — asks to confirm |
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

## Live reload

todo re-reads the open file about once a second and applies any change made
outside the app — so you can edit the raw markdown in your editor, or let an
agent update it, and watch it update live. Your selection and which items are
folded are kept across a reload. The app writes atomically and ignores its own
saves, so editing inside todo and watching from outside never fight each other.

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

todo also keeps a short **guide block** — an HTML comment (so it stays invisible
when the markdown is rendered) — at the very top of every saved file. It links
back to this repo and documents the format above for whoever edits the file
directly, human or agent. It's app-managed: todo rewrites it on every save, so
there's no need to edit it by hand.

## Develop

Requires Go 1.26+. The full toolchain also needs `golangci-lint`, `goreleaser`,
and `go-licence-detector`. todo has no runtime dependencies — it's a single
static binary.

```bash
make help         # list all targets
make test         # run tests with coverage
make lint         # golangci-lint
make vet          # go vet
make coverage     # enforce the per-package coverage threshold
```

### Run

```bash
make run          # runs against example.md
# or plain go:
go run . path/to/file.md
```

### Build

```bash
make build        # goreleaser snapshot build for the current OS/arch → ./dist
# or plain go:
go build ./...
```

### Release

Releases are published by pushing a semver tag from a clean `main` branch. The
tag triggers the [Release workflow](./.github/workflows/release.yml), which runs
GoReleaser to build and publish the archives, `.deb`, and Homebrew cask.

```bash
make tag version="v1.2.3"
```

`make tag` refuses to run unless you're on `main` with a clean working tree, then
creates and pushes the `vX.Y.Z` tag.
