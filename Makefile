.PHONY: all build server cli run run-server run-cli test test-race vet fmt lint clean config install install-systemd help

# Build metadata. Override on the make command line, e.g.:
#   make VERSION=v0.1.0 COMMIT=$(git rev-parse --short HEAD)
VERSION   ?= dev
COMMIT    ?= none
BUILD_TIME ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS   := -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.buildTime=$(BUILD_TIME)

# Output directory and binary names.
BINDIR    ?= bin
SERVER    := $(BINDIR)/llama-admin-server
CLI       := $(BINDIR)/llama-admin

# Install prefix and systemd paths (used by install-systemd).
PREFIX    ?= /usr/local
SYSCONF  ?= /etc
UNITDIR  ?= /etc/systemd/system
DATA_DIR ?= /var/lib/llama-admin

GOFLAGS   ?=
GO        := go

all: build ## Build server and CLI binaries

build: server cli

server: ## Build the llama-admin-server binary
	$(GO) build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o $(SERVER) ./cmd/server

cli: ## Build the llama-admin CLI
	$(GO) build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o $(CLI) ./cmd/llama-admin

run: run-server ## Run the server locally (alias for run-server)
run-server: server
	./$(SERVER)

run-cli: cli
	./$(CLI)

test: ## Run the test suite
	$(GO) test ./...

test-race: ## Run the test suite with the race detector
	$(GO) test -race ./...

vet: ## Run go vet
	$(GO) vet ./...

fmt: ## Format all Go sources
	$(GO) fmt ./...

lint: vet ## Run golangci-lint if available
	@command -v golangci-lint >/dev/null 2>&1 && golangci-lint run ./... || echo "golangci-lint not installed; skipping"

clean: ## Remove built binaries
	rm -rf $(BINDIR)

# Copy the example config into place if no config exists yet, then edit it.
config: ## Create config.yaml from config.example.yaml
	@if [ -f config.yaml ]; then \
		echo "config.yaml already exists; leaving it alone"; \
	else \
		cp config.example.yaml config.yaml; \
		echo "Created config.yaml from config.example.yaml — edit it now"; \
	fi

# Install binaries + example config + systemd unit (requires root).
install: build ## Install binaries and example config under PREFIX
	install -d $(DESTDIR)$(PREFIX)/bin
	install -m 0755 $(SERVER) $(DESTDIR)$(PREFIX)/bin/llama-admin-server
	install -m 0755 $(CLI)    $(DESTDIR)$(PREFIX)/bin/llama-admin
	install -d $(DESTDIR)$(SYSCONF)/llama-admin
	install -m 0644 config.example.yaml $(DESTDIR)$(SYSCONF)/llama-admin/config.example.yaml

install-systemd: install ## Install binaries, config, and systemd unit
	install -d $(DESTDIR)$(UNITDIR)
	install -m 0644 deploy/llama-admin.service $(DESTDIR)$(UNITDIR)/llama-admin.service
	install -d $(DESTDIR)$(DATA_DIR)
	@echo
	@echo "Installed. To enable and start:"
	@echo "  sudo systemctl daemon-reload"
	@echo "  sudo systemctl enable --now llama-admin"

help: ## Show this help
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-16s %s\n", $$1, $$2}' $(MAKEFILE_LIST)
