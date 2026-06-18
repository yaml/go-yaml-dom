# go-yaml-dom examples

Each example is a small standalone Go program that loads YAML into
`go.yaml.in/yaml/v4` representation nodes, applies `github.com/yaml/go-yaml-dom`,
and dumps YAML back out.

Run all examples:

```sh
make -C examples run
```

Build all examples:

```sh
make -C examples build
```

Clean built binaries:

```sh
make -C examples clean
```

Run one example:

```sh
make -C examples/basic-merge run
```

## Examples

- `basic-merge`: deep-merge one mapping into another in place.
- `append-sequences`: merge mappings while appending sequence items.
- `find-update`: find live scalar nodes and mutate them in place.
- `clone-merge`: clone first to perform a non-destructive merge.
