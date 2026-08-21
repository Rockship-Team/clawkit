---
name: sme-scheduler
description: "Time-based user-scheduled reminders & recurring tasks. UU TIEN cao hon sme-reminder khi message co TIME-EXPRESSION (14h, 6h chieu, mai, 25/5, moi ngay, 30 phut nua, every Monday, at 9am...). Khi user noi 'nhac toi 6h chieu report task', 'nhac toi 14:00 hom nay dang bai', 'moi thu sang nhac toi X', '30 phut nua nhac Y', 'huy/list reminder' → goi native tool cron (add/list/update/remove/run). Tach bach voi sme-reminder (trigger engine cho live BD data, KHONG time). Skill nay KHONG fetch data, KHONG suggest action — chi schedule prompt de gateway chay sau."
metadata: { "openclaw": { "emoji": "⏰" } }
---

# SME Scheduler — Lich hen + nhac viec tu chat

Skill nay la **thin wrapper quanh native tool `cron`** cua openclaw gateway. Xem description cua tool `cron` de biet chi tiet schema — skill nay chi huong dan khi nao goi + cach map tieng Viet.

**KHONG shell ra `openclaw cron ...` / `openclaw reminder ...` — `cron` la native tool, goi truc tiep.**

## TRIGGER — Kich hoat khi nao

Kich hoat NGAY khi message co **THOI GIAN CU THE** + y dinh lap lai / mot lan:

### Tieng Viet
- "nhac toi <time> <content>": "nhac toi 6h chieu report task", "nhac toi 9h mai call client A"
- "moi ngay/tuan/thu/sang/chieu <time> <action>": "moi thu 2 sang 9h gui daily plan"
- "sau X phut/gio nhac", "30 phut nua nhac toi Y"
- "<timestamp> nhac toi Z": "25/5 9h nhac toi gui proposal"
- "cancel / huy / xoa reminder/lich": list + confirm + remove
- "list reminder / xem cac lich"
- "pause reminder X" / "resume reminder X" → `update` voi `{enabled:false/true}`
- "chay luon reminder X" / "run now" → `run`

### English
- "remind me at <time>", "every Monday 9am do X"
- "in 30 min remind me to Y", "schedule Z daily"
- "cancel / pause / resume / list reminders"

## QUAN TRONG — TACH BACH VOI sme-reminder

| User noi | Dung skill nao |
|---|---|
| "nhac toi" (mo ho, KHONG time) | **sme-reminder** (fetch live BD data + suggest) |
| "ai can follow-up hom nay" | **sme-reminder** |
| "nhac toi 6h chieu report" (CO TIME) | **sme-scheduler** (skill nay — cron add) |
| "moi ngay 9h sang nhac ..." | **sme-scheduler** |
| "huy reminder 18h" | **sme-scheduler** |
| "xem cac lich" | **sme-scheduler** |

Quy tac vang: **chi dung sme-scheduler khi message co time-expression**. Khong thi la sme-reminder.

## 5 ACTION

### 1. ADD (tao job moi)

**Step 1 — Parse time-expression → schedule object.**

Truong `schedule` la OBJECT voi 3 kind:

- **"at"** — one-shot tai thoi diem tuyet doi:
  ```json
  { "kind": "at", "at": "2026-05-01T09:00:00+07:00" }
  ```
- **"every"** — recurring theo interval (milliseconds):
  ```json
  { "kind": "every", "everyMs": 1800000 }        // moi 30 phut
  ```
- **"cron"** — cron expression:
  ```json
  { "kind": "cron", "expr": "0 18 * * *", "tz": "Asia/Ho_Chi_Minh" }
  ```

Map tieng Viet pho bien:

| User noi | schedule |
|---|---|
| "6h chieu hang ngay" / "moi ngay 18h" | `{kind:"cron", expr:"0 18 * * *", tz:"Asia/Ho_Chi_Minh"}` |
| "9h sang thu 2 den thu 6" | `{kind:"cron", expr:"0 9 * * 1-5", tz:"Asia/Ho_Chi_Minh"}` |
| "moi thu hai 10h sang" | `{kind:"cron", expr:"0 10 * * 1", tz:"Asia/Ho_Chi_Minh"}` |
| "moi chu nhat 8h" | `{kind:"cron", expr:"0 8 * * 0", tz:"Asia/Ho_Chi_Minh"}` |
| "30 phut nua" | `{kind:"at", at:"<now+30m ISO+07:00>"}` |
| "2 tieng nua" | `{kind:"at", at:"<now+2h ISO+07:00>"}` |
| "ngay mai 8h sang" | `{kind:"at", at:"<tomorrow>T08:00:00+07:00"}` |
| "25/5 9h" | `{kind:"at", at:"2026-05-25T09:00:00+07:00"}` |

**Step 2 — Soan payload self-contained.**

Cron chay trong session moi (isolated agent). Payload.message la PROMPT cho agent do — KHONG phai text gui thang cho user. Neu chi viet "Nhac anh X..." agent se hieu nham la co ai nhac NO va tra loi meta ("No action taken..."). Phai frame nhu INSTRUCTION.

**Pattern DUNG — reminder thuan (chi gui 1 cau):**
```json
{
  "kind": "agentTurn",
  "message": "Gui DUY NHAT tin nhan sau vao chat hien tai (KHONG add comment, KHONG tool call, KHONG meta-reply, KHONG suffix): \"⏰ Nhac anh @akhoa2174: da den 15:30 hom nay — dang bai LinkedIn thoi!\"",
  "timeoutSeconds": 60
}
```

**Pattern DUNG — reminder co action (goi skill, chay CLI):**
```json
{
  "kind": "agentTurn",
  "message": "Trigger skill sme-reminder o che do DAILY_MORNING_BRIEFING. Chao @akhoa2174, chay 'sme-cli cosmo daily-plan --mode morning', render theo QUY TAC VANG.",
  "timeoutSeconds": 300
}
```

**Pattern SAI — dung viet:**
> ❌ `"message": "Nhac anh X: da den gio Y, vui long lam Z."` — agent doc nhu incoming request, response meta thay vi forward cho user.

**Step 3 — Delivery (thuong omit).**

Mac dinh isolated agentTurn → delivery = `"announce"` tu dong ve chat hien tai (gateway infer tu session key). Chi set explicit khi user rao ro "gui cho group X":
```json
{ "mode": "announce", "channel": "telegram", "to": "-5147613854" }
```

**Step 4 — Goi tool.**

```
cron(
  action="add",
  job={
    "name": "<short slug user-facing>",
    "schedule": { ...Step 1... },
    "sessionTarget": "isolated",
    "payload": { ...Step 2... }
  }
)
```

**Step 5 — Confirm ngan gon.**

> ✅ Da set: **<name>**
> Lich: <schedule human> — lan toi: <nextRunAtMs format>
> (id: <prefix>) — noi "huy reminder <name>" neu muon cancel.

### 2. LIST (BAT BUOC khi user noi "list reminder", "xem lich nhac", "scheduled tasks", "co lich gi", "show reminders")

```
cron(action="list", includeDisabled=false)
```

**Render output dung format markdown table** (de Telegram + browser deu hien clean):

```
📋 Lich nhac hien tai (3):

| # | Ten | Lich | Lan toi | Status |
|---|-----|------|---------|--------|
| 1 | ⏰ Daily morning briefing | hang ngay 08:00 | mai 08:00 | ✅ active |
| 2 | 📅 Daily evening review | hang ngay 15:00 | hom nay 15:00 | ✅ active |
| 3 | 🔔 Daily task report | hang ngay 18:00 | hom nay 18:00 | ⚠️ 2 errors |

Action keywords:
- "huy reminder <ten/so>" — xoa
- "pause reminder <ten/so>" — tam dung
- "resume reminder <ten/so>" — bat lai
- "chay luon reminder <ten/so>" — trigger ngay
- "sua reminder <ten/so> ..." — chinh gio/noi dung
```

**BAT BUOC kem:**
- Tong so reminder o dau output
- Cot Status hien co `consecutiveErrors > 0` thi warn `⚠️ N errors` (data tu state.consecutiveErrors)
- Cuoi list LUON co block "Action keywords" de user discover commands
- Schedule format human-readable: "hang ngay 08:00" thay vi "0 8 * * *", "30 phut nua" thay vi ISO timestamp

**Khi list trong:**
```
📋 Khong co reminder nao dang hoat dong.

Anh muon set lich nhac? Vi du:
- "Nhac toi 18h hang ngay report task"
- "9h sang thu 2 nhac call client A"
- "30 phut nua nhac toi gui email"
```
→ Output trong cung phai discoverability — gioi thieu pattern command.

### 3. REMOVE

1. **BAT BUOC list truoc** de lay `jobId`. KHONG guess.
2. Match theo context user ("reminder 18h" → 🕕 Report task daily).
3. Neu >1 match, ask confirm.
4. Goi `cron(action="remove", jobId="<id>")`
5. Confirm: "🗑 Da huy <name>."

### 4. UPDATE (pause / resume / doi gio)

Khong co action rieng cho pause/resume — dung `update` voi `patch.enabled`:

- Pause: `cron(action="update", jobId=<id>, patch={"enabled": false})`
- Resume: `cron(action="update", jobId=<id>, patch={"enabled": true})`
- Doi cron expr: `cron(action="update", jobId=<id>, patch={"schedule": {"kind":"cron","expr":"0 19 * * *","tz":"Asia/Ho_Chi_Minh"}})`

### 5. RUN NOW / RUNS (history)

- `cron(action="run", jobId=<id>)` — chay luon 1 lan (force)
- `cron(action="runs", jobId=<id>)` — xem lich su chay cua job

## QUY TAC BAT BUOC

1. **KHONG TU SCHEDULE khi khong co time-expression**. User noi "nhac toi" tru hong → hand-off sme-reminder, KHONG tao cron.

2. **KHONG schedule recursive**. Trong cron session (isolated agentTurn chay tu cron), KHONG duoc tao them cron. Tool tu block — nhung skill cung nhac.

3. **Message cron phai self-contained**. Cron isolated session khong biet chat hien tai. Lay du ten nguoi, context, y dinh vao `payload.message`.

4. **Timezone mac dinh: Asia/Ho_Chi_Minh** tru khi user noi khac. Luon set `schedule.tz` cho cron-kind; dung suffix `+07:00` cho at-kind.

5. **Confirm sau khi tao** (khong wait truoc) neu schedule <2 lan/ngay va intent ro. Neu ambiguous (vd "nhac toi tuan sau" — khong ro ngay) → ask clarify TRUOC khi call tool.

6. **Delivery target**: omit `delivery` de auto ve current chat (thread/topic preserve). Chi set explicit `delivery={mode:"announce", channel:..., to:...}` khi user rao ro "gui cho group X".

7. **KHONG dump jobId day du** — chi show 8 ky tu dau cho readable. User neu hoi full thi show.

8. **KHONG fallback shell** (`openclaw cron ...`, `openclaw reminder ...`, `edit /home/.../cron/jobs.json`). Luon dung native tool `cron(...)`. Neu tool fail, hoi user — dung tu sua file.

## VI DU

**User:** "nhac toi 6h chieu moi ngay report task"

**Ban:**
```
cron(
  action="add",
  job={
    "name": "Report task daily",
    "schedule": {"kind":"cron", "expr":"0 18 * * *", "tz":"Asia/Ho_Chi_Minh"},
    "sessionTarget": "isolated",
    "payload": {
      "kind": "agentTurn",
      "message": "Trigger skill sme-reminder mode DAILY_TASK_REPORT: nhac @akhoa2174 va team report task hom nay — (1) Done, (2) Dang lam, (3) Blocker. Plain Vietnamese, max 3 y, khong thuat ngu tech.",
      "timeoutSeconds": 300
    }
  }
)
```
→ confirm "✅ Da set: Report task daily — 18:00 hang ngay — lan toi hom nay 18:00".

---

**User:** "30 phut nua nhac goi khach Acme"

**Ban:** (hien tai 14:22, +30m = 14:52)
```
cron(
  action="add",
  job={
    "name": "Goi khach Acme",
    "schedule": {"kind":"at", "at":"2026-04-24T14:52:00+07:00"},
    "sessionTarget": "isolated",
    "payload": {
      "kind": "agentTurn",
      "message": "Gui DUY NHAT tin nhan sau vao chat hien tai (khong comment, khong tool call, khong meta): \"⏰ Nhac anh goi khach Acme — nho xem lai note cuoc meeting truoc khi goi.\"",
      "timeoutSeconds": 60
    }
  }
)
```

---

**User:** "moi thu 2 sang 9h gui daily plan vao group BD"

**Ban:**
```
cron(
  action="add",
  job={
    "name": "Weekly BD plan",
    "schedule": {"kind":"cron", "expr":"0 9 * * 1", "tz":"Asia/Ho_Chi_Minh"},
    "sessionTarget": "isolated",
    "payload": {
      "kind": "agentTurn",
      "message": "Moi thu 2 9h: trigger sme-reminder mode DAILY_MORNING_BRIEFING, chay 'sme-cli cosmo daily-plan --mode morning' va render vao group BD. Apply QUY TAC VANG cua sme-reminder (plain Vietnamese, data-aware).",
      "timeoutSeconds": 300
    },
    "delivery": {"mode":"announce", "channel":"telegram", "to":"-5147613854"}
  }
)
```

---

**User:** "huy reminder 18h"

**Ban:**
1. `cron(action="list")` → tim job co schedule.expr `0 18 * * *`
2. Confirm: "Anh muon huy **Report task daily** (6h chieu moi ngay)?"
3. User "ok" → `cron(action="remove", jobId=...)` → "🗑 Da huy."

---

**User:** "xem cac lich"

**Ban:** `cron(action="list")` → render 1 dong/job theo format plain.

---

**User:** "pause reminder 18h tam 1 tuan"

**Ban:** list → match → `cron(action="update", jobId=..., patch={"enabled":false})` → confirm + ghi chu "Anh resume khi nao can nhe, bot khong tu resume."

## PHAN BIET VOI CAC SKILL KHAC

- **`sme-reminder`**: trigger engine — fetch live BD data khi user muon "biet ai can follow up". KHONG schedule.
- **`sme-scheduler`** (skill nay): pure scheduler — set/list/remove cron jobs tu chat qua native tool `cron`.
- **`sme-campaign`**: tao email campaign (event / cold / follow-up). Khong phai reminder.

Moi thu khac → skill tuong ung.
