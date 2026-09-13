# ------------------------------------------------------------------------------
# Project
# ------------------------------------------------------------------------------

APP_NAME := bram

GO       ?= go
PREFIX   ?= $(HOME)/.local
GOFLAGS  ?=

.PHONY: all
all: build

.PHONY: build compile
build: install
compile:
	GO="$(GO)" PREFIX="$(PREFIX)" bash scripts/build-local.sh --compile-only

.PHONY: run
run: compile
	./.build/$(APP_NAME)

.PHONY: test
test:
	$(GO) test ./...

.PHONY: vet
vet:
	$(GO) vet ./...

.PHONY: fmt
fmt:
	$(GO) fmt ./...

.PHONY: clean
clean:
	rm -rf .build

.PHONY: install
install:
	GO="$(GO)" PREFIX="$(PREFIX)" bash scripts/build-local.sh

.PHONY: help
help:
	@echo "Targets:"
	@echo "  build         Build and install ~/.local/bin/bram"
	@echo "  compile       Compile only to .build/bram"
	@echo "  run           Build and run"
	@echo "  test          Run tests"
	@echo "  vet           Run go vet"
	@echo "  fmt           Run go fmt"
	@echo "  clean         Remove build artifacts"
	@echo "  install       Build and install using PREFIX (default ~/.local)"

