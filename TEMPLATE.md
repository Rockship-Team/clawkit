# Template

Reference for authoring a new skill. Use this as the checklist when creating `skills/<name>/` (flat) or `skills/<group>/` (grouped).

## Layout

```text
skills/
  [...skill]/                 Flat skill
    _bootstrap/               Markdown persona files copied to workspace root on install
    _engine/                     Runtime payload (binary, data, …)
    engine.json                 Runtime metadata: { exclude, data_paths, bins }
    config.json              Dev metadata: { version, setup_prompts }
    SKILL.md                  Frontmatter + agent prompt
  [...group]/                 Grouped skills (share _bootstrap / _engine / engine.json)
    _bootstrap/
    _engine/
    engine.json
    [...skill]/
      config.json
      SKILL.md
```

The four files prefixed with `_` are **never copied into the installed skill directory**. See below for where each one actually lands at install time.

## SKILL.md frontmatter

OpenClaw-native metadata, consumed by the agent and by `gen-registry`:

```yaml
---
name: my-skill
description: One-line purpose
metadata:
  openclaw:
    os: [darwin, linux, windows]
    requires:
      bins: [node]
      config: []
---
```

Fields: `name`, `description`, `metadata.openclaw.os`, `metadata.openclaw.requires.bins`, `metadata.openclaw.requires.config`.

`SKILL.md` is the only file from the skill directory that ships to the installed location. `{key}` placeholders in the body are replaced at install time with values collected from `setup_prompts`.

## config.json

Dev-time clawkit metadata, consumed only by `gen-registry`:

```json
{
  "version": "1.0.0",
  "setup_prompts": [{"key": "skill_title", "label": "Skill title"}]
}
```

Fields: `version`, `setup_prompts`. Not copied to the install.

## engine.json

Runtime install rules, colocated with `_engine/`:

```json
{
  "exclude":    ["cmd"],
  "data_paths": ["data"],
  "bins":       ["my-cli"]
}
```

- `exclude` — paths inside `_engine/` skipped on install (source directories like `cmd/`, tests, …).
- `data_paths` — paths preserved across re-installs (shared databases, user-written files).
- `bins` — names symlinked into `~/.clawkit/bin/` (on `PATH`).

On install, the installer copies `_engine/` → `~/.clawkit/engines/<key>/`, where `<key>` is the group name for grouped skills or the skill name for flat skills. Every member of a group shares one engine directory, so a single `sa-cli` binary and a single `sa.db` are reused across every member — no duplication, no diverged state.

## _bootstrap/

Markdown persona files (IDENTITY.md, SOUL.md, safety_rules.md, …). On install, every `.md` in `_bootstrap/` is copied to the **workspace root**, overwriting any existing file of the same name. For grouped skills, every member installs the same set.

## Install summary

| Source | Destination | Overwrites? |
|--------|-------------|-------------|
| `SKILL.md` (with baked placeholders) | `<workspace>/skills/<skill>/SKILL.md` | Yes |
| `_bootstrap/*.md` | `<workspace>/<file>.md` (workspace root) | Yes, every install |
| `_engine/` (honoring `engine.json#exclude` and `data_paths`) | `~/.clawkit/engines/<key>/` | Binaries yes; data_paths preserved |
| `engine.json#bins` | Symlinks in `~/.clawkit/bin/` | Yes |
| — | `<workspace>/skills/<skill>/clawkit.json` (written fresh) | — |

`config.json`, `_engine/`, `engine.json`, and `_bootstrap/` are never copied into the installed skill directory.
