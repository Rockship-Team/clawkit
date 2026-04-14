# Gold Vertical

Jewelry and gold trading bot. Handles transaction management, product inventory, price board tracking, and customer records.

## Tables

| Table | Purpose |
|-------|---------|
| `transactions` | Buy/sell/exchange transactions with payment tracking |
| `products` | Jewelry and gold product inventory with purity and weight |
| `price_board` | Live gold price quotes by type |
| `contacts` | Customer information with ID documents |

## Quick Start

```bash
# 1. Create your skill
cp -r templates/verticals/gold skills/my-goldshop
cp templates/cli.js skills/my-goldshop/cli.js

# 2. Customize
#    - Edit SKILL.md: add your shop name, gold types, persona
#    - Edit schema.json: add/remove fields for your domain

# 3. Register
make generate

# 4. Install
clawkit install my-goldshop
```

## Profiles

Use profiles to run the same skill for different shops:

```
skills/my-goldshop/
  profiles/
    main-store/
      profile.yaml      # shop_name: Tiem Vang Kim Thanh
    branch/
      profile.yaml      # shop_name: Chi Nhanh 2
```

```bash
clawkit install my-goldshop --profile main-store
clawkit install my-goldshop --profile branch
```

## Storage Options

Set `db_target` in profile.yaml:

- `local` — JSON files (default, for testing/small shops)
- `supabase` — Supabase cloud database
- `api` — Your own REST API backend
