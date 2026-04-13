"""
Crawl BĐS TP Hồ Chí Minh — hỗ trợ tất cả quận/huyện
Phân loại: căn hộ cũ vs căn hộ mới
Lưu dữ liệu JSON + ảnh vào listings/

Cách dùng:
  python3 crawl_bds.py                          # crawl toàn TP.HCM, 3-5 tỷ
  python3 crawl_bds.py --quan "Quận 1"          # chỉ Quận 1
  python3 crawl_bds.py --quan "Bình Thạnh"      # quận khác
  python3 crawl_bds.py --price-min 2 --price-max 8  # đổi ngân sách
  python3 crawl_bds.py --limit 10               # số lượng mỗi loại
  python3 crawl_bds.py --sources chotot,bds,dothi,mogi,homedy,alonhadat  # chọn nguồn

Nguồn:
  - Chợ Tốt Nhà (nhatot.com) — public REST API          [chotot]
  - BatDongSan.com.vn — curl_cffi (bypass Cloudflare)   [bds]
  - Dothi.net — curl_cffi (bypass Cloudflare)            [dothi]
  - Mogi.vn — web scraping                              [mogi]
  - Homedy.com — curl_cffi (bypass Cloudflare)           [homedy]
  - Alonhadat.com.vn — web scraping                     [alonhadat]

Yêu cầu: pip install curl-cffi beautifulsoup4 requests
"""
from __future__ import annotations

import argparse
import json
import os
import re
import sqlite3
import time
import hashlib
from datetime import datetime, timezone, timedelta
from pathlib import Path
from typing import Optional, List, Dict
from urllib.parse import urljoin

import requests
try:
    from curl_cffi import requests as cf_requests
    CF_AVAILABLE = True
except ImportError:
    CF_AVAILABLE = False

from bs4 import BeautifulSoup

# ──────────────────────────────────────────────
# CONFIG
# ──────────────────────────────────────────────
BASE_DIR = Path(__file__).parent
LISTINGS_DIR = BASE_DIR / "listings"
DB_PATH = BASE_DIR / "bds.db"
VN_TZ = timezone(timedelta(hours=7))

HEADERS = {
    "User-Agent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) "
                  "AppleWebKit/537.36 (KHTML, like Gecko) "
                  "Chrome/125.0.0.0 Safari/537.36",
    "Accept-Language": "vi-VN,vi;q=0.9,en;q=0.7",
    "Accept": "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8",
}

# Chợ Tốt area_v2 codes for HCM districts
# region_v2=13000 = TP.HCM
# area_v2=None means all of HCM
CHOTOT_AREA_MAP: Dict[str, Optional[str]] = {
    # Nội thành
    "quận 1": "13001",
    "quận 2": "13002",
    "quận 3": "13003",
    "quận 4": "13004",
    "quận 5": "13005",
    "quận 6": "13006",
    "quận 7": "13007",
    "quận 8": "13008",
    "quận 9": "13119",   # Nay là Thủ Đức
    "quận 10": "13010",
    "quận 11": "13011",
    "quận 12": "13012",
    "bình chánh": "13013",
    "bình tân": "13014",
    "bình thạnh": "13015",
    "cần giờ": "13016",
    "củ chi": "13017",
    "gò vấp": "13018",
    "hóc môn": "13019",
    "nhà bè": "13020",
    "phú nhuận": "13021",
    "tân bình": "13022",
    "tân phú": "13023",
    "thủ đức": "13119",  # TP Thủ Đức (gồm Q2+Q9+Q12 cũ)
    "tp thủ đức": "13119",
    "thành phố thủ đức": "13119",
}

# Alonhadat.com.vn slug mapping
ALONHADAT_SLUG_MAP: Dict[str, str] = {
    "quận 1": "quan-1",
    "quận 2": "quan-2",
    "quận 3": "quan-3",
    "quận 4": "quan-4",
    "quận 5": "quan-5",
    "quận 6": "quan-6",
    "quận 7": "quan-7",
    "quận 8": "quan-8",
    "quận 9": "quan-9",
    "quận 10": "quan-10",
    "quận 11": "quan-11",
    "quận 12": "quan-12",
    "bình chánh": "huyen-binh-chanh",
    "bình tân": "quan-binh-tan",
    "bình thạnh": "quan-binh-thanh",
    "cần giờ": "huyen-can-gio",
    "củ chi": "huyen-cu-chi",
    "gò vấp": "quan-go-vap",
    "hóc môn": "huyen-hoc-mon",
    "nhà bè": "huyen-nha-be",
    "phú nhuận": "quan-phu-nhuan",
    "tân bình": "quan-tan-binh",
    "tân phú": "quan-tan-phu",
    "thủ đức": "tp-thu-duc",
    "tp thủ đức": "tp-thu-duc",
    "thành phố thủ đức": "tp-thu-duc",
    "quận 9": "quan-9",
}

# BatDongSan.com.vn district slug (suffix: -tp-hcm)
BDS_SLUG_MAP: Dict[str, str] = {
    "quận 1": "quan-1",
    "quận 2": "quan-2",
    "quận 3": "quan-3",
    "quận 4": "quan-4",
    "quận 5": "quan-5",
    "quận 6": "quan-6",
    "quận 7": "quan-7",
    "quận 8": "quan-8",
    "quận 9": "quan-9",
    "quận 10": "quan-10",
    "quận 11": "quan-11",
    "quận 12": "quan-12",
    "bình chánh": "huyen-binh-chanh",
    "bình tân": "quan-binh-tan",
    "bình thạnh": "quan-binh-thanh",
    "cần giờ": "huyen-can-gio",
    "củ chi": "huyen-cu-chi",
    "gò vấp": "quan-go-vap",
    "hóc môn": "huyen-hoc-mon",
    "nhà bè": "huyen-nha-be",
    "phú nhuận": "quan-phu-nhuan",
    "tân bình": "quan-tan-binh",
    "tân phú": "quan-tan-phu",
    "thủ đức": "thanh-pho-thu-duc",
    "tp thủ đức": "thanh-pho-thu-duc",
    "thành phố thủ đức": "thanh-pho-thu-duc",
}

# Mogi.vn district slug
MOGI_SLUG_MAP: Dict[str, str] = {
    "quận 1": "quan-1-tp-hcm",
    "quận 2": "quan-2-tp-hcm",
    "quận 3": "quan-3-tp-hcm",
    "quận 4": "quan-4-tp-hcm",
    "quận 5": "quan-5-tp-hcm",
    "quận 6": "quan-6-tp-hcm",
    "quận 7": "quan-7-tp-hcm",
    "quận 8": "quan-8-tp-hcm",
    "quận 9": "quan-9-tp-hcm",
    "quận 10": "quan-10-tp-hcm",
    "quận 11": "quan-11-tp-hcm",
    "quận 12": "quan-12-tp-hcm",
    "bình chánh": "huyen-binh-chanh-tp-hcm",
    "bình tân": "quan-binh-tan-tp-hcm",
    "bình thạnh": "quan-binh-thanh-tp-hcm",
    "cần giờ": "huyen-can-gio-tp-hcm",
    "củ chi": "huyen-cu-chi-tp-hcm",
    "gò vấp": "quan-go-vap-tp-hcm",
    "hóc môn": "huyen-hoc-mon-tp-hcm",
    "nhà bè": "huyen-nha-be-tp-hcm",
    "phú nhuận": "quan-phu-nhuan-tp-hcm",
    "tân bình": "quan-tan-binh-tp-hcm",
    "tân phú": "quan-tan-phu-tp-hcm",
    "thủ đức": "thanh-pho-thu-duc-tp-hcm",
    "tp thủ đức": "thanh-pho-thu-duc-tp-hcm",
    "thành phố thủ đức": "thanh-pho-thu-duc-tp-hcm",
}


def log(msg: str):
    ts = datetime.now(VN_TZ).strftime("%H:%M:%S")
    print(f"[{ts}] {msg}")


def safe_get(url: str, retries: int = 3, delay: float = 1.0,
             headers: Optional[dict] = None) -> Optional[requests.Response]:
    hdrs = headers or HEADERS
    for attempt in range(retries):
        try:
            time.sleep(delay)
            resp = requests.get(url, headers=hdrs, timeout=20)
            if resp.status_code == 200:
                return resp
            log(f"  ⚠ HTTP {resp.status_code} — {url[:120]}")
        except requests.RequestException as e:
            log(f"  ⚠ Request error (attempt {attempt+1}): {e}")
    return None


def cf_get(url: str, retries: int = 3, delay: float = 1.0) -> Optional[object]:
    """Fetch with curl_cffi Chrome impersonation (bypasses Cloudflare).
    Falls back to regular requests if curl_cffi is not installed."""
    for attempt in range(retries):
        try:
            time.sleep(delay)
            if CF_AVAILABLE:
                resp = cf_requests.get(url, impersonate="chrome124", timeout=20)
            else:
                resp = requests.get(url, headers=HEADERS, timeout=20)
            if resp.status_code == 200:
                return resp
            log(f"  ⚠ HTTP {resp.status_code} — {url[:120]}")
        except Exception as e:
            log(f"  ⚠ cf_get error (attempt {attempt+1}): {e}")
    return None


def download_image(url: str, filepath: Path) -> bool:
    try:
        if not url.startswith("http"):
            return False
        resp = safe_get(url, retries=2, delay=0.3)
        if resp and resp.content and len(resp.content) > 1000:
            filepath.write_bytes(resp.content)
            return True
    except Exception:
        pass
    return False


# ══════════════════════════════════════════════
# SOURCE 1: Chợ Tốt Nhà — Public API
# ══════════════════════════════════════════════
CHOTOT_API = "https://gateway.chotot.com/v1/public/ad-listing"
CHOTOT_DETAIL_API = "https://gateway.chotot.com/v2/public/ad-listing"


def classify_ad_type(ad: dict) -> str:
    """Classify ad as 'can_ho_cu' or 'can_ho_moi' based on content."""
    subject = ad.get("subject", "").lower()
    body = ad.get("body", "").lower()
    combined = subject + " " + body

    new_keywords = [
        "cđt", "chủ đầu tư", "mở bán", "booking", "giỏ hàng",
        "từ cđt", "ưu đãi", "chiết khấu", "thanh toán chỉ",
        "ra mắt", "giai đoạn", "đợt mở bán", "nhận booking",
        "suất nội bộ", "giá gốc", "rổ hàng", "hàng chủ đầu tư"
    ]
    old_keywords = [
        "chính chủ", "cần bán gấp", "đã ra sổ", "sổ hồng", "sổ đỏ",
        "bán lại", "bán cắt lỗ", "định cư", "đi nước ngoài",
        "sang nhượng", "view đẹp", "full nội thất", "đang ở",
        "nhà đang cho thuê"
    ]

    new_score = sum(1 for kw in new_keywords if kw in combined)
    old_score = sum(1 for kw in old_keywords if kw in combined)

    if new_score > old_score:
        return "can_ho_moi"
    elif old_score > new_score:
        return "can_ho_cu"
    else:
        if ad.get("pty_project_name"):
            return "can_ho_moi"
        return "can_ho_cu"


def crawl_chotot_all(area_v2: Optional[str], price_min_ty: float,
                     price_max_ty: float, max_per_cat: int,
                     location_label: str) -> Dict[str, List[dict]]:
    """Crawl all apartments from Chợ Tốt and classify into old/new."""
    log(f"\n  📡 Chợ Tốt Nhà — {location_label}")

    api_headers = {
        "User-Agent": HEADERS["User-Agent"],
        "Accept": "application/json",
    }

    all_ads = []
    for cg in ["1010", "1020"]:
        params: dict = {
            "region_v2": "13000",  # TP.HCM
            "cg": cg,
            "limit": 50,
            "w": 1,
            "page": 1,
        }
        if area_v2:
            params["area_v2"] = area_v2

        try:
            time.sleep(1)
            resp = requests.get(CHOTOT_API, params=params, headers=api_headers, timeout=15)
            if resp.status_code == 200:
                data = resp.json()
                ads = data.get("ads", [])
                log(f"    cg={cg}: {len(ads)} ads (total={data.get('total', 0)})")
                all_ads.extend(ads)
        except Exception as e:
            log(f"    ⚠ API error cg={cg}: {e}")

    # Filter: selling only + price range (no location keyword filter)
    filtered = []
    for ad in all_ads:
        if ad.get("type", "") != "s":
            continue
        price = ad.get("price", 0)
        if price < price_min_ty * 1_000_000_000 or price > price_max_ty * 1_000_000_000:
            continue
        filtered.append(ad)

    log(f"    ✅ Sell + giá {price_min_ty}-{price_max_ty} tỷ: {len(filtered)} ads")

    # Classify and split
    cu_ads: List[dict] = []
    moi_ads: List[dict] = []
    seen_ids: set = set()

    for ad in filtered:
        lid = ad.get("list_id")
        if lid in seen_ids:
            continue
        seen_ids.add(lid)

        cat = classify_ad_type(ad)
        if cat == "can_ho_cu" and len(cu_ads) < max_per_cat:
            cu_ads.append(ad)
        elif cat == "can_ho_moi" and len(moi_ads) < max_per_cat:
            moi_ads.append(ad)

    # If one category is short, fill from remaining
    if len(cu_ads) < max_per_cat or len(moi_ads) < max_per_cat:
        for ad in filtered:
            lid = ad.get("list_id")
            if lid in seen_ids:
                continue
            seen_ids.add(lid)
            if len(cu_ads) < max_per_cat:
                cu_ads.append(ad)
            elif len(moi_ads) < max_per_cat:
                moi_ads.append(ad)

    log(f"    📊 Phân loại: {len(cu_ads)} cũ + {len(moi_ads)} mới")

    results: Dict[str, List[dict]] = {"can_ho_cu": [], "can_ho_moi": []}
    for ad in cu_ads:
        listing = parse_chotot_ad(ad, "can_ho_cu")
        if listing:
            results["can_ho_cu"].append(listing)
    for ad in moi_ads:
        listing = parse_chotot_ad(ad, "can_ho_moi")
        if listing:
            results["can_ho_moi"].append(listing)

    return results


def parse_chotot_ad(ad: dict, category: str) -> Optional[dict]:
    """Parse Chợ Tốt ad into our listing format."""
    list_id = str(ad.get("list_id", ""))
    subject = ad.get("subject", "Không có tiêu đề")
    price = ad.get("price", 0)
    price_str = ad.get("price_string", format_price(price))
    size = ad.get("size", 0)
    rooms = ad.get("rooms", 0)
    toilets = ad.get("toilets", 0)
    ward_name = ad.get("ward_name", "")
    area_name = ad.get("area_name", "")
    region_name = ad.get("region_name", "")
    body = ad.get("body", "")
    street = ad.get("street_name", "")
    street_number = ad.get("street_number", "")

    # Direction mapping
    direction = ""
    direction_code = ad.get("direction")
    direction_map = {
        1: "Đông", 2: "Tây", 3: "Nam", 4: "Bắc",
        5: "Đông Bắc", 6: "Đông Nam", 7: "Tây Bắc", 8: "Tây Nam"
    }
    if isinstance(direction_code, int):
        direction = direction_map.get(direction_code, "")

    # Image URLs
    image_urls = []
    for img in ad.get("images", []):
        if isinstance(img, str):
            if img.startswith("http"):
                image_urls.append(img)
            else:
                image_urls.append(f"https://cdn.chotot.com/unsafe/640x480/{img}")

    # Build address
    address_parts = [street_number, street, ward_name, area_name, region_name]
    address = ", ".join(filter(None, address_parts))

    location = ", ".join(filter(None, [ward_name, area_name, region_name]))
    if not location:
        location = region_name or "TP Hồ Chí Minh"

    result = {
        "source": "nhatot.com",
        "source_id": list_id,
        "title": subject,
        "url": f"https://www.nhatot.com/{list_id}.htm",
        "loai": category,
        "price": price_str,
        "price_raw": price,
        "area": f"{size} m²" if size else "",
        "bedrooms": rooms,
        "bathrooms": toilets,
        "direction": direction,
        "address": address,
        "location": location,
        "images": image_urls,
        "description": body[:2000] if body else "",
        "legal_status": ad.get("property_legal_document", ""),
        "project_name": ad.get("pty_project_name", ""),
    }

    if category == "can_ho_cu":
        result["bank_appraisal"] = extract_bank_appraisal(body, price)
    else:
        result.update({
            "developer": "",
            "amenities": "",
            "location_detail": address,
            "current_floors": "",
            "payment_schedule": "",
            "deposit_percent": "",
            "bank_support_percent": "",
            "interest_rate": "",
            "supporting_bank": "",
            "handover_date": "",
        })
        enrich_nha_moi_from_text(result, body)

    return result


def enrich_from_chotot_detail(listing: dict):
    """Fetch full detail from Chợ Tốt API v2."""
    list_id = listing.get("source_id", "")
    if not list_id:
        return

    try:
        api_url = f"{CHOTOT_DETAIL_API}/{list_id}"
        time.sleep(0.5)
        resp = requests.get(api_url, headers={
            "User-Agent": HEADERS["User-Agent"],
            "Accept": "application/json",
        }, timeout=10)

        if resp.status_code != 200:
            return

        data = resp.json()
        ad = data.get("ad", data)
        body = ad.get("body", "")

        if body and len(body) > len(listing.get("description", "")):
            listing["description"] = body[:2000]

        for img in ad.get("images", []):
            url = ""
            if isinstance(img, str):
                url = f"https://cdn.chotot.com/unsafe/640x480/{img}" if not img.startswith("http") else img
            if url and url not in listing["images"]:
                listing["images"].append(url)

        if listing["loai"] == "can_ho_moi":
            enrich_nha_moi_from_text(listing, body)
        elif listing["loai"] == "can_ho_cu":
            listing["bank_appraisal"] = extract_bank_appraisal(body, listing.get("price_raw", 0))

    except Exception as e:
        log(f"    ⚠ Detail API error: {e}")


# ══════════════════════════════════════════════
# SOURCE 2: Alonhadat.com.vn — Web Scraping
# ══════════════════════════════════════════════

def crawl_alonhadat(category: str, district_slug: str,
                    price_min_ty: float, price_max_ty: float,
                    max_per_cat: int) -> List[dict]:
    """Crawl from alonhadat.com.vn."""
    label = "CĂN HỘ CŨ" if category == "can_ho_cu" else "CĂN HỘ MỚI"
    log(f"\n  🌐 Alonhadat.com.vn — {label} ({district_slug})")

    price_slug = f"gia-tu-{int(price_min_ty)}-ty-den-{int(price_max_ty)}-ty"

    search_urls = [
        f"https://alonhadat.com.vn/nha-dat/can-ban/can-ho-chung-cu/2/{district_slug}/{price_slug}.html",
        f"https://alonhadat.com.vn/nha-dat/can-ban/can-ho-chung-cu/2/{district_slug}.html",
        # Fallback: all HCM
        f"https://alonhadat.com.vn/nha-dat/can-ban/can-ho-chung-cu/1/tp-ho-chi-minh/{price_slug}.html",
    ]

    all_cards = []
    for url in search_urls:
        resp = safe_get(url)
        if not resp:
            continue
        soup = BeautifulSoup(resp.text, "html.parser")
        cards = soup.select("article.property-item")
        if cards:
            log(f"    ✅ {len(cards)} kết quả từ {url.split('/')[-1]}")
            all_cards = cards
            break

    if not all_cards:
        log("    ⚠ Không tìm thấy kết quả trên alonhadat")
        return []

    results = []
    for card in all_cards:
        try:
            link_el = card.select_one("a[href]")
            if not link_el:
                continue
            href = link_el.get("href", "")
            title = link_el.get_text(strip=True)
            if not href or not title or len(title) < 5:
                continue

            url = urljoin("https://alonhadat.com.vn", href)

            price_el = card.select_one("span.price")
            price_text = ""
            if price_el:
                price_text = re.sub(r"^Giá:\s*", "", price_el.get_text(strip=True))

            if price_text:
                price_num = extract_price_ty(price_text)
                if price_num and (price_num < price_min_ty or price_num > price_max_ty):
                    continue

            area_text = ""
            for span in card.select(".property-details span"):
                t = span.get_text(strip=True)
                if "m²" in t:
                    area_text = t
                    break

            bedrooms = 0
            for span in card.select(".property-details span"):
                t = span.get_text(strip=True)
                if "phòng ngủ" in t:
                    m = re.search(r"(\d+)", t)
                    if m:
                        bedrooms = int(m.group(1))
                    break

            desc_el = card.select_one(".property-content, p")
            desc = desc_el.get_text(strip=True)[:300] if desc_el else ""

            img_el = card.select_one("img")
            thumb = ""
            if img_el:
                thumb = img_el.get("data-src", "") or img_el.get("src", "")
                if thumb and not thumb.startswith("http"):
                    thumb = urljoin("https://alonhadat.com.vn", thumb)

            title_lower = title.lower() + " " + desc.lower()
            is_new_project = any(kw in title_lower for kw in [
                "dự án", "du an", "mở bán", "mo ban", "booking",
                "chủ đầu tư", "cdt", "giỏ hàng", "block", "tòa"
            ])

            if category == "can_ho_cu" and is_new_project:
                continue
            if category == "can_ho_moi" and not is_new_project:
                continue

            listing = {
                "source": "alonhadat.com.vn",
                "source_id": hashlib.md5(url.encode()).hexdigest()[:8],
                "title": title[:200],
                "url": url,
                "loai": category,
                "price": price_text,
                "area": area_text,
                "bedrooms": bedrooms,
                "bathrooms": 0,
                "direction": "",
                "address": "",
                "location": "",  # Will be enriched from detail page
                "images": [thumb] if thumb else [],
                "description": desc,
                "legal_status": "",
            }

            if category == "can_ho_cu":
                listing["bank_appraisal"] = "Chưa có thông tin — cần liên hệ ngân hàng"
            else:
                listing.update({
                    "developer": "", "amenities": "", "location_detail": "",
                    "current_floors": "", "payment_schedule": "",
                    "deposit_percent": "", "bank_support_percent": "",
                    "interest_rate": "", "supporting_bank": "",
                    "handover_date": "",
                })

            results.append(listing)
            if len(results) >= max_per_cat:
                break

        except Exception as e:
            log(f"    ⚠ Parse error: {e}")

    return results


def enrich_from_alonhadat_detail(listing: dict):
    """Fetch detail page from alonhadat for more info."""
    url = listing.get("url", "")
    if not url:
        return

    resp = safe_get(url)
    if not resp:
        return

    soup = BeautifulSoup(resp.text, "html.parser")
    full_text = soup.get_text()

    for img in soup.select("img"):
        src = img.get("data-src", "") or img.get("src", "")
        if src and not src.endswith(".svg") and "logo" not in src.lower():
            full_url = urljoin("https://alonhadat.com.vn", src)
            if full_url not in listing["images"] and "thumbnails" not in full_url:
                listing["images"].append(full_url)

    desc_el = (soup.select_one("div.detail") or
               soup.select_one("div.content") or
               soup.select_one("div[class*='description']"))
    if desc_el:
        listing["description"] = desc_el.get_text(strip=True)[:2000]

    # Extract location from breadcrumb or address
    location_el = soup.select_one("span.address, div.address, .location")
    if location_el:
        listing["location"] = location_el.get_text(strip=True)[:200]

    for row in soup.select("tr, div[class*='info'] > div"):
        text = row.get_text(strip=True).lower()
        cells = row.select("td, span")
        value = cells[-1].get_text(strip=True) if len(cells) > 1 else ""

        if "hướng" in text:
            listing["direction"] = value
        elif "diện tích" in text and not listing.get("area"):
            listing["area"] = value
        elif "phòng ngủ" in text:
            try:
                listing["bedrooms"] = int(re.search(r"\d+", value).group())
            except Exception:
                pass
        elif "pháp lý" in text or "sổ" in text:
            listing["legal_status"] = value

    if listing.get("loai") == "can_ho_cu":
        if not listing.get("bank_appraisal") or listing["bank_appraisal"] == "Chưa có thông tin":
            listing["bank_appraisal"] = extract_bank_appraisal(full_text, 0)
    else:
        enrich_nha_moi_from_text(listing, full_text)


# ══════════════════════════════════════════════
# SOURCE 3: BatDongSan.com.vn — curl_cffi
# (Dùng Chrome impersonation để bypass Cloudflare)
# ══════════════════════════════════════════════

def crawl_batdongsan(category: str, district_key: str,
                     price_min_ty: float, price_max_ty: float,
                     max_per_cat: int) -> List[dict]:
    """Crawl batdongsan.com.vn dùng curl_cffi để bypass Cloudflare."""
    label = "CĂN HỘ CŨ" if category == "can_ho_cu" else "CĂN HỘ MỚI"
    bds_slug = BDS_SLUG_MAP.get(district_key, "")
    log(f"\n  🏢 BatDongSan.com.vn — {label} ({bds_slug or 'tp-hcm'})")

    if not CF_AVAILABLE:
        log("    ⚠ curl_cffi chưa cài — bỏ qua BatDongSan (pip install curl-cffi)")
        return []

    if bds_slug:
        search_urls = [
            f"https://batdongsan.com.vn/ban-can-ho-chung-cu-{bds_slug}-tp-hcm"
            f"?gia_tu={int(price_min_ty)}&gia_den={int(price_max_ty)}",
            f"https://batdongsan.com.vn/ban-can-ho-chung-cu-{bds_slug}-tp-hcm",
        ]
    else:
        search_urls = [
            f"https://batdongsan.com.vn/ban-can-ho-chung-cu-tp-hcm"
            f"?gia_tu={int(price_min_ty)}&gia_den={int(price_max_ty)}",
            "https://batdongsan.com.vn/ban-can-ho-chung-cu-tp-hcm",
        ]

    cards = []
    for url in search_urls:
        resp = cf_get(url)
        if not resp:
            continue
        soup = BeautifulSoup(resp.text, "html.parser")
        # Confirmed selector: div.re__card-full (20 cards per page)
        cards = soup.select("div.re__card-full")
        if cards:
            log(f"    ✅ {len(cards)} kết quả từ {url.split('?')[0].split('/')[-1]}")
            break

    if not cards:
        log("    ⚠ Không tìm thấy kết quả trên BatDongSan")
        return []

    results = []
    for card in cards:
        try:
            # Title & link
            a = card.select_one("a.js__product-link-for-product, h3 a, h2 a")
            if not a:
                continue
            href = a.get("href", "")
            title = a.get("title", "") or a.get_text(strip=True)
            if not href or not title or len(title) < 5:
                continue

            # Strip leading digits (VIP badge artifacts like "9Giảm mạnh...")
            title = re.sub(r"^\d+", "", title).strip()
            url_full = urljoin("https://batdongsan.com.vn", href)

            # Price
            price_el = card.select_one("span.re__card-config-price")
            price_text = price_el.get_text(strip=True) if price_el else ""

            if price_text:
                price_num = extract_price_ty(price_text)
                if price_num and (price_num < price_min_ty or price_num > price_max_ty):
                    continue

            # Area
            area_el = card.select_one("span.re__card-config-area")
            area_text = area_el.get_text(strip=True) if area_el else ""

            # Bedrooms
            bed_el = card.select_one("span.re__card-config-bedroom")
            bedrooms = 0
            if bed_el:
                m = re.search(r"(\d+)", bed_el.get_text())
                if m:
                    bedrooms = int(m.group(1))

            # Location — "·Q. Bình Thạnh (...)" → strip leading dot
            loc_el = card.select_one("div.re__card-location span, span.re__card-location")
            location = re.sub(r"^[·•\s]+", "", loc_el.get_text(strip=True)) if loc_el else ""

            # Description
            desc_el = card.select_one("div.re__card-description")
            desc = desc_el.get_text(strip=True)[:300] if desc_el else ""

            # Thumbnail
            img_el = card.select_one("img")
            thumb = ""
            if img_el:
                thumb = img_el.get("data-src") or img_el.get("src") or ""

            # Classify
            title_lower = title.lower() + " " + desc.lower()
            is_new_project = any(kw in title_lower for kw in [
                "dự án", "mở bán", "booking", "chủ đầu tư", "cđt",
                "giỏ hàng", "block", "tòa", "rổ hàng", "suất nội bộ"
            ])
            if category == "can_ho_cu" and is_new_project:
                continue
            if category == "can_ho_moi" and not is_new_project:
                continue

            listing = {
                "source": "batdongsan.com.vn",
                "source_id": hashlib.md5(url_full.encode()).hexdigest()[:8],
                "title": title[:200],
                "url": url_full,
                "loai": category,
                "price": price_text,
                "area": area_text,
                "bedrooms": bedrooms,
                "bathrooms": 0,
                "direction": "",
                "address": location,
                "location": location,
                "images": [thumb] if thumb else [],
                "description": desc,
                "legal_status": "",
            }

            if category == "can_ho_cu":
                listing["bank_appraisal"] = "Chưa có thông tin — cần liên hệ ngân hàng"
            else:
                listing.update({
                    "developer": "", "amenities": "", "location_detail": location,
                    "current_floors": "", "payment_schedule": "",
                    "deposit_percent": "", "bank_support_percent": "",
                    "interest_rate": "", "supporting_bank": "",
                    "handover_date": "",
                })

            results.append(listing)
            if len(results) >= max_per_cat:
                break

        except Exception as e:
            log(f"    ⚠ Parse error: {e}")

    log(f"    📊 BatDongSan: {len(results)} {label}")
    return results


def enrich_from_batdongsan_detail(listing: dict):
    """Fetch detail page từ batdongsan.com.vn dùng curl_cffi."""
    url = listing.get("url", "")
    if not url or not CF_AVAILABLE:
        return

    resp = cf_get(url)
    if not resp:
        return

    soup = BeautifulSoup(resp.text, "html.parser")
    full_text = soup.get_text()

    # Images from CDN
    for img in soup.select("img[src*='batdongsan'], img[data-src*='batdongsan'], img[src*='file4']"):
        src = img.get("data-src") or img.get("src") or ""
        if src.startswith("http") and src not in listing["images"]:
            listing["images"].append(src)

    # Description
    desc_el = (soup.select_one("div.re__section-description--content") or
               soup.select_one("div[class*='re__pr-description']") or
               soup.select_one("div[class*='description']"))
    if desc_el:
        listing["description"] = desc_el.get_text(strip=True)[:2000]

    # Structured specs table
    for row in soup.select("div.re__pr-specs-content-item"):
        lbl_el = row.select_one("span.re__pr-specs-content-item-title")
        val_el = row.select_one("span.re__pr-specs-content-item-value")
        if not lbl_el or not val_el:
            continue
        lbl = lbl_el.get_text(strip=True).lower()
        val = val_el.get_text(strip=True)

        if "diện tích" in lbl and not listing.get("area"):
            listing["area"] = val
        elif "hướng" in lbl and not listing.get("direction"):
            listing["direction"] = val
        elif "phòng ngủ" in lbl and not listing.get("bedrooms"):
            m = re.search(r"\d+", val)
            if m:
                listing["bedrooms"] = int(m.group())
        elif "pháp lý" in lbl or "giấy tờ" in lbl:
            listing["legal_status"] = val

    if listing.get("loai") == "can_ho_cu":
        if not listing.get("bank_appraisal") or "Chưa có" in listing.get("bank_appraisal", ""):
            listing["bank_appraisal"] = extract_bank_appraisal(full_text, 0)
    else:
        enrich_nha_moi_from_text(listing, full_text)


# ══════════════════════════════════════════════
# SOURCE 4: Mogi.vn — Web Scraping
# ══════════════════════════════════════════════

def crawl_mogi(category: str, district_key: str,
               price_min_ty: float, price_max_ty: float,
               max_per_cat: int) -> List[dict]:
    """Crawl from mogi.vn."""
    label = "CĂN HỘ CŨ" if category == "can_ho_cu" else "CĂN HỘ MỚI"
    mogi_slug = MOGI_SLUG_MAP.get(district_key, "tp-hcm")
    log(f"\n  🟠 Mogi.vn — {label} ({mogi_slug})")

    # Correct Mogi URL: https://mogi.vn/mua-can-ho-chung-cu-{slug}
    search_urls = [
        f"https://mogi.vn/mua-can-ho-chung-cu-{mogi_slug}",
        "https://mogi.vn/mua-can-ho-chung-cu-tp-hcm",
    ]

    mogi_headers = {**HEADERS, "Referer": "https://mogi.vn/"}
    cards = []

    for url in search_urls:
        resp = safe_get(url, headers=mogi_headers)
        if not resp:
            continue
        soup = BeautifulSoup(resp.text, "html.parser")
        # ul.props > li is the confirmed card container (15 items per page)
        cards = (soup.select("ul.props > li") or
                 soup.select("ul.props li"))
        if cards:
            log(f"    ✅ {len(cards)} kết quả từ {url.split('/')[-1]}")
            break

    if not cards:
        log("    ⚠ Không tìm thấy kết quả trên Mogi")
        return []

    results = []
    for card in cards:
        try:
            # Mogi structure (confirmed): ul.props > li
            #   div.prop-info > a (title + href)
            #   div.prop-addr (location)
            #   div.prop-extra > b.price (price)
            #   div.prop-img > img (thumbnail)
            info_el = card.select_one("div.prop-info")
            link_el = info_el.select_one("a[href]") if info_el else card.select_one("a[href]")
            if not link_el:
                continue
            href = link_el.get("href", "")
            title = link_el.get_text(strip=True)
            if not href or not title or len(title) < 5:
                continue

            url_full = urljoin("https://mogi.vn", href)

            # Price — div.prop-extra b.price
            price_el = card.select_one("b.price, div.prop-extra [class*='price'], [class*='price']")
            price_text = price_el.get_text(strip=True) if price_el else ""

            if price_text:
                price_num = extract_price_ty(price_text)
                if price_num and (price_num < price_min_ty or price_num > price_max_ty):
                    continue

            # Area — look inside prop-extra or full card text
            area_text = ""
            m = re.search(r"([\d.,]+\s*m[²2])", card.get_text())
            if m:
                area_text = m.group(0)

            # Location — div.prop-addr
            loc_el = card.select_one("div.prop-addr, [class*='addr'], [class*='location']")
            location = loc_el.get_text(strip=True) if loc_el else ""

            # Description — not in card, will be enriched from detail page
            desc = ""

            # Thumbnail — div.prop-img img
            img_el = card.select_one("div.prop-img img, img")
            thumb = ""
            if img_el:
                thumb = img_el.get("data-src") or img_el.get("src") or ""
                if thumb and not thumb.startswith("http"):
                    thumb = urljoin("https://mogi.vn", thumb)

            # Classify
            title_lower = title.lower() + " " + desc.lower()
            is_new_project = any(kw in title_lower for kw in [
                "dự án", "mở bán", "booking", "chủ đầu tư", "cđt",
                "giỏ hàng", "block", "tòa", "rổ hàng"
            ])
            if category == "can_ho_cu" and is_new_project:
                continue
            if category == "can_ho_moi" and not is_new_project:
                continue

            listing = {
                "source": "mogi.vn",
                "source_id": hashlib.md5(url_full.encode()).hexdigest()[:8],
                "title": title[:200],
                "url": url_full,
                "loai": category,
                "price": price_text,
                "area": area_text,
                "bedrooms": 0,
                "bathrooms": 0,
                "direction": "",
                "address": location,
                "location": location,
                "images": [thumb] if thumb else [],
                "description": desc,
                "legal_status": "",
            }

            if category == "can_ho_cu":
                listing["bank_appraisal"] = "Chưa có thông tin — cần liên hệ ngân hàng"
            else:
                listing.update({
                    "developer": "", "amenities": "", "location_detail": location,
                    "current_floors": "", "payment_schedule": "",
                    "deposit_percent": "", "bank_support_percent": "",
                    "interest_rate": "", "supporting_bank": "",
                    "handover_date": "",
                })

            results.append(listing)
            if len(results) >= max_per_cat:
                break

        except Exception as e:
            log(f"    ⚠ Parse error: {e}")

    log(f"    📊 Mogi: {len(results)} {label}")
    return results


def enrich_from_mogi_detail(listing: dict):
    """Fetch detail page from mogi.vn."""
    url = listing.get("url", "")
    if not url:
        return

    resp = safe_get(url, headers={**HEADERS, "Referer": "https://mogi.vn/"})
    if not resp:
        return

    soup = BeautifulSoup(resp.text, "html.parser")
    full_text = soup.get_text()

    for img in soup.select("img"):
        src = img.get("data-src") or img.get("src") or ""
        if src.startswith("http") and "logo" not in src.lower() and src not in listing["images"]:
            listing["images"].append(src)

    desc_el = soup.select_one("div.description, div[class*='desc'], div.content")
    if desc_el:
        listing["description"] = desc_el.get_text(strip=True)[:2000]

    for row in soup.select("tr, div.info-row, li.info-item"):
        text = row.get_text(" ", strip=True).lower()
        if "diện tích" in text and not listing.get("area"):
            m = re.search(r"([\d.,]+\s*m[²2])", row.get_text())
            if m:
                listing["area"] = m.group(1)
        elif "hướng" in text and not listing.get("direction"):
            cells = row.select("td, span")
            if len(cells) >= 2:
                listing["direction"] = cells[-1].get_text(strip=True)
        elif "phòng ngủ" in text:
            m = re.search(r"(\d+)", text)
            if m:
                listing["bedrooms"] = int(m.group(1))
        elif ("pháp lý" in text or "sổ" in text) and not listing.get("legal_status"):
            cells = row.select("td, span")
            if len(cells) >= 2:
                listing["legal_status"] = cells[-1].get_text(strip=True)

    if listing.get("loai") == "can_ho_cu":
        if not listing.get("bank_appraisal") or "Chưa có" in listing.get("bank_appraisal", ""):
            listing["bank_appraisal"] = extract_bank_appraisal(full_text, 0)
    else:
        enrich_nha_moi_from_text(listing, full_text)


# ══════════════════════════════════════════════
# HELPER FUNCTIONS
# ══════════════════════════════════════════════

def format_price(price_vnd: int) -> str:
    if price_vnd >= 1_000_000_000:
        ty = price_vnd / 1_000_000_000
        return f"{ty:.1f} tỷ" if ty != int(ty) else f"{int(ty)} tỷ"
    elif price_vnd >= 1_000_000:
        return f"{price_vnd / 1_000_000:.0f} triệu"
    return str(price_vnd)


def extract_price_ty(text: str) -> Optional[float]:
    m = re.search(r"([\d.,]+)\s*tỷ", text, re.IGNORECASE)
    if m:
        return float(m.group(1).replace(",", "."))
    return None


def extract_bank_appraisal(body: str, price_raw: int) -> str:
    if body:
        for pat in [
            r"ngân\s*hàng\s*định\s*giá[:\s]*([^\n,.]{5,100})",
            r"định\s*giá\s*(?:ngân\s*hàng)?[:\s]*([^\n,.]{5,100})",
            r"NH\s*(?:cho\s*vay|định\s*giá)[:\s]*([^\n,.]{5,100})",
        ]:
            m = re.search(pat, body, re.IGNORECASE)
            if m:
                return m.group(1).strip()

    if price_raw > 0:
        low = format_price(int(price_raw * 0.7))
        high = format_price(int(price_raw * 0.8))
        return f"Ước tính {low} - {high} (70-80% giá thị trường, cần liên hệ NH)"
    return "Chưa có thông tin — cần liên hệ ngân hàng để định giá"


def enrich_nha_moi_from_text(listing: dict, text: str):
    if not text:
        return

    def find(field_name, patterns, default="Chưa có thông tin"):
        current = listing.get(field_name, "")
        if current and current != "Chưa có thông tin":
            return
        for pat in patterns:
            m = re.search(pat, text, re.IGNORECASE)
            if m:
                listing[field_name] = m.group(1).strip()[:200]
                return
        listing[field_name] = default

    find("developer", [
        r"chủ\s*đầu\s*tư[:\s\-]*([^\n,\.]{3,100})",
        r"CĐT[:\s\-]*([^\n,\.]{3,100})",
    ])

    find("amenities", [
        r"tiện\s*ích[^:]*[:\s\-]*([^\n]{10,300})",
    ], "Xem chi tiết trong mô tả")

    find("current_floors", [
        r"(\d+\s*tầng\s*(?:nổi)?[^\n]{0,50})",
        r"quy\s*mô[:\s\-]*([^\n]{5,100})",
        r"(\d+\s*(?:block|tòa)[^\n]{0,80})",
    ])

    find("payment_schedule", [
        r"tiến\s*độ\s*thanh\s*toán[:\s\-]*([^\n]{10,300})",
        r"thanh\s*toán[:\s\-]*(\d+[%\s][^\n]{5,200})",
    ])

    find("deposit_percent", [
        r"(?:đặt\s*cọc|cọc)[:\s\-]*(\d+[%\s][^\n]{0,100})",
    ])

    find("bank_support_percent", [
        r"(?:ngân\s*hàng|NH)\s*(?:hỗ\s*trợ|cho\s*vay)\s*(?:lên\s*đến\s*)?(\d+\s*%[^\n]{0,100})",
        r"vay\s*(?:lên\s*đến|tối\s*đa)?\s*(\d+\s*%[^\n]{0,50})",
    ])

    find("interest_rate", [
        r"lãi\s*(?:suất|xuất)[:\s\-]*([^\n]{3,100})",
    ])

    find("supporting_bank", [
        r"(?:ngân\s*hàng|NH)\s*(?:hỗ\s*trợ|liên\s*kết|cho\s*vay)[:\s\-]*([^\n]{3,100})",
    ])

    bank_names = ["Vietcombank", "BIDV", "Techcombank", "VPBank", "ACB",
                  "Sacombank", "MBBank", "VIB", "TPBank", "HDBank", "Agribank", "OCB"]
    found_banks = [b for b in bank_names if b.lower() in text.lower()]
    if found_banks and (not listing.get("supporting_bank") or listing["supporting_bank"] == "Chưa có thông tin"):
        listing["supporting_bank"] = ", ".join(found_banks)

    find("handover_date", [
        r"(?:nhận|bàn\s*giao)\s*nhà[:\s\-]*([^\n]{3,100})",
        r"(?:dự\s*kiến\s*)?bàn\s*giao[:\s\-]*([^\n]{3,100})",
        r"hoàn\s*thành[:\s\-]*(?:vào\s*)?([^\n]{3,100})",
    ])


# ══════════════════════════════════════════════
# SAVE & OUTPUT
# ══════════════════════════════════════════════

def save_listing(listing: dict, idx: int) -> str:
    """Save listing + images to listings/ folder."""
    cat = listing.get("loai", "can_ho_cu")
    source_id = listing.get("source_id", hashlib.md5(listing["url"].encode()).hexdigest()[:8])
    folder_name = f"{cat}_{idx+1:02d}_{source_id}"
    listing_dir = LISTINGS_DIR / folder_name
    listing_dir.mkdir(parents=True, exist_ok=True)

    saved_images = []
    for i, img_url in enumerate(listing.get("images", [])[:5]):
        ext = ".jpg"
        if ".png" in img_url.lower():
            ext = ".png"
        elif ".webp" in img_url.lower():
            ext = ".webp"
        fname = f"img_{i+1:02d}{ext}"
        fpath = listing_dir / fname
        if download_image(img_url, fpath):
            saved_images.append(fname)
            log(f"    📷 {fname} ({fpath.stat().st_size // 1024}KB)")

    listing["local_images"] = saved_images

    save_data = {k: v for k, v in listing.items() if k != "images"}
    save_data["image_count"] = len(saved_images)
    save_data["crawled_at"] = datetime.now(VN_TZ).isoformat()

    json_path = listing_dir / "metadata.json"
    with open(json_path, "w", encoding="utf-8") as f:
        json.dump(save_data, f, ensure_ascii=False, indent=2)

    log(f"    📝 metadata.json saved → {folder_name}/")
    return folder_name


def save_to_db(listing: dict, folder_name: str):
    """Save listing to SQLite database."""
    try:
        conn = sqlite3.connect(DB_PATH)

        # Check duplicate by source_id
        source_id = listing.get("source_id", "")
        if source_id:
            existing = conn.execute(
                "SELECT id FROM listings WHERE source_id=?", (source_id,)
            ).fetchone()
            if existing:
                log(f"    ↩ Skip duplicate source_id={source_id}")
                conn.close()
                return

        price_val = 0
        price_raw = listing.get("price_raw", 0)
        if price_raw:
            price_val = int(price_raw / 1_000_000)
        else:
            pt = extract_price_ty(listing.get("price", ""))
            if pt:
                price_val = int(pt * 1000)

        area_val = 0
        area_text = listing.get("area", "")
        if area_text:
            m = re.search(r"([\d.,]+)", area_text)
            if m:
                area_val = int(float(m.group(1).replace(",", ".")))

        sale_type = "secondary" if listing.get("loai") == "can_ho_cu" else "primary"

        conn.execute(
            """INSERT INTO listings
            (title, property_type, location, address, area, price, bedrooms, bathrooms,
             direction, legal_status, description, status, sale_type, source, source_id,
             url, price_raw, image_folder, image_count, created_at)
            VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)""",
            (
                listing.get("title", "")[:200],
                "căn hộ chung cư",
                listing.get("location", "TP Hồ Chí Minh"),
                listing.get("address", ""),
                area_val,
                price_val,
                listing.get("bedrooms", 0),
                listing.get("bathrooms", 0),
                listing.get("direction", ""),
                listing.get("legal_status", ""),
                listing.get("description", "")[:500],
                "available",
                sale_type,
                listing.get("source", ""),
                source_id,
                listing.get("url", ""),
                price_raw,
                folder_name,
                len(listing.get("local_images", [])),
                datetime.now(VN_TZ).isoformat(),
            ),
        )
        conn.commit()
        conn.close()
    except Exception as e:
        log(f"    ⚠ DB error: {e}")


def generate_summary(all_results: Dict[str, List[dict]], location_label: str,
                     price_min_ty: float, price_max_ty: float):
    """Generate SUMMARY.md with all crawled data."""
    summary_path = LISTINGS_DIR / "SUMMARY.md"
    now_str = datetime.now(VN_TZ).strftime("%d/%m/%Y %H:%M")
    total = sum(len(v) for v in all_results.values())

    lines = [
        f"# 🏠 BĐS {location_label} — Crawl {now_str}",
        "",
        f"**Khu vực:** {location_label}",
        f"**Giá:** {price_min_ty} - {price_max_ty} tỷ VND",
        f"**Tổng số:** {total} BĐS",
        "",
        "---",
    ]

    lines += ["", "## 🏚️ CĂN HỘ CŨ (Đã bàn giao)", ""]
    cu_list = all_results.get("can_ho_cu", [])
    if not cu_list:
        lines.append("_Không tìm thấy kết quả phù hợp._")
    for j, d in enumerate(cu_list):
        lines += [
            f"### {j+1}. {d.get('title', 'N/A')[:100]}",
            "",
            f"| Thông tin | Chi tiết |",
            f"|-----------|----------|",
            f"| **Giá** | {d.get('price', 'N/A')} |",
            f"| **Khu vực** | {d.get('location', 'N/A')} |",
            f"| **Diện tích** | {d.get('area', 'N/A')} |",
            f"| **Hướng** | {d.get('direction', 'Chưa rõ')} |",
            f"| **Phòng ngủ** | {d.get('bedrooms', 'N/A')} |",
            f"| **Pháp lý** | {d.get('legal_status', 'N/A')} |",
            f"| **NH định giá** | {d.get('bank_appraisal', 'N/A')} |",
            f"| **Ảnh** | {len(d.get('local_images', []))} ảnh đã tải |",
            f"| **Nguồn** | [{d.get('source', '')}]({d.get('url', '')}) |",
            "",
        ]

    lines += ["", "## 🏗️ CĂN HỘ MỚI (Dự án)", ""]
    moi_list = all_results.get("can_ho_moi", [])
    if not moi_list:
        lines.append("_Không tìm thấy kết quả phù hợp._")
    for j, d in enumerate(moi_list):
        lines += [
            f"### {j+1}. {d.get('title', 'N/A')[:100]}",
            "",
            f"| Thông tin | Chi tiết |",
            f"|-----------|----------|",
            f"| **Giá** | {d.get('price', 'N/A')} |",
            f"| **Khu vực** | {d.get('location', d.get('location_detail', 'N/A'))} |",
            f"| **Chủ đầu tư** | {d.get('developer', 'N/A')} |",
            f"| **Tiện ích** | {d.get('amenities', 'N/A')[:150]} |",
            f"| **Quy mô / Tầng** | {d.get('current_floors', 'N/A')} |",
            f"| **Tiến độ TT** | {d.get('payment_schedule', 'N/A')[:150]} |",
            f"| **Cọc** | {d.get('deposit_percent', 'N/A')} |",
            f"| **NH hỗ trợ %** | {d.get('bank_support_percent', 'N/A')} |",
            f"| **Lãi suất** | {d.get('interest_rate', 'N/A')} |",
            f"| **Ngân hàng** | {d.get('supporting_bank', 'N/A')} |",
            f"| **Nhận nhà** | {d.get('handover_date', 'N/A')} |",
            f"| **Ảnh** | {len(d.get('local_images', []))} ảnh đã tải |",
            f"| **Nguồn** | [{d.get('source', '')}]({d.get('url', '')}) |",
            "",
        ]

    with open(summary_path, "w", encoding="utf-8") as f:
        f.write("\n".join(lines))

    log(f"\n📋 Summary saved: {summary_path}")


# ══════════════════════════════════════════════
# MAIN
# ══════════════════════════════════════════════

ENRICH_DISPATCH = {
    "nhatot.com": enrich_from_chotot_detail,
    "batdongsan.com.vn": enrich_from_batdongsan_detail,
    "mogi.vn": enrich_from_mogi_detail,
    "alonhadat.com.vn": enrich_from_alonhadat_detail,
}

ALL_SOURCES = ["chotot", "bds", "mogi", "alonhadat"]


def merge_into(all_results: Dict[str, List[dict]], extra: List[dict],
               category: str, max_per_cat: int):
    """Merge extra listings into all_results dedup by title prefix."""
    existing = {re.sub(r'\s+', '', r["title"].lower())[:40]
                for r in all_results[category]}
    for e in extra:
        if len(all_results[category]) >= max_per_cat:
            break
        key = re.sub(r'\s+', '', e["title"].lower())[:40]
        if key not in existing:
            all_results[category].append(e)
            existing.add(key)


def main():
    parser = argparse.ArgumentParser(
        description="Crawl BĐS TP.HCM — hỗ trợ tất cả quận/huyện"
    )
    parser.add_argument(
        "--quan", type=str, default=None,
        help='Tên quận/huyện, ví dụ: "Quận 1", "Bình Thạnh", "Gò Vấp". '
             'Bỏ trống = crawl toàn TP.HCM.'
    )
    parser.add_argument(
        "--price-min", type=float, default=3.0,
        help="Giá tối thiểu (tỷ VND), mặc định 3"
    )
    parser.add_argument(
        "--price-max", type=float, default=5.0,
        help="Giá tối đa (tỷ VND), mặc định 5"
    )
    parser.add_argument(
        "--limit", type=int, default=5,
        help="Số lượng tối đa mỗi loại (cũ/mới), mặc định 5"
    )
    parser.add_argument(
        "--sources", type=str, default="chotot,bds,mogi,alonhadat",
        help=f"Nguồn dữ liệu, phân cách bằng dấu phẩy. Mặc định: tất cả. "
             f"Chọn từ: {', '.join(ALL_SOURCES)}"
    )
    args = parser.parse_args()

    price_min = args.price_min
    price_max = args.price_max
    max_per_cat = args.limit
    active_sources = {s.strip() for s in args.sources.split(",")}

    # Resolve district
    area_v2: Optional[str] = None
    district_key = ""
    alonhadat_slug = "tp-ho-chi-minh"
    location_label = "TP Hồ Chí Minh (tất cả quận)"

    if args.quan:
        district_key = args.quan.lower().strip()
        area_v2 = CHOTOT_AREA_MAP.get(district_key)
        alonhadat_slug = ALONHADAT_SLUG_MAP.get(district_key, alonhadat_slug)
        location_label = args.quan
        if not area_v2:
            log(f"⚠ Không tìm thấy mã khu vực cho '{args.quan}', crawl toàn TP.HCM")
            location_label = f"{args.quan} (toàn TP.HCM)"

    log(f"🚀 BĐS Crawler — {location_label}")
    log(f"   📁 Output: {LISTINGS_DIR}")
    log(f"   💰 Giá: {price_min}-{price_max} tỷ VND")
    log(f"   📊 Target: {max_per_cat * 2} BĐS ({max_per_cat} cũ + {max_per_cat} mới)")
    log(f"   🌐 Nguồn: {', '.join(sorted(active_sources))}")
    if area_v2:
        log(f"   🗺  area_v2={area_v2}")

    LISTINGS_DIR.mkdir(parents=True, exist_ok=True)

    all_results: Dict[str, List[dict]] = {"can_ho_cu": [], "can_ho_moi": []}

    # Source 1: Chợ Tốt (API — highest quality data)
    if "chotot" in active_sources:
        log(f"\n{'='*60}")
        log(f"  📡 NGUỒN 1: Chợ Tốt Nhà (nhatot.com)")
        log(f"{'='*60}")
        ct = crawl_chotot_all(area_v2, price_min, price_max, max_per_cat, location_label)
        for cat in ["can_ho_cu", "can_ho_moi"]:
            merge_into(all_results, ct[cat], cat, max_per_cat)

    # Source 2: BatDongSan.com.vn (curl_cffi bypass Cloudflare)
    if "bds" in active_sources:
        for category in ["can_ho_cu", "can_ho_moi"]:
            if len(all_results[category]) < max_per_cat:
                log(f"\n{'='*60}")
                log(f"  🏢 NGUỒN 2: BatDongSan.com.vn")
                log(f"{'='*60}")
                extra = crawl_batdongsan(category, district_key, price_min, price_max, max_per_cat)
                merge_into(all_results, extra, category, max_per_cat)

    # Source 3: Mogi.vn
    if "mogi" in active_sources:
        for category in ["can_ho_cu", "can_ho_moi"]:
            if len(all_results[category]) < max_per_cat:
                log(f"\n{'='*60}")
                log(f"  🟠 NGUỒN 3: Mogi.vn")
                log(f"{'='*60}")
                extra = crawl_mogi(category, district_key, price_min, price_max, max_per_cat)
                merge_into(all_results, extra, category, max_per_cat)

    # Source 4: Alonhadat.com.vn
    if "alonhadat" in active_sources:
        for category in ["can_ho_cu", "can_ho_moi"]:
            if len(all_results[category]) < max_per_cat:
                log(f"\n{'='*60}")
                log(f"  🌐 NGUỒN 4: Alonhadat.com.vn")
                log(f"{'='*60}")
                extra = crawl_alonhadat(category, alonhadat_slug, price_min, price_max, max_per_cat)
                merge_into(all_results, extra, category, max_per_cat)

    # Enrich details + save
    for category in ["can_ho_cu", "can_ho_moi"]:
        label = "🏚️ CĂN HỘ CŨ" if category == "can_ho_cu" else "🏗️ CĂN HỘ MỚI"
        results = all_results[category]

        log(f"\n{'='*60}")
        log(f"  {label} — Xử lý {len(results)} listings")
        log(f"{'='*60}")

        for idx, listing in enumerate(results):
            log(f"\n  [{idx+1}/{len(results)}] {listing['title'][:80]}")

            enrich_fn = ENRICH_DISPATCH.get(listing.get("source", ""), enrich_from_alonhadat_detail)
            enrich_fn(listing)

            folder = save_listing(listing, idx)
            save_to_db(listing, folder)

    generate_summary(all_results, location_label, price_min, price_max)

    total = sum(len(v) for v in all_results.values())
    log(f"\n{'='*60}")
    log(f"✅ HOÀN TẤT — Crawl được {total} BĐS")
    for cat, lbl in [("can_ho_cu", "Căn hộ cũ"), ("can_ho_moi", "Căn hộ mới")]:
        log(f"   {lbl}: {len(all_results.get(cat, []))}")
    log(f"   📂 Listings: {LISTINGS_DIR}")
    log(f"   📊 DB: {DB_PATH}")
    log(f"{'='*60}")


if __name__ == "__main__":
    main()
