BINARY_NAME := kidsmedia
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS := -ldflags "-X main.version=$(VERSION) -X main.buildTime=$(BUILD_TIME)"

.PHONY: build build-pi test test-cover clean install lint

build:
	CGO_ENABLED=1 go build $(LDFLAGS) -o $(BINARY_NAME) ./cmd/player/

build-pi:
	GOOS=linux GOARCH=arm64 CGO_ENABLED=1 CC=aarch64-linux-gnu-gcc go build $(LDFLAGS) -o $(BINARY_NAME) ./cmd/player/

test:
	CGO_ENABLED=1 go test ./... -v

test-cover:
	CGO_ENABLED=1 go test ./... -coverprofile=coverage.out
	go tool cover -func=coverage.out

clean:
	rm -f $(BINARY_NAME) coverage.out

install: build-pi
	sudo bash scripts/install.sh

lint:
	go vet ./...
