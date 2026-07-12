.PHONY: fmt lint install

GOCACHE ?= /tmp/gomut-gocache
GOLANGCI_LINT_CACHE ?= /tmp/golangci-lint

fmt:
	GOCACHE=$(GOCACHE) GOLANGCI_LINT_CACHE=$(GOLANGCI_LINT_CACHE) golangci-lint fmt
	GOCACHE=$(GOCACHE) GOLANGCI_LINT_CACHE=$(GOLANGCI_LINT_CACHE) golangci-lint run --fix
	dprint fmt

lint:
	GOCACHE=$(GOCACHE) GOLANGCI_LINT_CACHE=$(GOLANGCI_LINT_CACHE) golangci-lint run ./cmd/gomut ./internal/gomut ./sample

install:
	go install ./cmd/gomut
