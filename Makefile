GO ?= go
BINARY_NAME = symprint
# version is a package-level var in `main`, so inject into main.version
# (matches .goreleaser.yml). Injecting into the full import path silently no-ops.
VERSION_PKG = main

.PHONY: all
all: build test

.PHONY: build
build:
	CGO_ENABLED=0 $(GO) build -ldflags "-s -w -X main.version=dev" -o $(BINARY_NAME) ./cmd/symprint

.PHONY: build-version
build-version:
	CGO_ENABLED=0 $(GO) build -ldflags "-s -w -X $(VERSION_PKG).version=$(VERSION)" -o $(BINARY_NAME) ./cmd/symprint

.PHONY: test
test:
	CGO_ENABLED=0 $(GO) test ./...

.PHONY: test-verbose
test-verbose:
	CGO_ENABLED=0 $(GO) test -v ./...

.PHONY: test-race
test-race:
	$(GO) test -race ./...

.PHONY: lint
lint:
	$(GO) fmt ./...
	CGO_ENABLED=0 $(GO) vet ./...

.PHONY: clean
clean:
	rm -f $(BINARY_NAME)
	rm -rf dist/

.PHONY: install
install:
	CGO_ENABLED=0 $(GO) install -ldflags "-s -w -X $(VERSION_PKG).version=dev" ./cmd/symprint

# Render the example documents (requires `typst` on PATH).
.PHONY: examples
examples: build
	./$(BINARY_NAME) render examples/report.md -o dist/report.pdf
	./$(BINARY_NAME) render examples/behoerde.md -o dist/behoerde.pdf
