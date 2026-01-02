.PHONY: build run test clean fmt lint deps

# Binary name
BINARY_NAME=remnanode
BINARY_PATH=bin/$(BINARY_NAME)

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GORUN=$(GOCMD) run
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
GOFMT=$(GOCMD) fmt

# Build flags
LDFLAGS=-ldflags "-s -w"

# Main package
MAIN_PKG=./cmd/remnanode

all: deps build

deps:
	$(GOMOD) download
	$(GOMOD) tidy

build:
	$(GOBUILD) $(LDFLAGS) -o $(BINARY_PATH) $(MAIN_PKG)

run:
	$(GORUN) $(MAIN_PKG)

test:
	$(GOTEST) -v ./...

test-cover:
	$(GOTEST) -v -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html

clean:
	rm -rf bin/
	rm -f coverage.out coverage.html

fmt:
	$(GOFMT) ./...

lint:
	golangci-lint run ./...

# Development build with race detection
build-dev:
	$(GOBUILD) -race -o $(BINARY_PATH) $(MAIN_PKG)

# Production build for Linux
build-linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_PATH)-linux-amd64 $(MAIN_PKG)

# Docker build
docker-build:
	docker build -t remnawave/remnanode:latest .

# Generate protobuf (if needed)
proto:
	protoc --go_out=. --go-grpc_out=. proto/*.proto
