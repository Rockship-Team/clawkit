# clawkit

CLI tool cai dat va quan ly OpenClaw skills. Phat trien boi [Rockship](https://rockship.co).

clawkit xu ly toan bo quy trinh trien khai skill: tai skill template, xac thuc Zalo qua QR code, thu thap cau hinh khach hang, va cai dat skill vao dung thu muc OpenClaw — tat ca trong 1 lenh duy nhat.

## Cai Dat

**Mot dong lenh (khuyen dung):**

```bash
curl -fsSL https://raw.githubusercontent.com/Rockship-Team/clawkit/main/install.sh | bash
```

**Hoac build tu source:**

```bash
git clone git@github.com:Rockship-Team/clawkit.git
cd clawkit
make build
```

## Bat Dau Nhanh

```bash
# Xem danh sach skills
clawkit list

# Cai dat skill (bao gom Zalo QR login + cau hinh)
clawkit install shop-hoa-zalo
```

## Yeu Cau

- **OpenClaw** da cai tren may khach hang ([huong dan cai dat](https://docs.openclaw.ai/installation))
- **Go 1.22+** chi can khi build tu source

clawkit tu dong detect OpenClaw va cai skill vao `~/.openclaw/workspace/skills/`.

## Cac Lenh

| Lenh | Mo ta |
|------|-------|
| `clawkit list` | Xem danh sach skills va trang thai cai dat |
| `clawkit install <skill>` | Cai skill voi Zalo QR login va cau hinh |
| `clawkit update <skill>` | Cap nhat skill, giu nguyen token va config cu |
| `clawkit status` | Xem tat ca skills da cai |
| `clawkit package <skill>` | Dong goi skill thanh .tar.gz de phan phoi (dev) |
| `clawkit version` | In phien ban |

### Flag khi install

```bash
# Bo qua OAuth de test (chi danh cho dev)
clawkit install shop-hoa-zalo --skip-oauth
```

## Cach Hoat Dong

```
clawkit install shop-hoa-zalo
  |
  |-- 1. Detect OpenClaw tren may
  |-- 2. Tai skill (remote) hoac copy tu local skills/
  |-- 3. Check/cai Zalo plugin, hien QR code de KH scan
  |-- 4. Thu thap cau hinh KH (ten shop, email, bang gia, v.v.)
  |-- 5. Xu ly SKILL.md — thay placeholder bang gia tri cua KH
  |-- 6. Khoi tao database (neu co init_db.py)
  '-- 7. Luu config.json
```

### Xac Thuc Zalo Personal

clawkit dung `zca-js` tich hop san trong OpenClaw. KH khong can biet App ID hay App Secret — chi can scan QR code:

```
✓ Zalo plugin found
Opening QR code — scan it with your Zalo app.
QR code saved at: /tmp/openclaw/qr.png
Waiting for you to scan... (press Enter after scanning)
```

### He Thong Template

SKILL.md chua cac placeholder ma clawkit thay the khi install:

| Placeholder | Nguon |
|-------------|-------|
| `{shopName}` | KH nhap khi install |
| `{notifyEmailFrom}` | KH nhap |
| `{notifyEmailTo}` | KH nhap |
| `{notifyEmailAppPassword}` | KH nhap |
| `{catalogSection}` | Tu dong sinh tu `catalog.json` |
| `{baseDir}` | OpenClaw tu xu ly (clawkit khong thay the) |

### He Thong Catalog

Moi skill co the co `catalog.json` dinh nghia danh muc san pham va gia:

```json
{
  "categories": [
    {"folder": "hoa-hong", "label": "anh hoa hong"},
    {"folder": "hoa-huong-duong", "label": "anh hoa huong duong"}
  ],
  "price_tiers": [280000, 300000, 350000, 450000],
  "best_seller": true
}
```

Sau khi install, KH tu chinh bang gia truc tiep trong SKILL.md.

## Build

```bash
make build          # Build cho platform hien tai
make test           # Chay tests voi race detector
make fmt            # Format va vet code
make lint           # Chay golangci-lint
make coverage       # Bao cao test coverage
make dist           # Cross-compile cho macOS, Linux, Windows
make package SKILL=shop-hoa-zalo   # Dong goi skill thanh .tar.gz
make help           # Xem tat ca commands
```

## Cau Truc Du An

```
clawkit/
|-- cmd/clawkit/           # CLI entry point
|   '-- main.go
|-- internal/
|   |-- archive/           # Tao/giai nen tar.gz
|   |-- config/            # Detect OpenClaw, skill config
|   |-- installer/         # Cac lenh install, update, list, status, package
|   |-- template/          # Xu ly placeholder SKILL.md + catalog
|   '-- ui/                # Hien thi terminal (mau sac, prompt)
|-- oauth/                 # OAuth providers (tu dang ky)
|   |-- oauth.go           # Provider interface + registry
|   |-- zalo_personal.go   # Zalo QR code login (qua OpenClaw)
|   |-- zalo_oa.go         # Zalo Official Account OAuth
|   |-- google.go          # Google OAuth (Gmail, Sheets, Calendar)
|   '-- facebook.go        # Facebook OAuth (Pages, Messenger)
|-- skills/                # Skill templates
|   '-- shop-hoa-zalo/
|       |-- SKILL.md       # Skill voi placeholder
|       |-- catalog.json   # Danh muc san pham va gia
|       |-- init_db.py     # Khoi tao database
|       '-- flowers/       # Anh san pham mau
|-- registry.json          # Danh sach skills
|-- install.sh             # Installer qua curl
|-- Makefile               # Build, test, lint, dist
|-- .github/workflows/     # CI pipeline
|-- .golangci.yml          # Cau hinh linter
|-- .editorconfig          # Quy tac format code
'-- LICENSE                # MIT
```

## Dong Gop (Contributing)

### Them Skill Moi

1. Tao thu muc trong `skills/`:

```
skills/ten-skill/
|-- SKILL.md           # Dung {placeholder} cho gia tri rieng moi KH
|-- catalog.json       # Tuy chon: catalog san pham
|-- init_db.py         # Tuy chon: khoi tao database
'-- [cac file khac]
```

2. Them skill vao `registry.json`:

```json
{
  "skills": {
    "ten-skill": {
      "version": "1.0.0",
      "description": "Skill nay lam gi",
      "requires_oauth": ["zalo_personal"],
      "setup_prompts": [
        {"key": "shop_name", "label": "Ten shop"},
        {"key": "phone", "label": "So dien thoai"}
      ]
    }
  }
}
```

3. Test:

```bash
make build
./clawkit install ten-skill --skip-oauth
cat ~/.openclaw/workspace/skills/ten-skill/SKILL.md
```

### Them OAuth Provider Moi

Tao file moi trong `oauth/` — tu dang ky qua `init()`:

```go
// oauth/provider_moi.go
package oauth

func init() { Register(&ProviderMoi{}) }

type ProviderMoi struct{}

func (p *ProviderMoi) Name() string    { return "provider_moi" }
func (p *ProviderMoi) Display() string { return "Provider Moi" }
func (p *ProviderMoi) Authenticate() (map[string]string, error) {
    // Implement OAuth flow
}
```

Khong can sua bat ky file nao khac.

### Quy Trinh Dev

```bash
git clone git@github.com:Rockship-Team/clawkit.git
cd clawkit
make build
make test
./clawkit install shop-hoa-zalo --skip-oauth
```

### Tao Release

```bash
make dist
gh release create v0.1.0 dist/* --title "v0.1.0" --notes "Initial release"
```

### Quy Uoc Commit

Theo [Conventional Commits](https://www.conventionalcommits.org/):

```
feat: them OAuth cho Gmail
fix: xu ly catalog.json rong
refactor: don gian hoa xu ly template
docs: cap nhat README
```

## Giay Phep

MIT
