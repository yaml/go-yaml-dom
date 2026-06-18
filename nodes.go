package dom

import (
	"errors"
	"fmt"

	yaml "go.yaml.in/yaml/v4"
)

// FindNodes walks root in pre-order and returns live pointers to nodes for which
// pred returns true. Document nodes are traversed but not passed to pred. Alias
// targets are not followed.
func FindNodes(root *yaml.Node, pred func(*yaml.Node) bool) []*yaml.Node {
	var out []*yaml.Node
	seen := map[*yaml.Node]bool{}
	var walk func(*yaml.Node)
	walk = func(n *yaml.Node) {
		if n == nil || seen[n] {
			return
		}
		seen[n] = true
		if n.Kind != yaml.DocumentNode && pred(n) {
			out = append(out, n)
		}
		for _, c := range n.Content {
			walk(c)
		}
	}
	walk(root)
	return out
}

// FindNode returns exactly one live node matching pred.
func FindNode(root *yaml.Node, pred func(*yaml.Node) bool) (*yaml.Node, error) {
	ns := FindNodes(root, pred)
	switch len(ns) {
	case 1:
		return ns[0], nil
	case 0:
		return nil, errors.New("dom: FindNode found no matching node")
	default:
		return nil, fmt.Errorf("dom: FindNode found %d matches (want exactly 1)", len(ns))
	}
}

// Update applies fn to each live node in place, stopping at the first error.
func Update(nodes []*yaml.Node, fn func(*yaml.Node) error) error {
	for i, n := range nodes {
		if err := fn(n); err != nil {
			return fmt.Errorf("dom: updating node %d: %w", i, err)
		}
	}
	return nil
}

// Clone returns an independent deep copy. Aliases in the clone point at cloned
// anchor nodes, not the original graph.
func Clone(node *yaml.Node) *yaml.Node {
	remap := map[*yaml.Node]*yaml.Node{}
	var cp func(*yaml.Node) *yaml.Node
	cp = func(n *yaml.Node) *yaml.Node {
		if n == nil {
			return nil
		}
		if c, ok := remap[n]; ok {
			return c
		}
		c := *n
		remap[n] = &c
		if n.Content != nil {
			c.Content = make([]*yaml.Node, len(n.Content))
			for i, ch := range n.Content {
				c.Content[i] = cp(ch)
			}
		}
		c.Alias = cp(n.Alias)
		return &c
	}
	return cp(node)
}

// Equal reports deep content equality. It compares kind, tag, value, children,
// and alias targets, while ignoring style, comments, and source positions.
func Equal(a, b *yaml.Node) bool {
	return equalNode(a, b, map[[2]*yaml.Node]bool{})
}

func equalNode(a, b *yaml.Node, seen map[[2]*yaml.Node]bool) bool {
	if a == nil || b == nil {
		return a == b
	}
	key := [2]*yaml.Node{a, b}
	if seen[key] {
		return true
	}
	seen[key] = true
	if a.Kind != b.Kind || a.Tag != b.Tag || a.Value != b.Value || len(a.Content) != len(b.Content) {
		return false
	}
	for i := range a.Content {
		if !equalNode(a.Content[i], b.Content[i], seen) {
			return false
		}
	}
	return equalNode(a.Alias, b.Alias, seen)
}
