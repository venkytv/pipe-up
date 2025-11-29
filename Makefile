BINARY ?= tts-cached
CMD_PKG ?= ./cmd/tts-cached
GOCACHE ?= $(CURDIR)/.gocache

.PHONY: build test fmt tidy clean

build:
	@mkdir -p bin
	CGO_ENABLED=0 GOCACHE=$(GOCACHE) go build -o bin/$(BINARY) $(CMD_PKG)

build-linux-arm64:
	@mkdir -p bin
	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 GOCACHE=$(GOCACHE) go build -o bin/$(BINARY)-linux-arm64 $(CMD_PKG)

test:
	GOCACHE=$(GOCACHE) go test ./...

fmt:
	gofmt -w $(shell find . -name '*.go' -not -path './vendor/*')

tidy:
	go mod tidy

clean:
	rm -rf bin $(GOCACHE)
