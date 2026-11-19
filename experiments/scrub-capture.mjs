#!/usr/bin/env node
// experiments/scrub-capture.mjs — canonical capture scrubber (D-02).
// The ONLY path into experiments/captures/: raw captures are born OUTSIDE the repo
// and enter the tree only through this script. Tested by scrub-capture.test.mjs.
//
// Usage:
//   node experiments/scrub-capture.mjs <raw-capture> [output] [--redact-path <prefix>]... [--no-header]
//
//   <raw-capture>          path to the raw capture (never inside the repo)
//   [output]               output path (default: stdout); this is where a scrubbed
//                          experiments/captures/<experiment-id>.jsonl lands
//   --redact-path <prefix> repeatable; every occurrence of <prefix> (a private
//                          location supplied at run time, never committed) becomes <path>
//   --no-header            treat line 1 as a data line (for legitimately header-less
//                          raw captures); without it, line 1 must be a JSON object
//                          with a "capture" key or the scrubber fails loudly
//
// Canonical rules (defined once here; every rule has a unit test):
//   line 1 (the capture header) passes through BYTE-IDENTICAL (and is validated to
//     be a JSON object with a "capture" key unless --no-header is passed)
//   JWT-shaped values (three base64url segments)                      -> [REDACTED-JWT]
//   credential blobs: enc:v2:os: / legacy enc:v1: + long base64ish body -> [REDACTED-CREDENTIAL]
//   long token values under access_token / apiKey / api-key / token   -> [REDACTED-TOKEN]
//   UUID-v4-shaped identifiers -> stable per-file placeholders <uuid-N>
//     (the nth distinct value becomes the deterministic nth placeholder; the same
//      value always maps to the same placeholder — referential integrity for the
//      Phase 3 golden fixtures seeded from these captures)
//   "user_info" payload objects -> [REDACTED-USERINFO] (redacted wholesale,
//     in both the bare form and the probe's wrapped/escaped envelope form)
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
// Credential bodies may contain separators (dots/colons) inside the payload —
// consume the whole value up to the JSON string boundary, never stop mid-body.
const CRED_RX = /enc:v[12]:(os:)?[A-Za-z0-9+/=_:.-]{20,}/g;
const TOKEN_RX = /("(?:access_token|apiKey|api-key|token)"\s*:\s*")[A-Za-z0-9_-]{20,}(")/g;
const UUID_RX = /[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-4[0-9a-fA-F]{3}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}/g;
// CR-01 (re-review): match both the bare form ("user_info":{) and the escaped
// envelope form the probe writes (\"user_info\":{ — every wire line is stored
// as JSON.stringify({ ts, line }), so the wire JSON arrives as an escaped
// string and the key's quotes carry backslashes). The capture groups carry the
// backslashes (if any) so the replacement can mirror the escaping.
const USER_INFO_RX = /(\\?)"user_info(\\?)"\s*:\s*\{/g;

// --- UUID placeholder map (per run = per file: stable, deterministic) ---------
const uuidMap = new Map();
function uuidPlaceholder(matched) {
  if (!uuidMap.has(matched)) uuidMap.set(matched, `<uuid-${uuidMap.size + 1}>`);
  return uuidMap.get(matched);
}

// --- user_info wholesale redaction (string-level, string-aware brace matching;
// --- works regardless of whether the line is bare JSON or the wrapped/
// --- timestamped capture line the probe writes — JSON.stringify({ ts, line }) —
// --- where the wire JSON arrives as an ESCAPED envelope string and the key
// --- looks like \"user_info\":{; keeps the line parseable in both forms) --------
function replaceUserInfos(line) {
  let out = '';
  let rest = line;
  for (;;) {
    // CR-01: USER_INFO_RX is module-level with the /g flag — its lastIndex
    // persists across calls and is in the PREVIOUS string's coordinates, so a
    // stale value skips every user_info after the first on a line (the second
    // leaks verbatim) and breaks the fixed-point contract. Search the current
    // suffix from its start every iteration; rest begins right after the
    // previously consumed object, so a match can never overlap what is already
    // redacted.
    USER_INFO_RX.lastIndex = 0;
    const m = USER_INFO_RX.exec(rest);
    if (!m) return out + rest;
    // CR-01 (re-review): m[1] non-empty means the match sits inside the probe's
    // envelope form (\"user_info\":{). The replacement must mirror that
    // escaping, or the bare quotes it inserts would terminate the envelope
    // string early and corrupt the line.
    const escaped = m[1] === '\\';
    const objStart = m.index + m[0].length - 1; // index of the opening '{'
    const end = findObjectEnd(rest, objStart, escaped);
    if (end === -1) return out + rest; // truncated line: leave untouched
    out +=
      rest.slice(0, m.index) +
      (m[1] || '') +
      '"user_info' +
      (m[2] || '') +
      '":' +
      (m[2] || '') +
      '"' +
      REDACT.userinfo +
      (m[2] || '') +
      '"';
    rest = rest.slice(end + 1);
  }
}

// String-aware closing-brace search for the "user_info" object opened at
// rest[objStart]. escaped=false scans bare JSON (quotes toggle strings; a
// backslash inside a string escapes the next character). escaped=true scans
// the probe's envelope form, where every quote of the inner wire JSON arrives
// as an escaped pair (\") — CR-01: a bare-quote toggle there would read the
// whole inner JSON as one giant string and never find the closing brace (an
// escaped quote is never a string boundary), so the pairs are decoded level by
// level: an envelope-level \\ / \" pair yields the wire character it
// represents, and that wire character then feeds the ordinary string/brace
// logic (a decoded wire backslash escapes the NEXT decoded wire character; a
// decoded wire quote opens/closes a wire string).
function findObjectEnd(rest, objStart, escaped) {
  let depth = 0;
  let inStr = false; // inside a JSON string at the level being scanned
  let esc = false; // bare mode: the previous character was a backslash
  let wireEsc = false; // escaped mode: the previously decoded wire character was a backslash
  for (let i = objStart; i < rest.length; i++) {
    const ch = rest[i];
    if (!escaped) {
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
        if (depth === 0) return i;
      }
      continue;
    }
    if (inStr) {
      if (ch === '"') return -1; // bare quote inside the envelope string: envelope ended mid-object — leave untouched
      if (ch === '\\') {
        // envelope-level escape pair: \\ or \" decodes to the wire character it represents
        const wire = rest[i + 1] === '\\' ? '\\' : rest[i + 1] === '"' ? '"' : null;
        if (wire === null) return -1; // not an envelope-level escape pair: malformed — leave untouched
        if (wireEsc) wireEsc = false; // escaped wire character: content
        else if (wire === '\\') wireEsc = true; // wire-level escape begins: the next wire character is content
        else inStr = false; // decoded wire quote closes the wire string
        i++; // consume both bytes of the pair (the loop adds the second step)
        continue;
      }
      continue; // any other byte is wire content
    }
    if (ch === '\\' && rest[i + 1] === '"') {
      inStr = true; // envelope-level \" opens a wire string
      wireEsc = false;
      i++;
      continue;
    }
    if (ch === '"') return -1; // same anomaly, outside a wire string
    if (ch === '{') depth++;
    else if (ch === '}') {
      depth--;
      if (depth === 0) return i;
    }
  }
  return -1; // truncated line: leave untouched
}

// Escape regex metacharacters so a literal path can be used inside a RegExp.
function escapeRegExp(s) {
  return s.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
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
    // Step 1 (WR-03): home + '/' + rest -> '~/ + rest' stays unconditional.
    line = line.split(home + '/').join('~/');
    // Step 2 (WR-03, boundary-aware): replace a BARE home occurrence only when it
    // is NOT followed by an optional '/' plus a name character. A path that merely
    // shares the home prefix (home followed directly by a name character — a
    // sibling directory) must pass through verbatim, never be falsified into a
    // tilde path.
    line = line.replace(new RegExp(escapeRegExp(home) + '(?!/?[A-Za-z0-9._-])', 'g'), '~');
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
let noHeader = false;
for (let i = 0; i < argv.length; i++) {
  if (argv[i] === '--redact-path') {
    if (i + 1 >= argv.length) {
      usage();
      process.exit(1);
    }
    redactPathArgs.push(argv[++i]);
  } else if (argv[i] === '--no-header') {
    noHeader = true;
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

// IN-04: line 1 of a capture is the header — validate it before trusting the rest,
// so a malformed or misaligned capture fails loudly instead of passing an
// unvalidated first line through. --no-header is the escape hatch for legitimately
// header-less raw captures (line 1 is then scrubbed like any data line).
if (!noHeader) {
  let header = null;
  try {
    header = JSON.parse(lines[0]);
  } catch {
    header = null;
  }
  if (
    header === null ||
    typeof header !== 'object' ||
    !Object.prototype.hasOwnProperty.call(header, 'capture')
  ) {
    process.stderr.write(
      `scrub-capture: ${inFile}: line 1 is not a capture header (expected a JSON object with a "capture" key); pass --no-header for header-less raw captures\n`,
    );
    process.exit(1);
  }
}

const scrubbed = lines.map((line, i) => (i === 0 && !noHeader ? line : scrubLine(line, redactPaths, home)));
const output = scrubbed.join('\n');

if (outFile) {
  mkdirSync(path.dirname(outFile), { recursive: true });
  writeFileSync(outFile, output);
} else {
  process.stdout.write(output);
}
