"""
Follow-up cron job cho bds-broker.

Chạy mỗi phút. Nếu khách nhắn tin rồi im lặng 1–5 phút
mà chưa được follow-up thì gửi tin nhắn hỏi lại.

Cài cron (chạy 1 lần sau khi cài skill):
    crontab -e
    * * * * * python3 ~/.openclaw/workspace/skills/bds-broker/followup_cron.py >> ~/.openclaw/workspace/skills/bds-broker/followup.log 2>&1
"""

import sqlite3
import urllib.request
import urllib.parse
import json
import os
import ssl
from datetime import datetime, timezone, timedelta

# Bypass SSL verification (self-signed cert in network chain)
_ssl_ctx = ssl.create_default_context()
_ssl_ctx.check_hostname = False
_ssl_ctx.verify_mode = ssl.CERT_NONE

# ── config ────────────────────────────────────────────────────────────────────
TOKEN = "8623915046:AAFbs_UKB7YvqToEnovOKxz_uZOUIBzdFBQ"
DB = os.path.expanduser("~/.openclaw/workspace/skills/bds-broker/bds.db")

# Gửi follow-up khi im lặng >= IDLE_MIN giây (không giới hạn trên)
IDLE_MIN = 10   # giây

# Tối đa bao nhiêu lần follow-up mỗi cuộc hội thoại (reset khi khách nhắn lại)
MAX_FOLLOWUPS = 2

MESSAGES = [
    "Dạ không biết anh/chị còn thắc mắc gì không ạ? Mình sẵn sàng hỗ trợ thêm 😊",
    "Dạ anh/chị có cần mình tư vấn thêm về căn hộ hoặc dự án nào không ạ? Mình ở đây nếu anh/chị cần nhé 🏠",
]
# ─────────────────────────────────────────────────────────────────────────────


def send_message(chat_id: str, text: str) -> bool:
    url = f"https://api.telegram.org/bot{TOKEN}/sendMessage"
    data = urllib.parse.urlencode({"chat_id": chat_id, "text": text}).encode()
    try:
        with urllib.request.urlopen(url, data=data, timeout=10, context=_ssl_ctx) as resp:
            result = json.loads(resp.read())
            return result.get("ok", False)
    except Exception as e:
        print(f"[followup] send error chat_id={chat_id}: {e}")
        return False


def main():
    if not os.path.exists(DB):
        print("[followup] DB not found, skipping")
        return

    vn = timezone(timedelta(hours=7))
    now = datetime.now(vn)

    conn = sqlite3.connect(DB)
    conn.row_factory = sqlite3.Row

    # Đảm bảo bảng tồn tại (phòng trường hợp init_db chưa chạy)
    conn.execute("""
        CREATE TABLE IF NOT EXISTS conversations (
            chat_id TEXT PRIMARY KEY,
            last_message_at TEXT NOT NULL,
            follow_up_sent_at TEXT,
            follow_up_count INTEGER DEFAULT 0,
            stage TEXT DEFAULT 'new'
        )
    """)
    conn.commit()

    rows = conn.execute("SELECT * FROM conversations").fetchall()

    for row in rows:
        chat_id = row["chat_id"]
        follow_up_count = row["follow_up_count"] or 0
        stage = row["stage"] or "new"

        # Chỉ follow-up khi khách đã bắt đầu hỏi thật sự (không phải mới mở bot)
        if stage == "new":
            continue

        # Đã hết lượt follow-up cho cuộc hội thoại này — dừng, chờ khách nhắn lại
        if follow_up_count >= MAX_FOLLOWUPS:
            continue

        try:
            last_msg = datetime.fromisoformat(row["last_message_at"])
            if last_msg.tzinfo is None:
                last_msg = last_msg.replace(tzinfo=vn)
        except Exception:
            continue

        idle_seconds = (now - last_msg).total_seconds()

        if idle_seconds < IDLE_MIN:
            continue

        # Kiểm tra follow_up_sent_at — không gửi lại nếu đã gửi sau lần nhắn cuối
        if row["follow_up_sent_at"]:
            try:
                sent_at = datetime.fromisoformat(row["follow_up_sent_at"])
                if sent_at.tzinfo is None:
                    sent_at = sent_at.replace(tzinfo=vn)
                if sent_at > last_msg:
                    # Đã gửi follow-up cho lần im lặng này rồi
                    continue
            except Exception:
                pass

        msg = MESSAGES[min(follow_up_count, len(MESSAGES) - 1)]
        ok = send_message(chat_id, msg)

        if ok:
            conn.execute(
                "UPDATE conversations SET follow_up_sent_at=?, follow_up_count=? WHERE chat_id=?",
                (now.isoformat(), follow_up_count + 1, chat_id),
            )
            conn.commit()
            print(f"[followup] sent to {chat_id} (idle={idle_seconds:.1f}s, count={follow_up_count + 1})")
        else:
            print(f"[followup] failed to send to {chat_id}")

    conn.close()


if __name__ == "__main__":
    main()
