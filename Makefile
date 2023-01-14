.PHONY: build

GO_FILES = $(shell find . -name "*.go" | grep -v .go | uniq)
GO_PACKAGES = $(shell go list ./... | grep -v .go)
APP_NAME = github.com/julien-sobczak/the-notetaker
APP_VERSION = $(shell git rev-parse HEAD)

build:
	go build --tags "fts5" -o build/nt main.go

test:
	go test --tags "fts5" ./... -count=1

# Development-only target
build-example:
	env NT_HOME="./example" go run main.go build
