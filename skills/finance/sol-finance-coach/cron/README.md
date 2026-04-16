# Cron Jobs — sol-finance-coach

Cron jobs are registered with the OpenClaw Gateway via `openclaw cron add`.
Run once after install:

```bash
bash skills/sol-finance-coach/cron/setup-cron.sh
```

## Jobs

| Name | Schedule | Style | Purpose |
| ---- | -------- | ----- | ------- |
| sol-daily-digest | `30 7 * * *` | isolated + announce | Morning financial digest |
| sol-weekly-report | `0 20 * * 0` | isolated + announce | Weekly spending summary |
| sol-deal-alerts | `0 8,12,18 * * *` | isolated + announce | Deal matching 3x/day |
| sol-monthly-report | `0 9 1 * *` | isolated + announce | Monthly spending analysis |
| sol-data-refresh | `0 5 * * 1` | isolated + light-context | Weekly crawl data refresh |
| sol-loyalty-expiry | `0 9 * * 1,4` | isolated + announce | Loyalty points expiry check |
| sol-savings-checkin | `0 10 15 * *` | isolated + announce | Mid-month budget check-in |

All schedules use timezone `Asia/Ho_Chi_Minh`.

## Schedule overview

| Time | Job | Frequency |
| ---- | --- | --------- |
| 05:00 Mon | sol-data-refresh | Weekly (silent) |
| 07:30 Daily | sol-daily-digest | Daily |
| 08:00/12:00/18:00 | sol-deal-alerts | 3x daily (conditional) |
| 09:00 Mon+Thu | sol-loyalty-expiry | 2x weekly (conditional) |
| 09:00 1st | sol-monthly-report | Monthly |
| 10:00 15th | sol-savings-checkin | Monthly (conditional) |
| 20:00 Sun | sol-weekly-report | Weekly |

## Management

```bash
openclaw cron list                    # View all jobs
openclaw cron run <jobId>             # Force-run a job
openclaw cron runs --id <jobId>       # View run history
openclaw cron edit <jobId> --message "..."  # Update prompt
openclaw cron remove <jobId>          # Delete a job
```

## Remove all jobs

```bash
for id in sol-daily-digest sol-weekly-report sol-deal-alerts \
  sol-monthly-report sol-data-refresh sol-loyalty-expiry \
  sol-savings-checkin; do
  openclaw cron remove "$id"
done
```
