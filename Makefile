SHELL:=$(PREFIX)/bin/bash

all: verify test

verify: fmt vet lint nakedreturns duplicatecode constants

clean:
	# Cleaning test cache
	go clean -testcache

fmt:
	# Checking project code formatting...
	! gofmt -d . | read || ( gofmt -d . && exit 1 )

vet:
	# Checking for suspicious constructs
	go vet ./...

lint:
	@command -v golint >/dev/null 2>&1 || go get github.com/golang/lint/golint
	# Checking project code style...
	! ( golint ./... | grep -v "ALL_CAPS" )

imports:
	@command -v goimports >/dev/null 2>&1 || go get golang.org/x/tools/cmd/goimports
	# Fixing imports
	goimports -w .

nakedreturns:
	@command -v nakedret >/dev/null 2>&1 || go get github.com/alexkohler/nakedret
	# Checking for naked returns
	! nakedret ./... 2>&1 | read || ( nakedret ./... && exit 1 )

duplicatecode:
	@command -v dupl >/dev/null 2>&1 || go get github.com/mibk/dupl
	# Checking for duplicate code
	! find . -name '*.go' | grep -v '_test.go' | dupl -t 75 -plumbing -files | read || ( find . -name '*.go' | grep -v '_test.go' | dupl -t 75 -files && exit 1 )

constants:
	@command -v goconst >/dev/null 2>&1 || go get github.com/jgautheron/goconst/cmd/goconst
	# Checking for possible constants
	! goconst ./... | read || ( goconst ./... && exit 1 )

test:
	# Running tests
	go test -v -timeout 10s -race ./...

.PHONY: all verify clean fmt vet lint imports nakedreturns duplicatecode constants test