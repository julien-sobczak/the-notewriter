
GO_FILES = $(shell find . -name "*.go" | grep -v .go | uniq)
GO_PACKAGES = $(shell go list ./... | grep -v .go)
APP_NAME = github.com/julien-sobczak/the-notetaker
APP_VERSION = $(shell git rev-parse HEAD)

build:
	# go build -ldflags "-X \"$(APP_NAME)/cmd.version=$(APP_NAME):$(APP_VERSION)\"" -o build/the-notetaker main.go
	go build -o build/the-notetaker main.go
