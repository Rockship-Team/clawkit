"""Initialize the carehub-baby orders SQLite database."""
import sqlite3
from pathlib import Path

DB_PATH = Path(__file__).parent / "orders.db"

def init_db():
    conn = sqlite3.connect(DB_PATH)
    conn.execute("""
        CREATE TABLE IF NOT EXISTS orders (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            status TEXT NOT NULL DEFAULT 'new',
            customer_name TEXT,
            customer_phone TEXT,
            customer_zalo_id TEXT,
            customer_zalo_name TEXT,
            customer_address TEXT,
            items TEXT,
            quantity INTEGER DEFAULT 1,
            baby_age TEXT,
            price INTEGER,
            note TEXT,
            created_at TEXT NOT NULL,
            payment_status TEXT DEFAULT 'unpaid',
            deposit_amount INTEGER DEFAULT 0
        )
    """)
    # Bang theo doi hoi thoai va follow-up
    conn.execute("""
        CREATE TABLE IF NOT EXISTS conversations (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            customer_zalo_id TEXT UNIQUE NOT NULL,
            customer_zalo_name TEXT,
            last_customer_msg_at TEXT,
            last_bot_msg_at TEXT,
            stage TEXT DEFAULT 'greeting',
            follow_up_count INTEGER DEFAULT 0,
            last_follow_up_at TEXT,
            has_order INTEGER DEFAULT 0,
            last_order_id INTEGER,
            created_at TEXT NOT NULL
        )
    """)
    conn.commit()
    conn.close()
    print(f"Database initialized at {DB_PATH}")

if __name__ == "__main__":
    init_db()
