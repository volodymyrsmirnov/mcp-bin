VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)
DATE    ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS  = -s -w \
	-X github.com/volodymyrsmirnov/mcp-bin/internal/version.Version=$(VERSION) \
	-X github.com/volodymyrsmirnov/mcp-bin/internal/version.Commit=$(COMMIT) \
	-X github.com/volodymyrsmirnov/mcp-bin/internal/version.Date=$(DATE)

.PHONY: build test fmt lint vet clean

build: clean
	go build -ldflags="$(LDFLAGS)" -o mcp-bin ./cmd/mcp-bin/

test:
	go test ./...

fmt:
	gofmt -s -w .

lint: vet
	golangci-lint run ./...

vet:
	go vet ./...

clean:
	rm -f mcp-bin mcp-bin-compiled
