.PHONY: fmt lint install install-codex install-dprint install-gitleaks install-dev-tools

GOCACHE ?= /tmp/gomut-gocache
GOLANGCI_LINT_CACHE ?= /tmp/golangci-lint
LOCAL_BIN ?= $(HOME)/.local/bin
export PATH := $(LOCAL_BIN):$(PATH)

fmt:
	GOCACHE=$(GOCACHE) GOLANGCI_LINT_CACHE=$(GOLANGCI_LINT_CACHE) golangci-lint fmt
	GOCACHE=$(GOCACHE) GOLANGCI_LINT_CACHE=$(GOLANGCI_LINT_CACHE) golangci-lint run --fix
	dprint fmt

lint:
	gitleaks detect --no-banner --redact --source .
	GOCACHE=$(GOCACHE) GOLANGCI_LINT_CACHE=$(GOLANGCI_LINT_CACHE) golangci-lint run ./cmd/gomut ./internal/gomut ./sample

install:
	go install ./cmd/gomut

install-codex:
	@set -eu; \
	if ! command -v codex >/dev/null 2>&1; then \
		curl -fsSL https://chatgpt.com/codex/install.sh | sh; \
	fi; \
	codex --version

install-dprint:
	@set -eu; \
	mkdir -p '$(LOCAL_BIN)'; \
	if ! command -v dprint >/dev/null 2>&1; then \
		tmpdir="$$(mktemp -d)"; \
		url="$$(curl -fsSL https://api.github.com/repos/dprint/dprint/releases/latest | grep -Eo '"browser_download_url": *"[^"]+"' | cut -d'"' -f4 | grep -E 'linux.*(x86_64|x64).*\.tar\.gz$$' | head -n1)"; \
		[ -n "$$url" ]; \
		curl -fsSL "$$url" -o "$$tmpdir/dprint.tar.gz"; \
		tar -xzf "$$tmpdir/dprint.tar.gz" -C "$$tmpdir"; \
		binary="$$(find "$$tmpdir" -type f -name dprint -perm -u+x | head -n1)"; \
		install -m 755 "$$binary" '$(LOCAL_BIN)/dprint'; \
		rm -rf "$$tmpdir"; \
	fi

install-gitleaks:
	@set -eu; \
	mkdir -p '$(LOCAL_BIN)'; \
	if ! command -v gitleaks >/dev/null 2>&1; then \
		tmpdir="$$(mktemp -d)"; \
		url="$$(curl -fsSL https://api.github.com/repos/gitleaks/gitleaks/releases/latest | grep -Eo '"browser_download_url": *"[^"]+"' | cut -d'"' -f4 | grep -E 'linux.*(x64|amd64).*\.tar\.gz$$' | head -n1)"; \
		[ -n "$$url" ]; \
		curl -fsSL "$$url" -o "$$tmpdir/gitleaks.tar.gz"; \
		tar -xzf "$$tmpdir/gitleaks.tar.gz" -C "$$tmpdir"; \
		binary="$$(find "$$tmpdir" -type f -name gitleaks -perm -u+x | head -n1)"; \
		install -m 755 "$$binary" '$(LOCAL_BIN)/gitleaks'; \
		rm -rf "$$tmpdir"; \
	fi

install-dev-tools: install-codex install-dprint install-gitleaks
