package main

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

var vnLoc = time.FixedZone("Asia/Ho_Chi_Minh", 7*60*60)

func vnNow() time.Time { return time.Now().In(vnLoc) }
func vnNowISO() string { return vnNow().Format(time.RFC3339) }

// dataDir returns the root data directory.
// Resolution: $BDS_DATA → ~/.clawkit/runtimes/bds-broker/data
func dataDir() string {
	if v := os.Getenv("BDS_DATA"); v != "" {
		return expandHome(v)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "data"
	}
	return filepath.Join(home, ".clawkit", "runtimes", "bds-broker", "data")
}

func duAnDir() string { return filepath.Join(dataDir(), "du-an") }
func dbPath() string  { return filepath.Join(dataDir(), "bds.db") }

func expandHome(p string) string {
	if p == "~" {
		if h, err := os.UserHomeDir(); err == nil {
			return h
		}
	}
	if strings.HasPrefix(p, "~/") {
		if h, err := os.UserHomeDir(); err == nil {
			return filepath.Join(h, p[2:])
		}
	}
	return p
}

func openDB() *sql.DB {
	path := dbPath()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		die("cannot create db dir: %v", err)
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		die("cannot open db: %v", err)
	}
	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		die("WAL mode: %v", err)
	}
	return db
}

func initDB(db *sql.DB) {
	schema := `
	CREATE TABLE IF NOT EXISTS "lich-hen" (
		id             INTEGER PRIMARY KEY AUTOINCREMENT,
		ten_khach      TEXT,
		lien_he_khach  TEXT,
		du_an_id       TEXT,
		ten_du_an      TEXT,
		thoi_gian_hen  TEXT,
		trang_thai     TEXT DEFAULT 'cho_xac_nhan',
		ghi_chu        TEXT,
		created_at     TEXT NOT NULL
	);`
	if _, err := db.Exec(schema); err != nil {
		die("create schema: %v", err)
	}
	// migrations
	cols := map[string]string{"ten_du_an": "TEXT", "lien_he_khach": "TEXT"}
	rows, _ := db.Query(`PRAGMA table_info("lich-hen")`)
	existing := map[string]bool{}
	if rows != nil {
		for rows.Next() {
			var cid int
			var name, typ string
			var notnull, pk int
			var dflt interface{}
			_ = rows.Scan(&cid, &name, &typ, &notnull, &dflt, &pk)
			existing[name] = true
		}
		rows.Close()
	}
	for col, def := range cols {
		if !existing[col] {
			db.Exec(fmt.Sprintf(`ALTER TABLE "lich-hen" ADD COLUMN %s %s`, col, def))
		}
	}
}

func initDirs() {
	categories := []string{
		"biet-thu-lien-ke", "can-ho-chung-cu", "cao-oc-van-phong",
		"khu-cong-nghiep", "khu-do-thi-moi", "khu-nghi-duong-sinh-thai",
		"nha-mat-pho", "nha-o-xa-hoi", "shophouse", "trung-tam-thuong-mai",
	}
	base := duAnDir()
	for _, cat := range categories {
		os.MkdirAll(filepath.Join(base, cat), 0o755)
	}
}

func die(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, `{"status":"error","message":`+"%q}\n", fmt.Sprintf(format, args...))
	os.Exit(1)
}
