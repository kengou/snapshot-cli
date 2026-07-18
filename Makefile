# Makefile for snapshot-cli
BINARY_NAME = snapshot-cli

## Version metadata injected into github.com/sapcc/go-api-declarations/bininfo
## via ldflags. Overridable for reproducible/container builds (no .git present):
##   make build VERSION=v1.2.3 COMMIT=abc1234
BININFO_PKG = github.com/sapcc/go-api-declarations/bininfo
VERSION    ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT     ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)
BUILD_DATE ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
GO_LDFLAGS  = -X $(BININFO_PKG).binName=$(BINARY_NAME) \
              -X $(BININFO_PKG).version=$(VERSION) \
              -X $(BININFO_PKG).commit=$(COMMIT) \
              -X $(BININFO_PKG).buildDate=$(BUILD_DATE)

.PHONY: build clean run docker test lint golint

build: build-$(BINARY_NAME)

build-%:
	go build -ldflags "$(GO_LDFLAGS)" -o bin/$* ./cmd/snapshot-cli

clean:
	rm -rf bin cover.out

run: build
	./bin/$(BINARY_NAME)

docker:
	docker build -t $(BINARY_NAME):latest .

test:
	go test ./... -coverprofile cover.out -v

## Location to install dependencies an GO binaries
LOCALBIN ?= $(shell pwd)/bin
$(LOCALBIN):
	mkdir -p $(LOCALBIN)

GOLINT ?= $(LOCALBIN)/golangci-lint
GOLINT_VERSION ?= 2.12.2

lint: golint
	$(GOLINT) run -v --timeout 5m

golint: $(GOLINT)
$(GOLINT): $(LOCALBIN)
	GOBIN=$(LOCALBIN) go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v$(GOLINT_VERSION)
