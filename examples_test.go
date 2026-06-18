package dom_test

import (
	"fmt"

	"github.com/yaml/go-yaml-dom"
	yaml "go.yaml.in/yaml/v4"
)

func ExampleMerge() {
	var base, overlay yaml.Node
	_ = yaml.Unmarshal([]byte("items: [a]\nmeta:\n  env: dev\n"), &base)
	_ = yaml.Unmarshal([]byte("items: [b]\nmeta:\n  version: v1\n"), &overlay)

	_ = dom.Merge(&base, &overlay, dom.WithAppendSequences())

	out, _ := yaml.Marshal(&base)
	fmt.Print(string(out))
	// Output:
	// items: [a, b]
	// meta:
	//     env: dev
	//     version: v1
}

func ExampleClone() {
	var base, overlay yaml.Node
	_ = yaml.Unmarshal([]byte("a: 1\n"), &base)
	_ = yaml.Unmarshal([]byte("b: 2\n"), &overlay)

	merged := dom.Clone(&base)
	_ = dom.Merge(merged, &overlay)

	fmt.Println(dom.Equal(&base, merged))
	// Output:
	// false
}
