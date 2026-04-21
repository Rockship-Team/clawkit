# Essay Review Rubric

Detailed scoring criteria for all 6 dimensions used in the Essay Scorecard.
Each dimension is scored 1–10 by the Claude API evaluator in `submit_essay.py`.

## Table of Contents
- [1. Authenticity](#1-authenticity)
- [2. Structure](#2-structure)
- [3. Specificity](#3-specificity)
- [4. Voice](#4-voice)
- [5. "So What?" (Insight/Growth)](#5-so-what)
- [6. Grammar & Style](#6-grammar--style)
- [Cliché Risk Flags](#cliché-risk-flags)
- [AI-Flag Heuristics](#ai-flag-heuristics)

---

## 1. Authenticity

**Question**: Does this essay feel like it was written by *this specific student*?

| Score | Description |
|-------|-------------|
| 9–10 | Deeply personal. Only this student could have written this. Specific names, places, sensory details. |
| 7–8 | Genuine voice, mostly personal. A few generic passages. |
| 5–6 | Some personal elements, but feels like it could apply to many students. |
| 3–4 | Generic story. Relies on common tropes without personal angle. |
| 1–2 | Could have been written by anyone. No personal detail. |

**Feedback cues:**
- Strength: "Đoạn '{quote}' rất riêng của em — admissions officers nhớ được điều này."
- Weakness: "Đoạn '{quote}' nghe rất chung chung. Thêm chi tiết cụ thể hơn: tên người, địa điểm, cảm giác lúc đó."

---

## 2. Structure

**Question**: Does the essay have a clear arc — beginning, development, resolution?

| Score | Description |
|-------|-------------|
| 9–10 | Compelling hook. Clear narrative arc. Ending ties back to opening or opens to the future. |
| 7–8 | Good flow. Minor structural issue (e.g. weak ending or slow start). |
| 5–6 | Story is there but structure is loose. Reader has to work to follow. |
| 3–4 | Chronological list of events without narrative tension. |
| 1–2 | No discernible structure. |

**Common structural issues:**
- **Starting with birth/childhood**: Usually too broad — push student to start closer to the core moment.
- **Ending with "this experience taught me to..."**: Predictable. Encourage ending with a forward-looking image instead.
- **No hook**: First sentence is boring. The first line must make the reader curious.

---

## 3. Specificity

**Question**: Does the essay show rather than tell? Are there concrete details?

| Score | Description |
|-------|-------------|
| 9–10 | Rich sensory detail. Specific names, numbers, quotes, scenes. |
| 7–8 | Good detail in most sections. One or two vague passages. |
| 5–6 | Tells emotions rather than showing them ("I felt happy" vs "my hands were shaking"). |
| 3–4 | Almost entirely abstract. No scenes, no dialogue, no tangible images. |
| 1–2 | Pure abstraction. No concrete grounding at all. |

**Coaching prompt when score is low:**
```
Hãy kể lại khoảnh khắc đó như đang quay phim: 
Lúc đó em ở đâu? Thời tiết thế nào? Ai đang ở cạnh em? 
Em nghe thấy gì, ngửi thấy gì, tay em đang làm gì?
```

---

## 4. Voice

**Question**: Does the writing sound like a real human teenager, or is it formal/AI-sounding?

| Score | Description |
|-------|-------------|
| 9–10 | Clear, distinctive voice. Reads naturally. Has personality and rhythm. |
| 7–8 | Generally good voice. A few stilted or over-formal passages. |
| 5–6 | Inconsistent voice — some sections feel natural, others robotic. |
| 3–4 | Mostly formal or generic. Sounds like a school report. |
| 1–2 | No voice. Flat, corporate, or clearly AI-generated. |

**Voice red flags (trigger AI-flag check):**
- Passive voice overuse: "It was discovered that...", "I was taught the lesson of..."
- Overly structured transitions: "Firstly...", "In conclusion..."
- Vocabulary inconsistent with student's other messages
- Perfect grammar in a non-native English writer with no tutoring history

---

## 5. "So What?" (Insight / Growth) {#5-so-what}

**Question**: Does the student show self-reflection? What did they *learn* about themselves — not just what happened?

| Score | Description |
|-------|-------------|
| 9–10 | Profound, nuanced self-insight. The reflection feels earned, not stated. Shows complexity. |
| 7–8 | Clear growth narrative. Insight is genuine though perhaps slightly surface-level. |
| 5–6 | Some reflection but it feels tacked on at the end rather than woven through. |
| 3–4 | "This experience taught me to work hard." Generic lessons. |
| 1–2 | No reflection at all. Purely narrative with no introspection. |

**Coaching prompt:**
```
Em học được gì về bản thân mình từ trải nghiệm này — 
không phải về chủ đề (lãnh đạo, kiên trì) mà về *em* cụ thể?
Điều gì bây giờ em làm khác so với trước đó?
```

---

## 6. Grammar & Style

**Question**: Is the writing clean, concise, and free of errors?

| Score | Description |
|-------|-------------|
| 9–10 | No grammar errors. Varied sentence length. Punchy and precise. |
| 7–8 | Minor errors (1–3). Generally clear writing. |
| 5–6 | Several grammar issues. Some run-on sentences or awkward phrasing. |
| 3–4 | Frequent errors that distract from content. |
| 1–2 | Significant grammar issues that undermine readability. |

**Non-native English guidance:**
- Small grammar errors are acceptable and often authentic for international students.
- Do NOT penalize accent or rhythm if the voice is genuine.
- Only flag errors that genuinely confuse meaning.

---

## Cliché Risk Flags

Topics that need extra scrutiny — not disqualifying, but require a unique angle:

| Topic | Risk Level | How to salvage |
|-------|-----------|----------------|
| Mission trip / volunteering abroad | High | Focus on what the student was *wrong about* before going |
| Sports injury comeback | High | Only works if insight is about identity, not resilience cliché |
| Immigrant family struggle | Medium | Works if the story is specific and unexpected — not "my parents sacrificed" |
| Death of a grandparent | Medium | Must be about the student's change, not grief description |
| "I'm a leader" essay | High | Show leadership through a specific decision, not a list of titles |
| Learning to code / first program | Medium | Must go beyond "I was curious and now I love CS" |

---

## AI-Flag Heuristics

`submit_essay.py` sets `ai_flag=True` when 2+ of these are detected:

1. Vocabulary level inconsistent with prior messages in the conversation
2. Sentence structure unusually varied and complex for a high school student
3. Multiple passive constructions in a row
4. Transition words: "Furthermore", "Moreover", "In conclusion", "It is evident that"
5. Opening line is a philosophical question or quote unattributed to a real source
6. The word "tapestry", "journey", "testament", "delve", "navigate", "realm"

When `ai_flag=True`, do NOT provide further feedback. Display the flag message from SKILL.md and stop.
