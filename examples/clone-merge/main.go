package main

import (
	"fmt"
	"log"

	"github.com/yaml/go-yaml-dom"
	yaml "go.yaml.in/yaml/v4"
)

const baseYAML = `
app:
  image: app:v1
`

const overlayYAML = `
app:
  replicas: 2
`

func main() {
	base := load(baseYAML)
	overlay := load(overlayYAML)

	merged := dom.Clone(base)
	if err := dom.Merge(merged, overlay); err != nil {
		log.Fatal(err)
	}

	fmt.Println("--- original")
	dump(base)
	fmt.Println("--- merged")
	dump(merged)
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
