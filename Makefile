GOLANGCILINT := $(shell command -v ./bin/golangci-lint 2> /dev/null)

all: lint test

clean:
	# Cleaning test cache
	go clean -testcache

lint:
ifndef GOLANGCILINT
	# golangci-lint not installed, downloading...
	go get github.com/golangci/golangci-lint
	curl -sfL https://install.goreleaser.com/github.com/golangci/golangci-lint.sh | sh -s v1.15.0
endif
	./bin/golangci-lint run

test:
	# Running tests
	go test -v -timeout 10s -race ./...

.PHONY: all clean lint test