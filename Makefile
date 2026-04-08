VERSION := 0.1.0
BINARY  := clawkit
LDFLAGS := -s -w -X main.version=$(VERSION)

.PHONY: build test lint clean dist

## build: Build for current platform
build:
	CGO_ENABLED=0 go build -ldflags "$(LDFLAGS)" -o $(BINARY) .

## test: Run all tests
test:
	CGO_ENABLED=0 go test -v ./...

## lint: Run linter
lint:
	golangci-lint run

## clean: Remove build artifacts
clean:
	rm -rf $(BINARY) dist/

## dist: Cross-compile for all platforms
dist: clean
	@mkdir -p dist
	@echo "Building $(BINARY) v$(VERSION)..."
	CGO_ENABLED=0 GOOS=darwin  GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o dist/$(BINARY)-darwin-arm64 .
	CGO_ENABLED=0 GOOS=darwin  GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o dist/$(BINARY)-darwin-amd64 .
	CGO_ENABLED=0 GOOS=linux   GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o dist/$(BINARY)-linux-amd64 .
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o dist/$(BINARY)-windows-amd64.exe .
	@echo "Done. Binaries in dist/"
	@ls -lh dist/

## package: Package a skill for distribution
package:
	@test -n "$(SKILL)" || (echo "Usage: make package SKILL=shop-hoa-zalo" && exit 1)
	./$(BINARY) package $(SKILL)

## help: Show this help
help:
	@grep -E '^## ' Makefile | sed 's/## //' | column -t -s ':'
