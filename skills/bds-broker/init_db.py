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
            bathrooms INTEGER DEFAULT 0,
            floor INTEGER DEFAULT 0,
            direction TEXT,
            legal_status TEXT DEFAULT 'pending',
            certificate TEXT,
            planning_status TEXT,
            legal_note TEXT,
            description TEXT,
            status TEXT DEFAULT 'available',
            sale_type TEXT DEFAULT 'primary',
            images TEXT,
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

    # Migrations: add missing columns to existing tables
    def add_column_if_missing(table, column, col_def):
        existing = {row[1] for row in conn.execute(f"PRAGMA table_info({table})")}
        if column not in existing:
            conn.execute(f"ALTER TABLE {table} ADD COLUMN {column} {col_def}")
            print(f"Migrated: {table}.{column} added")

    add_column_if_missing("listings", "bathrooms", "INTEGER DEFAULT 0")
    add_column_if_missing("listings", "floor", "INTEGER DEFAULT 0")
    add_column_if_missing("listings", "certificate", "TEXT")
    add_column_if_missing("listings", "planning_status", "TEXT")
    add_column_if_missing("listings", "legal_note", "TEXT")
    add_column_if_missing("listings", "sale_type", "TEXT DEFAULT 'primary'")
    add_column_if_missing("listings", "images", "TEXT")
    add_column_if_missing("appointments", "gcal_event_id", "TEXT")
    add_column_if_missing("appointments", "listing_title", "TEXT")
    add_column_if_missing("appointments", "customer_contact", "TEXT")

    conn.commit()
    conn.close()
    print(f"Database initialized at {DB_PATH}")


if __name__ == "__main__":
    init_db()
