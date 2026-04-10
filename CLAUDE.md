# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

```bash
make build          # Build binary for current platform → ./clawkit
make test           # Run all tests with race detector
make fmt            # go fmt + go vet
make lint           # golangci-lint run ./...
make generate       # Regenerate registry.json from skills/*/SKILL.md
make check-generate # Verify registry.json is in sync (CI check)
make dist           # Cross-compile for darwin/linux/windows
make package SKILL=<name>  # Package a skill to .tar.gz
```

Run a single test package:
```bash
CGO_ENABLED=0 go test -v ./internal/archive/...
```

**Important:** After editing any `skills/*/SKILL.md` frontmatter, always run `make generate` to keep `registry.json` in sync. CI will fail if they diverge (`make check-generate`).

## Architecture

Clawkit is a CLI skill manager for OpenClaw AI agents. Skills are AI prompt files with an install lifecycle (OAuth, config, binaries, templates). The binary is distributed via npm wrapping platform-specific Go binaries (`npm/binaries/`).

### Data flow

1. `registry.json` is generated from `skills/*/SKILL.md` YAML frontmatter by `cmd/gen-registry`. Never edit it by hand.
2. At install time, `internal/installer` fetches the registry (remote GitHub raw URL → local fallback), downloads the skill package from GitHub Releases (or copies from local `skills/` in dev mode), runs OAuth flows, processes templates, and saves `config.json` per skill.
3. Tokens from OAuth are written to the skill's config dir (`~/Library/Application Support/clawkit/<skill>/` on macOS, `~/.config/clawkit/<skill>/` on Linux). Runtime OAuth refresh is handled by OpenClaw/gog, not clawkit.

### Key packages

- **`cmd/clawkit/main.go`** — CLI dispatcher (list, install, update, status, package, version)
- **`internal/installer/commands.go`** — All command implementations; install flow: preflight → download → OAuth → template processing → DB init → config save → gateway restart
- **`internal/installer/registry.go`** — Registry loading (remote + local merge) and skill package downloading
- **`internal/archive/`** — tar.gz / zip extraction and creation; strips top-level directory from archives
- **`internal/config/`** — `SkillConfig` struct, config file read/write, OpenClaw detection
- **`internal/template/`** — SKILL.md placeholder substitution (e.g. `{shopName}`); `catalog.json` loading
- **`internal/ui/`** — ANSI terminal output helpers (Info/Ok/Warn/Fatal) and `PromptInput`
- **`oauth/`** — OAuth providers; each self-registers via `init()`. Add a new provider by creating a new file and calling `Register()` in its `init()`

### Adding a skill

1. Create `skills/<name>/SKILL.md` with required frontmatter (`name`, `description`, `version`).
2. Run `make generate`.
3. Add any OAuth providers referenced in `requires_oauth` to `oauth/` if they don't exist.
4. Skills listed in `requires_skills` must be installed manually by the user first. `clawkit install <skill>` does NOT auto-install dependencies — it checks they exist and fails with install instructions if missing. This keeps each skill's OAuth flow isolated.
5. Binaries listed in `requires_bins` are downloaded from their GitHub Releases at install time.

### Adding an OAuth provider

Implement the `oauth.Provider` interface (`Name()`, `Display()`, `Authenticate() (map[string]string, error)`) and call `oauth.Register(yourProvider{})` in `init()`. The returned map is merged into the skill's `config.json` tokens.

### Cross-platform rules

The binary targets Linux, macOS, and Windows. Follow these rules for any new code:

| Concern | Do | Don't |
|---|---|---|
| File paths | `filepath.Join(a, b)` | `a + "/" + b` |
| Temp directory | `os.TempDir()` | Hardcode `/tmp` |
| Binary names | `name := "gog"; if goos == "windows" { name += ".exe" }` | Assume no `.exe` |
| Open browser | `oauth.OpenBrowser(url)` (already handles all 3 platforms) | Call `open`/`xdg-open` directly |
| Unix-only syscalls | Guard with `if goos != "windows"` | Call `chmod`, `sudo` unconditionally |
| PATH update | Use `ensureInPath()` in `commands.go` | Write shell profile files directly |
| Archive format | `.zip` on Windows, `.tar.gz` elsewhere | Assume tar.gz |

Platform detection uses `runtime.GOOS` values `"darwin"`, `"linux"`, `"windows"` and `runtime.GOARCH` values `"amd64"`, `"arm64"`.

The existing `installGog()` and `ensureInPath()` in `internal/installer/commands.go` are the canonical reference implementations for cross-platform binary installation.

### Zero external dependencies

The module uses only the Go standard library. The YAML frontmatter parser in `cmd/gen-registry` is hand-written. Do not add external dependencies without discussion.

### Release

Releases are triggered by pushing a `v*` tag. The release workflow cross-compiles all platform binaries (`make dist`), packages skills, creates a GitHub Release, and publishes to npm as `@rockship/clawkit`.
