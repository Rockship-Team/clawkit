#!/usr/bin/env node
// Generic schema-driven CLI for clawkit skills.
// Reads schema.json for field definitions and clawkit.json for storage target.
// Supports local JSON file storage and Supabase REST API.
//
// Commands:
//   add <field1> <field2> ...           Create a record (positional args match non-auto fields)
//   list [filter]                       List all records (admin scope)
//   list-mine <owner_id> [filter]       List records owned by owner_id
//   done <id>                           Set status to completed (statuses[1])
//   cancel <id>                         Set status to cancelled (statuses[2])
//   update <id> <field> <value>         Update a single field on a record
//   revenue                             Sum price field for completed records
//   images <folder>                     List image files in a product folder
//   folders                             List product image folders
//   send-images-telegram <f> <cid> [n]  Upload images to Telegram chat

'use strict';

const fs = require('fs');
const path = require('path');
const os = require('os');
const { spawnSync } = require('child_process');

const SKILL_DIR = __dirname;

// ---------------------------------------------------------------------------
// Load schema + config
// ---------------------------------------------------------------------------

const rawSchema = JSON.parse(fs.readFileSync(path.join(SKILL_DIR, 'schema.json'), 'utf8'));
let skillConfig = {};
try {
  skillConfig = JSON.parse(fs.readFileSync(path.join(SKILL_DIR, 'clawkit.json'), 'utf8'));
} catch (_) { /* config may not exist yet during dev */ }

// Normalize schema: support both legacy single-table and multi-table format.
const schema = (() => {
  if (rawSchema.tables) return rawSchema;
  // Legacy: convert "table" + "fields" to multi-table format.
  const name = rawSchema.table || 'records';
  return {
    ...rawSchema,
    tables: { [name]: { fields: rawSchema.fields, statuses: rawSchema.statuses } },
    primary: name,
  };
})();

// Parse --table flag from args, remove it so commands see clean args.
let targetTable = schema.primary;
const tableIdx = process.argv.indexOf('--table');
if (tableIdx !== -1 && process.argv[tableIdx + 1]) {
  targetTable = process.argv[tableIdx + 1];
  process.argv.splice(tableIdx, 2);
}

const tableDef = schema.tables[targetTable];
if (!tableDef) {
  console.log(JSON.stringify({ ok: false, error: `unknown table: ${targetTable}`, available: Object.keys(schema.tables) }));
  process.exit(1);
}

const TABLE = targetTable;
const TIMEZONE = schema.timezone || 'UTC';
const STATUSES = tableDef.statuses || ['new', 'completed', 'cancelled'];
const DB_TARGET = skillConfig.db_target || schema.db_target || 'local';
const IMAGES_DIR = path.join(SKILL_DIR, schema.images_dir || 'products');
const OPENCLAW_CONFIG = path.join(os.homedir(), '.openclaw', 'openclaw.json');
const MAX_RECENT = 10;        // Default limit for 'recent' filter.
const MAX_IMAGES_SEND = 5;    // Max images per Telegram send.

// Field lookups by role (per current table).
const fields = tableDef.fields;
const idField = fields.find(f => f.auto === 'increment');
const statusField = fields.find(f => f.role === 'status');
const ownerField = fields.find(f => f.role === 'owner');
const priceField = fields.find(f => f.role === 'price');
const timestampField = fields.find(f => f.role === 'timestamp');

// Input fields = fields the user passes as positional args.
// Excluded: auto fields, and fields with defaults (auto-filled, not user-provided).
const inputFields = fields.filter(f => !f.auto && !f.default);

// ---------------------------------------------------------------------------
// Utilities
// ---------------------------------------------------------------------------

function localNow() {
  const parts = new Intl.DateTimeFormat('en-CA', {
    timeZone: TIMEZONE,
    year: 'numeric', month: '2-digit', day: '2-digit',
    hour: '2-digit', minute: '2-digit', second: '2-digit',
    hour12: false,
  }).formatToParts(new Date());
  const get = t => parts.find(p => p.type === t).value;
  // Compute UTC offset for the timezone.
  const now = new Date();
  const utc = new Date(now.toLocaleString('en-US', { timeZone: 'UTC' }));
  const local = new Date(now.toLocaleString('en-US', { timeZone: TIMEZONE }));
  const diffMin = (local - utc) / 60000;
  const sign = diffMin >= 0 ? '+' : '-';
  const absMin = Math.abs(diffMin);
  const offH = String(Math.floor(absMin / 60)).padStart(2, '0');
  const offM = String(absMin % 60).padStart(2, '0');
  return `${get('year')}-${get('month')}-${get('day')}T${get('hour')}:${get('minute')}:${get('second')}${sign}${offH}:${offM}`;
}

function localToday() {
  return new Intl.DateTimeFormat('en-CA', { timeZone: TIMEZONE }).format(new Date());
}

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

function coerce(value, field) {
  const v = String(value).trim();
  if (field.type === 'integer') {
    return field.role === 'price' ? parsePrice(v) : parseInt(v, 10) || 0;
  }
  return v;
}

// ---------------------------------------------------------------------------
// Store: local JSON file
// ---------------------------------------------------------------------------

function createJsonStore() {
  const dbPath = path.join(SKILL_DIR, TABLE + '.json');

  function ensureDB() {
    if (!fs.existsSync(dbPath)) fs.writeFileSync(dbPath, '[]');
  }

  function loadAll() {
    ensureDB();
    const raw = fs.readFileSync(dbPath, 'utf8').trim();
    return raw ? JSON.parse(raw) : [];
  }

  function saveAll(records) {
    fs.writeFileSync(dbPath, JSON.stringify(records, null, 2));
  }

  return {
    async loadAll() { return loadAll(); },

    async insert(record) {
      const records = loadAll();
      records.push(record);
      saveAll(records);
      return record;
    },

    async update(id, changes) {
      const records = loadAll();
      const idx = records.findIndex(r => r[idField.name] === id);
      if (idx < 0) return null;
      Object.assign(records[idx], changes);
      saveAll(records);
      return records[idx];
    },

    nextId() {
      const records = loadAll();
      return records.reduce((m, r) => Math.max(m, r[idField.name] || 0), 0) + 1;
    },
  };
}

// ---------------------------------------------------------------------------
// Store: Remote REST API (supabase, custom API, etc.)
//
// Store contract — the server must implement:
//   GET  <baseURL>       → [records...]
//   POST <baseURL>       → created record (with server-assigned id)
//   PATCH <baseURL>/<id> → updated record
//
// Config (from clawkit.json tokens):
//   db_url     — full endpoint URL
//   db_key     — API key (sets apikey + Authorization: Bearer headers)
//   db_headers — JSON object of extra headers
// ---------------------------------------------------------------------------

function createRemoteStore() {
  const tokens = skillConfig.tokens || {};
  const dbURL = tokens.db_url;
  if (!dbURL) throw new Error('db_url missing in clawkit.json tokens');

  // Build base URL per table.
  // Supabase: <project_url>/rest/v1/<table>
  // API:      <base_url>/<table>
  const isSupabase = DB_TARGET === 'supabase';
  const baseURL = isSupabase
    ? `${dbURL.replace(/\/$/, '')}/rest/v1/${TABLE}`
    : `${dbURL.replace(/\/$/, '')}/${TABLE}`;

  // Build headers.
  const headers = { 'Content-Type': 'application/json' };
  if (tokens.db_key) {
    headers['apikey'] = tokens.db_key;
    headers['Authorization'] = `Bearer ${tokens.db_key}`;
  }
  if (tokens.db_headers) {
    try { Object.assign(headers, JSON.parse(tokens.db_headers)); } catch {}
  }

  async function request(urlSuffix, method, body) {
    const opts = { method: method || 'GET', headers: { ...headers, 'Prefer': 'return=representation' } };
    if (body) opts.body = JSON.stringify(body);
    const res = await fetch(`${baseURL}${urlSuffix}`, opts);
    if (!res.ok) throw new Error(`${method || 'GET'} ${urlSuffix || '/'} failed: ${res.status} ${await res.text()}`);
    const data = await res.json();
    return Array.isArray(data) ? data : data.records || data.data || [data];
  }

  // Supabase uses query-string filters; generic API uses /:id path.
  const updatePath = isSupabase
    ? (id) => `?${idField.name}=eq.${id}`
    : (id) => `/${id}`;
  const listSuffix = isSupabase ? `?order=${idField.name}.desc` : '';

  return {
    async loadAll()           { return request(listSuffix); },
    async insert(record)      { return (await request('', 'POST', record))[0] || record; },
    async update(id, changes) { return (await request(updatePath(id), 'PATCH', changes))[0] || null; },
    nextId()                  { return undefined; },
  };
}

// ---------------------------------------------------------------------------
// Store factory
// ---------------------------------------------------------------------------

function createStore() {
  if (DB_TARGET === 'supabase' || DB_TARGET === 'api') return createRemoteStore();
  return createJsonStore();
}

const store = createStore();

// ---------------------------------------------------------------------------
// Filter logic
// ---------------------------------------------------------------------------

function applyFilter(records, filter) {
  if (!filter || filter === 'recent') return records.slice(-MAX_RECENT);
  if (filter === 'all') return records;

  const statusName = statusField ? statusField.name : 'status';
  const tsName = timestampField ? timestampField.name : 'created_at';

  // Filter by status name (matches any value in statuses array).
  if (STATUSES.includes(filter)) {
    return records.filter(r => r[statusName] === filter);
  }
  if (filter === 'today') {
    const today = localToday();
    return records.filter(r => (r[tsName] || '').startsWith(today));
  }
  if (filter.startsWith('id:')) {
    const id = parseInt(filter.slice(3), 10);
    return records.filter(r => r[idField.name] === id);
  }
  // Unknown filter → return all.
  return records;
}

// ---------------------------------------------------------------------------
// Commands
// ---------------------------------------------------------------------------

async function cmdAdd() {
  const args = process.argv.slice(3);

  // Map positional args to input fields.
  const record = {};
  for (let i = 0; i < inputFields.length; i++) {
    const f = inputFields[i];
    const val = args[i] || '';
    record[f.name] = coerce(val, f);
  }

  // Auto-fill fields.
  for (const f of fields) {
    if (f.auto === 'increment') {
      const nextId = store.nextId();
      if (nextId !== undefined) record[f.name] = nextId;
    }
    if (f.auto === 'timestamp') {
      record[f.name] = localNow();
    }
    if (f.default && record[f.name] === undefined) {
      record[f.name] = f.type === 'integer' ? parseInt(f.default, 10) : f.default;
    }
  }

  // Validate required fields.
  for (const f of fields) {
    if (f.required && !record[f.name] && record[f.name] !== 0) {
      console.log(JSON.stringify({ ok: false, error: `${f.name} is required` }));
      return;
    }
  }

  const saved = await store.insert(record);
  console.log(JSON.stringify({ ok: true, record: saved }));
}

async function cmdList() {
  const filter = process.argv[3] || 'recent';
  const records = await store.loadAll();
  const filtered = applyFilter(records, filter);
  const result = filtered.slice().reverse();
  console.log(JSON.stringify({ ok: true, scope: 'admin', filter, count: result.length, records: result }));
}

async function cmdListMine() {
  const ownerId = (process.argv[3] || '').trim();
  const filter = process.argv[4] || 'recent';
  if (!ownerId) {
    console.log(JSON.stringify({ ok: false, error: 'list-mine requires owner ID as first arg' }));
    return;
  }
  if (!ownerField) {
    console.log(JSON.stringify({ ok: false, error: 'no owner field defined in schema' }));
    return;
  }
  const all = await store.loadAll();
  const mine = all.filter(r => String(r[ownerField.name]) === ownerId);
  const filtered = applyFilter(mine, filter);
  const result = filtered.slice().reverse();
  console.log(JSON.stringify({
    ok: true,
    scope: 'customer',
    owner_id: ownerId,
    filter,
    count: result.length,
    records: result,
  }));
}

async function cmdSetStatus(statusIndex, label) {
  const id = parseInt(process.argv[3], 10);
  if (!id) { console.log(JSON.stringify({ ok: false, error: `${label} requires record id` })); return; }
  if (!statusField) { console.log(JSON.stringify({ ok: false, error: 'no status field in schema' })); return; }
  const updated = await store.update(id, { [statusField.name]: STATUSES[statusIndex] });
  if (!updated) { console.log(JSON.stringify({ ok: false, error: `record #${id} not found` })); return; }
  console.log(JSON.stringify({ ok: true, record: updated }));
}

async function cmdRevenue() {
  if (!priceField || !statusField) {
    console.log(JSON.stringify({ ok: false, error: 'schema must define price and status roles' }));
    return;
  }
  const sn = statusField.name;
  const all = await store.loadAll();
  const completed = all.filter(r => r[sn] === STATUSES[1]);
  const total = completed.reduce((s, r) => s + (r[priceField.name] || 0), 0);
  console.log(JSON.stringify({
    ok: true,
    total,
    count: completed.length,
    new_count: all.filter(r => r[sn] === STATUSES[0]).length,
    cancelled_count: all.filter(r => r[sn] === STATUSES[2]).length,
  }));
}

// ---------------------------------------------------------------------------
// Update command — set arbitrary field on a record
// ---------------------------------------------------------------------------

async function cmdUpdate() {
  const id = parseInt(process.argv[3], 10);
  const field = process.argv[4];
  const value = process.argv[5];
  if (!id || !field) {
    console.log(JSON.stringify({ ok: false, error: 'usage: update <id> <field> <value>' }));
    return;
  }
  const f = fields.find(sf => sf.name === field);
  if (!f) {
    console.log(JSON.stringify({ ok: false, error: `unknown field: ${field}` }));
    return;
  }
  const updated = await store.update(id, { [field]: coerce(value || '', f) });
  if (!updated) {
    console.log(JSON.stringify({ ok: false, error: `record #${id} not found` }));
    return;
  }
  console.log(JSON.stringify({ ok: true, record: updated }));
}

// ---------------------------------------------------------------------------
// Image gallery commands
// ---------------------------------------------------------------------------

function safePath(base, name) {
  const resolved = path.resolve(base, name);
  if (!resolved.startsWith(base + path.sep) && resolved !== base) return null;
  return resolved;
}

function cmdImages() {
  const folder = process.argv[3];
  if (!folder) { console.log(JSON.stringify({ ok: false, error: 'images requires folder name' })); return; }
  const dir = safePath(IMAGES_DIR, folder);
  if (!dir) { console.log(JSON.stringify({ ok: false, error: 'invalid folder name' })); return; }
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
  if (!fs.existsSync(IMAGES_DIR)) {
    console.log(JSON.stringify({ ok: false, error: 'images dir missing', dir: IMAGES_DIR }));
    return;
  }
  const dirs = fs.readdirSync(IMAGES_DIR)
    .filter(f => fs.statSync(path.join(IMAGES_DIR, f)).isDirectory())
    .sort();
  console.log(JSON.stringify({ ok: true, folders: dirs }));
}

// ---------------------------------------------------------------------------
// Telegram image upload
// ---------------------------------------------------------------------------

function readTelegramBotToken() {
  if (!fs.existsSync(OPENCLAW_CONFIG)) {
    throw new Error('openclaw.json not found — is OpenClaw installed?');
  }
  const cfg = JSON.parse(fs.readFileSync(OPENCLAW_CONFIG, 'utf8'));
  const tg = (cfg.channels || {}).telegram || {};
  if (tg.botToken) return tg.botToken;
  // Multi-account: pick first account with a botToken.
  const accounts = tg.accounts || {};
  for (const id of Object.keys(accounts)) {
    if (accounts[id].botToken) return accounts[id].botToken;
  }
  throw new Error('Telegram botToken not found in openclaw.json');
}

function sendPhotoViaCurl(token, chatId, filePath, caption) {
  const url = `https://api.telegram.org/bot${token}/sendPhoto`;
  const args = ['-s', '-X', 'POST', url,
    '-F', `chat_id=${chatId}`,
    '-F', `photo=@${filePath}`,
  ];
  if (caption) args.push('-F', `caption=${caption}`);
  const res = spawnSync('curl', args, { timeout: 30000, encoding: 'utf8' });
  if (res.error) return { ok: false, error: res.error.message };
  try {
    const body = JSON.parse(res.stdout);
    return body.ok
      ? { ok: true, fileId: (body.result.photo || []).slice(-1)[0]?.file_id }
      : { ok: false, error: body.description || 'Telegram API error' };
  } catch {
    return { ok: false, error: res.stderr || 'curl failed' };
  }
}

function cmdSendImagesTelegram() {
  const folder = process.argv[3];
  const chatId = process.argv[4];
  let count = parseInt(process.argv[5], 10) || 5;
  if (!folder || !chatId) {
    console.log(JSON.stringify({ ok: false, error: 'usage: send-images-telegram <folder> <chat_id> [count]' }));
    return;
  }
  count = Math.min(count, MAX_IMAGES_SEND);
  const dir = safePath(IMAGES_DIR, folder);
  if (!dir) { console.log(JSON.stringify({ ok: false, error: 'invalid folder name' })); return; }
  if (!fs.existsSync(dir)) {
    console.log(JSON.stringify({ ok: false, error: `folder not found: ${folder}` }));
    return;
  }
  const files = fs.readdirSync(dir)
    .filter(f => /\.(jpg|jpeg|png|webp)$/i.test(f))
    .sort()
    .slice(0, count);
  if (!files.length) {
    console.log(JSON.stringify({ ok: false, error: `no images in ${folder}` }));
    return;
  }
  const token = readTelegramBotToken();
  let sent = 0;
  const results = [];
  for (const file of files) {
    const fp = path.join(dir, file);
    const res = sendPhotoViaCurl(token, chatId, fp);
    results.push({ file, ...res });
    if (res.ok) sent++;
  }
  console.log(JSON.stringify({ ok: sent > 0, sent, total: files.length, folder, chat_id: chatId, results }));
}

// ---------------------------------------------------------------------------
// Dispatcher
// ---------------------------------------------------------------------------

const USAGE = `Usage: node cli.js [--table <name>] <command> [args...]

Tables: ${Object.keys(schema.tables).join(', ')} (primary: ${schema.primary})
Current: ${TABLE}

Commands:
  add <${inputFields.map(f => f.name).join('> <')}>
  list [filter]                     Filter: recent|${STATUSES.join('|')}|today|all|id:<N>
  list-mine <owner_id> [filter]
  done <id>
  cancel <id>
  update <id> <field> <value>
  revenue
  tables                            List available tables
  images <folder>
  folders
  send-images-telegram <folder> <chat_id> [count]`;

async function main() {
  const cmd = process.argv[2];
  switch (cmd) {
    case 'add':       await cmdAdd(); break;
    case 'list':      await cmdList(); break;
    case 'list-mine': await cmdListMine(); break;
    case 'done':      await cmdSetStatus(1, 'done'); break;
    case 'cancel':    await cmdSetStatus(2, 'cancel'); break;
    case 'update':    await cmdUpdate(); break;
    case 'revenue':   await cmdRevenue(); break;
    case 'tables':    console.log(JSON.stringify({ ok: true, tables: Object.keys(schema.tables), primary: schema.primary })); break;
    case 'images':    cmdImages(); break;
    case 'folders':   cmdFolders(); break;
    case 'send-images-telegram': cmdSendImagesTelegram(); break;
    default:
      console.error(USAGE);
      process.exit(2);
  }
}

main().catch(err => {
  console.log(JSON.stringify({ ok: false, error: err.message }));
  process.exit(1);
});
