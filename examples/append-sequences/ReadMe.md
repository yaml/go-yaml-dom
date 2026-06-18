# append-sequences

Demonstrates sequence merge configuration.

The program loads two YAML documents with pipeline steps, then merges them with
`dom.WithAppendSequences()` so source sequence items are appended instead of
replacing the destination sequence.

```sh
make run
make build
make clean
```

Expected output:

```yaml
pipeline:
    steps:
        - checkout
        - test
        - package
        - deploy
```
