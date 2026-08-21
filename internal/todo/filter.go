package todo

import "strings"

// Matches reports whether query occurs, case-insensitively, in the item's
// title — or, for a task, in its description. The query is a plain substring
// (grep-like), not a pattern. An empty query never matches; callers treat an
// empty query as "no filter" (see VisibleItems).
func (it *Item) Matches(query string) bool {
	if query == "" {
		return false
	}
	q := strings.ToLower(query)
	if strings.Contains(strings.ToLower(it.Title), q) {
		return true
	}
	return it.Kind == Task && strings.Contains(strings.ToLower(it.Description), q)
}

// VisibleItems returns the set of items to show when filtering by query, under
// "path to matches" semantics: an item is visible when it matches, when a
// descendant matches (so the path from the root down to every match is kept),
// or when an ancestor matches (so a match keeps its whole subtree). Branches
// with no match anywhere on their root-to-node path or in their subtree are
// pruned. A blank query returns nil, meaning "no filter".
func (d *Document) VisibleItems(query string) map[*Item]bool {
	if strings.TrimSpace(query) == "" {
		return nil
	}
	visible := map[*Item]bool{}

	// subtreeMatch[it] is true when it, or any descendant, matches. It marks
	// every item on a path down to a match, so a match is never orphaned.
	subtreeMatch := map[*Item]bool{}
	var mark func(it *Item) bool
	mark = func(it *Item) bool {
		m := it.Matches(query)
		for _, c := range it.Children {
			if mark(c) {
				m = true
			}
		}
		subtreeMatch[it] = m
		return m
	}
	for _, r := range d.Roots {
		mark(r)
	}

	// Collect: an item is visible if its subtree holds a match, or it sits under
	// a matching ancestor — the latter keeps a match's whole subtree.
	var collect func(it *Item, underMatch bool)
	collect = func(it *Item, underMatch bool) {
		if subtreeMatch[it] || underMatch {
			visible[it] = true
		}
		under := underMatch || it.Matches(query)
		for _, c := range it.Children {
			collect(c, under)
		}
	}
	for _, r := range d.Roots {
		collect(r, false)
	}
	return visible
}
