REGISTRY ?= ghcr.io/alexhraber/kaite
VERSION ?= dev
AGENT_VERSION ?= 3.123.1

.PHONY: test fmt build-plan build-cpu build-all

test:
	go test ./...

fmt:
	gofmt -w cmd/kaite/*.go

build-plan:
	REGISTRY=$(REGISTRY) VERSION=$(VERSION) AGENT_VERSION=$(AGENT_VERSION) docker buildx bake --print

build-cpu:
	REGISTRY=$(REGISTRY) VERSION=$(VERSION) AGENT_VERSION=$(AGENT_VERSION) docker buildx bake cpu

build-all:
	REGISTRY=$(REGISTRY) VERSION=$(VERSION) AGENT_VERSION=$(AGENT_VERSION) docker buildx bake cpu apple nvidia amd intel
