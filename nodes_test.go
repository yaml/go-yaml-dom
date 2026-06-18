package dom

import (
	"errors"
	"testing"

	yaml "go.yaml.in/yaml/v4"
)

func TestFindNodesUpdateLiveMutation(t *testing.T) {
	doc := mustParse(t, "images:\n- app:v1\n- sidecar:v1\n")
	nodes := FindNodes(doc, func(n *yaml.Node) bool {
		return n.Kind == yaml.ScalarNode && n.Tag == "!!str" && n.Value == "app:v1"
	})
	if len(nodes) != 1 {
		t.Fatalf("matches = %d", len(nodes))
	}
	if err := Update(nodes, func(n *yaml.Node) error {
		n.Value = "app:v2"
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if got := mappingValue(t, doc, "images").Content[0].Value; got != "app:v2" {
		t.Fatalf("live update got %q", got)
	}
}

func TestUpdateStopsOnError(t *testing.T) {
	errStop := errors.New("stop")
	nodes := []*yaml.Node{{Kind: yaml.ScalarNode}, {Kind: yaml.ScalarNode}}
	err := Update(nodes, func(n *yaml.Node) error {
		return errStop
	})
	if !errors.Is(err, errStop) {
		t.Fatalf("error = %v", err)
	}
}

func TestFindTraversalCorrectness(t *testing.T) {
	doc := mustParse(t, `
a: &a
  self: *a
b: 2
`)
	var sawDoc bool
	keys := map[string]bool{}
	aliases := 0
	nodes := FindNodes(doc, func(n *yaml.Node) bool {
		if n.Kind == yaml.DocumentNode {
			sawDoc = true
		}
		if n.Kind == yaml.AliasNode {
			aliases++
		}
		if n.Kind == yaml.ScalarNode {
			keys[n.Value] = true
		}
		return true
	})
	if sawDoc {
		t.Fatal("DocumentNode was passed to predicate")
	}
	if !keys["a"] || !keys["b"] {
		t.Fatalf("mapping keys not visited: %#v", keys)
	}
	if aliases != 1 {
		t.Fatalf("aliases visited = %d", aliases)
	}
	if len(nodes) == 0 {
		t.Fatal("no nodes returned")
	}
}

func TestFindNodeStrictness(t *testing.T) {
	doc := mustParse(t, "a: 1\nb: 2\n")
	if _, err := FindNode(doc, func(n *yaml.Node) bool { return n.Value == "missing" }); err == nil {
		t.Fatal("expected zero-match error")
	}
	if _, err := FindNode(doc, func(n *yaml.Node) bool { return n.Kind == yaml.ScalarNode }); err == nil {
		t.Fatal("expected multi-match error")
	}
	if n, err := FindNode(doc, func(n *yaml.Node) bool { return n.Value == "1" }); err != nil || n.Value != "1" {
		t.Fatalf("single match = %v, %v", n, err)
	}
}

func TestCloneDetachedAndAliasRemapped(t *testing.T) {
	doc := mustParse(t, `
a: &a
  b: 1
ref: *a
`)
	cp := Clone(doc)
	if cp == doc {
		t.Fatal("clone returned same pointer")
	}
	if !Equal(doc, cp) {
		t.Fatal("clone is not Equal to original")
	}
	origA := mappingValue(t, doc, "a")
	cpA := mappingValue(t, cp, "a")
	cpRef := mappingValue(t, cp, "ref")
	if cpA == origA {
		t.Fatal("anchor node was not detached")
	}
	if cpRef.Alias != cpA {
		t.Fatal("alias does not point at cloned anchor")
	}
	mappingValue(t, cpA, "b").Value = "2"
	if got := mappingValue(t, origA, "b").Value; got != "1" {
		t.Fatalf("mutating clone changed original: %q", got)
	}
}

func TestEqualSemantics(t *testing.T) {
	a := mustParse(t, "v: 1 # comment\n")
	b := mustParse(t, "v: 1\n")
	mappingValue(t, a, "v").Style = yaml.DoubleQuotedStyle
	if !Equal(a, b) {
		t.Fatal("Equal considered style/comment")
	}
	if Equal(a, mustParse(t, "v: 2\n")) {
		t.Fatal("Equal ignored value")
	}
	if Equal(a, mustParse(t, "v: !tag 1\n")) {
		t.Fatal("Equal ignored tag")
	}

	cycA := &yaml.Node{Kind: yaml.AliasNode}
	cycA.Alias = cycA
	cycB := &yaml.Node{Kind: yaml.AliasNode}
	cycB.Alias = cycB
	if !Equal(cycA, cycB) {
		t.Fatal("Equal is not cycle-safe for aliases")
	}
}
