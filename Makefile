GO ?= $(shell command -v go 2>/dev/null || echo /usr/local/go/bin/go)
BINARY := academy
GOCACHE ?= $(CURDIR)/.gocache
GOMODCACHE ?= $(CURDIR)/.gomodcache

.PHONY: build test install clean

build:
	@mkdir -p bin $(GOCACHE) $(GOMODCACHE)
	GOCACHE=$(GOCACHE) GOMODCACHE=$(GOMODCACHE) $(GO) build -trimpath -ldflags "-s -w -X github.com/Perttulands/agent-academy/internal/cli.Version=dev" -o bin/$(BINARY) ./cmd/academy

test:
	@mkdir -p $(GOCACHE) $(GOMODCACHE)
	GOCACHE=$(GOCACHE) GOMODCACHE=$(GOMODCACHE) $(GO) test ./...

install:
	@mkdir -p $(GOCACHE) $(GOMODCACHE)
	GOCACHE=$(GOCACHE) GOMODCACHE=$(GOMODCACHE) $(GO) install ./cmd/academy

clean:
	rm -rf bin dist coverage.out $(GOCACHE) $(GOMODCACHE)
