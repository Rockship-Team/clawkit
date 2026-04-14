"""Khởi tạo SQLite database và cấu trúc thư mục cho bds-broker."""
import sqlite3
from pathlib import Path

DB_PATH = Path(__file__).parent / "bds.db"

CATEGORIES = [
    "biet-thu-lien-ke",
    "can-ho-chung-cu",
    "cao-oc-van-phong",
    "khu-cong-nghiep",
    "khu-do-thi-moi",
    "khu-nghi-duong-sinh-thai",
    "nha-mat-pho",
    "nha-o-xa-hoi",
    "shophouse",
    "trung-tam-thuong-mai",
]


def init_db():
    conn = sqlite3.connect(DB_PATH)

    # Bảng lịch hẹn xem nhà
    conn.execute("""
        CREATE TABLE IF NOT EXISTS "lich-hen" (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            ten_khach TEXT,
            lien_he_khach TEXT,
            du_an_id TEXT,
            ten_du_an TEXT,
            thu_muc_anh TEXT,
            thoi_gian_hen TEXT,
            trang_thai TEXT DEFAULT 'cho_xac_nhan',
            ghi_chu TEXT,
            created_at TEXT NOT NULL
        )
    """)

    # Migration: thêm cột thiếu vào bảng cũ nếu tồn tại
    def add_column_if_missing(table, column, col_def):
        existing = {row[1] for row in conn.execute(f'PRAGMA table_info("{table}")')}
        if column not in existing:
            conn.execute(f'ALTER TABLE "{table}" ADD COLUMN {column} {col_def}')
            print(f"Migrated: {table}.{column} added")

    for col, defn in [
        ("ten_du_an", "TEXT"),
        ("lien_he_khach", "TEXT"),
        ("thu_muc_anh", "TEXT"),
    ]:
        add_column_if_missing("lich-hen", col, defn)

    conn.commit()
    conn.close()
    print(f"Database khởi tạo tại {DB_PATH}")

    # Tạo thư mục du-an/<category> mặc định
    du_an_root = Path(__file__).parent / "du-an"
    for category in CATEGORIES:
        category_dir = du_an_root / category
        category_dir.mkdir(parents=True, exist_ok=True)
    print(f"Thư mục du-an/ khởi tạo với {len(CATEGORIES)} categories")


if __name__ == "__main__":
    init_db()
