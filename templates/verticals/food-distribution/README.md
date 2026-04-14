# Food Distribution Vertical

Food distribution bot. Handles order management, inventory tracking with batch numbers and expiry dates, and customer accounts.

## Tables

| Table | Purpose |
|-------|---------|
| `orders` | Wholesale and retail orders with delivery tracking |
| `order_items` | Line items per order with batch numbers |
| `products` | Product catalog with pricing, shelf life, and storage |
| `inventory` | Batch-level inventory with expiry and warehouse location |
| `contacts` | Customer and retailer contact information |

## Quick Start

```bash
# 1. Create your skill
cp -r templates/verticals/food-distribution skills/my-distributor
cp templates/cli.js skills/my-distributor/cli.js

# 2. Customize
#    - Edit SKILL.md: add your company name, product lines, persona
#    - Edit schema.json: add/remove fields for your domain

# 3. Register
make generate

# 4. Install
clawkit install my-distributor
```

## Profiles

Use profiles to run the same skill for different operations:

```
skills/my-distributor/
  profiles/
    fresh-produce/
      profile.yaml      # shop_name: Fresh Produce Co
    frozen-food/
      profile.yaml      # shop_name: Frozen Food Distributor
```

```bash
clawkit install my-distributor --profile fresh-produce
clawkit install my-distributor --profile frozen-food
```

## Storage Options

Set `db_target` in profile.yaml:

- `local` — JSON files (default, for testing/small operations)
- `supabase` — Supabase cloud database
- `api` — Your own REST API backend
