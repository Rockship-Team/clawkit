# 02 — ASSESS

**Giai đoạn:** Profile đủ → scorecard 5 chiều → đánh giá EC tier → transition PLAN.
**Skills:** [`profile-assessment`](../skills/profile-assessment/SKILL.md), [`ec-strategy`](../skills/ec-strategy/SKILL.md)
**Skill test IDs:** TC-PA-004, TC-EC-001..007

---

## Scenario 2.1 — Scorecard 5 chiều (happy path)

**Setup:** Profile từ 01_onboard scenario 1.1 đã lưu (Nguyễn Minh Anh, lớp 11, GPA 8.7/10, chưa thi, EC: Tin HSG giải Ba + Python GitHub).

**Input:** "Đánh giá hồ sơ em"

**Expected CLI sequence:**
```bash
sa-cli student query zalo 12345
sa-cli ec list {student_id}               # lấy danh sách EC hiện có
sa-cli student scorecard {student_id}      # trả scores + commentary + next_actions
```

**Expected output (5 dimensions theo doc section 1 Module 1):**
```
📊 ĐÁNH GIÁ HỒ SƠ — Nguyễn Minh Anh (lớp 11, Chuyên LHP)
─────────────────────────────────────
Học thuật      [████████░░]  8/10   (GPA 8.7/10 ≈ 3.6–3.7/4.0 UW + chuyên Toán top VN)
Test Scores    [░░░░░░░░░░]  —/10   (chưa thi SAT/TOEFL — ưu tiên #1)
Ngoại khoá    [██████░░░░]  6/10   (Tier 2 Giải Ba QG, thiếu leadership)
Essay          [░░░░░░░░░░]  N/A    (chưa viết)
Tổng thể       [██████░░░░]  6/10
─────────────────────────────────────

NHẬN XÉT:
✅ Mạnh:
- GPA mạnh + context trường chuyên top VN
- Giải Ba QG Tin → Tier 2 theo framework

⚠️ Gap:
- Chưa có SAT/TOEFL → ưu tiên ngay
- EC thuần STEM, thiếu leadership/community impact
- Chưa có project/research có personal brand

📋 HÀNH ĐỘNG TIẾP THEO (ưu tiên):
1️⃣ Đăng ký thi SAT và TOEFL/IELTS ngay — Trong vòng 2 tuần
2️⃣ Nâng cấp EC: từ "member" Đội tuyển Tin → workshop/mentor → Tier 2+ — Học kỳ tới
3️⃣ Bắt đầu brainstorm Common App Personal Statement — Từ bây giờ

→ Chuyển PLAN: chọn trường (Reach/Target/Safety) hoặc tạo study plan?
```

**Pass criteria:**
- [ ] 5 chiều đầy đủ (Học thuật, Test Scores, Ngoại khoá, Essay, Tổng thể)
- [ ] Test Scores hiển thị `—/10` khi chưa thi, KHÔNG phải `0/10`
- [ ] Essay `N/A` khi chưa viết (không phải 0)
- [ ] EC tier được classify đúng: Giải Ba QG → Tier 2
- [ ] Overall là weighted (35% academics, 30% tests, 25% EC, 10% essay theo code)
- [ ] 3 actions có thứ tự ưu tiên, kèm timeline cụ thể
- [ ] Transition prompt rõ ràng sang PLAN (không tự chạy school matching)

**Fail signals:**
- Test Scores = 0/10 → sai (chưa thi ≠ 0)
- Overall ≥ 8 khi Test Scores N/A → nịnh
- Không có transition prompt sang PLAN

---

## Scenario 2.2 — EC tier detail + gap analysis (TC-EC-002, TC-EC-004)

**Input:** "EC của em mạnh chưa? Cần bổ sung gì?"

**Expected CLI:**
```bash
sa-cli student query zalo 12345
sa-cli ec list {student_id}
```

**Expected output (dùng emoji tier marker từ SKILL.md):**
```
🏆 EC PORTFOLIO — CS major
─────────────────────────────────
1. Đội tuyển Tin HSG (Giải Ba QG)   ⭐ Tier 2
   Vai trò: member | 15h/tuần
   → Mạnh: recognition cấp quốc gia
   → Upgrade Tier 2+: mentor junior, tổ chức workshop nội bộ

2. Python GitHub projects (2 projects) ✦ Tier 3
   Vai trò: self | 5h/tuần
   → Mạnh: code công khai
   → Upgrade Tier 2: README chuẩn, demo deploy, 100+ stars
     HOẶC tool được người thật sử dụng

GAP CHO CS MAJOR:
❌ Chưa có: hackathon, open-source contribution, competitive programming (USACO/VNOI)
❌ Chưa có: leadership cộng đồng (founder CLB, lớp dạy miễn phí)

📅 TIMELINE:
• Ngay: 1 commit/meaningful PR tuần này để activate GitHub
• Hè 2026: research summer program HOẶC founding CLB tin học
• Năm học tới: duy trì GitHub + 1 hackathon/quý
```

**Pass criteria:**
- [ ] Mỗi EC có tier marker (🌟 T1 / ⭐ T2 / ✦ T3 / · T4)
- [ ] Upgrade path cụ thể cho từng EC (không chung chung "cố gắng lên")
- [ ] Gap phân tích theo `intended_major` (CS ≠ pre-med ≠ business)
- [ ] Timeline chia theo: ngay / hè / năm học

---

## Scenario 2.3 — Inactive GitHub project warning (TC-EC-006)

**Setup:** EC "Python GitHub projects" có `last_updated > 6 tháng`.

**Input:** "Xem EC của em"

**Expected behavior:**
- Flag ⚠️ trên project stale:
```
Project này có vẻ chưa được cập nhật gần đây (6+ tháng). Nếu em dự định đưa vào hồ sơ,
admissions officers thường ấn tượng hơn với project đang active hoặc có README rõ ràng.

Em có muốn mình gợi ý cách "activate" lại project này không?
```
- Đưa 2 options: revive (commit lại tuần này) HOẶC remove khỏi application list

**Pass criteria:**
- [ ] Flag staleness rõ ràng
- [ ] 2 options: revive vs remove
- [ ] Không tự động remove — chỉ đưa lựa chọn

---

## Scenario 2.4 — EC update mid-cycle (TC-EC-003)

**Input:** "Em vừa được bầu Phó Chủ tịch CLB Tin học trường, 10h/tuần"

**Expected CLI:**
```bash
# Option A — tạo mới
sa-cli ec add {student_id} "CLB Tin học trường" "VP" 10 "Phó Chủ tịch"

# Option B — nếu đã có activity "CLB Tin học", update tier
sa-cli ec update-tier {activity_id} 2 "Leadership: VP CLB Tin học - tổ chức workshop/mentor"
```

**Expected output:**
- Tier 2 (vì VP = leadership cấp trường, có thể lên T2+ nếu có impact đo được)
- Gợi ý biến leadership → impact đo được:
  ```
  VP CLB → bước tiếp: pitch goal cụ thể
  Ví dụ: "Từ 15 member → 30 member + 3 workshop/kỳ = có impact đo được = Tier 2+"
  ```
- Nhắc update Common App Activities section (150 chars)

**Pass criteria:**
- [ ] Tier được classify đúng (VP CLB trường = T2 baseline)
- [ ] Gợi ý impact cụ thể, đo được (không chỉ "cố gắng lên")
- [ ] Nhắc 150-char activity description

---

## Scenario 2.5 — Profile yếu + dream Ivy (TC-PA-004)

**Setup:** Lớp 12, GPA 3.2/4.0, SAT 1280, TOEFL 85, EC chủ yếu Tier 3–4. Dream Harvard.

**Input:** "Đánh giá em có cửa Harvard không?"

**Expected CLI:**
```bash
sa-cli student query zalo {uid}
sa-cli student scorecard {student_id}
```

**Expected behavior:**
- Scorecard Overall ~4–5/10
- Commentary THẲNG:
  - "GPA 3.2 cần cải thiện đáng kể"
  - "SAT 1280 thấp hơn median của hầu hết trường Target. Đây là gap lớn nhất"
  - "EC chủ yếu Tier 3–4 — EC cần được nâng cấp"
- Về Harvard cụ thể:
  ```
  Harvard admitted GPA avg ~4.18 (weighted), SAT mid range 1500s, acceptance international ~3%.
  Với profile hiện tại, Harvard không phải Reach — đây là "Far Reach" và không thực tế trong cycle này.

  Alternatives phù hợp ngành em:
  • Target top 50–100 có CS tốt + merit scholarship cho international
  • Gap year + retake SAT + xây EC Tier 1–2 → apply lại cycle sau
  ```
- Không dùng "chắc chắn" / "đảm bảo" / "cố gắng là được"

**Pass criteria:**
- [ ] Không nịnh
- [ ] Không phân Harvard là "Reach" — phải là "Far Reach" hoặc khuyên off-list
- [ ] Đưa alternative path rõ (gap year + retake, hoặc trường tier thấp hơn)
- [ ] Tone: thẳng thắn + mentor (không công kích)

**Fail signals:**
- "Em cố gắng thi lại SAT là được Harvard thôi" (nịnh + phi thực tế)
- Phân Harvard là Reach
