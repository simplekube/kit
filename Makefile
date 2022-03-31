# credit: https://github.com/fastly/go-fastly/blob/main/Makefile

SHELL := /bin/bash -o pipefail

# List of tests to run
FILES ?= ./...

# List all our actual files, excluding vendor
GOPKGS ?= $(shell go list $(FILES) | grep -v /vendor/)
GOFILES ?= $(shell find . -name '*.go' | grep -v /vendor/)

# Tags specific for building
GOTAGS ?=

NAME := $(notdir $(shell pwd))

## Runs all of the required cleaning and verification targets.
all: mod-download dev-dependencies tidy fmt fiximports test
.PHONY: all

tidy: ## Cleans the Go module.
	@echo "==> Tidying module"
	@go mod tidy
.PHONY: tidy

mod-download: ## Downloads the Go module.
	@echo "==> Downloading Go module"
	@go mod download
.PHONY: mod-download

dev-dependencies: ## Downloads the necessesary dev dependencies.
	@echo "==> Downloading development dependencies"
	@go install honnef.co/go/tools/cmd/staticcheck@latest
	@go install golang.org/x/tools/cmd/goimports@latest
.PHONY: dev-dependencies

test: vet staticcheck ## Runs the test suite with VCR mocks enabled.
	@echo "==> Testing ${NAME}"
	@go test -timeout=30s -parallel=20 -tags="${GOTAGS}" ${GOPKGS} ${TESTARGS}
.PHONY: test

test-race: ## Runs the test suite with the -race flag to identify race conditions, if they exist.
	@echo "==> Testing ${NAME} (race)"
	@go test -timeout=60s -race -tags="${GOTAGS}" ${GOPKGS} ${TESTARGS}
.PHONY: test-race

test-full: ## Runs the tests with VCR disabled (i.e., makes external calls).
	@echo "==> Testing ${NAME} with VCR disabled"
	@VCR_DISABLE=1 \
		bash -c \
		'go test -timeout=60s -parallel=20 ${GOPKGS} ${TESTARGS}'
.PHONY: test-full


fiximports: ## Properly formats and orders imports.
	@echo "==> Fixing imports"
	@goimports -w {pkg,}
.PHONY: fiximports

fmt: ## Properly formats Go files and orders dependencies.
	@echo "==> Running gofmt"
	@gofmt -s -w ${GOFILES}
.PHONY: fmt

vet: ## Identifies common errors.
	@echo "==> Running go vet"
	@go vet ./...
.PHONY: vet

staticcheck: ## Runs the staticcheck linter.
	@echo "==> Running staticcheck"
	@staticcheck ./...
.PHONY: staticcheck

.PHONY: help
help: ## Prints this help menu.
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'
