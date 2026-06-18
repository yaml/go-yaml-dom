# go-yaml-dom

`go-yaml-dom` is a dependency-free Go library of structural operations over
go-yaml v4 representation nodes (`*yaml.Node`): merge, find, update in place,
clone, and compare.

Its only dependency is `go.yaml.in/yaml/v4`.

```sh
go get github.com/yaml/go-yaml-dom
```

Status: prototype.

This module supports Go 1.18 and is tested with Go 1.18. The repository
Makefiles install Go 1.18.10 locally through Makes, so a system Go installation
is not required for development.

## Purpose

Use `go-yaml-dom` when you want direct structural operations on go-yaml's
representation graph without pulling in an expression engine or non-YAML format
adapters. It is the live, in-place companion to `go-yaml-yq`.

## Contract

`dom` works on live `*yaml.Node` graphs from `go.yaml.in/yaml/v4`.

- `Merge` mutates the destination node in place.
- `FindNodes` and `FindNode` return live interior pointers.
- `Update` mutates those live pointers in place.
- `Clone` returns a detached deep copy.
- `Equal` compares node content and ignores style, comments, and source positions.

This is intentionally different from expression engines that return copies. If you
pass a copied node into `Merge` or `Update`, only that copy changes.

## Loading And Dumping YAML

Use go-yaml v4 directly to load and dump data:

```go
package main

import (
	"fmt"
	"log"

	yaml "go.yaml.in/yaml/v4"
)

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
```

## API

```go
func Merge(dst, src *yaml.Node, opts ...MergeOption) error

func FindNodes(root *yaml.Node, pred func(*yaml.Node) bool) []*yaml.Node
func FindNode(root *yaml.Node, pred func(*yaml.Node) bool) (*yaml.Node, error)

func Update(nodes []*yaml.Node, fn func(*yaml.Node) error) error

func Clone(node *yaml.Node) *yaml.Node
func Equal(a, b *yaml.Node) bool
```

## Merge

`Merge` deep-merges `src` into `dst` in place.

```go
err := dom.Merge(base, overlay)
```

For a non-destructive merge, clone first:

```go
merged := dom.Clone(base)
err := dom.Merge(merged, overlay)
```

By default, source nodes are deep-copied into the destination so later source
mutations do not bleed into the merged tree.

Merge options:

- `WithSequenceMerge(dom.SequenceReplace)` replaces sequences, the default.
- `WithAppendSequences()` appends source sequence items.
- `WithSequenceMerge(dom.SequenceByIndex)` merges sequence items by position.
- `WithOnlyExistingKeys()` updates only keys already present in the destination.
- `WithOnlyNewKeys()` adds only keys absent from the destination.
- `WithClobberTags()` lets source custom tags replace destination tags.
- `WithNullMerge(dom.NullOverwrite)` lets source null replace destination values, the default.
- `WithNullMerge(dom.NullIgnore)` ignores source null values.
- `WithNullMerge(dom.NullDelete)` removes destination mapping keys when source values are null.
- `WithSharedSource()` grafts source nodes by pointer instead of deep-copying them.

## Find And Update

`FindNodes` walks a node tree in pre-order and returns live pointers to matching
nodes. It takes a Go predicate, not a path DSL. Mapping keys are visited. Document
wrappers are traversed but not passed to the predicate. Alias targets are not
followed.

`FindNode` is strict: it returns an error unless exactly one node matches.

`Update` applies a function to live nodes in place and stops at the first error.

## Clone

`Clone` returns a fully detached deep copy. Anchor and alias structure is
preserved: aliases in the clone point at cloned anchors, not the original graph.

## Equal

`Equal` compares deep node content: kind, tag, value, children, and alias
structure. It ignores style, comments, and source positions.

## Example: Basic Merge

```go
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
```

Output:

```yaml
service:
    image: app:v1
    replicas: 3
    env:
        LOG_LEVEL: info
        FEATURE_FLAG: enabled
```

## Example: Append Sequences

```go
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
```

Output:

```yaml
pipeline:
    steps:
        - checkout
        - test
        - package
        - deploy
```

## Example: Find And Update

```go
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
```

Output:

```yaml
services:
    api:
        image: app:v2
    worker:
        image: worker:v2
```

## Example: Non-Destructive Merge

```go
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
```

Output:

```yaml
--- original
app:
    image: app:v1
--- merged
app:
    image: app:v1
    replicas: 2
```

## Running The Examples

The runnable versions of these examples live under `examples/`.

```sh
make -C examples build
make -C examples run
make -C examples clean
```

Run one example:

```sh
make -C examples/basic-merge run
```

Each example directory has its own `ReadMe.md` and supports:

```sh
make build
make run
make clean
```

## Development

The root Makefile bootstraps Makes into `.cache/makes` and installs the pinned
Go toolchain into `.cache/local`.

Common targets:

```sh
make test        # go test ./...
make vet         # go vet ./...
make verify      # fmt, tidy, vet, test
make examples    # build all example programs
make test-all    # tests plus example smoke runs
make clean       # remove example binaries
make deps        # print the module graph
```

Create a release tag:

```sh
make release VERSION=0.1.1
```

`VERSION` is required, must not include a leading `v`, and must be a semantic
version like `0.1.1`. The release target runs verification, requires a clean
working tree, and creates an annotated tag named `v0.1.1`.

## CI

GitHub Actions runs tests, hygiene checks, example smoke tests, Staticcheck,
CodeQL, and a dependency guard. The dependency guard verifies that the module
graph contains only this module and `go.yaml.in/yaml/v4 v4.0.0-rc.5`.

Dependabot is configured for Go modules and GitHub Actions.

## Works With go-yaml-yq

`go-yaml-dom` and `go-yaml-yq` compose only through `*yaml.Node`; neither imports
the other. Use yq expressions when you want path/query/expression power, then use
`dom.Merge` or `dom.Update` for live in-place structural operations.

Keep both modules on the same `go.yaml.in/yaml/v4` version while yaml/v4 is
pre-1.0, otherwise `*yaml.Node` can resolve to distinct types.
