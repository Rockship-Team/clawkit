# Education Vertical

Education and training bot. Handles course enrollment, progress tracking, and student management.

## Tables

| Table | Purpose |
|-------|---------|
| `enrollments` | Student enrollments with progress and completion tracking |
| `courses` | Course catalog with pricing, levels, and scheduling |
| `contacts` | Student and lead contact information |

## Quick Start

```bash
# 1. Create your skill
cp -r templates/verticals/education skills/my-school
cp templates/cli.js skills/my-school/cli.js

# 2. Customize
#    - Edit SKILL.md: add your school name, courses, persona
#    - Edit schema.json: add/remove fields for your domain

# 3. Register
make generate

# 4. Install
clawkit install my-school
```

## Profiles

Use profiles to run the same skill for different schools:

```
skills/my-school/
  profiles/
    english-center/
      profile.yaml      # shop_name: English Center
    coding-bootcamp/
      profile.yaml      # shop_name: Coding Bootcamp
```

```bash
clawkit install my-school --profile english-center
clawkit install my-school --profile coding-bootcamp
```

## Storage Options

Set `db_target` in profile.yaml:

- `local` — JSON files (default, for testing/small operations)
- `supabase` — Supabase cloud database
- `api` — Your own REST API backend
