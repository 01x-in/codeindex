.PHONY: build test lint typecheck dev clean

BINARY=code-index
VERSION?=dev
LDFLAGS=-ldflags "-s -w -X github.com/01x/codeindex/internal/cli.Version=$(VERSION)"

build:
	go build $(LDFLAGS) -o bin/$(BINARY) ./cmd/code-index

test:
	go test ./... -v

test-race:
	go test -race ./...

lint:
	golangci-lint run ./...

typecheck:
	go vet ./...

dev: build
	./bin/$(BINARY)

clean:
	rm -rf bin/
	rm -rf .code-index/

install:
	go install $(LDFLAGS) ./cmd/code-index
