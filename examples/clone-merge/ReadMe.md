# clone-merge

Demonstrates a non-destructive merge.

The program loads a base document and overlay document with `go.yaml.in/yaml/v4`,
clones the base with `dom.Clone`, merges into the clone, and then dumps both the
unchanged original and the merged copy.

```sh
make run
make build
make clean
```

Expected output:

```yaml
--- original
app:
    image: app:v1
--- merged
app:
    image: app:v1
    replicas: 2
```
