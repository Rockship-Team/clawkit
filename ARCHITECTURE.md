# Architecture

Technical reference for contributors and developers.

---

## System Overview

```
User Machine                          External
┌──────────────────────────┐         ┌──────────────────┐
│ clawkit CLI (Go)         │         │ npm registry     │
│   ├── install/update     │◄────────│ GitHub Releases  │
│   ├── registry lookup    │         │                  │
│   └── template render    │         └──────────────────┘
│                          │
│ Skill runtime (per skill)│
│   └── _cli/ (any lang)   │
│                          │
│ OpenClaw Runtime         │
│   ├── AI agent           │
│   └── Channels           │
└──────────────────────────┘
```

---

## Install Flow

```
clawkit install <skill>
  │
  ├── 1. Preflight: detect OpenClaw
  ├── 2. Registry lookup: load skill metadata
  ├── 3. Download: local dev → embedded → GitHub Releases
  ├── 4. Group _cli merge: if skill is under skills/<group>/<skill>/ and has
  │      no _cli/ of its own, copy the group's shared _cli/ into the install
  ├── 5. Install bins: download required CLIs declared in requires_bins
  ├── 6. Collect setup prompts: read user_inputs interactively
  ├── 7. Allowlist update: openclaw config set agents.defaults.skills [...]
  ├── 8. Template processing: replace {key} placeholders in SKILL.md
  └── 9. Save clawkit.json: { version, user_inputs }
```

Update re-runs steps 3, 4, 8, and 9, reusing the stored `user_inputs` —
setup prompts are not asked again.

---

## Skill Layout

Flat skill:

```
skills/<skill>/
  _cli/                   cli.js or any runtime helpers
  config.json             { version, setup_prompts, exclude }
  SKILL.md                frontmatter + agent prompt
```

Grouped skills share one `_cli/` at the group level:

```
skills/<group>/
  _cli/                   shared runtime for every child skill
  <skill-a>/
    config.json
    SKILL.md
  <skill-b>/
    config.json
    SKILL.md
```

The installer copies the group's `_cli/` into each installed child skill if
the child doesn't define its own.

---

## Metadata Sources

Skill metadata is split into two files:

**`SKILL.md` frontmatter** — OpenClaw-native fields, consumed by the agent
runtime and by `gen-registry`:

```yaml
---
name: my-skill
description: What this skill does
metadata:
  openclaw:
    os: [darwin, linux, windows]
    requires:
      bins: [node]
      config: []
---
```

**`config.json`** — clawkit-specific, consumed only by `gen-registry`:

```json
{
  "version": "1.0.0",
  "setup_prompts": [{"key": "shop_name", "label": "Shop name"}],
  "exclude": ["*.tmp"]
}
```

**`registry.json`** — generated from both; each entry contains:
`description, os, requires_bins, requires_config, version, setup_prompts, exclude`.
Regenerate with `make generate`; CI enforces sync via `make check-generate`.

**`clawkit.json`** — written at install time into the installed skill
directory:

```json
{
  "version": "1.0.0",
  "user_inputs": {"shop_name": "Hoa Xuan"}
}
```

Used on update to re-bake placeholders without re-prompting.

---

## Registry Generation

`cmd/gen-registry` walks `skills/` recursively, skipping any `_cli/`
directory. For each `SKILL.md` it finds, it:

1. Parses the YAML frontmatter with a hand-written indent-aware parser
   (zero deps) — supports nested maps and inline flow arrays `[a, b, c]`.
2. Flattens `metadata.openclaw.os`, `metadata.openclaw.requires.bins`,
   `metadata.openclaw.requires.config` into the entry.
3. Reads the sibling `config.json` for `version`, `setup_prompts`, `exclude`.
4. Emits a merged record keyed by directory name.

The directory name is the canonical skill key — the `name:` field in
frontmatter is informational only.

---

## Skill Resolution Order

When installing `<name>`:

1. **Local dev** — `skills/<name>/` or `skills/<group>/<name>/` (one level
   of nesting). `findLocalSkill` returns the first match with a `SKILL.md`.
2. **Embedded** — `skills.FindSkill` searches the `//go:embed` FS.
3. **Remote** — `.tar.gz` from
   `github.com/Rockship-Team/clawkit/releases/latest/download/<name>.tar.gz`.

For local and embedded sources, if the matched path is under a group and
the skill has no `_cli/`, the group's `_cli/` is merged into the target
directory after the main copy.

---

## Workspace Allowlist

clawkit updates OpenClaw's skill allowlist so installed skills show up in
`<available_skills>`:

- **Install** appends the skill to `agents.defaults.skills`.
- **Uninstall** removes it; when the last skill is removed, the allowlist
  entry is cleared (`openclaw config unset agents.defaults.skills`).

No workspace persona files, backups, or bootstrap copies are managed —
skills customise behaviour purely through their own SKILL.md and `_cli/`.

---

## Directory Structure

```
clawkit/
  cmd/
    clawkit/                CLI entry point
    gen-registry/           Registry generator + frontmatter parser
  internal/
    archive/                tar.gz / zip
    config/                 SkillConfig, OpenClaw detection
    installer/              Commands, registry, allowlist
    template/               {key} placeholder substitution
    dashboard/              Web dashboard
    ui/                     Terminal output helpers
  skills/                   Built-in skills (grouped by vertical)
  templates/                Scaffolding for new skills
    skill/                  Flat-skill template
    group/                  Group-with-shared-_cli template
  npm/                      npm package wrapper with platform binaries
```

---

## Release

1. `make release-check` — local dry run: `fmt + check-generate + test + dist`.
2. `make bump V=1.2.0` — syncs VERSION in `Makefile` and `npm/package.json`
   so dev view and published view can't drift.
3. Commit, tag `v1.2.0`, push tag:

```bash
git commit -am 'Release v1.2.0'
git tag v1.2.0
git push && git push --tags
```

The `v*` tag triggers `.github/workflows/release.yml`, which:

- Re-runs `make check-generate` and `make test` on the tag
- Runs `make dist` to cross-compile 5 binaries
- Packages each Unix binary into `clawkit-v<ver>-<os>-<arch>.tar.gz`,
  keeps Windows as a raw `.exe`, and uploads them to the GitHub Release
- Copies the raw `dist/` binaries into `npm/binaries/` and
  `npm publish --access public` as `@rockship/clawkit`

The workflow also `sed`s the Makefile VERSION in-place as a safety net
in case step 2 was skipped, but the canonical flow is `make bump` first.
