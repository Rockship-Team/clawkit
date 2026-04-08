VERSION := 0.2.0
BINARY  := clawkit
CMD     := ./cmd/clawkit
LDFLAGS := -s -w -X main.version=$(VERSION)

.PHONY: build test lint fmt clean dist package coverage generate check-generate help

## build: Build for current platform
build:
	CGO_ENABLED=0 go build -ldflags "$(LDFLAGS)" -o $(BINARY) $(CMD)

## test: Run all tests with race detector
test:
	CGO_ENABLED=0 go test -v ./...

## coverage: Run tests with coverage report
coverage:
	CGO_ENABLED=0 go test -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out
	@echo ""
	@echo "HTML report: go tool cover -html=coverage.out"

## generate: Generate registry.json from skills/*/SKILL.md frontmatter
generate:
	go run ./cmd/gen-registry

## check-generate: Verify registry.json is up to date (fails if outdated)
check-generate:
	go run ./cmd/gen-registry -check

## lint: Run golangci-lint
lint:
	golangci-lint run ./...

## fmt: Format and vet code
fmt:
	go fmt ./...
	go vet ./...

## clean: Remove build artifacts
clean:
	rm -rf $(BINARY) dist/ coverage.out

## dist: Cross-compile for all platforms
dist: clean
	@mkdir -p dist
	@echo "Building $(BINARY) v$(VERSION)..."
	CGO_ENABLED=0 GOOS=darwin  GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o dist/$(BINARY)-darwin-arm64 $(CMD)
	CGO_ENABLED=0 GOOS=darwin  GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o dist/$(BINARY)-darwin-amd64 $(CMD)
	CGO_ENABLED=0 GOOS=linux   GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o dist/$(BINARY)-linux-amd64 $(CMD)
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o dist/$(BINARY)-windows-amd64.exe $(CMD)
	@echo "Done. Binaries in dist/"
	@ls -lh dist/

## package: Package a skill for distribution
package:
	@test -n "$(SKILL)" || (echo "Usage: make package SKILL=shop-hoa-zalo" && exit 1)
	./$(BINARY) package $(SKILL)

## help: Show this help
help:
	@grep -E '^## ' Makefile | sed 's/## //' | column -t -s ':'
