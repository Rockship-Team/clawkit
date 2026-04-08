# clawkit

CLI tool cai dat va quan ly OpenClaw skills. Phat trien boi [Rockship](https://rockship.co).

clawkit xu ly toan bo quy trinh trien khai skill: tai skill template, chay OAuth authorization, thu thap cau hinh cua khach hang, va cai dat skill vao dung thu muc OpenClaw — tat ca trong 1 lenh duy nhat.

## Bat Dau Nhanh

```bash
# Build tu source
git clone git@github.com:Rockship-Team/clawkit.git
cd clawkit
CGO_ENABLED=0 go build -o clawkit .

# Xem danh sach skills
./clawkit list

# Cai dat skill
./clawkit install shop-hoa-zalo
```

## Yeu Cau

- **Go 1.22+** (de build tu source)
- **OpenClaw** da cai tren may khach hang ([huong dan cai dat](https://docs.openclaw.ai/installation))

clawkit tu dong detect OpenClaw va cai skill vao `~/.openclaw/workspace/skills/`.

## Cac Lenh

| Lenh | Mo ta |
|------|-------|
| `clawkit list` | Xem danh sach skills va trang thai cai dat |
| `clawkit install <skill>` | Cai skill kem OAuth va cau hinh |
| `clawkit update <skill>` | Cap nhat skill, giu nguyen token va config cu |
| `clawkit status` | Xem tat ca skills da cai |
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
  |-- 2. Copy skill template vao ~/.openclaw/workspace/skills/
  |-- 3. Chay OAuth (mo browser de KH xac thuc Zalo/Google)
  |-- 4. Thu thap thong tin KH (ten shop, email, v.v.)
  |-- 5. Xu ly SKILL.md.tmpl -> thay placeholder -> tao SKILL.md
  '-- 6. Luu config.json
```

### He Thong Template

Skills dung template de tai su dung. File `SKILL.md.tmpl` chua cac placeholder ma clawkit thay the khi install:

| Placeholder | Nguon |
|-------------|-------|
| `{shopName}` | KH nhap khi install |
| `{notifyEmailFrom}` | KH nhap |
| `{notifyEmailTo}` | KH nhap |
| `{notifyEmailAppPassword}` | KH nhap |
| `{catalogSection}` | Tu dong sinh tu `catalog.json` |
| `{baseDir}` | OpenClaw tu xu ly (clawkit khong thay the) |

### He Thong Catalog

Moi skill co the co `catalog.json` dinh nghia danh muc san pham va gia. clawkit sinh phan listing trong SKILL.md tu file nay:

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

## Build Da Nen Tang

```bash
chmod +x build.sh
./build.sh
```

Tao binary trong `dist/` cho:
- macOS ARM64 (Apple Silicon)
- macOS AMD64 (Intel)
- Linux AMD64
- Windows AMD64

## Cau Truc Du An

```
clawkit/
|-- main.go          # Entry point va routing lenh
|-- installer.go     # Lenh install, update, list, status
|-- oauth.go         # OAuth flow cho Zalo Personal/OA
|-- template.go      # Xu ly template SKILL.md + sinh catalog
|-- config.go        # Detect OpenClaw, doc/ghi skill config
|-- registry.go      # Load registry.json
|-- ui.go            # Hien thi terminal (mau sac, prompt)
|-- registry.json    # Danh sach skills
|-- build.sh         # Script build da nen tang
'-- skills/          # Cac skill template
    '-- shop-hoa-zalo/
        |-- SKILL.md.tmpl    # Template voi placeholder
        |-- catalog.json     # Danh muc san pham va gia
        |-- init_db.py       # Script khoi tao database
        '-- flowers/         # Anh san pham (theo danh muc/gia)
```

## Dong Gop (Contributing)

### Them Skill Moi

1. Tao thu muc trong `skills/`:

```
skills/ten-skill/
|-- SKILL.md.tmpl      # Template (dung placeholder cho gia tri rieng moi KH)
|-- catalog.json       # Tuy chon: catalog san pham/dich vu
'-- [cac file khac]    # Script, tai nguyen, v.v.
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

3. Neu skill can OAuth provider moi, them flow trong `oauth.go`:

```go
func oauthProviderMoi(skillDir string) error {
    // Implement OAuth flow
}
```

Va dang ky trong `runOAuthFlow()`.

4. Test:

```bash
CGO_ENABLED=0 go build -o clawkit .
./clawkit install ten-skill --skip-oauth
```

### Quy Trinh Dev

```bash
# Clone
git clone git@github.com:Rockship-Team/clawkit.git
cd clawkit

# Build
CGO_ENABLED=0 go build -o clawkit .

# Test voi skip-oauth
./clawkit install shop-hoa-zalo --skip-oauth

# Kiem tra SKILL.md da sinh
cat ~/.openclaw/workspace/skills/shop-hoa-zalo/SKILL.md

# Chay test
CGO_ENABLED=0 go test ./...
```

### Quy Uoc Commit

Theo [Conventional Commits](https://www.conventionalcommits.org/):

```
feat: them OAuth cho Gmail
fix: xu ly catalog.json rong
refactor: don gian hoa xu ly template
docs: cap nhat README
```

### Them OAuth Provider Moi

1. Them function OAuth trong `oauth.go`
2. Dang ky ten provider trong switch cua `runOAuthFlow()`
3. Dung `waitForOAuthCallback()` cho local callback server (dung chung)
4. Luu token vao `SkillConfig.Tokens` map

## Giay Phep

MIT
