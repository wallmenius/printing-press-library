.PHONY: build test lint install clean

build:
	go build -o bin/se-prices-pp-cli ./cmd/se-prices-pp-cli

test:
	go test ./...

lint:
	golangci-lint run

install:
	go install ./cmd/se-prices-pp-cli

clean:
	rm -rf bin/

build-mcp:
	go build -o bin/se-prices-pp-mcp ./cmd/se-prices-pp-mcp

install-mcp:
	go install ./cmd/se-prices-pp-mcp

build-all: build build-mcp
