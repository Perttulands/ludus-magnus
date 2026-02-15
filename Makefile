GO ?= $(shell command -v go 2>/dev/null || echo /usr/local/go/bin/go)
BINARY := ludus-magnus

.PHONY: build test install clean

build:
	@mkdir -p bin
	$(GO) build -trimpath -ldflags "-s -w" -o bin/$(BINARY) .

test:
	$(GO) test ./...

test-integration:
	$(GO) test ./test/integration/... -v -count=1

install:
	$(GO) install .

clean:
	rm -rf bin dist coverage.out
