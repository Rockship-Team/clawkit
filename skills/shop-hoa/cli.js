#!/usr/bin/env node
// Shop Hoa helper — manage orders (JSON file) and list flower images.
// Called by the skill via: node ~/.openclaw/workspace/skills/shop-hoa/cli.js <cmd> ...
// Orders live in orders.json next to this script. Images live in flowers/ next to this script.

'use strict';

const fs = require('fs');
const path = require('path');
const os = require('os');

const SKILL_DIR = path.join(os.homedir(), '.openclaw', 'workspace', 'skills', 'shop-hoa');
const DB = path.join(SKILL_DIR, 'orders.json');
const FLOWERS = path.join(SKILL_DIR, 'flowers');

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

// add <customer> <recipient> <phone> <address> <items> <price> <delivery_time> [note]
// Positional, single-line.
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
  ] = process.argv.slice(3);
  if (!customer_name || !recipient_name || !recipient_phone || !recipient_address || !items || !priceStr || !delivery_time) {
    throw new Error('usage: add <customer> <recipient_name> <phone> <address> <items> <price> <delivery_time> [note]');
  }
  const orders = load();
  const order = {
    id: nextId(orders),
    status: 'new',
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

function cmdList() {
  const filter = process.argv[3] || 'recent';
  const orders = load();
  let result = orders;
  if (filter === 'new') {
    result = orders.filter(o => o.status === 'new');
  } else if (filter === 'today') {
    const today = vnToday();
    result = orders.filter(o => (o.created_at || '').startsWith(today));
  } else if (filter === 'completed') {
    result = orders.filter(o => o.status === 'completed');
  } else if (filter === 'cancelled') {
    result = orders.filter(o => o.status === 'cancelled');
  } else if (filter === 'all') {
    result = orders;
  } else if (filter.startsWith('id:')) {
    const id = parseInt(filter.slice(3), 10);
    result = orders.filter(o => o.id === id);
  } else if (filter.startsWith('customer:')) {
    const q = filter.slice(9).toLowerCase();
    result = orders.filter(o => (o.customer_name || '').toLowerCase().includes(q));
  } else {
    // 'recent' — last 10 of any status
    result = orders.slice(-10);
  }
  result = result.slice().reverse();
  console.log(JSON.stringify({ ok: true, filter, count: result.length, orders: result }));
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

const USAGE = 'usage: cli.js <add|list|revenue|done|cancel|images|folders> [args]';

const cmd = process.argv[2];
try {
  switch (cmd) {
    case 'add': cmdAdd(); break;
    case 'list': cmdList(); break;
    case 'revenue': cmdRevenue(); break;
    case 'done': cmdDone(); break;
    case 'cancel': cmdCancel(); break;
    case 'images': cmdImages(); break;
    case 'folders': cmdFolders(); break;
    default:
      console.error(USAGE);
      process.exit(2);
  }
} catch (e) {
  console.log(JSON.stringify({ ok: false, error: e.message }));
  process.exit(1);
}
