.PHONY: build test fmt lint vet clean

build: clean
	go build -o mcp-bin ./cmd/mcp-bin/

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
