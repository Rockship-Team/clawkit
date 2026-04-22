package main

import (
	"database/sql"
	"fmt"
)

func cmdApptBook(args []string) {
	if len(args) < 5 {
		die("usage: bds-cli appt book CUSTOMER CONTACT LISTING_PATH TITLE DATETIME [NOTE]")
	}
	customer, contact, listingPath, title, dt := args[0], args[1], args[2], args[3], args[4]
	note := ""
	if len(args) > 5 {
		note = args[5]
	}

	db := openDB()
	defer db.Close()
	initDB(db)

	res, err := db.Exec(
		`INSERT INTO "lich-hen" (ten_khach, lien_he_khach, du_an_id, ten_du_an, thoi_gian_hen, ghi_chu, created_at) VALUES (?,?,?,?,?,?,?)`,
		customer, contact, listingPath, title, dt, note, vnNowISO(),
	)
	if err != nil {
		die("insert: %v", err)
	}
	id, _ := res.LastInsertId()
	jsonOut(map[string]interface{}{
		"status":  "ok",
		"id":      id,
		"message": fmt.Sprintf("Lịch #%d đã lưu — %s %s", id, customer, dt),
	})
}

func cmdApptList(args []string) {
	keyword := ""
	if len(args) > 0 {
		keyword = "%" + args[0] + "%"
	}

	db := openDB()
	defer db.Close()
	initDB(db)

	var (
		rows *sql.Rows
		err  error
	)
	if keyword != "" {
		rows, err = db.Query(`SELECT id,ten_khach,lien_he_khach,du_an_id,ten_du_an,thoi_gian_hen,trang_thai,ghi_chu FROM "lich-hen" WHERE lien_he_khach LIKE ? OR ten_khach LIKE ? ORDER BY thoi_gian_hen DESC LIMIT 10`, keyword, keyword)
	} else {
		rows, err = db.Query(`SELECT id,ten_khach,lien_he_khach,du_an_id,ten_du_an,thoi_gian_hen,trang_thai,ghi_chu FROM "lich-hen" ORDER BY thoi_gian_hen DESC LIMIT 20`)
	}
	if err != nil {
		die("query: %v", err)
	}
	defer rows.Close()

	data := scanRows(rows)
	jsonOut(map[string]interface{}{"status": "ok", "count": len(data), "appointments": data})
}

func cmdApptUpdate(args []string) {
	if len(args) < 2 {
		die("usage: bds-cli appt update ID STATUS")
	}
	id, status := args[0], args[1]
	valid := map[string]bool{"cho_xac_nhan": true, "da_xac_nhan": true, "da_huy": true}
	if !valid[status] {
		die("invalid status: %s (cho_xac_nhan|da_xac_nhan|da_huy)", status)
	}

	db := openDB()
	defer db.Close()

	res, err := db.Exec(`UPDATE "lich-hen" SET trang_thai=? WHERE id=?`, status, id)
	if err != nil {
		die("update: %v", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		die("appointment #%s not found", id)
	}
	jsonOut(map[string]interface{}{
		"status":  "ok",
		"message": fmt.Sprintf("Lịch #%s → %s", id, status),
	})
}

func scanRows(rows *sql.Rows) []map[string]interface{} {
	cols, _ := rows.Columns()
	var out []map[string]interface{}
	for rows.Next() {
		vals := make([]interface{}, len(cols))
		ptrs := make([]interface{}, len(cols))
		for i := range vals {
			ptrs[i] = &vals[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			continue
		}
		row := map[string]interface{}{}
		for i, c := range cols {
			row[c] = vals[i]
		}
		out = append(out, row)
	}
	return out
}
