GOLANGCI_VERSION ?= v2.11.4

.PHONY: help mod-verify vet lint staticcheck build test test-race all

help:
	@echo "Targets:"
	@echo "  mod-verify   go mod verify"
	@echo "  vet          go vet ./..."
	@echo "  lint         golangci-lint via go run ($(GOLANGCI_VERSION))"
	@echo "  staticcheck  staticcheck ./...        (via go run; no global install)"
	@echo "  build        go build -v ./..."
	@echo "  test         go test ./..."
	@echo "  test-race    go test -race ./..."
	@echo "  all          mod-verify, lint, build, test-race"

mod-verify:
	go mod verify

vet:
	go vet ./...

lint:
	go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_VERSION) run ./...

staticcheck:
	go run honnef.co/go/tools/cmd/staticcheck@latest ./...

build:
	go build -v ./...

test:
	go test ./...

test-race:
	go test -race ./...

all: mod-verify lint build test-race
