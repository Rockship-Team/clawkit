#!/usr/bin/env bash
# Register sol-finance-coach cron jobs with OpenClaw Gateway.
# Run once after install: bash skills/sol-finance-coach/cron/setup-cron.sh
#
# Prerequisites: openclaw CLI available, Gateway running.
# Verify after: openclaw cron list
# Remove all:   for id in sol-daily-digest sol-weekly-report sol-deal-alerts sol-monthly-report sol-challenge-reminder sol-data-refresh sol-feedback-prompt sol-loyalty-expiry sol-savings-checkin sol-weekly-quiz; do openclaw cron remove "$id"; done
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
Doc JSON output va format thanh ban tin sang than thien theo SKILL.md muc 12 (Ban tin hang ngay). Bao gom: meo tiet kiem, deal hot, micro-lesson, nhac loyalty het han.
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

# 5. Challenge check-in reminder — 8 PM daily VN time
openclaw cron add \
  --name "sol-challenge-reminder" \
  --cron "0 20 * * *" \
  --tz "Asia/Ho_Chi_Minh" \
  --session isolated \
  --message "Chay: ${SKILL}/sol-cli challenge status
Neu co active challenge va chua check-in hom nay (checkins array khong chua ngay hom nay), gui nhac nho than thien: 'Nho check-in thu thach [ten] hom nay nha! Day [X]/[Y] roi.'
Neu da check-in hoac khong co active challenge thi KHONG gui gi." \
  --announce
echo "  + sol-challenge-reminder (0 20 * * *)"

# 6. Data refresh — Monday 5 AM VN time (weekly)
openclaw cron add \
  --name "sol-data-refresh" \
  --cron "0 5 * * 1" \
  --tz "Asia/Ho_Chi_Minh" \
  --session isolated \
  --message "Chay crawl tool de cap nhat du lieu:
cd skills/sol-finance-coach/tools/crawl && go run . all 2>&1
Crawl du lieu moi tu cac nguon (credit cards, interest rates, deals, loyalty, ecommerce, investment). Doc stderr output de xac nhan cac file da duoc ghi vao data/. Neu co loi crawl trang nao, ghi nhan nhung KHONG bao user." \
  --light-context
echo "  + sol-data-refresh (0 5 * * 1)"

# 7. Feedback prompt — noon daily VN time
openclaw cron add \
  --name "sol-feedback-prompt" \
  --cron "0 12 * * *" \
  --tz "Asia/Ho_Chi_Minh" \
  --session isolated \
  --message "Chay: ${SKILL}/sol-cli feedback stats
Neu total = 0, chay: ${SKILL}/sol-cli onboard status
Neu onboarded = true va created_at cach hom nay >= 7 ngay, hoi user: 'Ban da dung Tai duoc 1 tuan roi! Bot giup ich cho ban khong? Cho minh danh gia 1-5 sao nhe!'
Neu chua du 7 ngay hoac da feedback roi, KHONG gui gi." \
  --announce
echo "  + sol-feedback-prompt (0 12 * * *)"

# 8. Loyalty expiry check — Mon+Thu 9 AM VN time
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

# 9. Mid-month savings check-in — 15th of month, 10 AM VN time
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

# 10. Weekly quiz prompt — Wednesday 7 PM VN time
openclaw cron add \
  --name "sol-weekly-quiz" \
  --cron "0 19 * * 3" \
  --tz "Asia/Ho_Chi_Minh" \
  --session isolated \
  --message "Chay: ${SKILL}/sol-cli quiz random
Trinh bay cau hoi trac nghiem vui: 'Toi thu 4 roi — thoi diem hoc tai chinh! 🧠'
Cho user tra loi.
Chay: ${SKILL}/sol-cli quiz stats
Neu user da tra loi dung 20+ cau, kiem tra badge finance_101." \
  --announce
echo "  + sol-weekly-quiz (0 19 * * 3)"

echo ""
echo "Done! 10 cron jobs registered. Verify: openclaw cron list"
