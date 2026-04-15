# E-Commerce Vertical

Sales bot for online/chat-based shops. Handles product consultation, order creation, payment tracking, and order lookup.

## Tables

| Table | Purpose |
|-------|---------|
| `orders` | Customer orders with status tracking |
| `order_items` | Line items per order |
| `products` | Product catalog with pricing and stock |
| `contacts` | Customer information |

## Quick Start

```bash
# 1. Create your skill
cp -r templates/verticals/ecommerce skills/my-shop
cp templates/cli.js skills/my-shop/cli.js

# 2. Customize
#    - Edit SKILL.md: add your shop name, products, prices, persona
#    - Edit schema.json: add/remove fields for your domain
#    - Add product images to products/ folder

# 3. Register
make generate

# 4. Install
clawkit install my-shop
```

## Profiles

Use profiles to run the same skill for different shops:

```
skills/my-shop/
  profiles/
    shop-hoa/
      profile.yaml      # shop_name: Shop Hoa Tuoi
      catalog.json
      products/
    bakery/
      profile.yaml      # shop_name: My Bakery
      catalog.json
      products/
```

```bash
clawkit install my-shop --profile shop-hoa
clawkit install my-shop --profile bakery
```

## Storage Options

Set `db_target` in profile.yaml:

- `local` — JSON files (default, for testing/small shops)
- `supabase` — Supabase cloud database
- `api` — Your own REST API backend
