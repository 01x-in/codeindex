.PHONY: build test lint typecheck dev clean

BINARY=codeindex
VERSION?=dev
LDFLAGS=-ldflags "-s -w -X github.com/01x/codeindex/internal/cli.Version=$(VERSION)"

build:
	go build $(LDFLAGS) -o bin/$(BINARY) ./cmd/codeindex

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
	rm -rf .codeindex/

install:
	go install $(LDFLAGS) ./cmd/codeindex

release-dry-run:
	goreleaser release --snapshot --clean
