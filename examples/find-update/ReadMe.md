# find-update

Demonstrates live node lookup and in-place updates.

The program loads a YAML document, finds scalar image tags ending in `:v1` with
`dom.FindNodes`, updates the returned live nodes with `dom.Update`, and dumps the
changed document.

```sh
make run
make build
make clean
```

Expected output:

```yaml
services:
    api:
        image: app:v2
    worker:
        image: worker:v2
```
