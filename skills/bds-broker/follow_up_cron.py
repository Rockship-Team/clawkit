#!/usr/bin/env python3
"""
BDS Broker Follow-up Cron Wrapper
Chay moi 60 phut (8h-21h), kiem tra va gui tin follow-up qua Zalo.
"""
import subprocess
import sys
import json
import os
import sqlite3
from pathlib import Path
from datetime import datetime, timezone, timedelta

VN = timezone(timedelta(hours=7))
DB_PATH = os.path.expanduser("~/.openclaw/workspace/skills/bds-broker/bds.db")
FOLLOW_UP_SCRIPT = Path(os.path.expanduser("~/.openclaw/workspace/skills/bds-broker/follow_up.py"))

# Import zalouser tool (openclaw internal API)
try:
    from openclaw import tools
    ZALO_TOOL = tools.zalouser
except Exception as e:
    print(f"ERROR: Cannot import zalouser tool: {e}")
    sys.exit(1)


def update_follow_up_count(zalo_id: str, new_count: int):
    """Cap nhat follow_up_count va last_follow_up_at sau khi gui tin."""
    now = datetime.now(VN).isoformat()
    conn = sqlite3.connect(DB_PATH)
    conn.execute(
        "UPDATE conversations SET follow_up_count=?, last_follow_up_at=? WHERE customer_zalo_id=?",
        (new_count, now, zalo_id)
    )
    conn.commit()
    conn.close()


def main():
    # Chay follow_up.py de lay danh sach can follow-up
    result = subprocess.run(
        [sys.executable, str(FOLLOW_UP_SCRIPT)],
        capture_output=True,
        text=True,
        timeout=30
    )

    if result.returncode != 0:
        print(f"ERROR: follow_up.py failed: {result.stderr}")
        sys.exit(1)

    output = result.stdout.strip()

    if not output or output == "NO_FOLLOWUP":
        print("NO_FOLLOWUP")
        sys.exit(0)

    lines = output.strip().split('\n')
    sent_count = 0

    for line in lines:
        try:
            fu = json.loads(line)
            zalo_id = fu.get('zalo_id', '')
            zalo_name = fu.get('zalo_name', '')
            message = fu.get('message', '')
            new_count = fu.get('follow_up_count', 1)

            if not zalo_id or not message:
                continue

            # Gui tin qua Zalo
            ZALO_TOOL(
                action="message",
                user_id=zalo_id,
                message=message
            )

            # Cap nhat DB
            update_follow_up_count(zalo_id, new_count)

            print(f"Sent follow-up #{new_count} to {zalo_name} ({zalo_id})")
            sent_count += 1

        except json.JSONDecodeError:
            print(f"ERROR: Invalid JSON line: {line}")
            continue
        except Exception as e:
            print(f"ERROR: Failed to send to {zalo_id}: {e}")
            continue

    print(f"Follow-up complete: {sent_count} messages sent")


if __name__ == "__main__":
    main()
