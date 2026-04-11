"""
BDS Broker follow-up — gửi tin nhắn Telegram cho khách cần chăm sóc.
Chạy bởi cron job mỗi 60 phút (8h-21h).
Đọc config từ ~/.openclaw/workspace/skills/bds-broker/config.json
"""
import sqlite3
import sys
import json
import os
import urllib.request
from datetime import datetime, timezone, timedelta

sys.stdout.reconfigure(encoding='utf-8', errors='replace')

DB_PATH = os.path.expanduser("~/.openclaw/workspace/skills/bds-broker/bds.db")
CONFIG_PATH = os.path.expanduser("~/.openclaw/workspace/skills/bds-broker/config.json")
VN = timezone(timedelta(hours=7))
now = datetime.now(VN)

# Chỉ chạy trong giờ làm việc (8h-21h)
if now.hour < 8 or now.hour >= 21:
    print("NO_FOLLOWUP: ngoài giờ làm việc")
    sys.exit(0)

# Đọc config lấy bot token
try:
    with open(CONFIG_PATH) as f:
        config = json.load(f)
    TOKEN = config.get("telegram_bot_token", "")
    BROKER_CHAT_ID = config.get("broker_chat_id", "")
except Exception as e:
    print(f"ERROR: không đọc được config: {e}")
    sys.exit(1)

if not TOKEN:
    print("ERROR: telegram_bot_token chưa được cấu hình")
    sys.exit(1)


def send_telegram(chat_id: str, text: str) -> bool:
    """Gửi tin nhắn Telegram, trả về True nếu thành công."""
    try:
        data = json.dumps({"chat_id": chat_id, "text": text}).encode()
        req = urllib.request.Request(
            f"https://api.telegram.org/bot{TOKEN}/sendMessage",
            data=data,
            headers={"Content-Type": "application/json"},
        )
        urllib.request.urlopen(req, timeout=10)
        return True
    except Exception as e:
        print(f"WARN: gửi Telegram thất bại chat_id={chat_id}: {e}")
        return False


def notify_broker(text: str):
    """Gửi thông báo lỗi/cảnh báo cho broker nếu có broker_chat_id."""
    if BROKER_CHAT_ID:
        send_telegram(BROKER_CHAT_ID, text)


conn = sqlite3.connect(DB_PATH)
conn.row_factory = sqlite3.Row

rows = conn.execute("""
    SELECT * FROM conversations
    WHERE follow_up_count < 3
    ORDER BY last_customer_msg_at ASC
""").fetchall()

sent_count = 0

for r in rows:
    d = dict(r)
    tg_id = d["customer_tg_id"]
    tg_name = d.get("customer_tg_name", "")
    stage = d.get("stage", "greeting")
    follow_up_count = d.get("follow_up_count", 0)
    last_customer_msg = d.get("last_customer_msg_at")
    last_follow_up = d.get("last_follow_up_at")
    has_transaction = d.get("has_transaction", 0)

    if not last_customer_msg:
        continue

    last_msg_time = datetime.fromisoformat(last_customer_msg)
    if last_msg_time.tzinfo is None:
        last_msg_time = last_msg_time.replace(tzinfo=VN)

    minutes_since = (now - last_msg_time).total_seconds() / 60

    # Không follow-up nếu vừa gửi gần đây (< 120 phút)
    if last_follow_up:
        last_fu_time = datetime.fromisoformat(last_follow_up)
        if last_fu_time.tzinfo is None:
            last_fu_time = last_fu_time.replace(tzinfo=VN)
        if (now - last_fu_time).total_seconds() / 60 < 120:
            continue

    msg = None

    # Sau lịch xem nhà 1-3 ngày, chưa có giao dịch
    recent_appt = conn.execute("""
        SELECT * FROM appointments
        WHERE customer_tg_id=? AND status='completed'
        ORDER BY scheduled_at DESC LIMIT 1
    """, (tg_id,)).fetchone()

    if recent_appt and not has_transaction:
        appt_d = dict(recent_appt)
        scheduled = datetime.fromisoformat(appt_d["scheduled_at"])
        if scheduled.tzinfo is None:
            scheduled = scheduled.replace(tzinfo=VN)
        days_since_appt = (now - scheduled).total_seconds() / 86400
        if 1 <= days_since_appt <= 3 and follow_up_count == 0:
            msg = "Dạ sau buổi xem nhà hôm trước, anh/chị có thêm câu hỏi nào không ạ? 🏠 Mình có thể hỗ trợ kiểm tra pháp lý hoặc thương lượng giá nếu anh/chị quan tâm ạ."

    # Lead mới, chưa phản hồi sau 60 phút
    if not msg and stage in ("greeting", "analyzing") and follow_up_count == 0 and minutes_since >= 60:
        msg = "Dạ mình muốn hỏi thêm về nhu cầu BDS của anh/chị để tìm sản phẩm phù hợp ạ 😊 Anh/chị đang tìm mua hay thuê ạ?"

    # Đang tư vấn, chưa phản hồi sau 4 giờ
    elif not msg and stage == "consulting" and follow_up_count == 1 and minutes_since >= 240:
        msg = "Dạ bên mình vừa cập nhật thêm một số sản phẩm mới phù hợp với nhu cầu của anh/chị ạ 🏠 Anh/chị có muốn mình gửi thông tin không ạ?"

    # Đang đàm phán, chưa phản hồi sau 2 ngày
    elif not msg and stage == "negotiating" and follow_up_count < 2 and minutes_since >= 2880:
        msg = "Dạ chủ nhà đã phản hồi đề xuất của anh/chị rồi ạ 😊 Mình có thể chia sẻ thêm để anh/chị cân nhắc không ạ?"

    # Tư vấn xong chưa mua, sau 1 ngày
    elif not msg and not has_transaction and stage == "consulting" and follow_up_count == 2 and minutes_since >= 1440:
        msg = "Dạ không biết anh/chị thấy sản phẩm vừa xem thế nào ạ? 😊 Nếu cần thêm thông tin hoặc muốn xem thêm sản phẩm khác, mình sẵn sàng hỗ trợ ạ."

    if not msg:
        continue

    # Gửi Telegram
    if send_telegram(tg_id, msg):
        conn.execute("""
            UPDATE conversations
            SET follow_up_count = follow_up_count + 1,
                last_follow_up_at = ?
            WHERE customer_tg_id = ?
        """, (now.isoformat(), tg_id))
        conn.commit()
        sent_count += 1
        print(f"Sent follow-up to {tg_name} ({tg_id}): follow_up #{follow_up_count + 1}")

conn.close()

if sent_count == 0:
    print("NO_FOLLOWUP")
else:
    print(f"Done: {sent_count} follow-up(s) sent")
