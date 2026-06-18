package main

import (
	"fmt"
	"log"
	"strings"

	"github.com/yaml/go-yaml-dom"
	yaml "go.yaml.in/yaml/v4"
)

const inputYAML = `
services:
  api:
    image: app:v1
  worker:
    image: worker:v1
`

func main() {
	doc := load(inputYAML)

	images := dom.FindNodes(doc, func(n *yaml.Node) bool {
		return n.Kind == yaml.ScalarNode && strings.HasSuffix(n.Value, ":v1")
	})
	if err := dom.Update(images, func(n *yaml.Node) error {
		n.Value = strings.TrimSuffix(n.Value, ":v1") + ":v2"
		return nil
	}); err != nil {
		log.Fatal(err)
	}

	dump(doc)
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
