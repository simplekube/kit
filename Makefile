# credit: https://github.com/fastly/go-fastly/blob/main/Makefile

SHELL := /bin/bash -o pipefail

# List of tests to run
FILES ?= ./...

# List all our actual files, excluding vendor
GOPKGS ?= $(shell go list $(FILES) | grep -v /vendor/)
GOFILES ?= $(shell find . -name '*.go' | grep -v /vendor/)

# Tags specific to build the binary
GOTAGS ?=

NAME := $(notdir $(shell pwd))

## Default target
all: test
.PHONY: all

## Clean up project's Go modules
# Note: This should be run by humans & not CI systems so that
# the changes can be accounted for as git commits & also
# provide a chance for code reviews
#
# refer - https://go.dev/blog/supply-chain
tidy:
	@echo "==> Tidying module"
	@go mod tidy
.PHONY: tidy

## Download project's Go modules
mod-download:
	@echo "==> Downloading Go module"
	@go mod download
.PHONY: mod-download

## Download necessary dev dependencies
dev-dependencies:
	@echo "==> Downloading development dependencies"
	@go install honnef.co/go/tools/cmd/staticcheck@latest
	@go install golang.org/x/tools/cmd/goimports@latest
.PHONY: dev-dependencies

## Runs the test suite with VCR mocks enabled.
test: lint
	@echo "==> Testing ${NAME}"
	@go test -timeout=30s -parallel=20 -tags="${GOTAGS}" ${GOPKGS} ${TESTARGS}
.PHONY: test

## Runs the test suite with the -race flag to identify race conditions, if they exist.
test-race: lint
	@echo "==> Testing ${NAME} (race)"
	@go test -timeout=60s -race -tags="${GOTAGS}" ${GOPKGS} ${TESTARGS}
.PHONY: test-race

## Runs the tests with VCR disabled (i.e., makes external calls).
test-full: lint
	@echo "==> Testing ${NAME} with VCR disabled"
	@VCR_DISABLE=1 \
		bash -c \
		'go test -timeout=60s -parallel=20 ${GOPKGS} ${TESTARGS}'
.PHONY: test-full

## Properly formats and orders imports
fiximports:
	@echo "==> Fixing imports"
	@goimports -w {pkg,}
.PHONY: fiximports

## Format Go files
fmt:
	@echo "==> Running gofmt"
	@gofmt -s -w ${GOFILES}
.PHONY: fmt

## Identify common errors
vet:
	@echo "==> Running go vet"
	@go vet ./...
.PHONY: vet

## Run staticcheck linter
staticcheck:
	@echo "==> Running staticcheck"
	@staticcheck ./...
.PHONY: staticcheck

lint: dev-dependencies fiximports fmt vet staticcheck
.PHONY: lint

.PHONY: help
help: ## Prints this help menu.
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'
