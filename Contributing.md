Contributing to go-yaml-dom
===========================

Thank you for your interest in contributing to go-yaml-dom.

This repository provides live structural operations over `go.yaml.in/yaml/v4`
representation nodes. Keep changes small, well tested, and focused on that DOM
surface.


## Development

The Makefile bootstraps Makes into `.cache/makes` and installs Go locally, so a
system Go installation is not required.

This module supports Go 1.18. The repository Makefiles use Go 1.18.10.

Useful targets:

```sh
make test        # go test ./...
make vet         # go vet ./...
make verify      # fmt, tidy, vet, test
make examples    # build all example programs
make test-all    # tests plus example smoke runs
make clean       # remove example binaries
make deps        # print the module graph
```

Run a single example:

```sh
make -C examples/basic-merge run
```


## Coding Conventions

- Keep the public API small and node-oriented.
- Preserve the live contract: `Merge` and `Update` mutate in place; `Clone`
  returns detached copies.
- Keep the dependency graph limited to `go.yaml.in/yaml/v4`.
- Use `make verify` before sending changes.
- Add or update tests for behavior changes.
- Update `ReadMe.md` and example READMEs when public behavior changes.


## Release Tags

Create a release tag with:

```sh
make release VERSION=0.1.1
```

`VERSION` is required, must not include a leading `v`, and must be a semantic
version like `0.1.1`. The release target runs verification, requires a clean
working tree, and creates an annotated tag named `v0.1.1`.


## Commit Conventions

- Avoid merge commits.
- Commit subject line should:
  - Start with a capital letter.
  - Not end with a period.
  - Be between 20 and 50 characters.
  - Not use conventional-commit prefixes such as `fix:` or `feat:`.
- Separate subject and body with a blank line.


## Pull Requests

1. Create a focused branch.
1. Make the smallest practical change.
1. Add tests and documentation when behavior changes.
1. Run `make verify` and `make test-all`.
1. Submit a pull request.
