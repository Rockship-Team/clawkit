# Consulting Vertical

Study abroad consulting bot. Handles student management, university application tracking, test score records, and enrollment pipeline.

## Tables

| Table | Purpose |
|-------|---------|
| `students` | Student profiles with target schools and counselor assignments |
| `applications` | University application tracking with status and decisions |
| `test_scores` | IELTS, TOEFL, SAT and other test score records |
| `contacts` | Parent and student contact information |

## Quick Start

```bash
# 1. Create your skill
cp -r templates/verticals/consulting skills/my-agency
cp templates/cli.js skills/my-agency/cli.js

# 2. Customize
#    - Edit SKILL.md: add your agency name, services, persona
#    - Edit schema.json: add/remove fields for your domain

# 3. Register
make generate

# 4. Install
clawkit install my-agency
```

## Profiles

Use profiles to run the same skill for different agencies:

```
skills/my-agency/
  profiles/
    us-consulting/
      profile.yaml      # shop_name: US Study Abroad
    uk-consulting/
      profile.yaml      # shop_name: UK Study Abroad
```

```bash
clawkit install my-agency --profile us-consulting
clawkit install my-agency --profile uk-consulting
```

## Storage Options

Set `db_target` in profile.yaml:

- `local` — JSON files (default, for testing/small operations)
- `supabase` — Supabase cloud database
- `api` — Your own REST API backend
