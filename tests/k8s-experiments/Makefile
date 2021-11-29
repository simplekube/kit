# credit: https://github.com/fastly/go-fastly/blob/main/Makefile

SHELL := /bin/bash -o pipefail

# List of tests to run
FILES ?= ./...

IMG_TAG ?= $(shell git fetch --all --tags | echo "" | git describe --always --tags)

# List all our actual files, excluding vendor
GOPKGS ?= $(shell go list $(FILES) | grep -v /vendor/)
GOFILES ?= $(shell find . -name '*.go' | grep -v /vendor/)

# Kubernetes related tags
NAMESPACE := e2e-system
SERVICE_ACCOUNT_NAME := ${NAMESPACE}
POD_NAME := e2e
SUITE_NAMESPACE_LBL := e2e-testing=yes
CLUSTER_NAME ?= k3s-default

# Tags specific for building
GOTAGS ?=

NAME := $(notdir $(shell pwd))

# We default to local k3d registry as the default registry
K3D_REGISTRY_NAME ?= k3d-local-registry
REGISTRY_NAME ?= local
K3D_REGISTRY_PORT ?= 12345
REGISTRY_PORT ?= ${K3D_REGISTRY_PORT}
IMG_NAME ?= ${NAME}
IMG_REPO ?= ${REGISTRY_NAME}:${REGISTRY_PORT}/${IMG_NAME}

## Runs vendoring & unit testing checks if any
all: mod-download build-dependencies mod-tidy fmt fiximports vet staticcheck test
.PHONY: all

binary-clean:
	@echo "==> Deleting $(IMG_NAME) binary at bin/"
	@rm -rf bin/${IMG_NAME}
.PHONY: binary-clean

binary: binary-clean all $(IMG_NAME)
.PHONY: binary

$(IMG_NAME): $(GOFILES)
	@echo "==> Generating $(IMG_NAME) binary at bin/"
	@CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
		go build -o bin/$@ main.go

$(GOFILES): ;

mod-vendor: ## Vendors the Go module.
	@echo "==> Vendoring module"
	@go mod vendor
.PHONY: mod-vendor

mod-tidy: ## Cleans the Go module.
	@echo "==> Tidying module"
	@go mod tidy
.PHONY: mod-tidy

mod-download: ## Downloads the Go module.
	@echo "==> Downloading Go module"
	@go mod download
.PHONY: mod-download

wonderland2-k8s-dependencies:
	@echo "==> Generating Wonderland2 API deepcopy object"
	@controller-gen object \
		paths=./vendor/github.com/Jimdo/wonderland2-k8s-operator/api/v1/... \
		output:dir=./vendor/github.com/Jimdo/wonderland2-k8s-operator/api/v1/
.PHONY: wonderland2-k8s-dependencies

build-dependencies: ## Downloads the build related dependencies.
	@echo "==> Downloading build dependencies"
	@go install honnef.co/go/tools/cmd/staticcheck@latest
	@go install golang.org/x/tools/cmd/goimports@latest
	@go install github.com/banzaicloud/kurun@latest
.PHONY: build-dependencies

test: ## Runs the test suite with VCR mocks enabled.
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

check-imports: ## A check which lists improperly-formatted imports, if they exist.
	@$(shell pwd)/../../scripts/check-imports.sh
.PHONY: check-imports

check-fmt: ## A check which lists improperly-formatted files, if they exist.
	@$(shell pwd)/../../scripts/check-gofmt.sh
.PHONY: check-fmt

check-mod: ## A check which lists extraneous dependencies, if they exist.
	@$(shell pwd)/../../scripts/check-mod.sh
.PHONY: check-mod

fiximports: ## Properly formats and orders imports.
	@echo "==> Fixing imports"
	@goimports -w {checks,setup}
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

image: binary
	@echo "==> Build docker image for binary ${IMG_NAME}"
	@echo ""
	@docker rmi $(IMG_REPO):$(IMG_TAG) || true
	docker build -t $(IMG_REPO):$(IMG_TAG) .
.PHONY: image

k3d-push: image
	@echo "==> Push image to k3d cluster"
	@echo ""
	@k3d image import ${IMG_REPO}:${IMG_TAG} || echo "[ERR] Docker & K3d should be installed & running"
.PHONY: k3d-push

run: delete-stale teardown
	@echo "==> Run binary as a K8s Pod"
	@kubectl apply -f ./manifests/rbac.yaml
	@kubectl -n ${NAMESPACE} run ${POD_NAME} --image=$(IMG_REPO):$(IMG_TAG) \
		--overrides='{"spec": {"restartPolicy": "Never", "serviceAccountName": "${SERVICE_ACCOUNT_NAME}"} }'
.PHONY: run

logs:
	@echo "==> Logs of Pod ${NAMESPACE} / ${POD_NAME}"
	@kubectl -n ${NAMESPACE} logs ${POD_NAME}
.PHONY: logs

teardown: ## Removes the Kubernetes Pod
	@echo "==> Teardown e2e setup"
	@kubectl -n ${NAMESPACE} delete po ${POD_NAME} || true
	@kubectl delete -f ./manifests/rbac.yaml || true
 .PHONY: teardown

delete-stale: ## Removes the Kubernetes Pod
	@echo "==> Delete stale e2e suite resources if any"
	@kubectl delete ns -l ${SUITE_NAMESPACE_LBL} || true
 .PHONY: delete-stale

.PHONY: help
help: ## Prints this help menu.
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'
