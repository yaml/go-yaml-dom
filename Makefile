R := https://github.com/makeplus/makes
M ?= $(or $(MAKES_REPO_DIR),.cache/makes)

$(shell [ -d '$(M)' ] || git clone -q $(R) '$(M)')

include $(M)/init.mk

GO-VERSION := 1.18.10
include $(M)/go.mk
include $(M)/clean.mk
include $(M)/shell.mk


check: test vet examples

deps: $(GO)
	@go list -m all

examples:: $(GO) FORCE
	go build \
	  -o examples/append-sequences/append-sequences ./examples/append-sequences
	go build \
	  -o examples/basic-merge/basic-merge ./examples/basic-merge
	go build \
	  -o examples/clone-merge/clone-merge ./examples/clone-merge
	go build \
	  -o examples/find-update/find-update ./examples/find-update

fmt: $(GO)
	go fmt ./...

tidy: $(GO)
	go mod tidy

test: $(GO)
	go test ./...

test-examples:: FORCE
	@$(MAKE) --no-print-directory -C examples run

test-all:: test test-examples

vet: $(GO)
	go vet ./...

verify: fmt tidy vet test

release: release-check verify release-tag

release-check:
ifndef VERSION
	@echo "Set VERSION=x.y.z to use 'make release'"
	@exit 1
endif
	@case '$(VERSION)' in \
	  v*) echo "VERSION must not start with 'v'; use VERSION=$(patsubst v%,%,$(VERSION))" >&2; exit 1 ;; \
	esac
	@printf '%s\n' '$(VERSION)' | grep -Eq '^[0-9]+\.[0-9]+\.[0-9]+$$' || \
	  (echo "VERSION must be a semantic version like 0.1.1" >&2; exit 1)

release-tag: release-check
	@git diff --quiet -- . ':!.cache' || \
	  (echo "Working tree has uncommitted changes" >&2; exit 1)
	@git diff --cached --quiet -- . ':!.cache' || \
	  (echo "Index has staged changes" >&2; exit 1)
	@test -z "$$(git status --porcelain --untracked-files=all -- . ':!.cache')" || \
	  (echo "Working tree has untracked files" >&2; exit 1)
	@git rev-parse --verify 'v$(VERSION)' >/dev/null 2>&1 && \
	  (echo "Tag v$(VERSION) already exists" >&2; exit 1) || true
	git tag -a 'v$(VERSION)' -m 'Release v$(VERSION)'

clean::
	@$(MAKE) --no-print-directory -C examples clean

build-example:: $(GO)
ifndef EXAMPLE
	@echo "Set EXAMPLE=... to use 'make build-example'"
	@exit 1
endif
	go build -o examples/$(EXAMPLE)/$(EXAMPLE) ./examples/$(EXAMPLE)

run-example:: $(GO)
ifndef EXAMPLE
	@echo "Set EXAMPLE=... to use 'make run-example'"
	@exit 1
endif
	@$(MAKE) --no-print-directory -C examples/$(EXAMPLE) run

FORCE:
