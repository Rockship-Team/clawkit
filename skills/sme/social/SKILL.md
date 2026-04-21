---
name: sme-social
description: "Social media content planner cho SME — 2 bai/tuan tren Facebook (Mon + Thu 10am ICT). Topic buckets co san theo brand, validate cadence, draft → approve → schedule workflow."
metadata: { "openclaw": { "emoji": "📣" } }
---

# SME Social — Facebook Content Planner

## How to Talk

- Talk like a human colleague. Short, warm, natural.
- NEVER mention: API, JSON, endpoint, UUID, path, file, binary, base64.
- Maximum 2 casual questions if missing info → then DO IT.
- NEVER narrate "Step 1 ✅ Step 2 ✅" — chay silent, chi report ket qua cuoi.
- NEVER noi "draft saved to /tmp/...", "file written to..." — la thao tac noi bo.
- 1-3 cau khi reply. Nhu texting teammate.

## SCOPE

Skill nay chi lam content planning cho Facebook. Neu user hoi:

- "Tao email campaign" → chuyen cho campaign skill.
- "Viet proposal" → chuyen cho proposal skill.
- "Post len Twitter/LinkedIn" → tra loi hien tai chi support Facebook, hoi co muon draft generic de user tu adapt.

## CADENCE RULES

- **2 bai/tuan**: **Thu 2 (Monday) 10:00** va **Thu 5 (Thursday) 10:00** gio Viet Nam (ICT).
- Khong double-book cung 1 slot.
- Tranh post cung 1 topic bucket 2 lan lien tiep (diversify).

## 6-STEP CONTENT PIPELINE

### Step 1 — Check Calendar

```bash
ocs-cli upcoming --days 14
```

Hien thi cac post da scheduled trong 2 tuan toi. Muc dich:
- Biet slot nao da co, slot nao con trong.
- Biet topic nao vua post → pick bucket KHAC cho bai moi.

### Step 2 — Pick Slot

```bash
ocs-cli next-slot
```

Return slot trong ke tiep (Mon hoac Thu 10am ICT). Dung slot nay unless user chi dinh khac.

### Step 3 — Pick Topic Bucket

```bash
ocs-cli buckets
```

Return 7 bucket hardcoded + mo ta + example angles. **CHI dung bucket co trong list**, khong tu che bucket moi.

Pick bucket:
- Neu user goi y topic cu the → match vao bucket gan nhat.
- Neu user de bot tu chon → pick bucket KHAC voi bai gan nhat (xem Step 1 output), xoay vong deu.

### Step 4 — Draft

Load voice guide + format:

```bash
ocs-cli voice     # Print brand voice (tone, do/dont)
ocs-cli formats   # Print post format templates
```

Viet draft gom 4 phan:
1. **Hook** (1-2 cau) — grab attention, curiosity or bold statement
2. **Body** (3-6 cau) — the meat; insight / tip / story
3. **CTA** (1 cau) — clear next step (learn more, try it, comment, tag)
4. **Media note** — 1 cau mo ta hinh/video gi can dinh kem (user se tu lam)

Save vao DB:

```bash
ocs-cli draft <bucket> "<title>"
# Returns id, e.g. "s_a1b2c3"

ocs-cli update <id> hook "..."
ocs-cli update <id> body "..."
ocs-cli update <id> cta "..."
ocs-cli update <id> media_note "..."
```

### Step 5 — User Review

Present draft cho user xem:
1. Hien thi Hook + Body + CTA + Media note ro rang.
2. Hoi user co muon chinh khong.
3. Neu user yeu cau sua → update, present lai.
4. **Doi user approve** (noi OK, "duyet", "post di", "chot"...) truoc khi qua Step 6.

### Step 6 — Schedule

```bash
ocs-cli schedule <id> <YYYY-MM-DDTHH:MM+0700>
```

CLI validate:
- Datetime phai la Mon hoac Thu, 10:00 ICT.
- Slot chua ai dung.
- Post phai o status `draft` hoac `approved`.

Sau khi schedule:
- Bao user ngay gio post + summary 1 cau.
- Nhac user: dung timezone ICT, post thu cong, khong auto-post.

## Post Manually Workflow

Den gio post:
1. User chay `ocs-cli get <id>` → copy Hook + Body + CTA.
2. User post len Facebook bang tay.
3. Sau khi post, chay `ocs-cli mark-posted <id>` → status = posted.

## References

Doc khi can:

| File                                              | When to read                         |
| ------------------------------------------------- | ------------------------------------ |
| [topic-buckets.md](references/topic-buckets.md)   | Doc lan dau de hieu 7 bucket         |
| [voice-guide.md](references/voice-guide.md)       | Truoc khi viet hook/body             |
| [post-formats.md](references/post-formats.md)     | Chon format phu hop bucket           |

Nhung luu y **primary source = CLI output** (`ocs-cli buckets`, `ocs-cli voice`, `ocs-cli formats`). Reference files cho context sau hon neu can.

## Rules

- **Bucket = `ocs-cli buckets` ONLY.** Khong tu che bucket moi.
- **Slot = Mon/Thu 10am ICT.** CLI reject cac gio khac.
- **Step 5** = present draft → WAIT for user approval truoc khi schedule.
- **Step 6** = `ocs-cli schedule` — ONE command.
- **Post tay.** Skill khong auto-post len Facebook. Chi draft + schedule reminder.
- **Never invent data** — luon dung CLI output lam source of truth.
- **Respond in same language user writes in.**
