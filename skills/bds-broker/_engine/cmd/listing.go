package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

func cmdListingList(args []string) {
	// flags: --type TYPE --location LOC --min N --max N --limit N --giao-dich ban|cho_thue
	var typeKW, locKW, giaoDich string
	var minPrice, maxPrice int = 0, 999999999
	var limit int = 10

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--type":
			i++
			if i < len(args) {
				typeKW = strings.ToLower(args[i])
			}
		case "--location":
			i++
			if i < len(args) {
				locKW = strings.ToLower(args[i])
			}
		case "--min":
			i++
			if i < len(args) {
				minPrice, _ = strconv.Atoi(args[i])
			}
		case "--max":
			i++
			if i < len(args) {
				maxPrice, _ = strconv.Atoi(args[i])
			}
		case "--limit":
			i++
			if i < len(args) {
				limit, _ = strconv.Atoi(args[i])
			}
		case "--giao-dich":
			i++
			if i < len(args) {
				giaoDich = args[i]
			}
		}
	}

	base := duAnDir()
	type result struct {
		Path         string `json:"path"`
		Ten          string `json:"ten"`
		LoaiBDS      string `json:"loai_bds"`
		ViTri        string `json:"vi_tri"`
		DienTich     string `json:"dien_tich"`
		Gia          int    `json:"gia"`
		SoPhongNgu   string `json:"so_phong_ngu"`
		Huong        string `json:"huong"`
		PhapLy       string `json:"phap_ly"`
		LoaiGiaoDich string `json:"loai_giao_dich"`
	}
	var results []result

	loaiDirs, _ := os.ReadDir(base)
	for _, loaiEntry := range loaiDirs {
		if !loaiEntry.IsDir() {
			continue
		}
		loai := loaiEntry.Name()
		idDirs, _ := os.ReadDir(filepath.Join(base, loai))
		for _, idEntry := range idDirs {
			if !idEntry.IsDir() {
				continue
			}
			listingPath := loai + "/" + idEntry.Name()
			chiTiet := filepath.Join(base, listingPath, "chi-tiet.md")
			data, err := os.ReadFile(chiTiet)
			if err != nil {
				continue
			}
			fm := parseFrontmatter(string(data))

			status := fm["trang_thai"]
			if status == "" {
				status = "con_hang"
			}
			if status != "con_hang" {
				continue
			}

			if typeKW != "" {
				haystack := strings.ToLower(fm["loai_bds"] + " " + fm["ten"])
				if !strings.Contains(haystack, typeKW) {
					continue
				}
			}
			if locKW != "" {
				haystack := strings.ToLower(fm["vi_tri"] + " " + fm["dia_chi"])
				if !strings.Contains(haystack, locKW) {
					continue
				}
			}
			if giaoDich != "" && fm["loai_giao_dich"] != giaoDich {
				continue
			}

			gia, _ := strconv.Atoi(fm["gia"])
			if gia < minPrice || gia > maxPrice {
				continue
			}

			results = append(results, result{
				Path:         listingPath,
				Ten:          fm["ten"],
				LoaiBDS:      fm["loai_bds"],
				ViTri:        fm["vi_tri"],
				DienTich:     fm["dien_tich"],
				Gia:          gia,
				SoPhongNgu:   fm["so_phong_ngu"],
				Huong:        fm["huong"],
				PhapLy:       fm["phap_ly"],
				LoaiGiaoDich: fm["loai_giao_dich"],
			})
		}
	}

	sort.Slice(results, func(i, j int) bool { return results[i].Gia < results[j].Gia })
	if limit > 0 && len(results) > limit {
		results = results[:limit]
	}

	jsonOut(map[string]interface{}{
		"status":  "ok",
		"count":   len(results),
		"results": results,
	})
}

func cmdListingGet(args []string) {
	if len(args) == 0 {
		die("usage: bds-cli listing get LISTING_PATH")
	}
	listingPath := args[0]
	chiTiet := filepath.Join(duAnDir(), listingPath, "chi-tiet.md")
	data, err := os.ReadFile(chiTiet)
	if err != nil {
		die("listing not found: %s", listingPath)
	}
	fm := parseFrontmatter(string(data))
	jsonOut(map[string]interface{}{
		"status":  "ok",
		"path":    listingPath,
		"fields":  fm,
		"content": string(data),
	})
}

func cmdListingImages(args []string) {
	if len(args) == 0 {
		die("usage: bds-cli listing images LISTING_PATH [SUBFOLDER]")
	}
	listingPath := args[0]
	subfolder := ""
	if len(args) > 1 {
		subfolder = args[1]
	}

	imgDir := filepath.Join(duAnDir(), listingPath, "hinh-anh")
	if subfolder != "" && subfolder != "." {
		imgDir = filepath.Join(imgDir, subfolder)
	}

	exts := map[string]bool{".jpg": true, ".jpeg": true, ".png": true, ".webp": true}

	type folderInfo struct {
		Name  string   `json:"name"`
		Count int      `json:"count"`
		Files []string `json:"files"`
	}

	entries, err := os.ReadDir(imgDir)
	if err != nil {
		jsonOut(map[string]interface{}{"status": "ok", "images": []string{}, "subfolders": []folderInfo{}})
		return
	}

	var images []string
	var subfolders []folderInfo
	for _, e := range entries {
		if e.IsDir() {
			subEntries, _ := os.ReadDir(filepath.Join(imgDir, e.Name()))
			var files []string
			for _, f := range subEntries {
				if !f.IsDir() && exts[strings.ToLower(filepath.Ext(f.Name()))] {
					files = append(files, filepath.Join(imgDir, e.Name(), f.Name()))
				}
			}
			subfolders = append(subfolders, folderInfo{Name: e.Name(), Count: len(files), Files: files})
		} else if exts[strings.ToLower(filepath.Ext(e.Name()))] {
			images = append(images, filepath.Join(imgDir, e.Name()))
		}
	}

	jsonOut(map[string]interface{}{
		"status":     "ok",
		"path":       listingPath,
		"subfolder":  subfolder,
		"images":     images,
		"subfolders": subfolders,
	})
}

func cmdListingNextID(args []string) {
	if len(args) == 0 {
		die("usage: bds-cli listing next-id LOAI_FOLDER")
	}
	dir := filepath.Join(duAnDir(), args[0])
	os.MkdirAll(dir, 0o755)
	entries, _ := os.ReadDir(dir)
	max := 0
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		n, err := strconv.Atoi(e.Name())
		if err == nil && n > max {
			max = n
		}
	}
	jsonOut(map[string]interface{}{"status": "ok", "next_id": max + 1})
}

func cmdListingCreate(args []string) {
	// args: LOAI_FOLDER ID TITLE LOAI_VI LOCATION ADDRESS AREA PRICE BEDROOMS DIRECTION LEGAL_STATUS DESCRIPTION
	if len(args) < 12 {
		die("usage: bds-cli listing create LOAI_FOLDER ID TITLE LOAI_VI LOCATION ADDRESS AREA PRICE BEDROOMS DIRECTION LEGAL_STATUS DESCRIPTION")
	}
	loai, id := args[0], args[1]
	title, loaiBDSVi, location, address := args[2], args[3], args[4], args[5]
	area, price, bedrooms, direction, legal, desc := args[6], args[7], args[8], args[9], args[10], args[11]

	dst := filepath.Join(duAnDir(), loai, id)
	imgDir := filepath.Join(dst, "hinh-anh")
	if err := os.MkdirAll(imgDir, 0o755); err != nil {
		die("mkdir: %v", err)
	}

	content := fmt.Sprintf(`---
ten: %s
loai_bds: %s
vi_tri: %s
dia_chi: %s
dien_tich: %s
gia: %s
so_phong_ngu: %s
huong: %s
phap_ly: %s
trang_thai: con_hang
loai_giao_dich: ban
chu_dau_tu:
tien_ich:
ngay_ban_giao:
created_at: %s
---

%s
`, title, loaiBDSVi, location, address, area, price, bedrooms, direction, legal, vnNowISO(), desc)

	if err := os.WriteFile(filepath.Join(dst, "chi-tiet.md"), []byte(content), 0o644); err != nil {
		die("write: %v", err)
	}

	jsonOut(map[string]interface{}{
		"status":       "ok",
		"listing_path": loai + "/" + id,
		"message":      fmt.Sprintf("Đã tạo %s/%s — %s", loai, id, title),
	})
}

func cmdListingSetField(args []string) {
	if len(args) < 3 {
		die("usage: bds-cli listing set-field LISTING_PATH FIELD VALUE")
	}
	listingPath, field, value := args[0], args[1], args[2]
	chiTiet := filepath.Join(duAnDir(), listingPath, "chi-tiet.md")
	data, err := os.ReadFile(chiTiet)
	if err != nil {
		die("listing not found: %s", listingPath)
	}
	updated := setFrontmatter(string(data), field, value)
	if err := os.WriteFile(chiTiet, []byte(updated), 0o644); err != nil {
		die("write: %v", err)
	}
	jsonOut(map[string]interface{}{
		"status":  "ok",
		"message": fmt.Sprintf("Đã cập nhật %s.%s = %s", listingPath, field, value),
	})
}

func jsonOut(v interface{}) {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	_ = enc.Encode(v)
}
