#!/usr/bin/env node
// Finance Tracker helper — append/query local transactions.csv.
// Called by the skill via: node ~/.openclaw/workspace/skills/finance-tracker/cli.js <cmd> ...
// All data lives in transactions.csv in the same directory as this script.

'use strict';

const fs = require('fs');
const path = require('path');
const os = require('os');

const SKILL_DIR = path.join(os.homedir(), '.openclaw', 'workspace', 'skills', 'finance-tracker');
const CSV = path.join(SKILL_DIR, 'transactions.csv');
const HEADER = 'date,place,amount,category,note';

function ensureFile() {
  if (!fs.existsSync(SKILL_DIR)) {
    fs.mkdirSync(SKILL_DIR, { recursive: true });
  }
  if (!fs.existsSync(CSV)) {
    fs.writeFileSync(CSV, HEADER + '\n');
  }
}

function esc(s) {
  s = String(s == null ? '' : s);
  return /[",\n\r]/.test(s) ? '"' + s.replace(/"/g, '""') + '"' : s;
}

function parseCSV(text) {
  const rows = [];
  const lines = text.split(/\r?\n/);
  for (let i = 1; i < lines.length; i++) {
    const line = lines[i];
    if (!line || !line.trim()) continue;
    const cells = [];
    let cur = '';
    let inQ = false;
    for (let j = 0; j < line.length; j++) {
      const ch = line[j];
      if (inQ) {
        if (ch === '"' && line[j + 1] === '"') { cur += '"'; j++; }
        else if (ch === '"') { inQ = false; }
        else cur += ch;
      } else {
        if (ch === '"') inQ = true;
        else if (ch === ',') { cells.push(cur); cur = ''; }
        else cur += ch;
      }
    }
    cells.push(cur);
    rows.push({
      date: cells[0] || '',
      place: cells[1] || '',
      amount: parseInt(cells[2], 10) || 0,
      category: cells[3] || 'other',
      note: cells[4] || '',
    });
  }
  return rows;
}

function readRows() {
  ensureFile();
  return parseCSV(fs.readFileSync(CSV, 'utf8'));
}

function writeRows(rows) {
  const body = rows.map(r => [r.date, r.place, r.amount, r.category, r.note].map(esc).join(',')).join('\n');
  fs.writeFileSync(CSV, HEADER + '\n' + body + (rows.length ? '\n' : ''));
}

function todayVN() {
  // en-CA locale yields YYYY-MM-DD.
  return new Intl.DateTimeFormat('en-CA', { timeZone: 'Asia/Ho_Chi_Minh' }).format(new Date());
}

// Parse amounts leniently so agents can pass "55k", "1tr", "1.5tr", "55.000", "55,000".
function parseAmount(input) {
  if (input == null) return 0;
  let s = String(input).trim().toLowerCase().replace(/[₫đ\s]/g, '');
  if (!s) return 0;
  let mult = 1;
  if (/tr$/.test(s)) { mult = 1_000_000; s = s.replace(/tr$/, ''); }
  else if (/m$/.test(s)) { mult = 1_000_000; s = s.replace(/m$/, ''); }
  else if (/k$/.test(s)) { mult = 1_000; s = s.replace(/k$/, ''); }
  // Strip VN thousand-separators ("55.000" or "55,000"). If it looks like a decimal (e.g. "1.5"), keep dot.
  if (/^\d+[.,]\d{3}(?:[.,]\d{3})*$/.test(s)) {
    s = s.replace(/[.,]/g, '');
  } else {
    s = s.replace(/,/g, '.');
  }
  const n = parseFloat(s);
  return isFinite(n) ? Math.round(n * mult) : 0;
}

const CATEGORY_ALIASES = {
  food: 'food', an: 'food', 'ăn': 'food', 'ăn-uống': 'food', 'an-uong': 'food', eat: 'food', meal: 'food', lunch: 'food', dinner: 'food', breakfast: 'food',
  cafe: 'cafe', coffee: 'cafe', cà: 'cafe', 'cà-phê': 'cafe', caphe: 'cafe', drink: 'cafe', tea: 'cafe',
  shopping: 'shopping', shop: 'shopping', 'mua-sắm': 'shopping', 'mua-sam': 'shopping', clothes: 'shopping', quần: 'shopping',
  transport: 'transport', 'di-chuyển': 'transport', 'di-chuyen': 'transport', grab: 'transport', taxi: 'transport', xe: 'transport', xăng: 'transport', fuel: 'transport', gas: 'transport', ride: 'transport',
  health: 'health', 'y-tế': 'health', 'y-te': 'health', medicine: 'health', doctor: 'health', pharmacy: 'health', gym: 'health',
  entertainment: 'entertainment', 'giải-trí': 'entertainment', 'giai-tri': 'entertainment', movie: 'entertainment', game: 'entertainment', fun: 'entertainment',
  education: 'education', 'học-tập': 'education', 'hoc-tap': 'education', book: 'education', course: 'education', study: 'education', school: 'education',
  home: 'home', 'nhà-cửa': 'home', 'nha-cua': 'home', rent: 'home', electricity: 'home', water: 'home', internet: 'home', utilities: 'home',
  work: 'work', 'công-việc': 'work', 'cong-viec': 'work', office: 'work', software: 'work',
  other: 'other', 'khác': 'other', khac: 'other', misc: 'other',
};

function normalizeCategory(input) {
  if (!input) return 'other';
  const key = String(input).trim().toLowerCase().replace(/\s+/g, '-');
  return CATEGORY_ALIASES[key] || 'other';
}

// add <place> <amount> <category> [note] [date]
// Positional, single-line. Date defaults to today VN.
function cmdAdd() {
  const [place, amountStr, categoryRaw, note = '', dateArg = ''] = process.argv.slice(3);
  if (!place) throw new Error('usage: add <place> <amount> <category> [note] [date]');
  if (!amountStr) throw new Error('amount is required');

  const row = {
    date: dateArg.trim() || todayVN(),
    place: String(place).trim(),
    amount: parseAmount(amountStr),
    category: normalizeCategory(categoryRaw),
    note: String(note).trim(),
  };
  if (!row.place) throw new Error('place is required');
  if (!row.amount) throw new Error('amount must be > 0');

  ensureFile();
  fs.appendFileSync(CSV, [row.date, row.place, row.amount, row.category, row.note].map(esc).join(',') + '\n');

  const rows = readRows();
  const today = todayVN();
  const todayTotal = rows.filter(r => r.date === today).reduce((s, r) => s + r.amount, 0);

  console.log(JSON.stringify({ ok: true, saved: row, today_total: todayTotal }));
}

function periodFilter(rows, period) {
  const today = todayVN();
  if (period === 'today') {
    return { rows: rows.filter(r => r.date === today), label: today };
  }
  if (period === 'week') {
    const now = new Date(today + 'T00:00:00');
    const start = new Date(now); start.setDate(now.getDate() - 6);
    const s = start.toISOString().slice(0, 10);
    return { rows: rows.filter(r => r.date >= s && r.date <= today), label: s + ' → ' + today };
  }
  if (period === 'all') {
    return { rows, label: 'all time' };
  }
  // default month
  const ym = today.slice(0, 7);
  return { rows: rows.filter(r => r.date.startsWith(ym)), label: ym };
}

function cmdReport() {
  const period = process.argv[3] || 'month';
  const all = readRows();
  const { rows, label } = periodFilter(all, period);
  const total = rows.reduce((s, r) => s + r.amount, 0);
  const byCat = {};
  for (const r of rows) byCat[r.category] = (byCat[r.category] || 0) + r.amount;
  const sorted = Object.entries(byCat)
    .sort((a, b) => b[1] - a[1])
    .map(([category, amount]) => ({
      category,
      amount,
      pct: total ? Math.round((amount / total) * 100) : 0,
    }));
  const recent = rows.slice(-5).reverse().map(r => ({
    date: r.date,
    place: r.place,
    amount: r.amount,
    category: r.category,
  }));
  console.log(JSON.stringify({
    ok: true,
    period,
    label,
    total,
    count: rows.length,
    by_category: sorted,
    recent,
  }));
}

function cmdLast() {
  const n = parseInt(process.argv[3], 10) || 5;
  const rows = readRows();
  console.log(JSON.stringify({
    ok: true,
    transactions: rows.slice(-n).reverse(),
  }));
}

function cmdUndo() {
  const rows = readRows();
  if (rows.length === 0) {
    console.log(JSON.stringify({ ok: false, error: 'nothing to undo' }));
    return;
  }
  const removed = rows.pop();
  writeRows(rows);
  console.log(JSON.stringify({ ok: true, removed }));
}

function cmdStats() {
  // Quick meta: file path, row count, date range.
  const rows = readRows();
  const dates = rows.map(r => r.date).sort();
  console.log(JSON.stringify({
    ok: true,
    file: CSV,
    count: rows.length,
    first: dates[0] || null,
    last: dates[dates.length - 1] || null,
  }));
}

const USAGE = 'usage: cli.js <add|report|last|undo|stats> [args]';

const cmd = process.argv[2];
try {
  switch (cmd) {
    case 'add': cmdAdd(); break;
    case 'report': cmdReport(); break;
    case 'last': cmdLast(); break;
    case 'undo': cmdUndo(); break;
    case 'stats': cmdStats(); break;
    default:
      console.error(USAGE);
      process.exit(2);
  }
} catch (e) {
  console.log(JSON.stringify({ ok: false, error: e.message }));
  process.exit(1);
}
