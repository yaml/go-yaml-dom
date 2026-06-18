package main

import (
	"fmt"
	"log"

	"github.com/yaml/go-yaml-dom"
	yaml "go.yaml.in/yaml/v4"
)

const baseYAML = `
service:
  image: app:v1
  replicas: 1
  env:
    LOG_LEVEL: info
`

const overlayYAML = `
service:
  replicas: 3
  env:
    FEATURE_FLAG: enabled
`

func main() {
	base := load(baseYAML)
	overlay := load(overlayYAML)

	if err := dom.Merge(base, overlay); err != nil {
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
