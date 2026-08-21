---
name: sme-strategy
description: "Strategic layer cho BD/GTM SME — research market+competitor, propose tactics adjust theo pipeline data, validate hypothesis. KICH HOAT khi user noi 'research thi truong', 'phan tich pipeline', 'dieu chinh chien luoc', 'pivot tactic', 'tai sao win/lose', 'so sanh competitor X', 'next quarter plan', 'review thang qua', 'ROI campaign', 'hypothesis nay co dung khong', 'data noi gi'. KHONG execute action (delegate sang sme-crm/sme-marketing). Output luon co: insight + recommendation + measurable next step."
metadata:
  openclaw:
    requires:
      bins: ["sme-cli"]
---

# sme-strategy

Strategic advisor layer. Khac sales (execute) — strategy is "*think before act*".

## TRIGGERS

User noi:
- "research thi truong X", "phan tich competitor Y"
- "tai sao deal Z lose?", "tai sao 8 contact PROPOSAL chua reply?"
- "dieu chinh chien luoc", "pivot tactic", "thay doi approach"
- "ROI campaign tuan qua", "so sanh 2 campaign A vs B"
- "hypothesis: SaaS founder tra loi nhanh hon SME owner — dung khong?"
- "next quarter plan", "Q3 strategy"
- "data noi gi ve cohort khach hang nay"

## 4 OUTPUT TYPES

### A. RESEARCH (market/competitor scan)

Input: tu khoa thi truong / competitor name / niche.

Workflow:
1. `gog gmail search "from:<competitor>" --max 20` (xem ho gui gi qua mail neu da chu y)
2. `web_search` (neu co plugin) cho company background, product, pricing
3. Delegate `sme-crm` neu can list khach hien tai trong niche
4. Output:
   - Tom tat 1 paragraph
   - 3 differentiator competitor co
   - 2 gap minh khai thac duoc
   - 1 recommendation cu the (vd "thu A/B test pricing low-tier 30%")

### B. PIPELINE ANALYSIS

Input: question ve pipeline state (vd "8 PROPOSAL idle 3 ngay co binh thuong khong?")

Workflow:
1. `sme-cli cosmo api POST /v2/contacts/search` lay data thuc
2. `sme-cli cosmo api GET /v1/interactions/contacts/{id}` lay lich su (sau khi rule log work)
3. Phan tich:
   - Median days-to-reply theo customer_type
   - Conversion rate moi stage
   - Stuck deals (>2x median)
4. Output: insight + so + recommendation (vd "Prospect median reply 4d, 8 contact >7d la outlier → suggest follow-up tay")

### C. TACTIC ADJUSTMENT

Input: "campaign reply rate thap" / "deal duration tang"

Workflow:
1. Pull data 2 cohort (truoc vs sau adjust)
2. So sanh metric: open rate, reply rate, meeting rate, win rate
3. Hypothesis test: cause = subject line / time / channel / persona?
4. Output: cohort A vs B table + recommendation tac thiet ke moi

### D. HYPOTHESIS VALIDATION

Input: "SaaS CEO reply nhanh hon SME owner — dung khong?"

Workflow:
1. Split data: cohort `customer_type=Prospect` + role contains "CEO" + industry segmentation
2. Tinh mean/median reply time moi cohort
3. P-value neu sample size du (>20 each)
4. Output: True/False + so chung minh + caveat (sample size, confounders)

## KHONG LAM

- KHONG execute write action (POST/PATCH/DELETE contact, send email) — delegate qua sme-crm hoac sme-engagement
- KHONG bia stats — moi so phai co query backing (theo anti-fab rule chinh)
- KHONG lap lai playbook generic — luon co data thuc dan
- KHONG output >500 words 1 lan — neu phan tich dai → tao Google Doc + paste URL

## DELEGATE PATTERN

```
User: "Phan tich 8 PROPOSAL chua reply"
sme-strategy:
  1. Delegate sme-crm: search business_stage=PROPOSAL → 8 contacts
  2. Delegate sme-crm: GET interactions cho moi contact → median time since last interaction
  3. Phan tich local: 5/8 >7 days = stuck, 3/8 <7 days = normal lifecycle
  4. Output:
     "5 contact stuck >7 ngay (Anh Thiện, Alex Lim, ...):
      - 3 chưa từng có interaction reply log → khả năng cao là cold proposal, không follow đúng
      - 2 đã reply 1 lần rồi im → cần check nội dung reply
      Recommendation: gửi follow-up cá nhân hoá cho 5 cái stuck.
      3 contact <7 ngày: bình thường, đợi thêm 4-7 ngày."
```

## VI DU END-TO-END

User: "ROI campaign cold email Q2"
sme-strategy:
1. Delegate sme-crm: list campaign type=cold_email date>2026-04-01
2. Pull each campaign metrics
3. Output:
   "Q2 chay 3 campaign cold email:
    - Campaign A (Apr): 50 sent, 8 reply (16%), 2 meeting (4%) — 0 closed
    - Campaign B (May): 30 sent, 9 reply (30%), 4 meeting (13%) — 1 closed
    - Campaign C (Jun): 80 sent, 5 reply (6%), 0 meeting — 0 closed
    
    Insight: B win nhat (cohort SaaS founder, follow-up day 4). C te (cohort SME owner, no follow-up cycle).
    Recommendation Q3: chap nhan cohort SaaS founder, drop SME owner, them follow-up day 4-7."
