package oauth

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

func init() {
	Register(&GoogleSheets{})
}

// GoogleSheets implements OAuth for Google Sheets.
// After authorization it creates a Finance Tracker spreadsheet with
// a transactions sheet, a summary sheet with SUMIF formulas, and a pie chart.
type GoogleSheets struct{}

func (g *GoogleSheets) Name() string    { return "google_sheets" }
func (g *GoogleSheets) Display() string { return "Google Sheets" }

const sheetsScope = "https://www.googleapis.com/auth/spreadsheets"

var financeCategories = []string{
	"Ăn uống", "Cafe", "Mua sắm", "Di chuyển",
	"Y tế", "Giải trí", "Học tập", "Nhà cửa", "Công việc", "Khác",
}

func (g *GoogleSheets) Authenticate() (map[string]string, error) {
	fmt.Println()
	fmt.Println("  ╔════════════════════════════════════════════╗")
	fmt.Println("  ║   Kết nối Google Sheets                    ║")
	fmt.Println("  ╚════════════════════════════════════════════╝")
	fmt.Println()
	fmt.Println("  Bước 1: Tạo OAuth2 credentials tại Google Cloud Console")
	fmt.Println("  → https://console.cloud.google.com/apis/credentials")
	fmt.Println()
	fmt.Println("  Lưu ý: Bật Google Sheets API trước tại")
	fmt.Println("  → https://console.cloud.google.com/apis/library/sheets.googleapis.com")
	fmt.Println()
	fmt.Println("  Redirect URI cần thêm vào OAuth app:")
	fmt.Printf("  → http://localhost:%d/callback\n", CallbackPort)
	fmt.Println()

	clientID := PromptInput("  Google Client ID")
	clientSecret := PromptInput("  Google Client Secret")

	redirectURI := fmt.Sprintf("http://localhost:%d/callback", CallbackPort)
	authURL := fmt.Sprintf(
		"https://accounts.google.com/o/oauth2/v2/auth?client_id=%s&redirect_uri=%s&response_type=code&scope=%s&access_type=offline&prompt=consent",
		url.QueryEscape(clientID),
		url.QueryEscape(redirectURI),
		url.QueryEscape(sheetsScope),
	)

	fmt.Println()
	fmt.Println("  Bước 2: Đăng nhập Google và cấp quyền Google Sheets")
	fmt.Println("  Đang mở trình duyệt...")
	fmt.Println("  Nếu trình duyệt không mở, truy cập:")
	fmt.Printf("  %s\n\n", authURL)

	OpenBrowser(authURL)

	code, err := WaitForCallback()
	if err != nil {
		return nil, err
	}

	fmt.Println()
	fmt.Println("  Bước 3: Đang lấy access token...")
	tokens, err := exchangeGoogleCode(code, clientID, clientSecret, redirectURI)
	if err != nil {
		return nil, err
	}

	fmt.Println("  ✓ Xác thực thành công")
	fmt.Println()
	fmt.Println("  Bước 4: Đang tạo Google Sheets và thiết lập cấu trúc...")

	sheetID, sheetURL, err := createFinanceSpreadsheet(tokens["access_token"])
	if err != nil {
		return nil, fmt.Errorf("tạo Google Sheets thất bại: %w", err)
	}

	fmt.Println("  ✓ Đã tạo spreadsheet")
	fmt.Println()
	fmt.Println("  Bước 5: Đang kiểm tra kết nối...")

	if err := verifySheetAccess(tokens["access_token"], sheetID); err != nil {
		return nil, fmt.Errorf("kiểm tra kết nối thất bại: %w", err)
	}

	fmt.Println("  ✓ Kết nối Google Sheets thành công")
	fmt.Println()
	fmt.Printf("  Spreadsheet: %s\n", sheetURL)
	fmt.Println()

	tokens["google_client_id"] = clientID
	tokens["google_client_secret"] = clientSecret
	tokens["spreadsheet_id"] = sheetID
	tokens["spreadsheet_url"] = sheetURL
	return tokens, nil
}

// createFinanceSpreadsheet creates a spreadsheet with transactions + report sheets and a pie chart.
func createFinanceSpreadsheet(accessToken string) (sheetID, sheetURL string, err error) {
	// Step 1: Create spreadsheet with 2 sheets
	createBody := map[string]any{
		"properties": map[string]any{
			"title":  "Finance Tracker — Chi tiêu cá nhân",
			"locale": "vi_VN",
		},
		"sheets": []map[string]any{
			{"properties": map[string]any{"title": "Giao dịch", "index": 0}},
			{"properties": map[string]any{"title": "Báo cáo", "index": 1}},
		},
	}

	respBody, err := sheetsAPI("POST", "https://sheets.googleapis.com/v4/spreadsheets", accessToken, createBody)
	if err != nil {
		return "", "", err
	}

	var created struct {
		SpreadsheetID string `json:"spreadsheetId"`
		SpreadsheetURL string `json:"spreadsheetUrl"`
		Sheets        []struct {
			Properties struct {
				SheetID int    `json:"sheetId"`
				Title   string `json:"title"`
			} `json:"properties"`
		} `json:"sheets"`
	}
	if err := json.Unmarshal(respBody, &created); err != nil {
		return "", "", fmt.Errorf("parse response: %w", err)
	}

	txSheetID := created.Sheets[0].Properties.SheetID
	reportSheetID := created.Sheets[1].Properties.SheetID

	// Step 2: Batch update — headers, formulas, formatting, chart
	batchURL := fmt.Sprintf("https://sheets.googleapis.com/v4/spreadsheets/%s:batchUpdate", created.SpreadsheetID)

	requests := []map[string]any{
		// Headers for "Giao dịch" sheet
		updateCellsRequest(txSheetID, 0, 0, [][]cellValue{
			{
				{v: "Ngày", bold: true, bg: "4285F4", fg: "FFFFFF"},
				{v: "Nơi mua", bold: true, bg: "4285F4", fg: "FFFFFF"},
				{v: "Số tiền (đ)", bold: true, bg: "4285F4", fg: "FFFFFF"},
				{v: "Danh mục", bold: true, bg: "4285F4", fg: "FFFFFF"},
				{v: "Ghi chú", bold: true, bg: "4285F4", fg: "FFFFFF"},
			},
		}),
		// Freeze header row in "Giao dịch"
		freezeRowRequest(txSheetID, 1),
		// Set column widths for "Giao dịch"
		setColumnWidthRequest(txSheetID, 0, 110), // Ngày
		setColumnWidthRequest(txSheetID, 1, 200), // Nơi mua
		setColumnWidthRequest(txSheetID, 2, 130), // Số tiền
		setColumnWidthRequest(txSheetID, 3, 140), // Danh mục
		setColumnWidthRequest(txSheetID, 4, 200), // Ghi chú
		// Headers for "Báo cáo" sheet
		updateCellsRequest(reportSheetID, 0, 0, [][]cellValue{
			{
				{v: "Danh mục", bold: true, bg: "34A853", fg: "FFFFFF"},
				{v: "Số tiền (đ)", bold: true, bg: "34A853", fg: "FFFFFF"},
				{v: "Tỷ lệ (%)", bold: true, bg: "34A853", fg: "FFFFFF"},
			},
		}),
		// Category labels + SUMIF formulas in "Báo cáo"
		updateCellsRequest(reportSheetID, 1, 0, buildReportRows()),
		// Total row in "Báo cáo"
		updateCellsRequest(reportSheetID, len(financeCategories)+1, 0, [][]cellValue{
			{
				{v: "TỔNG", bold: true},
				{formula: fmt.Sprintf("=SUM(B2:B%d)", len(financeCategories)+1), bold: true},
				{formula: "=100%", bold: true},
			},
		}),
		// Freeze header row in "Báo cáo"
		freezeRowRequest(reportSheetID, 1),
		// Set column widths for "Báo cáo"
		setColumnWidthRequest(reportSheetID, 0, 160),
		setColumnWidthRequest(reportSheetID, 1, 150),
		setColumnWidthRequest(reportSheetID, 2, 120),
		// Pie chart
		addPieChartRequest(reportSheetID, len(financeCategories)),
	}

	_, err = sheetsAPI("POST", batchURL, accessToken, map[string]any{"requests": requests})
	if err != nil {
		return "", "", fmt.Errorf("setup spreadsheet: %w", err)
	}

	return created.SpreadsheetID, created.SpreadsheetURL, nil
}

// verifySheetAccess writes a test row then deletes it to confirm write access.
func verifySheetAccess(accessToken, spreadsheetID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	checkURL := fmt.Sprintf("https://sheets.googleapis.com/v4/spreadsheets/%s?fields=spreadsheetId", spreadsheetID)
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, checkURL, nil)
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("không thể truy cập spreadsheet (HTTP %d)", resp.StatusCode)
	}
	return nil
}

// --- helpers ---

type cellValue struct {
	v       string
	formula string
	bold    bool
	bg      string // hex without #
	fg      string // hex without #
}

func buildReportRows() [][]cellValue {
	rows := make([][]cellValue, len(financeCategories))
	for i, cat := range financeCategories {
		sumif := fmt.Sprintf(`=IFERROR(SUMIF('Giao dịch'!D:D,"%s",'Giao dịch'!C:C),0)`, cat)
		pct := fmt.Sprintf(`=IFERROR(B%d/B$%d,0)`, i+2, len(financeCategories)+2)
		rows[i] = []cellValue{
			{v: cat},
			{formula: sumif},
			{formula: pct},
		}
	}
	return rows
}

func updateCellsRequest(sheetID, startRow, startCol int, rows [][]cellValue) map[string]any {
	var apiRows []map[string]any
	for _, row := range rows {
		var values []map[string]any
		for _, cell := range row {
			userEnteredValue := map[string]any{}
			if cell.formula != "" {
				userEnteredValue["formulaValue"] = cell.formula
			} else {
				userEnteredValue["stringValue"] = cell.v
			}

			format := map[string]any{}
			if cell.bold {
				format["textFormat"] = map[string]any{"bold": true}
			}
			if cell.bg != "" {
				format["backgroundColor"] = hexToRGB(cell.bg)
			}
			if cell.fg != "" {
				tf, ok := format["textFormat"].(map[string]any)
				if !ok {
					tf = map[string]any{}
				}
				tf["bold"] = cell.bold
				tf["foregroundColor"] = hexToRGB(cell.fg)
				format["textFormat"] = tf
			}

			v := map[string]any{"userEnteredValue": userEnteredValue}
			if len(format) > 0 {
				v["userEnteredFormat"] = format
			}
			values = append(values, v)
		}
		apiRows = append(apiRows, map[string]any{"values": values})
	}

	return map[string]any{
		"updateCells": map[string]any{
			"rows": apiRows,
			"fields": "userEnteredValue,userEnteredFormat",
			"start": map[string]any{
				"sheetId":     sheetID,
				"rowIndex":    startRow,
				"columnIndex": startCol,
			},
		},
	}
}

func freezeRowRequest(sheetID, count int) map[string]any {
	return map[string]any{
		"updateSheetProperties": map[string]any{
			"properties": map[string]any{
				"sheetId": sheetID,
				"gridProperties": map[string]any{
					"frozenRowCount": count,
				},
			},
			"fields": "gridProperties.frozenRowCount",
		},
	}
}

func setColumnWidthRequest(sheetID, colIndex, width int) map[string]any {
	return map[string]any{
		"updateDimensionProperties": map[string]any{
			"range": map[string]any{
				"sheetId":    sheetID,
				"dimension":  "COLUMNS",
				"startIndex": colIndex,
				"endIndex":   colIndex + 1,
			},
			"properties": map[string]any{
				"pixelSize": width,
			},
			"fields": "pixelSize",
		},
	}
}

func addPieChartRequest(reportSheetID, categoryCount int) map[string]any {
	endRow := categoryCount + 1 // rows 1..N (0-indexed: 1 to categoryCount+1)
	return map[string]any{
		"addChart": map[string]any{
			"chart": map[string]any{
				"spec": map[string]any{
					"title": "Chi tiêu theo danh mục",
					"pieChart": map[string]any{
						"legendPosition": "RIGHT_LEGEND",
						"threeDimensional": false,
						"domain": map[string]any{
							"sourceRange": map[string]any{
								"sources": []map[string]any{{
									"sheetId":          reportSheetID,
									"startRowIndex":    1,
									"endRowIndex":      endRow,
									"startColumnIndex": 0,
									"endColumnIndex":   1,
								}},
							},
						},
						"series": map[string]any{
							"sourceRange": map[string]any{
								"sources": []map[string]any{{
									"sheetId":          reportSheetID,
									"startRowIndex":    1,
									"endRowIndex":      endRow,
									"startColumnIndex": 1,
									"endColumnIndex":   2,
								}},
							},
						},
					},
				},
				"position": map[string]any{
					"overlayPosition": map[string]any{
						"anchorCell": map[string]any{
							"sheetId":     reportSheetID,
							"rowIndex":    0,
							"columnIndex": 4,
						},
						"widthPixels":  520,
						"heightPixels": 380,
					},
				},
			},
		},
	}
}

func hexToRGB(hex string) map[string]any {
	if len(hex) != 6 {
		return map[string]any{"red": 1.0, "green": 1.0, "blue": 1.0}
	}
	r := hexByte(hex[0:2])
	gr := hexByte(hex[2:4])
	b := hexByte(hex[4:6])
	return map[string]any{
		"red":   float64(r) / 255.0,
		"green": float64(gr) / 255.0,
		"blue":  float64(b) / 255.0,
	}
}

func hexByte(s string) int {
	val := 0
	for _, c := range s {
		val *= 16
		switch {
		case c >= '0' && c <= '9':
			val += int(c - '0')
		case c >= 'a' && c <= 'f':
			val += int(c-'a') + 10
		case c >= 'A' && c <= 'F':
			val += int(c-'A') + 10
		}
	}
	return val
}

func sheetsAPI(method, apiURL, accessToken string, body any) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, method, apiURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("Sheets API error %d: %s", resp.StatusCode, string(respBody))
	}
	return respBody, nil
}
