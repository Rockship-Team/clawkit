# Templates

Generic scaffolding for a new skill. Copy `skill/` into `skills/` and customize.

## Layout

```
skills/
  [...skill]/               Flat skill
    _cli/                   cli.js (and any helpers)
    config.json             version, setup_prompts, exclude
    SKILL.md                name, description, metadata.openclaw.*
  [...group]/               Grouped skills
    _cli/                   cli.js shared across the group
    [...skill]/
      config.json
      SKILL.md
```

## SKILL.md frontmatter

```yaml
name: my-skill
description: One-line purpose
metadata:
  openclaw:
    os: [darwin, linux, windows]
    requires:
      bins: [node]
      config: []
```

Fields: `name`, `description`, `metadata.openclaw.os`, `metadata.openclaw.requires.bins`, `metadata.openclaw.requires.config`.

## config.json

```json
{
  "version": "1.0.0",
  "setup_prompts": [{"key": "skill_title", "label": "Skill title"}],
  "exclude": ["*.tmp"]
}
```

Fields: `version`, `setup_prompts`, `exclude`.
