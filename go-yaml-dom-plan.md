# go-yaml-dom — Prototype Implementation Plan

> A tight, dependency-free Go library of structural operations over go-yaml v4
> representation-graph nodes (`*yaml.Node`): **merge**, find, update-in-place, clone,
> compare. Its only dependency is `go.yaml.in/yaml/v4`.
>
> Companion to **go-yaml-yq** (the yq expression engine binding). The two compose
> through the standard `*yaml.Node` type; neither imports the other.

This document is written to be handed to a CLI coding agent.

---

## 0. Objective and strategy

Pure Go. No vendored engine, no expression DSL, no third-party dependency beyond the
YAML library itself. Every operation is a hand-written walk over `*yaml.Node`. This
is the part of the family that ports conceptually to PyYAML / libyaml / serde-yaml —
including **merge**, which lives here (not in go-yaml-yq) precisely so it ports.

Owning merge natively is the one substantial piece of work in this module: it is the
~200-line recursive operator, not a wrapper. The reward is portability and a clean,
engine-free dependency graph. The risk is **fidelity** — anchors, aliases, and the
`!!merge <<` tag must survive the merge — so the fidelity round-trip is a blocking
test (§5), and the harder semantics are explicitly staged (§6).

Expression-language power (path queries, `del`, the other ~60 yq operators) lives in
the separate **go-yaml-yq** module and is opt-in. A consumer who only wants merge or
find+update pulls *nothing* but go-yaml here.

---

## 1. Locked design decisions

- **Module path:** `github.com/yaml/go-yaml-dom` (placeholder).
- **Package name:** `dom`. Call sites: `dom.Merge(...)`, `dom.FindNodes(...)`.
- **Sole dependency:** `go.yaml.in/yaml/v4`, pinned to **`v4.0.0-rc.5`** — must match
  go-yaml-yq and any consumer using both (§1.3).
- **Naming convention:** node-resolving functions come as a singular/plural pair with
  a `…Node`/`…Nodes` suffix. Non-resolvers (`Merge`, `Clone`, `Equal`) take bare
  verbs.
- **Options convention (mirrors go-yaml v4):** configuration is via functional
  `WithX()` options passed variadically. Boolean toggles are callable with no
  argument (enable) or an explicit bool, e.g. `WithClobberTags()` /
  `WithClobberTags(false)` — matching v4's `WithCompactSeqIndent()` /
  `WithCompactSeqIndent(false)`. Multi-choice settings take a typed constant
  (`WithSequenceMerge(SequenceAppend)`). Each function family gets its own option type
  (`MergeOption` now) so an option can't be passed to a function it doesn't apply to.

### 1.1 Contract — this module is LIVE (in-place mutation)

- `Merge(dst, src, …)` mutates **dst in place** and returns `error`. For a
  non-destructive merge, clone first: `m := Clone(dst); Merge(m, src, …)`.
- `FindNodes`/`FindNode` return **live interior pointers** into the passed tree.
- `Update` applies a function to those pointers **in place**.
- `Clone` is the deliberate exception: it returns a fully **detached** deep copy.
- `Equal` is read-only.

This is the opposite of go-yaml-yq, which is pure/copy. Feeding a **yq** result (a
copy) into `Merge`/`Update` mutates the copy and does nothing to the source;
document this.

### 1.2 MVP public API

```go
// Merge — in-place, native, configurable.
func Merge(dst, src *yaml.Node, opts ...MergeOption) error

// Find — predicate-based, returns LIVE references.
func FindNodes(root *yaml.Node, pred func(*yaml.Node) bool) []*yaml.Node
func FindNode(root *yaml.Node, pred func(*yaml.Node) bool) (*yaml.Node, error) // strict: 0 or >1 = error

// Update — in-place mutation of live references.
func Update(nodes []*yaml.Node, fn func(*yaml.Node) error) error

// Utilities.
func Clone(node *yaml.Node) *yaml.Node       // detached deep copy; preserves anchor/alias structure
func Equal(a, b *yaml.Node) bool             // deep content equality; ignores style/comments/position
```

#### Merge options (the focus)

```go
type MergeOption func(*mergeConfig)

// --- Sequences (arrays). Default: SequenceReplace. ---
type SequenceStrategy int
const (
    SequenceReplace SequenceStrategy = iota // dst sequence replaced by src      (default)
    SequenceAppend                          // src elements appended to dst       (yq '+')
    SequenceByIndex                         // merge element-wise by position      (yq 'd')  [STAGED]
)
func WithSequenceMerge(s SequenceStrategy) MergeOption
func WithAppendSequences() MergeOption       // convenience for WithSequenceMerge(SequenceAppend)

// --- Map key filtering. Default: KeyAll. ---
type KeyFilter int
const (
    KeyAll          KeyFilter = iota // merge every src key                       (default)
    KeyOnlyExisting                  // only keys already present in dst          (yq '?')
    KeyOnlyNew                       // only keys absent from dst                 (yq 'n')
)
func WithKeyFilter(f KeyFilter) MergeOption
func WithOnlyExistingKeys() MergeOption      // convenience
func WithOnlyNewKeys() MergeOption           // convenience

// --- Custom tags. Default: keep dst tag. ---
func WithClobberTags(enable ...bool) MergeOption // src custom tag wins           (yq 'c')

// --- Null handling (src null merging into dst). Default: NullOverwrite. ---
type NullStrategy int
const (
    NullOverwrite NullStrategy = iota // src null replaces dst value             (default)
    NullIgnore                        // src null leaves dst unchanged
    NullDelete                        // src null removes the key from dst
)
func WithNullMerge(s NullStrategy) MergeOption

// --- Source graft semantics. Default: deep-copy src into dst (no aliasing). ---
func WithSharedSource(enable ...bool) MergeOption // graft src pointers directly (faster, aliases src)
```

`Find` takes a Go predicate, not a path DSL — path/pattern lookups are go-yaml-yq's
job (`yq.Nodes`/`yq.Node`). This keeps `dom` free of an expression parser.

### 1.3 Interop with go-yaml-yq

- Interop surface is exclusively `*yaml.Node`. Compose: build a document with
  `yq.Node`, then `dom.Merge` overlays into it live; or locate live targets with
  `dom.FindNodes` and compute replacement values with `yq.Node`.
- **Cross-module version coordination (critical):** go-yaml-dom, go-yaml-yq, and any
  consumer using both must resolve to the **same** `go.yaml.in/yaml/v4` version, or
  `*yaml.Node` is two distinct types. While yaml/v4 is pre-1.0, keep `require`
  versions identical and bump in lockstep.
- Neither module imports the other.

---

## 2. Repository scaffold

```
go-yaml-dom/
├── go.mod                  # module …/go-yaml-dom; require go.yaml.in/yaml/v4 v4.0.0-rc.5
├── LICENSE
├── README.md               # §4
├── merge.go                # Merge + MergeOption + mergeConfig (§3.1)
├── nodes.go                # FindNodes/Node, Update, Clone, Equal (§3.2)
├── merge_test.go           # merge semantics + fidelity (§5)
├── nodes_test.go           # find/update/clone/equal (§5)
└── examples_test.go
```

```bash
go mod init github.com/yaml/go-yaml-dom
go get go.yaml.in/yaml/v4@v4.0.0-rc.5
```

**Verify:** `go list -m all` shows exactly `go.yaml.in/yaml/v4` (plus its own small
tail) and nothing else. Anything else means the engine leaked in — this module must
stay engine-free.

---

## 3. Implementation

go-yaml v4 node fields used: `Kind` (constants `DocumentNode`, `MappingNode`,
`SequenceNode`, `ScalarNode`, `AliasNode`), `Tag`, `Value`, `Content`, `Alias`,
`Anchor`, `Style`, `HeadComment`/`LineComment`/`FootComment`. Mapping `Content` is
`[k0,v0,k1,v1,…]`. The null tag is `!!null`.

### 3.1 `merge.go`

**Option layer** (mechanical — implement exactly as below):

```go
package dom

import (
	"errors"
	"fmt"

	yaml "go.yaml.in/yaml/v4"
)

type SequenceStrategy int
const (
	SequenceReplace SequenceStrategy = iota
	SequenceAppend
	SequenceByIndex
)

type KeyFilter int
const (
	KeyAll KeyFilter = iota
	KeyOnlyExisting
	KeyOnlyNew
)

type NullStrategy int
const (
	NullOverwrite NullStrategy = iota
	NullIgnore
	NullDelete
)

type mergeConfig struct {
	sequence SequenceStrategy
	keys     KeyFilter
	clobber  bool
	null     NullStrategy
	shared   bool
}

func defaultMergeConfig() mergeConfig {
	return mergeConfig{sequence: SequenceReplace, keys: KeyAll, null: NullOverwrite}
}

type MergeOption func(*mergeConfig)

func WithSequenceMerge(s SequenceStrategy) MergeOption { return func(c *mergeConfig) { c.sequence = s } }
func WithAppendSequences() MergeOption                 { return WithSequenceMerge(SequenceAppend) }
func WithKeyFilter(f KeyFilter) MergeOption            { return func(c *mergeConfig) { c.keys = f } }
func WithOnlyExistingKeys() MergeOption                { return WithKeyFilter(KeyOnlyExisting) }
func WithOnlyNewKeys() MergeOption                     { return WithKeyFilter(KeyOnlyNew) }
func WithNullMerge(s NullStrategy) MergeOption         { return func(c *mergeConfig) { c.null = s } }

func firstBool(def bool, b []bool) bool { if len(b) > 0 { return b[0] }; return def }
func WithClobberTags(enable ...bool) MergeOption   { return func(c *mergeConfig) { c.clobber = firstBool(true, enable) } }
func WithSharedSource(enable ...bool) MergeOption  { return func(c *mergeConfig) { c.shared = firstBool(true, enable) } }
```

**Merge algorithm** (spec — implement against the staged tests in §5):

```go
// Merge deep-merges src into dst IN PLACE. dst is mutated; src is not (unless
// WithSharedSource, which grafts src subtrees by pointer). Returns error on
// structurally impossible merges.
func Merge(dst, src *yaml.Node, opts ...MergeOption) error {
	cfg := defaultMergeConfig()
	for _, o := range opts {
		o(&cfg)
	}
	return mergeNode(unwrapDoc(dst), unwrapDoc(src), cfg)
}
```

Required behavior, in priority order:

1. **Unwrap documents.** If either node is a `DocumentNode`, operate on its single
   content child. (`unwrapDoc` returns `n.Content[0]` for a DocumentNode, else `n`.)
2. **Null handling (src side).** If src is `!!null`: `NullOverwrite` → replace dst
   with null; `NullIgnore` → leave dst; `NullDelete` → caller (mergeMapping) removes
   the key. (For MVP, treat any `!!null` src as null; the implicit-vs-explicit
   distinction is STAGED — §6.)
3. **Kind mismatch or scalar.** If dst and src are not both mappings or both
   sequences, src wins: copy src's content/kind/tag/style into dst (deep-copy unless
   `cfg.shared`). Tag: keep dst's tag unless `cfg.clobber`.
4. **Both mappings.** For each `(key, val)` in src (iterating Content pairwise):
   locate a dst entry whose key is `Equal` to the src key.
   - found: recurse `mergeNode(dstVal, srcVal, cfg)`, **unless** `cfg.keys ==
     KeyOnlyNew` (then skip).
   - not found: append the pair (deep-copied unless shared), **unless** `cfg.keys ==
     KeyOnlyExisting` (then skip).
   - Preserve dst key ordering; appended keys go at the end.
5. **Both sequences.** Apply `cfg.sequence`: `SequenceReplace` → dst.Content =
   copy(src.Content); `SequenceAppend` → dst.Content = append(dst.Content,
   copy(src.Content)...); `SequenceByIndex` → element-wise `mergeNode` up to
   min(len), append the remainder [STAGED].
6. **Copy semantics.** Whenever src material lands in dst, deep-copy it via `Clone`
   unless `cfg.shared` is set. This is the copy-by-value vs copy-by-pointer decision
   from PR #353 — default to value (no aliasing surprises); shared is the opt-in.
7. **Fidelity (non-negotiable for MVP scope).** Preserve dst's comments and style on
   merged-through nodes; carry src's comments/style on newly added nodes. Preserve
   anchors and aliases — a merged result that loses an `&anchor`/`*alias` or drops the
   `!!merge <<` tag is a bug (see §5 test 2). Do not follow `n.Alias` into infinite
   recursion; guard with a visited-set as in `FindNodes`.

### 3.2 `nodes.go`

```go
package dom

import (
	"errors"
	"fmt"

	yaml "go.yaml.in/yaml/v4"
)

// FindNodes walks root in pre-order and returns LIVE pointers to every node for which
// pred returns true. The DocumentNode wrapper is not passed to pred (traversal
// descends into it). Mapping KEYS are visited. Alias targets are NOT followed; a
// visited-set guards cycles.
func FindNodes(root *yaml.Node, pred func(*yaml.Node) bool) []*yaml.Node {
	var out []*yaml.Node
	seen := map[*yaml.Node]bool{}
	var walk func(n *yaml.Node)
	walk = func(n *yaml.Node) {
		if n == nil || seen[n] {
			return
		}
		seen[n] = true
		if n.Kind != yaml.DocumentNode && pred(n) {
			out = append(out, n)
		}
		for _, c := range n.Content { // never follow n.Alias
			walk(c)
		}
	}
	walk(root)
	return out
}

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

// Update applies fn to each node in place, stopping at the first error. Pair with
// FindNodes (live refs). There is no single-node variant: applying one function to
// one node is just fn(node), so for a lone live node, call your function directly.
// Passing a go-yaml-yq result here mutates a copy and has no effect on any source
// document.
func Update(nodes []*yaml.Node, fn func(*yaml.Node) error) error {
	for i, n := range nodes {
		if err := fn(n); err != nil {
			return fmt.Errorf("dom: updating node %d: %w", i, err)
		}
	}
	return nil
}

// Clone returns an independent deep copy, preserving anchor/alias structure (aliases
// in the clone point at the cloned anchor, not the original).
func Clone(node *yaml.Node) *yaml.Node {
	remap := map[*yaml.Node]*yaml.Node{}
	var cp func(n *yaml.Node) *yaml.Node
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
		if n.Alias != nil {
			c.Alias = cp(n.Alias)
		}
		return &c
	}
	return cp(node)
}

// Equal reports deep content equality (Kind, Tag, Value, children); ignores Style,
// comments, and source position. Aliases compared structurally. Cycle-guarded.
func Equal(a, b *yaml.Node) bool { return equalNode(a, b, map[[2]*yaml.Node]bool{}) }

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
	return true
}
```

(Provide `unwrapDoc(*yaml.Node) *yaml.Node` in `merge.go`.)

**Verify:** `go build ./...` and `go vet ./...` pass.

---

## 4. README

- One-line: dependency-free structural operations over go-yaml v4 nodes, incl. merge.
- Install line.
- **State the live contract early:** `Merge`/`Update*` mutate in place; `Find*`
  returns live references; `Clone` detaches; `Equal` compares.
- Examples:
  - merge with options:
    `dom.Merge(base, overlay, dom.WithAppendSequences(), dom.WithClobberTags())`
  - non-destructive merge: `m := dom.Clone(base); dom.Merge(m, overlay)`
  - find + mutate: `dom.Update(dom.FindNodes(doc, isImageScalar), bumpTag)`
  - compare: `dom.Equal(a, b)`
- A "works with go-yaml-yq" section (locate with `dom`, compute with `yq`).
- "Status: prototype."

---

## 5. Tests

### merge_test.go (the headline)

1. **Merge semantics** (mirror yq's documented examples): deep map merge; sequence
   `SequenceReplace` (default) vs `WithAppendSequences()`; `WithOnlyExistingKeys()`;
   `WithOnlyNewKeys()`; `WithClobberTags()`; null per `NullOverwrite`/`NullIgnore`/
   `NullDelete`.
2. **Anchor / merge-key fidelity (BLOCKER).** Merge documents using `&anchor`/`*alias`
   and an explicit `!!merge <<`; assert anchors, aliases, and the `!!merge` tag
   survive in the (re-encoded) result. This is the regression that hit go-yaml main —
   a failure blocks the prototype.
3. **In-place + copy semantics.** Default `Merge(dst, src)` mutates `dst` and leaves
   `src` unchanged (re-encode src, assert equal); after merge, mutating `src` does NOT
   bleed into `dst` (proves deep-copy default). With `WithSharedSource()`, document
   that grafted subtrees alias `src`.
4. **Non-destructive via Clone.** `m := Clone(dst); Merge(m, src)` leaves `dst`
   unchanged.
5. **Comment/style preservation.** Merged-through nodes keep dst comments/style;
   added nodes carry src comments/style.

### nodes_test.go

6. **Live mutation visible** through `FindNodes`+`Update`.
7. **Find traversal correctness:** keys visited; recursive anchor doesn't infinite-
   loop; alias nodes returned as aliases, targets not re-walked via `.Alias`;
   DocumentNode wrapper not passed to pred.
8. **FindNode strictness** (0 and >1 error).
9. **Clone** is `Equal` but not same pointer; mutating clone doesn't touch original;
   `&anchor`/`*alias` doc clones with alias pointing at the clone's anchor.
10. **Equal** ignores style/comments; distinguishes kind/tag/value/shape; cycle-safe.

**Verify (prototype acceptance):** `go test ./...` passes, with tests **1, 2, 3, 7,
and 9** green.

---

## 6. Out of scope / deferred

- **Staged merge semantics** (ship after the MVP core lands):
  - `SequenceByIndex` element-wise array merge (yq `d`).
  - **Merge arrays of objects by a key field** (yq's complex case) — likely a new
    option `WithSequenceMergeByKey(keyPath)`.
  - **Implicit vs explicit null** distinction in `NullStrategy`.
  - **Full `!!merge <<` expansion** during merge (vs. preservation, which MVP does).
- **Options for other functions.** The functional-option pattern is reusable: when
  `Find` or a future `Walk` needs configuration, give it its own typed option set
  (`FindOption`, …) the same way. Don't share one `Option` type across functions
  (loses type safety).
- **Merge presets** (mirroring v4's `WithV3Defaults()`): named bundles for known merge
  dialects, e.g. `WithJSONMergePatch()` = `WithSequenceMerge(SequenceReplace)` +
  `WithNullMerge(NullDelete)` (RFC 7386), or `WithStrategicMerge()` for k8s-style.
- **`MergeOptsYAML(yamlString)`** — configure merge from a YAML document, mirroring
  v4's `yaml.OptsYAML`. On-brand; defer to post-MVP.
- **`Walk` / `WalkInPlace`** with a position-aware visitor — the two capabilities it
  must add beyond `Find`+`Update`: structural mutation mid-traversal (needs
  parent+index) and position/path-aware bulk mutation.
- **Structured-segment paths** and **live lookup by path** — native walkers; v2.
- **Path-based set/delete are not here and not in go-yaml-yq as methods** — they are
  expressions: `yq.Node(".a = $2", doc, v)`, `yq.Node("del(.a)", doc)`. A native DOM
  variant that would fit (no path-DSL parser) is a *keyed structural* form operating
  on an already-found live mapping node — `SetKey(mapNode, key, val)` /
  `DeleteKey(mapNode, key)`, composing with `FindNode`. Not in MVP; noted as the shape
  that belongs here if wanted.

This module **does** port conceptually to the rest of the YAML family — the
operations (merge included) and the "representation graph"/"node" vocabulary are
language-neutral. The expression engine (go-yaml-yq) is the Go-only piece that does
not port.
