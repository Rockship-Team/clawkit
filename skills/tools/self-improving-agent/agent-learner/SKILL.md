---
name: agent-learner
description: "Tu hoc va cai thien — luu quy trinh thanh cong, cap nhat khi phat hien thieu sot, tim kiem kinh nghiem cu."
metadata: { "openclaw": { "emoji": "🔄" } }
---

# Agent Learner (Hermes Self-Improvement)

Ban co kha nang TU HOC. Sau moi tac vu phuc tap, ban luu lai quy trinh thanh cong. Truoc moi tac vu moi, ban tim kinh nghiem cu. Khi phat hien thieu sot, ban cap nhat ngay. Day la vong phan hoi lien tuc de ban ngay cang gioi hon.

## KHI NAO LUU KY NANG MOI (save-skill)

Luu khi:

- Hoan thanh tac vu phuc tap (5+ buoc)
- Sua duoc loi kho
- Phat hien quy trinh hieu qua
- User xac nhan cach lam dung

KHONG luu khi:

- Cuoc hoi thoai don gian, khong co quy trinh
- Thong tin nhay cam (mat khau, token, CCCD)
- Noi dung chua duoc xac nhan

## KHI NAO CAP NHAT KY NANG (patch-skill)

Cap nhat khi:

- Ky nang da luu nhung thieu buoc → patch TRUOC khi ket thuc tac vu
- User sua lai quy trinh → cap nhat ngay
- Phat hien truong hop dac biet → bo sung

## KHI NAO TIM KINH NGHIEM CU (search)

Tim khi:

- Truoc khi bat dau tac vu moi → `vault-cli learn list` de kiem tra
- Gap loi → `vault-cli session search` tim tinh huong tuong tu
- User hoi "truoc day minh lam the nao" → tim trong session va learn

## NHAC NHO DINH KY (Nudge Protocol)

Moi ~10 luot hoi thoai, tu hoi:

1. "Co thong tin nao dang luu vao memory khong?"
2. "Minh da hoc duoc quy trinh nao dang ghi lai khong?"
3. "Co memory nao cu/sai can cap nhat khong?"

Neu co, thuc hien. Neu khong, tiep tuc binh thuong.

## LENH VAULT-CLI

### Quan ly ky nang da hoc

Luu ky nang moi:

```
vault-cli learn save-skill <name> <description> <procedure> [tags]
```

Cap nhat ky nang (tim va thay noi dung):

```
vault-cli learn patch-skill <name> <find_text> <replace_text>
```

Liet ke tat ca ky nang:

```
vault-cli learn list
```

Doc chi tiet ky nang:

```
vault-cli learn get <name>
```

### Tim kiem

Tim trong lich su phien:

```
vault-cli session search <query>
```

Tim trong toan bo vault:

```
vault-cli search <query>
```

### Bo nho

Luu thong tin:

```
vault-cli memory set MEMORY.md <info>
```

Cap nhat thong tin:

```
vault-cli memory replace MEMORY.md <old> <new>
```

## QUY TAC TUYET DOI

- Moi lenh `vault-cli` phai goi qua exec, TREN 1 DONG DUY NHAT.
- TUYET DOI KHONG dung pipe (`|`), redirect (`>`), heredoc, `&&`, `;`, subshell.
- Moi argument co khoang trang -> boc trong `"double quotes"`.
- Luon kiem tra `ok:true` trong ket qua truoc khi bao thanh cong.
- Chi luu quy trinh THUC SU huu ich, khong phai moi cuoc hoi thoai.
- KHONG BAO GIO luu du lieu nhay cam (mat khau, token, CCCD, so tai khoan).
- Khong luu nguyen cuoc hoi thoai — chung cat thanh cac buoc cu the.
- Khi memory day (>2200 ky tu), PHAI rut gon truoc khi them.
- Moi file ky nang PHAI co: name, description, quy trinh tung buoc, ngay tao.

## CAU TRUC FILE KY NANG

Moi ky nang duoc luu phai co:

- **name**: ten ngan gon, de nho (vi du: `payroll-monthly`)
- **description**: mo ta 1 cau ve muc dich
- **procedure**: cac buoc cu the, ro rang, co the lam theo
- **created**: ngay tao

## VI DU TUONG TAC

### Sau khi hoan thanh tinh luong

User vua hoan thanh tinh luong thang voi cac buoc: tai bang cham cong, doi chieu hop dong, tinh thue TNCN, xuat bang luong.

Hanh dong:

```
vault-cli learn save-skill "payroll-monthly" "Quy trinh tinh luong hang thang" "1. Tai bang cham cong tu HR\n2. Doi chieu hop dong lao dong\n3. Tinh thue TNCN theo bieu luy tien\n4. Tru bao hiem bat buoc\n5. Xuat bang luong va gui ke toan truong duyet" "payroll,finance,monthly"
```

### Truoc khi doi soat ngan hang

User: "Minh can doi soat so du ngan hang thang nay"

Hanh dong — kiem tra kinh nghiem cu:

```
vault-cli learn list
```

→ Tim thay ky nang `bank-reconciliation` da luu truoc do.

```
vault-cli learn get "bank-reconciliation"
```

→ Doc quy trinh va lam theo tung buoc.

### Phat hien thieu buoc

Dang lam theo ky nang `payroll-monthly` nhung phat hien thieu buoc kiem tra ngay nghi phep.

Hanh dong:

```
vault-cli learn patch-skill "payroll-monthly" "2. Doi chieu hop dong lao dong" "2. Doi chieu hop dong lao dong\n3. Kiem tra ngay nghi phep va ngay le trong thang"
```

→ Ky nang duoc cap nhat, lan sau se day du hon.
