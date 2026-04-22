VERSION := 0.5.1
BINARY  := clawkit
CMD     := ./cmd/clawkit
LDFLAGS := -s -w -X github.com/rockship-co/clawkit/internal/version.Version=$(VERSION)

.PHONY: build test test-race lint fmt clean dist coverage generate check-generate \
        release-check bump npm-stage npm-pack npm-publish help

## build: Build for current platform
build:
	CGO_ENABLED=0 go build -ldflags "$(LDFLAGS)" -o $(BINARY) $(CMD)

## test: Run all tests
test:
	CGO_ENABLED=0 go test -v ./...

## test-race: Run tests with the race detector (requires CGO)
test-race:
	CGO_ENABLED=1 go test -race ./...

## coverage: Run tests with coverage report
coverage:
	CGO_ENABLED=0 go test -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out
	@echo ""
	@echo "HTML report: go tool cover -html=coverage.out"

## generate: Generate registry.json from skills/**/{SKILL.md,config.json}
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
	rm -rf npm/binaries/$(BINARY)-* npm/skills npm/registry.json npm/*.tgz

## dist: Cross-compile for all platforms into dist/
dist:
	@rm -rf dist/
	@mkdir -p dist
	@echo "Building $(BINARY) v$(VERSION)..."
	CGO_ENABLED=0 GOOS=darwin  GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o dist/$(BINARY)-darwin-arm64 $(CMD)
	CGO_ENABLED=0 GOOS=darwin  GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o dist/$(BINARY)-darwin-amd64 $(CMD)
	CGO_ENABLED=0 GOOS=linux   GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o dist/$(BINARY)-linux-amd64 $(CMD)
	CGO_ENABLED=0 GOOS=linux   GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o dist/$(BINARY)-linux-arm64 $(CMD)
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o dist/$(BINARY)-windows-amd64.exe $(CMD)
	@echo "Done. Binaries in dist/"
	@ls -lh dist/

## npm-stage: Copy binaries, skills, and registry.json into npm/ for packaging
npm-stage: dist
	@echo "Staging npm/ package..."
	@rm -rf npm/skills npm/registry.json
	cp dist/$(BINARY)-darwin-arm64      npm/binaries/$(BINARY)-darwin-arm64
	cp dist/$(BINARY)-darwin-amd64      npm/binaries/$(BINARY)-darwin-amd64
	cp dist/$(BINARY)-linux-amd64       npm/binaries/$(BINARY)-linux-amd64
	cp dist/$(BINARY)-linux-arm64       npm/binaries/$(BINARY)-linux-arm64
	cp dist/$(BINARY)-windows-amd64.exe npm/binaries/$(BINARY)-windows-amd64.exe
	@mkdir -p npm/skills
	@cp -R skills/. npm/skills/
	@find npm/skills -name skills.go -delete
	@cp internal/installer/registry.json npm/registry.json
	@cd npm && npm version $(VERSION) --no-git-tag-version --allow-same-version >/dev/null
	@echo "Staged npm/ with binaries, skills, registry.json (version $(VERSION))."

## npm-pack: Stage and run `npm pack` for local smoke testing
npm-pack: npm-stage
	cd npm && npm pack
	@echo "Local tarball ready: npm/rockship-clawkit-$(VERSION).tgz"

## npm-publish: Stage and publish to GitHub Packages (needs NODE_AUTH_TOKEN)
npm-publish: npm-stage
	cd npm && npm publish

## release-check: Run everything the release workflow will run (dry run)
release-check: fmt check-generate test npm-stage
	@echo ""
	@echo "Release check passed. To release:"
	@echo "  make bump V=x.y.z"
	@echo "  git commit -am 'Release vx.y.z' && git tag vx.y.z && git push && git push --tags"

## bump: Sync VERSION across Makefile and npm/package.json (pass V=x.y.z)
bump:
	@test -n "$(V)" || (echo "Usage: make bump V=x.y.z" && exit 1)
	@sed -i.bak 's/^VERSION := .*/VERSION := $(V)/' Makefile && rm Makefile.bak
	@cd npm && npm version $(V) --no-git-tag-version --allow-same-version
	@echo "Bumped to $(V). Review changes, then:"
	@echo "  git commit -am 'Release v$(V)' && git tag v$(V) && git push && git push --tags"

## help: Show this help
help:
	@grep -E '^## ' Makefile | sed 's/## //' | column -t -s ':'
