// Package dom provides live structural operations over go-yaml v4
// representation nodes.
//
// The package works directly with [yaml.Node] values from go.yaml.in/yaml/v4.
// Merge and Update mutate live node graphs in place. FindNodes and FindNode
// return live interior pointers. Clone returns detached deep copies, and Equal
// compares node content while ignoring presentation details such as comments
// and style.
//
// Merge mutates the destination:
//
//	var base, overlay yaml.Node
//	_ = yaml.Unmarshal([]byte("a: 1\n"), &base)
//	_ = yaml.Unmarshal([]byte("b: 2\n"), &overlay)
//	err := dom.Merge(&base, &overlay)
//
// Clone first for non-destructive merge:
//
//	merged := dom.Clone(&base)
//	err := dom.Merge(merged, &overlay)
//
// FindNodes and Update compose for in-place edits:
//
//	nodes := dom.FindNodes(doc, func(n *yaml.Node) bool {
//	    return n.Kind == yaml.ScalarNode && n.Value == "old"
//	})
//	err := dom.Update(nodes, func(n *yaml.Node) error {
//	    n.Value = "new"
//	    return nil
//	})
//
// Use github.com/yaml/go-yaml-yq when you want yq's expression language.
// The two packages compose only through *yaml.Node; neither imports the other.
package dom

import yaml "go.yaml.in/yaml/v4"

var _ *yaml.Node
