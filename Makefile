VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)
DATE    ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS  = -s -w \
	-X github.com/volodymyrsmirnov/mcp-bin/internal/version.Version=$(VERSION) \
	-X github.com/volodymyrsmirnov/mcp-bin/internal/version.Commit=$(COMMIT) \
	-X github.com/volodymyrsmirnov/mcp-bin/internal/version.Date=$(DATE)

.PHONY: build test fmt fmt-check lint vet vulncheck clean help

build:
	go build -ldflags="$(LDFLAGS)" -o mcp-bin ./cmd/mcp-bin/

test:
	go test -race -coverprofile=coverage.out -covermode=atomic ./...

fmt:
	gofmt -s -w .

fmt-check:
	@gofmt -s -l . | tee /dev/stderr | (! read)

lint: vet
	golangci-lint run ./...

vet:
	go vet ./...

vulncheck:
	govulncheck ./...

clean:
	rm -f mcp-bin mcp-bin-compiled coverage.out

help:
	@echo "Available targets:"
	@echo "  build      - Build the mcp-bin binary"
	@echo "  test       - Run tests with race detector and coverage"
	@echo "  fmt        - Format Go source files"
	@echo "  fmt-check  - Check formatting without modifying files"
	@echo "  lint       - Run go vet and golangci-lint"
	@echo "  vet        - Run go vet"
	@echo "  vulncheck  - Run vulnerability scanner"
	@echo "  clean      - Remove built binaries and coverage files"
	@echo "  help       - Show this help message"
