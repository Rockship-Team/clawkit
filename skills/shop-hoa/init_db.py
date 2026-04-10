"""Initialize the flower-shop orders SQLite database."""
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
            recipient_name TEXT,
            recipient_phone TEXT,
            recipient_address TEXT,
            items TEXT,
            price INTEGER,
            delivery_time TEXT,
            note TEXT,
            created_at TEXT NOT NULL
        )
    """)
    conn.commit()
    conn.close()
    print(f"Database initialized at {DB_PATH}")

if __name__ == "__main__":
    init_db()
