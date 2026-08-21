package todo

// Path returns the titles from the root down to it (inclusive). It identifies
// an item's position by title at each level, so a reload — which replaces the
// tree with freshly parsed pointers — can re-find the previously selected or
// collapsed item. See FindByPath.
func (it *Item) Path() []string {
	var p []string
	for n := it; n != nil; n = n.Parent {
		p = append([]string{n.Title}, p...)
	}
	return p
}

// FindByPath returns the item reached by following a title path from the roots,
// or nil if no item matches at some level. The first match at each level wins,
// which is good enough to restore selection/fold across a reload (duplicate
// sibling titles are rare and only cost a slightly-off restore).
func (d *Document) FindByPath(path []string) *Item {
	if len(path) == 0 {
		return nil
	}
	items := d.Roots
	var cur *Item
	for _, title := range path {
		cur = nil
		for _, it := range items {
			if it.Title == title {
				cur = it
				break
			}
		}
		if cur == nil {
			return nil
		}
		items = cur.Children
	}
	return cur
}
