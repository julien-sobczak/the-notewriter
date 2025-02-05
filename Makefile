.PHONY: build

GO_FILES = $(shell find . -name "*.go" | grep -v .go | uniq)
GO_PACKAGES = $(shell go list ./... | grep -v .go)
APP_NAME = github.com/julien-sobczak/the-notewriter
APP_VERSION = $(shell git rev-parse HEAD)


deps:
	go install github.com/hhatto/gocloc/cmd/gocloc@latest

build:
	go build --tags "fts5" -o build/nt cmd/nt/*.go
	go build --tags "fts5" -o build/ntlite cmd/ntlite/*.go
	go build --tags "fts5" -o build/ntreference cmd/nt-reference/*.go
	go build --tags "fts5" -o build/ntanki cmd/nt-anki/*.go

test:
	go test --tags "fts5" ./... -count=1 -v

cover:
	go test -cover --tags "fts5" ./... -count=1
cover-html:
	@go test -coverprofile="cover.out" --tags "fts5" ./... -count=1
	@go tool cover -html=cover.out

cloc:
	@echo "Counting lines of code..."
	@gocloc --not-match-d="(^website|testdata)" --not-match="_test.go$$" --exclude-ext="json" .
	@echo "Counting lines of code (including tests)..."
	@gocloc --not-match-d="(^website|testdata)" --exclude-ext="json" .

docs:
	npm run --prefix ./website start

# go install gotest.tools/gotestsum@latest
testsum:
	gotestsum -- -tags=fts5 ./... -count=1

test-all:
	go test --tags "fts5 integration" ./... -count=1 -v

install: build
	cp build/nt /Users/julien/go/bin/nt
	cp build/ntreference /Users/julien/go/bin/nt-reference
	cp build/ntanki /Users/julien/go/bin/nt-anki
# go install --tags "fts5" cmd/nt/*.go => FIXME build an invalid main executable instead of a nt file
