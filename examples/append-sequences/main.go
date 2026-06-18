package main

import (
	"fmt"
	"log"

	"github.com/yaml/go-yaml-dom"
	yaml "go.yaml.in/yaml/v4"
)

const baseYAML = `
pipeline:
  steps:
    - checkout
    - test
`

const overlayYAML = `
pipeline:
  steps:
    - package
    - deploy
`

func main() {
	base := load(baseYAML)
	overlay := load(overlayYAML)

	if err := dom.Merge(base, overlay, dom.WithAppendSequences()); err != nil {
		log.Fatal(err)
	}

	dump(base)
}

func load(s string) *yaml.Node {
	var n yaml.Node
	if err := yaml.Unmarshal([]byte(s), &n); err != nil {
		log.Fatal(err)
	}
	return &n
}

func dump(n *yaml.Node) {
	out, err := yaml.Marshal(n)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Print(string(out))
}
