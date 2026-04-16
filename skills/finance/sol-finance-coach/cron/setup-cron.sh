#!/usr/bin/env bash
# Register sol-finance-coach cron jobs with OpenClaw Gateway.
# Run once after install: bash skills/sol-finance-coach/cron/setup-cron.sh
#
# Prerequisites: openclaw CLI available, Gateway running.
# Verify after: openclaw cron list
# Remove all:   for id in sol-daily-digest sol-weekly-report sol-deal-alerts sol-monthly-report sol-data-refresh sol-loyalty-expiry sol-savings-checkin; do openclaw cron remove "$id"; done
set -euo pipefail

SKILL="skills/sol-finance-coach"

echo "[sol-finance-coach] Registering cron jobs..."

# 1. Daily digest — 7:30 AM VN time
openclaw cron add \
  --name "sol-daily-digest" \
  --cron "30 7 * * *" \
  --tz "Asia/Ho_Chi_Minh" \
  --session isolated \
  --message "Chay: ${SKILL}/sol-cli digest generate
Doc JSON output va format thanh ban tin sang than thien theo SKILL.md muc 7 (Ban tin hang ngay). Bao gom: meo tiet kiem, deal hot, micro-lesson, nhac loyalty het han.
NEU co truong 'budget': them dong 'Budget thang: da chi X/Y (Z%)'.
Gui cho user." \
  --announce
echo "  + sol-daily-digest (30 7 * * *)"

# 2. Weekly spending report — Sunday 8 PM VN time
openclaw cron add \
  --name "sol-weekly-report" \
  --cron "0 20 * * 0" \
  --tz "Asia/Ho_Chi_Minh" \
  --session isolated \
  --message "Chay: ${SKILL}/sol-cli spend report week
Doc JSON output va render thanh ASCII bar chart phan loai chi tieu. Goi y tiet kiem cu the dua tren danh muc cao nhat. So sanh voi tuan truoc neu co du lieu." \
  --announce
echo "  + sol-weekly-report (0 20 * * 0)"

# 3. Deal alerts — 3x daily (8 AM, noon, 6 PM) VN time
openclaw cron add \
  --name "sol-deal-alerts" \
  --cron "0 8,12,18 * * *" \
  --tz "Asia/Ho_Chi_Minh" \
  --session isolated \
  --message "Chay: ${SKILL}/sol-cli deals match
Neu co deal phu hop (count > 0), format top 1-2 deal thanh tin nhan ngan.
Neu khong co match thi KHONG gui gi." \
  --announce
echo "  + sol-deal-alerts (0 8,12,18 * * *)"

# 4. Monthly spending report — 1st of month, 9 AM VN time
openclaw cron add \
  --name "sol-monthly-report" \
  --cron "0 9 1 * *" \
  --tz "Asia/Ho_Chi_Minh" \
  --session isolated \
  --message "Chay: ${SKILL}/sol-cli spend report month
Doc JSON output. Render bao cao tong hop thang voi ASCII bar chart. Goi y 2-3 hanh dong cu the de toi uu chi tieu thang toi. So sanh voi thang truoc neu co du lieu." \
  --announce
echo "  + sol-monthly-report (0 9 1 * *)"

# 5. Data refresh — Monday 5 AM VN time (weekly)
openclaw cron add \
  --name "sol-data-refresh" \
  --cron "0 5 * * 1" \
  --tz "Asia/Ho_Chi_Minh" \
  --session isolated \
  --message "Chay: ${SKILL}/sol-cli data refresh
Lam moi du lieu cards, deals, loyalty tu cac nguon crawl.
Neu loi thi chi log noi bo, KHONG gui thong bao cho user." \
  --light-context
echo "  + sol-data-refresh (0 5 * * 1)"

# 6. Loyalty expiry check — Mon+Thu 9 AM VN time
openclaw cron add \
  --name "sol-loyalty-expiry" \
  --cron "0 9 * * 1,4" \
  --tz "Asia/Ho_Chi_Minh" \
  --session isolated \
  --message "Chay: ${SKILL}/sol-cli loyalty expiring
Neu co diem sap het han (count > 0), gui tin nhan rieng voi:
- Ten chuong trinh + so diem + ngay het han
- Goi y cu the cach doi (vi du: '12,000 Lotusmiles = 1 ve HN-SGN')
Neu khong co diem sap het han, KHONG gui gi." \
  --announce
echo "  + sol-loyalty-expiry (0 9 * * 1,4)"

# 7. Mid-month savings check-in — 15th of month, 10 AM VN time
openclaw cron add \
  --name "sol-savings-checkin" \
  --cron "0 10 15 * *" \
  --tz "Asia/Ho_Chi_Minh" \
  --session isolated \
  --message "Chay: ${SKILL}/sol-cli spend report month
Chay: ${SKILL}/sol-cli profile get
Neu user co monthly_budget > 0:
- Tinh da chi bao nhieu so voi budget, con bao nhieu ngay
- Toc do chi tieu hien tai co vuot budget khong
- Gui: 'Giua thang roi! Ban da chi [X]d / [budget]d. [Goi y].'
Neu khong co budget, goi y: 'Ban chua dat budget thang. Nhan \"dat budget 10tr\" de minh theo doi giup!'
Neu khong co giao dich, KHONG gui gi." \
  --announce
echo "  + sol-savings-checkin (0 10 15 * *)"

echo ""
echo "Done! 7 cron jobs registered. Verify: openclaw cron list"
