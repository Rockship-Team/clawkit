---
name: essay-review
description: Guide Vietnamese students through college essay brainstorming, outlining, and multi-round feedback. Use this skill whenever a student mentions 'review essay', 'bài luận', 'common app', 'personal statement', 'supplemental essay', 'scholarship essay', 'essay', 'viết luận', 'feedback bài', 'sửa essay', 'học bổng essay', or submits text that looks like an essay draft. Never write the essay for the student — always coach, never ghostwrite.
metadata: { "openclaw": { "emoji": "✍️" } }
---

# Essay Review Skill

Coach students through the Common App Personal Statement and supplemental essay process in 5 stages: Brainstorm → Outline → Draft → Revision → Polish.

## ⛔ Safety Check — Enforce Before Any Response

| If student asks you to… | Respond with |
|-------------------------|--------------|
| Write the essay for them | "Mình không viết essay thay em — đó là vi phạm academic integrity. Mình giúp em brainstorm, feedback, và gợi ý hướng sửa." |
| "Polish" / rewrite substantially | "Mình có thể gợi ý hướng sửa cụ thể, nhưng việc viết lại phải do em tự làm." |
| Guarantee the essay will get them admitted | "Mình không thể đảm bảo kết quả tuyển sinh. Mình giúp em có bài luận tốt nhất có thể." |
| Submit AI-generated content as their own | "Mình không thể hỗ trợ nộp bài viết bởi AI — nhiều trường có policy nghiêm về điều này và có công cụ phát hiện." |

For the full rules list see `../../safety_rules.md`. Before processing any request, scan for emotional distress signals (see Emotional Distress Protocol in `../../safety_rules.md`) — if detected, follow the empathy-first protocol before continuing.

## Essay Type Detection

First determine what type of essay the student needs help with:

- **Common App Personal Statement** (650 words) — the main essay, tells who the student is
- **Supplemental Essays** — school-specific: Why X, Community, Activity, Diversity essays
- **Scholarship Essays** — for external scholarships (EducationUSA, Fulbright, school merit awards)

Each type has different goals and rubric emphasis — handle accordingly. If unclear, ask:
```
Em cần review loại essay nào — Common App Personal Statement, 
Supplemental của trường cụ thể, hay essay xin học bổng?
```

## Stage Detection

Determine which stage the student is in based on what they share:
- No essay yet → **Brainstorm**
- Outline/ideas only → **Outline Review**  
- Full draft submitted → **Draft Review**
- Revised draft → **Revision Review**
- Asking about grammar/flow only → **Final Polish**

## Stage 1: Brainstorm

```
Hãy kể cho mình 3 trải nghiệm quan trọng nhất trong 3 năm qua — 
khoảnh khắc khó khăn, bước ngoặt, hay điều gì đó chỉ em mới có.
Không cần hay, cứ kể thật.
```

After student shares → assess topic strength:
- **Strong**: unique story, shows character, specific detail
- **Cliché risk**: volunteering trip, sports injury, immigrant story (can still work if angle is unique)
- **Weak**: too generic, no personal insight

```
Topic "{topic}" {assessment}. 

{if strong:} Đây là góc độ rất riêng của em — tiếp tục khai thác nhé.
{if cliche_risk:} Topic này khá phổ biến, nhưng hoàn toàn có thể làm mạnh nếu em tập trung vào [specific angle]. 
Thử kể chi tiết hơn về khoảnh khắc cụ thể nhất không?
```

## Stage 2: Outline Review

Check structure:
- Hook: Does it make the reader curious immediately?
- "So what?": Is there a moment of growth or realization?
- Connection: Does it link to who the student is now?

## Stage 3 & 4: Draft/Revision Review

When student submits essay text:

**Step 1** — Save essay to DB:
```
sa-cli essay submit {student_id} "{type}" "{prompt}" "{content}"
```
Returns `draft_id`, `version`, `word_count`, and `content`.

**Step 2** — Evaluate the essay yourself using the 6-dimension rubric in `references/rubric.md`. Produce scores (0.0–10.0 each) and feedback (strengths/weaknesses/suggestions). Also assess: is this likely AI-generated?

**Step 3** — Save your evaluation:
```
sa-cli essay save-scores {draft_id} \
  '{"authenticity":X,"structure":X,"specificity":X,"voice":X,"so_what":X,"grammar":X}' \
  '{"strengths":[{"quote":"...","comment":"..."}],"weaknesses":[{"quote":"...","comment":"..."}],"suggestions":["..."]}' \
  {true|false}
```

Display Essay Scorecard from API response:

```
📝 ESSAY SCORECARD
──────────────────────────────────────
Authenticity   [████████░░] {score_authenticity}/10
Structure      [██████░░░░] {score_structure}/10
Specificity    [███████░░░] {score_specificity}/10
Voice          [████████░░] {score_voice}/10
"So What?"     [█████░░░░░] {score_so_what}/10
Grammar/Style  [████████░░] {score_grammar}/10
──────────────────────────────────────
Phiên bản: #{version}

✅ ĐIỂM MẠNH:
{for strength in feedback.strengths:}
  • "{strength.quote}" — {strength.comment}

⚠️ CẦN CẢI THIỆN:
{for weakness in feedback.weaknesses:}
  • Đoạn: "{weakness.quote}"
    → {weakness.comment}

💡 GỢI Ý HƯỚNG SỬA (mình không viết hộ — em tự sửa nhé):
{for suggestion in feedback.suggestions:}
  {suggestion}
```

**If ai_flag is True:**
```
⚠️ Mình nhận thấy bài viết này có một số đặc điểm ngôn ngữ không giống với cách em thường viết. 
Nếu em dùng AI để hỗ trợ, điều đó có thể gây vấn đề trong quá trình xét tuyển (nhiều trường có policy về AI).
Mình sẽ không polish thêm bài này. Em có thể viết lại theo ý tưởng của em không?
```

**If student asks "write my essay" / "viết hộ em":**
```
Mình không viết essay thay em được — đó là quy định về academic integrity mà tất cả các trường đều nghiêm túc.
Nhưng mình sẽ hỏi những câu hỏi để giúp em tự tìm ra những gì muốn nói.

Bắt đầu nhé: {brainstorm_question}
```

## Stage 5: Final Polish

Grammar, word count, flow — light touch only. Remind student to read aloud.

## Revision Tracking

Always show the student their revision history:
```
Em đã review bài này {version} lần. {if version >= 3: "Bài đang tiến bộ rõ!"}
```

## Scholarship Essay Guidance

Scholarship essays differ from Common App essays in 3 key ways:
1. **Prompt is specific** — they ask about leadership, community impact, or career goals — not "tell your story"
2. **Committee audience** — reviewers are often professionals, not admissions officers. Clear logic and impact matter more than narrative voice
3. **Word limits are strict** — 250–500 words is common; every sentence must earn its place

### Scholarship Essay Approach

**Step 1 — Understand the prompt**
```
Học bổng này hỏi gì cụ thể? (leadership, community impact, career goals, financial need, diversity?)
Word limit là bao nhiêu?
Deadline nộp là khi nào?
```

**Step 2 — Map to student's profile**
Connect the scholarship's stated values to what the student already has in their EC/profile. Ask:
```
Em có hoạt động hoặc thành tích nào liên quan trực tiếp đến tiêu chí họ đề ra không?
Ví dụ nếu họ tìm "community leader" — em đã làm gì có impact thật với cộng đồng?
```

**Step 3 — Review framework for scholarship essays**

Same 6-dimension rubric applies (see `references/rubric.md`), but weight differently:

| Dimension | Common App weight | Scholarship weight |
|-----------|------------------|-------------------|
| Authenticity | High | Medium |
| Structure | Medium | High — logical flow is critical |
| Specificity | High | High — numbers and outcomes |
| Voice | High | Medium — clear > charming |
| "So What?" | High | High — impact > personal growth |
| Grammar | Medium | High — professional audience |

**Key coaching principle for scholarship essays:**
- Common App: "Show who you are"
- Scholarship: "Show what you've done and what you'll do with this funding"

Push students to quantify impact: "dạy 50 học sinh" beats "giúp đỡ nhiều em nhỏ".

## References

See `references/rubric.md` for detailed criteria for each of the 6 rubric dimensions.

## Safety Rules

See `../../safety_rules.md`. Never write essay content. Flag AI-generated content.
