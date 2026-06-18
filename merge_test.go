package dom

import (
	"strings"
	"testing"

	yaml "go.yaml.in/yaml/v4"
)

func mustParse(t *testing.T, s string) *yaml.Node {
	t.Helper()
	var n yaml.Node
	if err := yaml.Unmarshal([]byte(s), &n); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	return &n
}

func mustYAML(t *testing.T, n *yaml.Node) string {
	t.Helper()
	out, err := yaml.Marshal(n)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return string(out)
}

func mappingValue(t *testing.T, n *yaml.Node, key string) *yaml.Node {
	t.Helper()
	n = unwrapDoc(n)
	if n.Kind != yaml.MappingNode {
		t.Fatalf("node is %v, not mapping", n.Kind)
	}
	for i := 0; i+1 < len(n.Content); i += 2 {
		if n.Content[i].Value == key {
			return n.Content[i+1]
		}
	}
	t.Fatalf("key %q not found", key)
	return nil
}

func hasMappingKey(n *yaml.Node, key string) bool {
	n = unwrapDoc(n)
	if n != nil && n.Kind == yaml.AliasNode {
		n = n.Alias
	}
	for i := 0; n != nil && i+1 < len(n.Content); i += 2 {
		if n.Content[i].Value == key {
			return true
		}
	}
	return false
}

func TestMergeSemantics(t *testing.T) {
	t.Run("deep map merge", func(t *testing.T) {
		dst := mustParse(t, "a:\n  b: 1\n  keep: true\n")
		src := mustParse(t, "a:\n  b: 2\n  c: 3\n")
		if err := Merge(dst, src); err != nil {
			t.Fatal(err)
		}
		a := mappingValue(t, dst, "a")
		if got := mappingValue(t, a, "b").Value; got != "2" {
			t.Fatalf("a.b = %q", got)
		}
		if got := mappingValue(t, a, "keep").Value; got != "true" {
			t.Fatalf("a.keep = %q", got)
		}
		if got := mappingValue(t, a, "c").Value; got != "3" {
			t.Fatalf("a.c = %q", got)
		}
	})

	t.Run("sequence strategies", func(t *testing.T) {
		dst := mustParse(t, "items: [a, b]\n")
		src := mustParse(t, "items: [c]\n")
		if err := Merge(dst, src); err != nil {
			t.Fatal(err)
		}
		if got := len(mappingValue(t, dst, "items").Content); got != 1 {
			t.Fatalf("replace len = %d", got)
		}

		dst = mustParse(t, "items: [a, b]\n")
		if err := Merge(dst, src, WithAppendSequences()); err != nil {
			t.Fatal(err)
		}
		if got := len(mappingValue(t, dst, "items").Content); got != 3 {
			t.Fatalf("append len = %d", got)
		}
	})

	t.Run("key filters", func(t *testing.T) {
		dst := mustParse(t, "a: 1\nb: 2\n")
		src := mustParse(t, "b: 20\nc: 30\n")
		if err := Merge(dst, src, WithOnlyExistingKeys()); err != nil {
			t.Fatal(err)
		}
		if hasMappingKey(dst, "c") {
			t.Fatal("new key c was added with existing-only filter")
		}
		if got := mappingValue(t, dst, "b").Value; got != "20" {
			t.Fatalf("b = %q", got)
		}

		dst = mustParse(t, "a: 1\nb: 2\n")
		if err := Merge(dst, src, WithOnlyNewKeys()); err != nil {
			t.Fatal(err)
		}
		if got := mappingValue(t, dst, "b").Value; got != "2" {
			t.Fatalf("existing key b changed to %q", got)
		}
		if got := mappingValue(t, dst, "c").Value; got != "30" {
			t.Fatalf("c = %q", got)
		}
	})

	t.Run("clobber tags", func(t *testing.T) {
		dst := mustParse(t, "v: !old 1\n")
		src := mustParse(t, "v: !new 2\n")
		if err := Merge(dst, src); err != nil {
			t.Fatal(err)
		}
		if got := mappingValue(t, dst, "v").Tag; got != "!old" {
			t.Fatalf("tag without clobber = %q", got)
		}
		if err := Merge(dst, src, WithClobberTags()); err != nil {
			t.Fatal(err)
		}
		if got := mappingValue(t, dst, "v").Tag; got != "!new" {
			t.Fatalf("tag with clobber = %q", got)
		}
	})

	t.Run("null strategies", func(t *testing.T) {
		src := mustParse(t, "a: null\n")
		dst := mustParse(t, "a: 1\n")
		if err := Merge(dst, src); err != nil {
			t.Fatal(err)
		}
		if got := mappingValue(t, dst, "a").Tag; got != "!!null" {
			t.Fatalf("overwrite tag = %q", got)
		}

		dst = mustParse(t, "a: 1\n")
		if err := Merge(dst, src, WithNullMerge(NullIgnore)); err != nil {
			t.Fatal(err)
		}
		if got := mappingValue(t, dst, "a").Value; got != "1" {
			t.Fatalf("ignore value = %q", got)
		}

		dst = mustParse(t, "a: 1\nb: 2\n")
		if err := Merge(dst, src, WithNullMerge(NullDelete)); err != nil {
			t.Fatal(err)
		}
		if hasMappingKey(dst, "a") {
			t.Fatal("null delete left key a")
		}
	})
}

func TestMergeAnchorAndMergeKeyFidelity(t *testing.T) {
	dst := mustParse(t, `
defaults: &defaults
  image: alpine
svc:
  !!merge <<: *defaults
  command: sh
`)
	src := mustParse(t, "svc:\n  replicas: 2\n")
	if err := Merge(dst, src); err != nil {
		t.Fatal(err)
	}

	defaults := mappingValue(t, dst, "defaults")
	if defaults.Anchor != "defaults" {
		t.Fatalf("anchor = %q", defaults.Anchor)
	}
	svc := mappingValue(t, dst, "svc")
	mergeKey := svc.Content[0]
	mergeVal := svc.Content[1]
	if mergeKey.Tag != "!!merge" || mergeKey.Value != "<<" {
		t.Fatalf("merge key lost: tag=%q value=%q", mergeKey.Tag, mergeKey.Value)
	}
	if mergeVal.Kind != yaml.AliasNode || mergeVal.Alias != defaults {
		t.Fatal("merge alias no longer points at defaults anchor")
	}
	if got := mappingValue(t, svc, "replicas").Value; got != "2" {
		t.Fatalf("replicas = %q", got)
	}
	out := mustYAML(t, dst)
	for _, want := range []string{"&defaults", "*defaults", "!!merge <<"} {
		if !strings.Contains(out, want) {
			t.Fatalf("encoded YAML missing %q:\n%s", want, out)
		}
	}
}

func TestMergeInPlaceAndCopySemantics(t *testing.T) {
	dst := mustParse(t, "a:\n  b: 1\n")
	src := mustParse(t, "a:\n  c: 2\n")
	srcBefore := mustYAML(t, src)
	if err := Merge(dst, src); err != nil {
		t.Fatal(err)
	}
	if !hasMappingKey(mappingValue(t, dst, "a"), "c") {
		t.Fatalf("dst was not mutated:\n%s", mustYAML(t, dst))
	}
	if got := mustYAML(t, src); got != srcBefore {
		t.Fatalf("src changed:\n%s", got)
	}

	mappingValue(t, mappingValue(t, src, "a"), "c").Value = "changed"
	if got := mappingValue(t, mappingValue(t, dst, "a"), "c").Value; got != "2" {
		t.Fatalf("dst aliased src under default copy semantics: %q", got)
	}
}

func TestMergeSharedSourceAliasesSource(t *testing.T) {
	dst := mustParse(t, "a: 1\n")
	src := mustParse(t, "b:\n  c: 2\n")
	if err := Merge(dst, src, WithSharedSource()); err != nil {
		t.Fatal(err)
	}
	mappingValue(t, mappingValue(t, src, "b"), "c").Value = "changed"
	if got := mappingValue(t, mappingValue(t, dst, "b"), "c").Value; got != "changed" {
		t.Fatalf("shared subtree did not alias src: %q", got)
	}
}

func TestNonDestructiveMergeViaClone(t *testing.T) {
	dst := mustParse(t, "a: 1\n")
	orig := mustYAML(t, dst)
	merged := Clone(dst)
	if err := Merge(merged, mustParse(t, "b: 2\n")); err != nil {
		t.Fatal(err)
	}
	if got := mustYAML(t, dst); got != orig {
		t.Fatalf("original changed:\n%s", got)
	}
	if !hasMappingKey(merged, "b") {
		t.Fatal("clone was not merged")
	}
}

func TestMergeCommentStylePreservation(t *testing.T) {
	dst := mustParse(t, "a: old # keep\n")
	src := mustParse(t, "a: new\nb: added # carry\n")
	dstA := mappingValue(t, dst, "a")
	dstA.Style = yaml.DoubleQuotedStyle
	if err := Merge(dst, src); err != nil {
		t.Fatal(err)
	}
	if got := mappingValue(t, dst, "a"); got.LineComment != "# keep" || got.Style != yaml.DoubleQuotedStyle {
		t.Fatalf("merged-through metadata not preserved: comment=%q style=%v", got.LineComment, got.Style)
	}
	if got := mappingValue(t, dst, "b"); got.LineComment != "# carry" {
		t.Fatalf("added metadata not carried: %q", got.LineComment)
	}
}
