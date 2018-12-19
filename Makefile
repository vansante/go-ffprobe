SHELL:=$(PREFIX)/bin/bash

all: verify test

verify: fmt vet lint nakedreturns duplicatecode constants

clean:
	# Cleaning test cache
	go clean -testcache

deps:
	# Fetching dependencies
	go get -u github.com/golang/lint/golint golang.org/x/tools/cmd/goimports github.com/alexkohler/nakedret \
		github.com/mibk/dupl github.com/jgautheron/goconst/cmd/goconst

fmt:
	# Checking project code formatting...
	! gofmt -d . | read || ( gofmt -d . && exit 1 )

vet:
	# Checking for suspicious constructs
	go vet ./...

lint: deps
	# Checking project code style...
	! ( golint ./... | grep -v "ALL_CAPS" )

imports: deps
	# Fixing imports
	goimports -w .

nakedreturns: deps
	# Checking for naked returns
	! nakedret ./... 2>&1 | read || ( nakedret ./... && exit 1 )

duplicatecode: deps
	# Checking for duplicate code
	! dupl -t 75 -plumbing | read || ( dupl -t 75 && exit 1 )

constants: deps
	# Checking for possible constants
	! goconst ./... | read || ( goconst ./... && exit 1 )

test:
	# Running tests
	go test -v -race ./...

