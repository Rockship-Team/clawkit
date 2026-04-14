# Templates

Everything needed to create a new skill. Pick a vertical, copy, customize.

## Structure

```
templates/
  cli.js                          Generic CLI (copy to every skill)
  verticals/
    ecommerce/                    Online/chat shop bot
      schema.json                 4 tables: orders, order_items, products, contacts
      SKILL.md                    AI prompt skeleton
      README.md                   Setup guide
    education/                    Course enrollment bot
      schema.json                 3 tables: enrollments, courses, contacts
      SKILL.md
      README.md
    consulting/                   Study abroad consulting bot
      schema.json                 4 tables: students, applications, test_scores, contacts
      SKILL.md
      README.md
    gold/                         Jewelry & gold trading bot
      schema.json                 4 tables: transactions, products, price_board, contacts
      SKILL.md
      README.md
    food-distribution/            Food wholesale/distribution bot
      schema.json                 5 tables: orders, order_items, products, inventory, contacts
      SKILL.md
      README.md
```

## Creating a New Skill

```bash
# 1. Pick a vertical and copy it
cp -r templates/verticals/ecommerce skills/my-shop

# 2. Copy the generic CLI
cp templates/cli.js skills/my-shop/cli.js

# 3. Customize
#    - SKILL.md: your shop name, products, prices, AI persona
#    - schema.json: add/remove fields for your domain

# 4. Register and install
make generate
clawkit install my-shop
```

## Standard Skill Layout

```
skills/my-skill/
  SKILL.md              Required  AI prompt (YAML frontmatter + markdown)
  schema.json           Required  Database schema (tables, fields, roles)
  cli.js                Required  Generic CLI (copy from templates/cli.js)
  catalog.json          Optional  Product categories for image directories
  workspace-overrides/  Optional  MD files to override agent persona
  products/             Optional  Product images organized by folder
  profiles/             Optional  Domain-specific overrides
```
