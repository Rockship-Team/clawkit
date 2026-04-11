#!/usr/bin/env node
// Shop Hoa helper — manage orders (JSON file) and list flower images.
// Called by the skill via: node ~/.openclaw/workspace/skills/shop-hoa/cli.js <cmd> ...
// Orders live in orders.json next to this script. Images live in flowers/ next to this script.

'use strict';

const fs = require('fs');
const path = require('path');
const os = require('os');
const { spawnSync } = require('child_process');

const SKILL_DIR = path.join(os.homedir(), '.openclaw', 'workspace', 'skills', 'shop-hoa');
const DB = path.join(SKILL_DIR, 'orders.json');
const FLOWERS = path.join(SKILL_DIR, 'flowers');
const OPENCLAW_CONFIG = path.join(os.homedir(), '.openclaw', 'openclaw.json');

function ensureDB() {
  if (!fs.existsSync(SKILL_DIR)) {
    fs.mkdirSync(SKILL_DIR, { recursive: true });
  }
  if (!fs.existsSync(DB)) {
    fs.writeFileSync(DB, '[]');
  }
}

function load() {
  ensureDB();
  const raw = fs.readFileSync(DB, 'utf8').trim();
  if (!raw) return [];
  return JSON.parse(raw);
}

function save(orders) {
  fs.writeFileSync(DB, JSON.stringify(orders, null, 2));
}

function nextId(orders) {
  return orders.reduce((m, o) => Math.max(m, o.id || 0), 0) + 1;
}

function vnNow() {
  // ISO timestamp in Asia/Ho_Chi_Minh.
  const parts = new Intl.DateTimeFormat('en-CA', {
    timeZone: 'Asia/Ho_Chi_Minh',
    year: 'numeric', month: '2-digit', day: '2-digit',
    hour: '2-digit', minute: '2-digit', second: '2-digit',
    hour12: false,
  }).formatToParts(new Date());
  const get = t => parts.find(p => p.type === t).value;
  return `${get('year')}-${get('month')}-${get('day')}T${get('hour')}:${get('minute')}:${get('second')}+07:00`;
}

function vnToday() {
  return new Intl.DateTimeFormat('en-CA', { timeZone: 'Asia/Ho_Chi_Minh' }).format(new Date());
}

// Parse prices leniently: "350k" → 350000, "1.5tr" → 1500000, "350.000" → 350000.
function parsePrice(input) {
  if (input == null) return 0;
  let s = String(input).trim().toLowerCase().replace(/[₫đ\s]/g, '');
  if (!s) return 0;
  let mult = 1;
  if (/tr$/.test(s)) { mult = 1_000_000; s = s.replace(/tr$/, ''); }
  else if (/m$/.test(s)) { mult = 1_000_000; s = s.replace(/m$/, ''); }
  else if (/k$/.test(s)) { mult = 1_000; s = s.replace(/k$/, ''); }
  if (/^\d+[.,]\d{3}(?:[.,]\d{3})*$/.test(s)) {
    s = s.replace(/[.,]/g, '');
  } else {
    s = s.replace(/,/g, '.');
  }
  const n = parseFloat(s);
  return isFinite(n) ? Math.round(n * mult) : 0;
}

// add <customer> <recipient> <phone> <address> <items> <price> <delivery_time> [note] [sender_id]
// Positional, single-line. sender_id is the customer's chat-channel identity
// (e.g. Telegram sender_id) — used later by `list-mine` to show only their
// own orders and never leak other customers' data.
function cmdAdd() {
  const [
    customer_name,
    recipient_name,
    recipient_phone,
    recipient_address,
    items,
    priceStr,
    delivery_time,
    note = '',
    sender_id = '',
  ] = process.argv.slice(3);
  if (!customer_name || !recipient_name || !recipient_phone || !recipient_address || !items || !priceStr || !delivery_time) {
    throw new Error('usage: add <customer> <recipient_name> <phone> <address> <items> <price> <delivery_time> [note] [sender_id]');
  }
  const orders = load();
  const order = {
    id: nextId(orders),
    status: 'new',
    sender_id: String(sender_id).trim(),
    customer_name: String(customer_name).trim() || 'Khách',
    recipient_name: String(recipient_name).trim(),
    recipient_phone: String(recipient_phone).trim(),
    recipient_address: String(recipient_address).trim(),
    items: String(items).trim(),
    price: parsePrice(priceStr),
    delivery_time: String(delivery_time).trim(),
    note: String(note).trim(),
    created_at: vnNow(),
  };
  if (!order.recipient_name) throw new Error('recipient_name is required');
  if (!order.recipient_phone) throw new Error('recipient_phone is required');
  if (!order.recipient_address) throw new Error('recipient_address is required');
  if (!order.items) throw new Error('items is required');
  if (!order.price) throw new Error('price must be > 0');
  orders.push(order);
  save(orders);
  console.log(JSON.stringify({ ok: true, order }));
}

// Apply filter keyword ('recent' | 'new' | 'today' | 'completed' | 'cancelled'
// | 'all' | 'id:N') to an array of orders and return the filtered subset.
// Extracted so both the admin-scope `list` and customer-scope `list-mine`
// can reuse the same filter logic without duplication.
function applyOrderFilter(orders, filter) {
  if (filter === 'new') {
    return orders.filter(o => o.status === 'new');
  }
  if (filter === 'today') {
    const today = vnToday();
    return orders.filter(o => (o.created_at || '').startsWith(today));
  }
  if (filter === 'completed') {
    return orders.filter(o => o.status === 'completed');
  }
  if (filter === 'cancelled') {
    return orders.filter(o => o.status === 'cancelled');
  }
  if (filter === 'all') {
    return orders;
  }
  if (filter.startsWith('id:')) {
    const id = parseInt(filter.slice(3), 10);
    return orders.filter(o => o.id === id);
  }
  // 'recent' (default) — last 10 of any status
  return orders.slice(-10);
}

// cmdList — ADMIN SCOPE — returns every order in the database regardless
// of owner. Only the shop owner should run this. Customers must NEVER use
// this command (SKILL.md explicitly bans it). Kept around because there is
// no owner-identity mechanism yet and manual dev usage is helpful.
function cmdList() {
  const filter = process.argv[3] || 'recent';
  // `customer:<name>` is admin-only (needs to scan across all customers).
  let result;
  if (filter.startsWith('customer:')) {
    const q = filter.slice(9).toLowerCase();
    result = load().filter(o => (o.customer_name || '').toLowerCase().includes(q));
  } else {
    result = applyOrderFilter(load(), filter);
  }
  result = result.slice().reverse();
  console.log(JSON.stringify({ ok: true, scope: 'admin', filter, count: result.length, orders: result }));
}

// cmdListMine — CUSTOMER SCOPE — returns only orders whose `sender_id`
// matches the argument. This is how the bot answers "đơn của tôi thế nào"
// queries on Telegram without ever leaking another customer's order data.
//
// Usage:
//   node cli.js list-mine <sender_id> [filter]
//
// sender_id is mandatory — if missing or empty the command refuses. The
// filter is the same keyword set as `list` (recent|new|today|completed|
// cancelled|all|id:N) applied AFTER the sender_id restriction.
function cmdListMine() {
  const senderId = (process.argv[3] || '').trim();
  const filter = process.argv[4] || 'recent';
  if (!senderId) {
    console.log(JSON.stringify({ ok: false, error: 'list-mine requires sender_id as first arg' }));
    return;
  }
  // Reject the admin-scope `customer:` filter here — customer scope can only
  // see their OWN orders, period.
  if (filter.startsWith('customer:')) {
    console.log(JSON.stringify({ ok: false, error: 'customer: filter is admin-only; list-mine rejects it' }));
    return;
  }
  const mine = load().filter(o => (o.sender_id || '') === senderId);
  const filtered = applyOrderFilter(mine, filter);
  const result = filtered.slice().reverse();
  console.log(JSON.stringify({
    ok: true,
    scope: 'customer',
    sender_id: senderId,
    filter,
    count: result.length,
    orders: result,
  }));
}

function cmdRevenue() {
  const orders = load();
  const completed = orders.filter(o => o.status === 'completed');
  const total = completed.reduce((s, o) => s + (o.price || 0), 0);
  console.log(JSON.stringify({
    ok: true,
    total,
    count: completed.length,
    new_count: orders.filter(o => o.status === 'new').length,
    cancelled_count: orders.filter(o => o.status === 'cancelled').length,
  }));
}

function setStatus(id, status) {
  const orders = load();
  const idx = orders.findIndex(o => o.id === id);
  if (idx < 0) {
    console.log(JSON.stringify({ ok: false, error: `order #${id} not found` }));
    return;
  }
  orders[idx].status = status;
  save(orders);
  console.log(JSON.stringify({ ok: true, order: orders[idx] }));
}

function cmdDone() {
  const id = parseInt(process.argv[3], 10);
  if (!id) throw new Error('done requires order id');
  setStatus(id, 'completed');
}

function cmdCancel() {
  const id = parseInt(process.argv[3], 10);
  if (!id) throw new Error('cancel requires order id');
  setStatus(id, 'cancelled');
}

function cmdImages() {
  const folder = process.argv[3];
  if (!folder) throw new Error('images requires folder name');
  const dir = path.join(FLOWERS, folder);
  if (!fs.existsSync(dir)) {
    console.log(JSON.stringify({ ok: false, error: `folder not found: ${folder}`, dir }));
    return;
  }
  const files = fs.readdirSync(dir)
    .filter(f => /\.(jpg|jpeg|png|webp)$/i.test(f))
    .sort()
    .map(f => path.join(dir, f));
  console.log(JSON.stringify({ ok: true, folder, count: files.length, files }));
}

function cmdFolders() {
  if (!fs.existsSync(FLOWERS)) {
    console.log(JSON.stringify({ ok: false, error: 'flowers dir missing', dir: FLOWERS }));
    return;
  }
  const dirs = fs.readdirSync(FLOWERS)
    .filter(f => fs.statSync(path.join(FLOWERS, f)).isDirectory())
    .sort();
  console.log(JSON.stringify({ ok: true, folders: dirs }));
}

// Read the Telegram bot token from ~/.openclaw/openclaw.json so cli.js can
// upload photos directly to the Telegram Bot API without going through
// OpenClaw's delivery pipeline (MEDIA: token) — that path is fragile.
// Accepts both `channels.telegram.botToken` (single account) and
// `channels.telegram.accounts.<id>.botToken` (multi-account).
function readTelegramBotToken() {
  if (!fs.existsSync(OPENCLAW_CONFIG)) {
    throw new Error('openclaw config not found at ' + OPENCLAW_CONFIG);
  }
  const cfg = JSON.parse(fs.readFileSync(OPENCLAW_CONFIG, 'utf8'));
  const tg = cfg.channels && cfg.channels.telegram;
  if (!tg) throw new Error('channels.telegram not configured in openclaw.json');
  if (typeof tg.botToken === 'string' && tg.botToken.trim()) return tg.botToken.trim();
  if (tg.accounts && typeof tg.accounts === 'object') {
    for (const acc of Object.values(tg.accounts)) {
      if (acc && typeof acc.botToken === 'string' && acc.botToken.trim()) return acc.botToken.trim();
    }
  }
  throw new Error('channels.telegram.botToken is empty');
}

// sendPhotoViaCurl uploads a single local image file to a Telegram chat
// using a multipart/form-data POST via the `curl` binary (Win10+, macOS,
// Linux all ship with curl). Bypasses OpenClaw's media pipeline entirely.
// Returns a result object {ok, description?, fileId?}.
function sendPhotoViaCurl(token, chatId, filePath, caption) {
  const url = 'https://api.telegram.org/bot' + token + '/sendPhoto';
  const args = [
    '-sS',
    '-F', 'chat_id=' + chatId,
    '-F', 'photo=@' + filePath,
  ];
  if (caption) args.push('-F', 'caption=' + caption);
  args.push(url);
  const r = spawnSync('curl', args, { encoding: 'utf8' });
  if (r.error) return { ok: false, error: 'curl spawn failed: ' + r.error.message };
  if (r.status !== 0) return { ok: false, error: 'curl exit ' + r.status + ': ' + r.stderr };
  try {
    const resp = JSON.parse(r.stdout);
    if (resp.ok) {
      const photos = resp.result && resp.result.photo;
      const largest = photos && photos[photos.length - 1];
      return { ok: true, fileId: largest && largest.file_id };
    }
    return { ok: false, error: resp.description || 'telegram api returned ok=false', telegram: resp };
  } catch (e) {
    return { ok: false, error: 'invalid JSON from telegram: ' + e.message, raw: r.stdout.slice(0, 500) };
  }
}

// cmdSendImagesTelegram: uploads up to N flower images from a folder to a
// Telegram chat directly. This is the canonical way for shop-hoa to send
// product photos on Telegram — it does NOT use OpenClaw's MEDIA: token
// pipeline.
//
// Usage:
//   node cli.js send-images-telegram <folder> <chat_id> [count]
//
// Example:
//   node cli.js send-images-telegram best-seller 2006815602 5
//
// Returns JSON:
//   {ok, sent, total, folder, results: [{file, ok, error?}]}
function cmdSendImagesTelegram() {
  const folder = process.argv[3];
  const chatId = process.argv[4];
  const count = Math.max(1, Math.min(parseInt(process.argv[5] || '5', 10) || 5, 5));

  if (!folder || !chatId) {
    throw new Error('usage: send-images-telegram <folder> <chat_id> [count]');
  }

  const dir = path.join(FLOWERS, folder);
  if (!fs.existsSync(dir)) {
    console.log(JSON.stringify({ ok: false, error: 'folder not found: ' + folder, dir }));
    return;
  }

  const files = fs.readdirSync(dir)
    .filter(f => /\.(jpg|jpeg|png|webp)$/i.test(f))
    .sort()
    .slice(0, count)
    .map(f => path.join(dir, f));

  if (files.length === 0) {
    console.log(JSON.stringify({ ok: false, error: 'no images in folder: ' + folder, dir }));
    return;
  }

  let token;
  try {
    token = readTelegramBotToken();
  } catch (e) {
    console.log(JSON.stringify({ ok: false, error: e.message }));
    return;
  }

  const results = [];
  let sent = 0;
  for (const f of files) {
    const r = sendPhotoViaCurl(token, chatId, f);
    results.push({ file: path.basename(f), ok: r.ok, ...(r.ok ? {} : { error: r.error }) });
    if (r.ok) sent++;
  }

  console.log(JSON.stringify({
    ok: sent > 0,
    sent,
    total: files.length,
    folder,
    chat_id: chatId,
    results,
  }));
}

const USAGE = 'usage: cli.js <add|list|list-mine|revenue|done|cancel|images|folders|send-images-telegram> [args]';

const cmd = process.argv[2];
try {
  switch (cmd) {
    case 'add': cmdAdd(); break;
    case 'list': cmdList(); break;
    case 'list-mine': cmdListMine(); break;
    case 'revenue': cmdRevenue(); break;
    case 'done': cmdDone(); break;
    case 'cancel': cmdCancel(); break;
    case 'images': cmdImages(); break;
    case 'folders': cmdFolders(); break;
    case 'send-images-telegram': cmdSendImagesTelegram(); break;
    default:
      console.error(USAGE);
      process.exit(2);
  }
} catch (e) {
  console.log(JSON.stringify({ ok: false, error: e.message }));
  process.exit(1);
}
