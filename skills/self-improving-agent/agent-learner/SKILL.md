---
name: agent-learner
description: "Self-learn and improve — save successful workflows, update when gaps are found, search past experience."
metadata:
  openclaw:
    emoji: "🔄"
    requires:
      bins: [vault-cli]
---

# Agent Learner (Hermes Self-Improvement)

You have the ability to SELF-LEARN. After every complex task, save the successful workflow. Before every new task, search for past experience. When a gap is found, update immediately. This is a continuous feedback loop to make you progressively better.

## WHEN TO SAVE A NEW SKILL (save-skill)

Save when:

- Completed a complex task (5+ steps)
- Fixed a difficult bug
- Discovered an efficient workflow
- User confirmed the approach was correct

Do NOT save when:

- Simple conversation with no real workflow
- Sensitive information (passwords, tokens, national ID)
- Content not yet confirmed by the user

## WHEN TO UPDATE A SKILL (patch-skill)

Update when:

- A saved skill is missing steps → patch BEFORE ending the task
- User corrects the workflow → update immediately
- A special case is discovered → add it

## WHEN TO SEARCH PAST EXPERIENCE (search)

Search when:

- Before starting a new task → `vault-cli learn list` to check
- Encountering an error → `vault-cli session search` for similar situations
- User asks "how did we do this before" → search sessions and learned skills

## PERIODIC REMINDER (Nudge Protocol)

Every ~10 conversation turns, ask yourself:

1. "Is there any information worth saving to memory?"
2. "Did I learn a workflow worth recording?"
3. "Is there any outdated or incorrect memory that needs updating?"

If yes, act. If no, continue normally.

## VAULT-CLI COMMANDS

### Managing learned skills

Save a new skill:

```
vault-cli learn save-skill <name> <description> <procedure> [tags]
```

Update a skill (find and replace content):

```
vault-cli learn patch-skill <name> <find_text> <replace_text>
```

List all skills:

```
vault-cli learn list
```

Read skill details:

```
vault-cli learn get <name>
```

### Search

Search session history:

```
vault-cli session search <query>
```

Search the entire vault:

```
vault-cli search <query>
```

### Memory

Save information:

```
vault-cli memory set MEMORY.md <info>
```

Update information:

```
vault-cli memory replace MEMORY.md <old> <new>
```

## ABSOLUTE RULES

- Every `vault-cli` command must be called via exec, ON A SINGLE LINE ONLY.
- NEVER use pipe (`|`), redirect (`>`), heredoc, `&&`, `;`, or subshell.
- Any argument containing spaces must be wrapped in `"double quotes"`.
- Always check `ok:true` in the result before reporting success.
- Only save GENUINELY useful workflows, not every conversation.
- NEVER save sensitive data (passwords, tokens, national ID, bank account numbers).
- Do not save raw conversations — distill them into concrete steps.
- When memory is full (>2200 chars), MUST condense before adding.
- Every skill file MUST have: name, description, step-by-step procedure, creation date.

## SKILL FILE STRUCTURE

Every saved skill must have:

- **name**: short, memorable name (e.g. `payroll-monthly`)
- **description**: one-sentence description of purpose
- **procedure**: specific, clear, actionable steps
- **created**: creation date

## INTERACTION EXAMPLES

### After completing payroll processing

User just completed monthly payroll with these steps: download timesheet, cross-check contracts, calculate personal income tax, export payroll.

Action:

```
vault-cli learn save-skill "payroll-monthly" "Monthly payroll processing workflow" "1. Download timesheet from HR\n2. Cross-check employment contracts\n3. Calculate PIT using progressive tax brackets\n4. Deduct mandatory insurance\n5. Export payroll and send to head accountant for approval" "payroll,finance,monthly"
```

### Before bank reconciliation

User: "I need to reconcile the bank balance this month"

Action — check past experience:

```
vault-cli learn list
```

→ Found skill `bank-reconciliation` saved previously.

```
vault-cli learn get "bank-reconciliation"
```

→ Read the workflow and follow it step by step.

### Discovered a missing step

Following skill `payroll-monthly` but found a missing step for checking annual leave days.

Action:

```
vault-cli learn patch-skill "payroll-monthly" "2. Cross-check employment contracts" "2. Cross-check employment contracts\n3. Check annual leave and public holidays for the month"
```

→ Skill updated, next run will be more complete.
