"""
CareHub Baby follow-up checker.
Chay boi cron job, output danh sach khach can follow-up de agent gui tin qua zalouser.
"""
import sqlite3
import sys
import json
from datetime import datetime, timezone, timedelta
import os

sys.stdout.reconfigure(encoding='utf-8', errors='replace')

DB_PATH = os.path.expanduser("~/.openclaw/workspace/skills/carehub-baby/orders.db")
VN = timezone(timedelta(hours=7))
now = datetime.now(VN)

conn = sqlite3.connect(DB_PATH)
conn.row_factory = sqlite3.Row

# Lay tat ca conversations can follow-up
rows = conn.execute("""
    SELECT * FROM conversations
    WHERE follow_up_count < 3
    ORDER BY last_customer_msg_at ASC
""").fetchall()

follow_ups = []

for r in rows:
    d = dict(r)
    zalo_id = d['customer_zalo_id']
    zalo_name = d.get('customer_zalo_name', '')
    stage = d.get('stage', 'greeting')
    follow_up_count = d.get('follow_up_count', 0)
    last_customer_msg = d.get('last_customer_msg_at')
    last_follow_up = d.get('last_follow_up_at')
    has_order = d.get('has_order', 0)
    last_order_id = d.get('last_order_id')

    if not last_customer_msg:
        continue

    last_msg_time = datetime.fromisoformat(last_customer_msg)
    if last_msg_time.tzinfo is None:
        last_msg_time = last_msg_time.replace(tzinfo=VN)

    minutes_since = (now - last_msg_time).total_seconds() / 60

    # Khong follow-up neu vua gui follow-up gan day (< 2 phut)
    if last_follow_up:
        last_fu_time = datetime.fromisoformat(last_follow_up)
        if last_fu_time.tzinfo is None:
            last_fu_time = last_fu_time.replace(tzinfo=VN)
        minutes_since_fu = (now - last_fu_time).total_seconds() / 60
        if minutes_since_fu < 2:
            continue

    # Chi follow-up trong gio lam viec (8h-22h)
    if now.hour < 8 or now.hour >= 22:
        continue

    msg = None

    # Check don da giao -> hoi feedback (2-3 ngay sau)
    if has_order and last_order_id:
        order = conn.execute(
            "SELECT * FROM orders WHERE id=? AND status='completed'",
            (last_order_id,)
        ).fetchone()
        if order:
            order_d = dict(order)
            created = datetime.fromisoformat(order_d['created_at'])
            if created.tzinfo is None:
                created = created.replace(tzinfo=VN)
            days_since_order = (now - created).total_seconds() / 86400
            if 2 <= days_since_order <= 5 and follow_up_count == 0:
                msg = "Dạ shop xin phép hỏi thăm 😊 Bé nhà mình dùng sữa có hợp không ạ? Nếu cần đổi loại phù hợp hơn, shop hỗ trợ mình ngay ạ 👍"

    # Khach dang tu van nhung chua phan hoi
    if not msg and not has_order:
        if follow_up_count == 0 and minutes_since >= 2:
            msg = "Dạ shop gửi thêm thông tin để bạn tham khảo ạ 😊 Không biết bé nhà mình hiện tại bao nhiêu tháng rồi ạ?"
        elif follow_up_count == 1 and minutes_since >= 2:
            msg = "Dạ không biết bạn còn quan tâm sản phẩm không ạ? Shop vẫn đang có ưu đãi + freeship hôm nay đó ạ 🎉"
        elif follow_up_count == 2 and minutes_since >= 2:
            msg = "Dạ shop nhắn lại để hỗ trợ mình ạ 😊 Nếu bé cần sữa dễ tiêu hóa, dòng này khá phù hợp đó ạ 👍"

    # Khach da tu van nhung chua mua (co tuong tac nhung khong co don)
    if not msg and not has_order and stage == 'consulted' and follow_up_count == 0 and minutes_since >= 2:
        msg = "Dạ hôm trước shop có tư vấn cho bé nhà mình 😊 Không biết bé đã dùng thử sữa nào chưa ạ? Hiện bên shop vẫn đang có ưu đãi tốt, mình cần shop hỗ trợ thêm không ạ?"

    if msg:
        follow_ups.append({
            "zalo_id": zalo_id,
            "zalo_name": zalo_name,
            "message": msg,
            "follow_up_count": follow_up_count + 1
        })

conn.close()

if not follow_ups:
    print("NO_FOLLOWUP")
else:
    for fu in follow_ups:
        print(json.dumps(fu, ensure_ascii=False))
