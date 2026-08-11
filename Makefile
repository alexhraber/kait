REGISTRY ?= ghcr.io/alexhraber/kait
VERSION ?= dev
AGENT_VERSION ?= 3.123.1
BIN ?= bin/kait
GOCACHE ?= $(CURDIR)/.gocache

export GOCACHE

.PHONY: all test vet fmt build clean \
	build-plan build-cpu build-slim build-full build-all build-all-full build-all-accelerators

all: test vet build

test:
	go test ./...

vet:
	go vet ./...

fmt:
	gofmt -w cmd/kait/*.go

build:
	mkdir -p $(dir $(BIN))
	CGO_ENABLED=0 go build -trimpath -ldflags='-s -w' -o $(BIN) ./cmd/kait

clean:
	rm -rf bin .gocache

build-plan:
	REGISTRY=$(REGISTRY) VERSION=$(VERSION) AGENT_VERSION=$(AGENT_VERSION) docker buildx bake --print

build-cpu:
	REGISTRY=$(REGISTRY) VERSION=$(VERSION) AGENT_VERSION=$(AGENT_VERSION) docker buildx bake cpu

build-slim:
	REGISTRY=$(REGISTRY) VERSION=$(VERSION) AGENT_VERSION=$(AGENT_VERSION) docker buildx bake cpu-slim apple-slim

build-full:
	REGISTRY=$(REGISTRY) VERSION=$(VERSION) AGENT_VERSION=$(AGENT_VERSION) docker buildx bake cpu-full apple-full

build-all:
	REGISTRY=$(REGISTRY) VERSION=$(VERSION) AGENT_VERSION=$(AGENT_VERSION) docker buildx bake cpu-slim apple-slim

build-all-full:
	REGISTRY=$(REGISTRY) VERSION=$(VERSION) AGENT_VERSION=$(AGENT_VERSION) docker buildx bake cpu-full apple-full

build-all-accelerators:
	REGISTRY=$(REGISTRY) VERSION=$(VERSION) AGENT_VERSION=$(AGENT_VERSION) docker buildx bake nvidia amd intel
