.PHONY: build test lint install clean

build:
	go build -o bin/blocket-pp-cli ./cmd/blocket-pp-cli

test:
	go test ./...

lint:
	golangci-lint run

install:
	go install ./cmd/blocket-pp-cli

clean:
	rm -rf bin/

build-mcp:
	go build -o bin/blocket-pp-mcp ./cmd/blocket-pp-mcp

install-mcp:
	go install ./cmd/blocket-pp-mcp

build-all: build build-mcp
