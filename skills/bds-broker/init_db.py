"""Initialize the bds-broker SQLite database."""
import sqlite3
from pathlib import Path

DB_PATH = Path(__file__).parent / "bds.db"


def init_db():
    conn = sqlite3.connect(DB_PATH)

    conn.execute("""
        CREATE TABLE IF NOT EXISTS listings (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            title TEXT NOT NULL,
            property_type TEXT,
            location TEXT,
            address TEXT,
            area INTEGER DEFAULT 0,
            price INTEGER DEFAULT 0,
            bedrooms INTEGER DEFAULT 0,
            direction TEXT,
            legal_status TEXT DEFAULT 'pending',
            description TEXT,
            status TEXT DEFAULT 'available',
            created_at TEXT NOT NULL
        )
    """)

    conn.execute("""
        CREATE TABLE IF NOT EXISTS appointments (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            customer_name TEXT,
            customer_contact TEXT,
            listing_id INTEGER,
            listing_title TEXT,
            scheduled_at TEXT,
            status TEXT DEFAULT 'scheduled',
            note TEXT,
            gcal_event_id TEXT,
            created_at TEXT NOT NULL,
            FOREIGN KEY (listing_id) REFERENCES listings(id)
        )
    """)

    # Migrate existing DB: add gcal_event_id if missing
    existing = {row[1] for row in conn.execute("PRAGMA table_info(appointments)")}
    if "gcal_event_id" not in existing:
        conn.execute("ALTER TABLE appointments ADD COLUMN gcal_event_id TEXT")

    conn.commit()
    conn.close()
    print(f"Database initialized at {DB_PATH}")


if __name__ == "__main__":
    init_db()
