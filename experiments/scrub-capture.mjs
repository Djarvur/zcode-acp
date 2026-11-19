#!/usr/bin/env node
// experiments/scrub-capture.mjs — canonical capture scrubber (D-02).
// The ONLY path into experiments/captures/: raw captures are born OUTSIDE the repo
// and enter the tree only through this script. Tested by scrub-capture.test.mjs.
//
// Usage:
//   node experiments/scrub-capture.mjs <raw-capture> [output] [--redact-path <prefix>]...
//
//   <raw-capture>          path to the raw capture (never inside the repo)
//   [output]               output path (default: stdout); this is where a scrubbed
//                          experiments/captures/<experiment-id>.jsonl lands
//   --redact-path <prefix> repeatable; every occurrence of <prefix> (a private
//                          location supplied at run time, never committed) becomes <path>
//
// Canonical rules (defined once here; every rule has a unit test):
//   line 1 (the capture header) passes through BYTE-IDENTICAL
//   JWT-shaped values (three base64url segments)                      -> [REDACTED-JWT]
//   credential blobs: enc:v2:os: / legacy enc:v1: + long base64ish body -> [REDACTED-CREDENTIAL]
//   long token values under access_token / apiKey / api-key / token   -> [REDACTED-TOKEN]
//   UUID-v4-shaped identifiers -> stable per-file placeholders <uuid-N>
//     (the nth distinct value becomes the deterministic nth placeholder; the same
//      value always maps to the same placeholder — referential integrity for the
//      Phase 3 golden fixtures seeded from these captures)
//   "user_info" payload objects -> [REDACTED-USERINFO] (redacted wholesale)
//   absolute paths under the user's home directory -> tilde-relative form
//   prefixes passed via --redact-path -> <path>
//
// Idempotent: scrubbing already-scrubbed output changes nothing (fixed point).
// Exit 0 on success, including empty input -> empty output.

import { mkdirSync, readFileSync, writeFileSync } from 'node:fs';
import os from 'node:os';
import path from 'node:path';

const REDACT = {
  jwt: '[REDACTED-JWT]',
  cred: '[REDACTED-CREDENTIAL]',
  token: '[REDACTED-TOKEN]',
  userinfo: '[REDACTED-USERINFO]',
  path: '<path>',
};

// Value-shape rules. Each regex matches VALUES only; a bare prefix mention in
// surrounding prose never matches (bodies below minimum lengths are left alone).
const JWT_RX = /eyJ[A-Za-z0-9_-]{10,}\.[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+/g;
const CRED_RX = /enc:v[12]:(os:)?[A-Za-z0-9+/=_-]{20,}/g;
const TOKEN_RX = /("(?:access_token|apiKey|api-key|token)"\s*:\s*")[A-Za-z0-9_-]{20,}(")/g;
const UUID_RX = /[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-4[0-9a-fA-F]{3}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}/g;
const USER_INFO_RX = /"user_info"\s*:\s*\{/g;

// --- UUID placeholder map (per run = per file: stable, deterministic) ---------
const uuidMap = new Map();
function uuidPlaceholder(matched) {
  if (!uuidMap.has(matched)) uuidMap.set(matched, `<uuid-${uuidMap.size + 1}>`);
  return uuidMap.get(matched);
}

// --- user_info wholesale redaction (string-level, string-aware brace matching;
// --- works regardless of whether the line is bare JSON or a wrapped/timestamped
// --- capture line, and keeps the line parseable) --------------------------------
function replaceUserInfos(line) {
  let out = '';
  let rest = line;
  for (;;) {
    const m = USER_INFO_RX.exec(rest);
    if (!m) return out + rest;
    const objStart = m.index + m[0].length - 1; // index of the opening '{'
    let depth = 0;
    let inStr = false;
    let esc = false;
    let end = -1;
    for (let i = objStart; i < rest.length; i++) {
      const ch = rest[i];
      if (inStr) {
        if (esc) esc = false;
        else if (ch === '\\') esc = true;
        else if (ch === '"') inStr = false;
      } else if (ch === '"') {
        inStr = true;
      } else if (ch === '{') {
        depth++;
      } else if (ch === '}') {
        depth--;
        if (depth === 0) {
          end = i;
          break;
        }
      }
    }
    if (end === -1) return out + rest; // truncated line: leave untouched
    out += rest.slice(0, m.index) + '"user_info":"' + REDACT.userinfo + '"';
    rest = rest.slice(end + 1);
  }
}

function scrubLine(line, redactPaths, home) {
  line = replaceUserInfos(line);
  line = line.replace(TOKEN_RX, '$1' + REDACT.token + '$2');
  line = line.replace(CRED_RX, REDACT.cred);
  line = line.replace(JWT_RX, REDACT.jwt);
  line = line.replace(UUID_RX, uuidPlaceholder);
  for (const p of redactPaths) {
    line = line.split(p).join(REDACT.path);
  }
  if (home && home !== '/') {
    line = line.split(home + '/').join('~/');
    line = line.split(home).join('~');
  }
  return line;
}

function usage() {
  process.stderr.write(
    'usage: node experiments/scrub-capture.mjs <raw-capture> [output] [--redact-path <prefix>]...\n',
  );
}

// --- CLI ------------------------------------------------------------------------
const argv = process.argv.slice(2);
const positional = [];
const redactPathArgs = [];
for (let i = 0; i < argv.length; i++) {
  if (argv[i] === '--redact-path') {
    if (i + 1 >= argv.length) {
      usage();
      process.exit(1);
    }
    redactPathArgs.push(argv[++i]);
  } else {
    positional.push(argv[i]);
  }
}
if (positional.length < 1 || positional.length > 2) {
  usage();
  process.exit(1);
}
const [inFile, outFile] = positional;

// Longest prefixes first so an overlapping prefix list cannot partially redact.
const redactPaths = redactPathArgs
  .map((p) => path.resolve(p).replace(/\/+$/, ''))
  .sort((a, b) => b.length - a.length);

let raw;
try {
  raw = readFileSync(inFile, 'utf8');
} catch (err) {
  process.stderr.write(`scrub-capture: cannot read raw capture: ${err.message}\n`);
  process.exit(1);
}

// Empty input -> empty output, exit 0.
if (raw.length === 0) {
  if (outFile) writeFileSync(outFile, '');
  else process.stdout.write('');
  process.exit(0);
}

const home = os.homedir();
const lines = raw.split('\n');
const scrubbed = lines.map((line, i) => (i === 0 ? line : scrubLine(line, redactPaths, home)));
const output = scrubbed.join('\n');

if (outFile) {
  mkdirSync(path.dirname(outFile), { recursive: true });
  writeFileSync(outFile, output);
} else {
  process.stdout.write(output);
}
