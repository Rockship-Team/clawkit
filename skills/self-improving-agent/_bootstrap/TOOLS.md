# TOOLS.md - Vault CLI Tools

Skills define _how_ tools work. This file is for _your_ specifics — vault path, db location, quick command reference.

## Configuration

Defaults (when no vault-config.json is found):
```
vault_path  → ~/.openclaw/workspace/skills/knowledge-vault
db_path     → ~/.openclaw/workspace/skills/knowledge-vault/session.db
config      → ~/.openclaw/workspace/skills/knowledge-vault/vault-config.json
```

To override, edit `vault-config.json` (supports `~` expansion):
```json
{ "vault_path": "~/MyVault", "db_path": "~/MyVault/session.db" }
```

### Config resolution order (first match wins)

```
1. $VAULT_CONFIG env var  →  absolute path to vault-config.json
2. vault-config.json in current working directory
3. ~/.openclaw/workspace/skills/knowledge-vault/vault-config.json
```

### vault-cli binary (resolve once, use first found)

```
1. which vault-cli
2. ~/.openclaw/workspace/skills/_cli/vault-cli
3. ~/.openclaw/workspace/skills/self-improving-agent/_cli/vault-cli
```

If not found → inform user once, skip all vault-cli calls, continue normally.

## vault-cli Commands

### note
```
vault-cli note add <path> <body> [key=value ...]   create note with frontmatter
vault-cli note get <path>                          read note + links + tags
vault-cli note list [dir]                          list .md files
vault-cli note search <query>                      keyword search across vault
vault-cli note append <path> <text>                append to existing note
```

### memory
```
vault-cli memory show                              show MEMORY.md + USER.md with char counts
vault-cli memory get <MEMORY.md|USER.md>           read one file
vault-cli memory set <file> <entry>                add entry (rejects duplicates + over-cap)
vault-cli memory replace <file> <old> <new>        replace entry containing old
vault-cli memory remove <file> <substring>         remove first matching entry
```

Limits: `MEMORY.md` → 2,200 chars · `USER.md` → 1,375 chars

**Priority:** save to vault-cli first. Only fall back to writing `MEMORY.md`/`USER.md` directly if vault-cli is unavailable. Never write the same info to both.

### session
```
vault-cli session save <id> <title> <skill> <role> <content>   save message
vault-cli session search <query>                               full-text search
vault-cli session list [limit]                                 list recent
```

### learn
```
vault-cli learn save-skill <name> <desc> <procedure> [tags]   save new skill
vault-cli learn patch-skill <name> <find> <replace>           update skill
vault-cli learn list                                           list skills
vault-cli learn get <name>                                     read skill
```

Skills stored in `<vault_path>/skills/`.

### search
```
vault-cli search <query>    search vault notes + session history simultaneously
```

## Note Frontmatter

```markdown
---
title: Note Title
tags: tag1,tag2
created: YYYY-MM-DD
---
```

## Folder Conventions

```
meetings/   YYYY-MM-DD-topic.md
projects/   project-name.md
notes/      general notes
reference/  reference material
daily/      YYYY-MM-DD.md
skills/     learned procedures (vault-cli learn)
```
