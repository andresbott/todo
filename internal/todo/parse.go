package todo

import (
	"os"
	"regexp"
	"strings"
)

var (
	// headerRe matches a markdown header line: 1..6 leading '#', a space, then
	// the title (trimmed).
	headerRe = regexp.MustCompile(`^(#{1,6})[ \t]+(.*)$`)
	// checkboxRe matches a task line: optional indentation, a list bullet, a
	// `[ ]`/`[x]` box, then the title. Group 1 is the indentation, group 2 the
	// box char, group 3 the title.
	checkboxRe = regexp.MustCompile(`^([ \t]*)[-*+][ \t]+\[([ xX])\][ \t]?(.*)$`)
)

// Load reads and parses the TODO file at path. A missing file is not an error:
// it yields an empty document, so `todo new-file.md` starts a fresh list that
// is written on the first save.
func Load(path string) (*Document, error) {
	b, err := os.ReadFile(path) //nolint:gosec // path is a user-provided CLI argument; reading it is the app's purpose
	if err != nil {
		if os.IsNotExist(err) {
			return &Document{}, nil
		}
		return nil, err
	}
	return Parse(string(b)), nil
}

// Save writes the document back to path in canonical markdown form.
func (d *Document) Save(path string) error {
	return os.WriteFile(path, []byte(d.Render()), 0o644) //nolint:gosec // 0644 is intentional: todo files are meant to be user-readable and greppable
}

// Parse turns markdown source into a Document. Headers become nested
// categories; `- [ ]` items become tasks nested by indentation; indented
// non-checkbox text under a task becomes that task's description. Text before
// the first header/task is kept verbatim as the preamble; stray non-indented
// prose elsewhere is dropped (the app owns the file format).
func Parse(src string) *Document {
	doc := &Document{}
	lines := strings.Split(src, "\n")

	var catStack []*Item // open categories, by increasing Level
	type taskEntry struct {
		indent int
		item   *Item
	}
	var taskStack []taskEntry // open tasks, by increasing indentation
	var curTask *Item         // most recent task; receives description lines
	var curTaskIndent int
	var descBuf []string  // buffered description lines for curTask
	var preamble []string // verbatim lines before the first node
	seenNode := false

	flushDesc := func() {
		if curTask != nil {
			for len(descBuf) > 0 && strings.TrimSpace(descBuf[len(descBuf)-1]) == "" {
				descBuf = descBuf[:len(descBuf)-1]
			}
			if len(descBuf) > 0 {
				curTask.Description = dedent(descBuf)
			}
		}
		descBuf = nil
	}

	attach := func(it *Item) {
		if len(catStack) > 0 {
			catStack[len(catStack)-1].AppendChild(it)
		} else {
			doc.AppendRoot(it)
		}
	}

	for _, line := range lines {
		if m := headerRe.FindStringSubmatch(line); m != nil {
			flushDesc()
			curTask = nil
			taskStack = nil
			level := len(m[1])
			cat := &Item{Kind: Category, Level: level, Title: strings.TrimRight(m[2], " \t")}
			for len(catStack) > 0 && catStack[len(catStack)-1].Level >= level {
				catStack = catStack[:len(catStack)-1]
			}
			attach(cat)
			catStack = append(catStack, cat)
			seenNode = true
			continue
		}
		if m := checkboxRe.FindStringSubmatch(line); m != nil {
			flushDesc()
			indent := len(m[1])
			task := &Item{
				Kind:  Task,
				Title: strings.TrimRight(m[3], " \t"),
				Done:  m[2] == "x" || m[2] == "X",
			}
			for len(taskStack) > 0 && taskStack[len(taskStack)-1].indent >= indent {
				taskStack = taskStack[:len(taskStack)-1]
			}
			if len(taskStack) > 0 {
				taskStack[len(taskStack)-1].item.AppendChild(task)
			} else {
				attach(task)
			}
			taskStack = append(taskStack, taskEntry{indent: indent, item: task})
			curTask = task
			curTaskIndent = indent
			seenNode = true
			continue
		}
		// Neither a header nor a checkbox.
		if curTask != nil {
			if strings.TrimSpace(line) == "" {
				descBuf = append(descBuf, "")
				continue
			}
			if leadingWhitespace(line) > curTaskIndent {
				descBuf = append(descBuf, line)
				continue
			}
			// A dedented non-blank line closes the current task's description.
			flushDesc()
			curTask = nil
		}
		if !seenNode {
			preamble = append(preamble, line)
		}
		// Otherwise: stray prose between items after a task closed — dropped.
	}
	flushDesc()
	doc.Preamble = strings.Trim(strings.Join(preamble, "\n"), "\n")
	return doc
}

// leadingWhitespace counts the leading space/tab characters of s (each as one).
func leadingWhitespace(s string) int {
	return len(s) - len(strings.TrimLeft(s, " \t"))
}

// dedent strips the smallest common leading-whitespace prefix from the buffered
// description lines (blank lines ignored for the measurement, emitted empty)
// and joins them with newlines.
func dedent(lines []string) string {
	min := -1
	for _, l := range lines {
		if strings.TrimSpace(l) == "" {
			continue
		}
		if n := leadingWhitespace(l); min == -1 || n < min {
			min = n
		}
	}
	if min <= 0 {
		return strings.Join(lines, "\n")
	}
	out := make([]string, len(lines))
	for i, l := range lines {
		if strings.TrimSpace(l) == "" {
			out[i] = ""
		} else {
			out[i] = l[min:]
		}
	}
	return strings.Join(out, "\n")
}
