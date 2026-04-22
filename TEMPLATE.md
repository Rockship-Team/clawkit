# Template

Reference for authoring a new skill. Use this as the checklist when creating `skills/<name>/` (flat) or `skills/<group>/` (grouped).

## Layout

```text
skills/
  [...skill]/                 Flat skill
    _bootstrap/               Markdown persona files copied to workspace root on install
    _cli/                     Runtime payload (binary, data, …)
    _cli.json                 Runtime metadata: { exclude, data_paths, bins }
    _config.json              Dev metadata: { version, setup_prompts }
    SKILL.md                  Frontmatter + agent prompt
  [...group]/                 Grouped skills (share _bootstrap / _cli / _cli.json)
    _bootstrap/
    _cli/
    _cli.json
    [...skill]/
      _config.json
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

## _config.json

Dev-time clawkit metadata, consumed only by `gen-registry`:

```json
{
  "version": "1.0.0",
  "setup_prompts": [{"key": "skill_title", "label": "Skill title"}]
}
```

Fields: `version`, `setup_prompts`. Not copied to the install.

## _cli.json

Runtime install rules, colocated with `_cli/`:

```json
{
  "exclude":    ["cmd"],
  "data_paths": ["data"],
  "bins":       ["my-cli"]
}
```

- `exclude` — paths inside `_cli/` skipped on install (source directories like `cmd/`, tests, …).
- `data_paths` — paths preserved across re-installs (shared databases, user-written files).
- `bins` — names symlinked into `~/.clawkit/bin/` (on `PATH`).

On install, the installer copies `_cli/` → `~/.clawkit/runtimes/<key>/`, where `<key>` is the group name for grouped skills or the skill name for flat skills. Every member of a group shares one runtime directory, so a single `sa-cli` binary and a single `sa.db` are reused across every member — no duplication, no diverged state.

## _bootstrap/

Markdown persona files (IDENTITY.md, SOUL.md, safety_rules.md, …). On install, every `.md` in `_bootstrap/` is copied to the **workspace root**, overwriting any existing file of the same name. For grouped skills, every member installs the same set.

## Install summary

| Source | Destination | Overwrites? |
|--------|-------------|-------------|
| `SKILL.md` (with baked placeholders) | `<workspace>/skills/<skill>/SKILL.md` | Yes |
| `_bootstrap/*.md` | `<workspace>/<file>.md` (workspace root) | Yes, every install |
| `_cli/` (honoring `_cli.json#exclude` and `data_paths`) | `~/.clawkit/runtimes/<key>/` | Binaries yes; data_paths preserved |
| `_cli.json#bins` | Symlinks in `~/.clawkit/bin/` | Yes |
| — | `<workspace>/skills/<skill>/clawkit.json` (written fresh) | — |

`_config.json`, `_cli/`, `_cli.json`, and `_bootstrap/` are never copied into the installed skill directory.
