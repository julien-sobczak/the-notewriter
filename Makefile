.PHONY: build

GO_FILES = $(shell find . -name "*.go" | grep -v .go | uniq)
GO_PACKAGES = $(shell go list ./... | grep -v .go)
APP_NAME = github.com/julien-sobczak/the-notewriter
APP_VERSION = $(shell git rev-parse HEAD)

build:
	go build --tags "fts5" -o build/nt cmd/nt/nt.go
	go build --tags "fts5" -o build/nt-lite cmd/nt-lite/nt-lite.go

test:
	go test --tags "fts5" ./... -count=1 -v

docs:
	npm run --prefix ./website start

# go install gotest.tools/gotestsum@latest
testsum:
	gotestsum -- -tags=fts5 ./... -count=1

test-all:
	go test --tags "fts5 integration" ./... -count=1 -v

install:
	go install --tags "fts5" cmd/nt/nt.go
