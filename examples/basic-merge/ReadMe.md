# basic-merge

Demonstrates an in-place deep mapping merge.

The program loads a base YAML document and an overlay YAML document with
`go.yaml.in/yaml/v4`, merges the overlay into the base with `dom.Merge`, and dumps
the mutated base document.

```sh
make run
make build
make clean
```

Expected output:

```yaml
service:
    image: app:v1
    replicas: 3
    env:
        LOG_LEVEL: info
        FEATURE_FLAG: enabled
```
