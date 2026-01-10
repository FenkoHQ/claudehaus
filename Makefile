.PHONY: build run check test fmt clean

VERSION ?= dev
BINARY := claudehaus

build:
	go build -ldflags="-X main.version=$(VERSION)" -o $(BINARY) ./cmd/claudehaus

run:
	go run ./cmd/claudehaus

check:
	go vet ./...
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "golangci-lint not installed, skipping"; \
	fi

test:
	go test -v ./...

fmt:
	gofmt -w .
	@if command -v goimports >/dev/null 2>&1; then \
		goimports -w .; \
	fi

clean:
	rm -f $(BINARY)
