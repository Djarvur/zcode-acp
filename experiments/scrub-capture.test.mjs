// experiments/scrub-capture.test.mjs — unit tests for every canonical scrub rule
// (D-02; the scrubber experiments/scrub-capture.mjs is the only path into
// experiments/captures/). Run: node --test experiments/
//
// IMPORTANT: forbidden shapes (JWT values, credential-blob payloads, long token
// values, UUIDs) are constructed here by STRING CONCATENATION so that no forbidden
// literal appears verbatim in this tracked file — `bash experiments/scan-secrets.sh`
// must stay green with this file committed.

import { test } from 'node:test';
import assert from 'node:assert/strict';
import { spawnSync } from 'node:child_process';
import { mkdtempSync, writeFileSync, readFileSync } from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import { tmpdir } from 'node:os';
import { fileURLToPath } from 'node:url';

const script = path.join(path.dirname(fileURLToPath(import.meta.url)), 'scrub-capture.mjs');

// --- fixtures (concatenated: no forbidden literal in tracked source) ----------
const JWT_FIXTURE = 'ey' + 'J' + 'abcDEFghij' + '.' + 'pQrsTuv01x' + '.' + 'wXyZ12345_y';
const CRED_V2 = 'enc:v' + '2:os:' + 'Aa09+/=_Bb11Cc22Dd33Ee44';
const CRED_V1 = 'enc:v' + '1:' + 'Zz88-99_Xx77__Qq66Ww55Vv44';
const TOKEN_A = 't' + 'k9' + 'E'.repeat(20); // 22 class chars
const TOKEN_B = 'k'.repeat(24);
const UUID_A = '0123ab' + 'cd-' + '45ef' + '-4' + 'c01-' + '89ab' + '-cdef01234567';
const UUID_B = '9f8e7d' + '6c-' + '12ab' + '-4' + 'd02-' + '9e8d' + '-123456abcdef';
// Byte-identity probe: odd spacing that re-serialization would destroy.
const HEADER =
  '{"capture": "EXP-SCRUBTEST-01",  "date":"2026-11-19T00:00:00Z","desktopVersion":"3.18.2","builtinRevision":"44","host":"test-host"}';

// --- harness ------------------------------------------------------------------
function runScrub(inputText, extraArgs = []) {
  const dir = mkdtempSync(path.join(tmpdir(), 'scrub-test-'));
  const inFile = path.join(dir, 'raw.jsonl');
  const outFile = path.join(dir, 'scrubbed.jsonl');
  writeFileSync(inFile, inputText);
  const res = spawnSync(process.execPath, [script, inFile, outFile, ...extraArgs], {
    encoding: 'utf8',
  });
  let out = null;
  try {
    out = readFileSync(outFile, 'utf8');
  } catch {
    out = null;
  }
  return { res, out };
}

// Defensive extractor: returns the parsed nth (0-based) line or null — so a missing
// scrubber fails tests on ASSERTIONS (null vs expected), never on parse errors.
function dataLine(out, n = 1) {
  const lines = (out ?? '').split('\n');
  try {
    return JSON.parse(lines[n] ?? '');
  } catch {
    return null;
  }
}

// --- the canonical rules -------------------------------------------------------

test('JWT-shaped token values are redacted and the line stays parseable JSON', () => {
  const input =
    HEADER + '\n' + JSON.stringify({ id: 1, result: { authHeader: 'Bearer ' + JWT_FIXTURE } }) + '\n';
  const { out } = runScrub(input);
  const parsed = dataLine(out);
  assert.equal(parsed?.result?.authHeader, 'Bearer [REDACTED-JWT]');
  assert.ok(!(out ?? '').includes(JWT_FIXTURE), 'raw JWT shape must not survive');
  assert.ok(!/eyJ/.test(out ?? ''), 'no JWT prefix may remain');
});

test('credential blobs with the enc:v2:os: prefix are fully redacted including the prefix', () => {
  const input = HEADER + '\n' + JSON.stringify({ id: 2, result: { blob: CRED_V2 } }) + '\n';
  const { out } = runScrub(input);
  const parsed = dataLine(out);
  assert.equal(parsed?.result?.blob, '[REDACTED-CREDENTIAL]');
  assert.ok(!(out ?? '').includes('enc:v'), 'prefix itself must be gone');
});

test('legacy enc:v1: credential blobs are fully redacted including the prefix', () => {
  const input = HEADER + '\n' + JSON.stringify({ id: 3, result: { blob: CRED_V1 } }) + '\n';
  const { out } = runScrub(input);
  const parsed = dataLine(out);
  assert.equal(parsed?.result?.blob, '[REDACTED-CREDENTIAL]');
  assert.ok(!(out ?? '').includes('enc:v'), 'legacy prefix itself must be gone');
});

test('credential blob bodies containing dots/colons are redacted WHOLE (no residue after the placeholder)', () => {
  // Real enc:v2 values can carry separators inside the body; the rule must consume
  // the entire value up to the JSON string boundary, not stop at the first dot.
  const CRED_DOTTED = 'enc:v' + '2:os:' + 'Aa09+/=_Bb11.Cc22:Dd33Ee44Ff55Gg66.' + 'Hh77Ii88Jj99';
  const input = HEADER + '\n' + JSON.stringify({ id: 31, result: { blob: CRED_DOTTED } }) + '\n';
  const { out } = runScrub(input);
  const parsed = dataLine(out);
  assert.equal(parsed?.result?.blob, '[REDACTED-CREDENTIAL]');
  assert.ok(!(out ?? '').includes('Cc22'), 'body fragment after an inner dot must not survive');
  assert.ok(!(out ?? '').includes('Hh77'), 'body tail must not survive');
});

test('long token values under credential keys are redacted', () => {
  const input =
    HEADER +
    '\n' +
    JSON.stringify({
      id: 4,
      result: {
        access_token: TOKEN_A,
        apiKey: TOKEN_B,
        'api-key': TOKEN_B,
        token: TOKEN_A,
        keep: 'short-value',
      },
    }) +
    '\n';
  const { out } = runScrub(input);
  const parsed = dataLine(out);
  assert.equal(parsed?.result?.access_token, '[REDACTED-TOKEN]');
  assert.equal(parsed?.result?.apiKey, '[REDACTED-TOKEN]');
  assert.equal(parsed?.result?.['api-key'], '[REDACTED-TOKEN]');
  assert.equal(parsed?.result?.token, '[REDACTED-TOKEN]');
  assert.equal(parsed?.result?.keep, 'short-value', 'short non-credential values stay');
  assert.ok(!(out ?? '').includes(TOKEN_A) && !(out ?? '').includes(TOKEN_B));
});

test('UUID-shaped identifiers become stable per-file placeholders (nth distinct -> nth)', () => {
  const input =
    HEADER +
    '\n' +
    JSON.stringify({ id: 5, result: { attemptId: UUID_A, ref: UUID_A, other: UUID_B } }) +
    '\n';
  const { out } = runScrub(input);
  const parsed = dataLine(out);
  assert.equal(parsed?.result?.attemptId, '<uuid-1>', 'first distinct value -> first placeholder');
  assert.equal(parsed?.result?.ref, '<uuid-1>', 'same value always maps to the same placeholder');
  assert.equal(parsed?.result?.other, '<uuid-2>', 'second distinct value -> second placeholder');
  assert.ok(!(out ?? '').includes(UUID_A) && !(out ?? '').includes(UUID_B));
});

test('absolute paths under the user home directory become tilde-relative', () => {
  const home = os.homedir();
  const input =
    HEADER +
    '\n' +
    JSON.stringify({ id: 6, result: { cwd: home + '/proj/e3', bare: home } }) +
    '\n';
  const { out } = runScrub(input);
  const parsed = dataLine(out);
  assert.equal(parsed?.result?.cwd, '~' + '/proj/e3');
  assert.equal(parsed?.result?.bare, '~');
  assert.ok(!(out ?? '').includes(home), 'home prefix must not survive');
});

test('the header line (line 1) passes through byte-identical', () => {
  const input = HEADER + '\n' + JSON.stringify({ id: 7, result: { authHeader: JWT_FIXTURE } }) + '\n';
  const { out } = runScrub(input);
  assert.equal((out ?? '').split('\n')[0], HEADER, 'header must not be re-serialized or scrubbed');
});

test('scrubbing is idempotent: scrubbing already-scrubbed output changes nothing', () => {
  const input =
    HEADER +
    '\n' +
    JSON.stringify({
      id: 8,
      result: { authHeader: JWT_FIXTURE, blob: CRED_V2, token: TOKEN_A, u: UUID_A, cwd: os.homedir() + '/w' },
    }) +
    '\n';
  const first = runScrub(input);
  assert.ok(first.out !== null && first.out.length > 0, 'first pass must produce output');
  const second = runScrub(first.out);
  assert.equal(second.out, first.out, 'second scrub pass must be a fixed point');
});

test('empty input produces empty output and exit 0', () => {
  const { res, out } = runScrub('');
  assert.equal(res.status, 0, 'expected exit 0, saw: ' + (res.stderr || '').slice(0, 200));
  assert.equal(out, '');
});

test('user_info payload objects are redacted wholesale', () => {
  const input =
    HEADER +
    '\n' +
    JSON.stringify({
      method: 'state.updated',
      params: { user_info: { email: 'x' + '@' + 'y.z', nested: { deep: true } }, sequence: 3 },
    }) +
    '\n';
  const { out } = runScrub(input);
  const parsed = dataLine(out);
  assert.equal(parsed?.params?.user_info, '[REDACTED-USERINFO]');
  assert.ok(!(out ?? '').includes('email'), 'no field of the payload may survive');
});

// --- CR-01: USER_INFO_RX lastIndex regression ------------------------------------
// USER_INFO_RX is a module-level /g regex driven by exec() in a loop over
// progressively shorter suffixes of the line; lastIndex persists in the
// PREVIOUS string's coordinates, so every user_info object after the first on
// a line is skipped (the second leaks verbatim) and pass 2 redacts it — the
// output is not a fixed point, violating the documented idempotency contract.

test('CR-01: two user_info objects on ONE line are both redacted in a single pass', () => {
  const input =
    HEADER +
    '\n' +
    JSON.stringify({
      method: 'state.updated',
      params: {
        events: [
          { user_info: { email: 'first' + '@' + 'example.test', marker: 'FIRST_USERINFO_MARKER' } },
          { user_info: { email: 'second' + '@' + 'example.test', marker: 'SECOND_USERINFO_MARKER' } },
        ],
      },
    }) +
    '\n';
  const { out } = runScrub(input);
  const parsed = dataLine(out);
  assert.equal(parsed?.params?.events?.[0]?.user_info, '[REDACTED-USERINFO]', 'first user_info must be redacted');
  assert.equal(
    parsed?.params?.events?.[1]?.user_info,
    '[REDACTED-USERINFO]',
    'the SECOND user_info on the same line must be redacted in the same pass',
  );
  assert.ok(!(out ?? '').includes('FIRST_USERINFO_MARKER'), 'first payload must not survive');
  assert.ok(!(out ?? '').includes('SECOND_USERINFO_MARKER'), 'second payload must not survive verbatim');
});

test('CR-01: scrubbing a multi-user_info line is a fixed point (re-scrub is byte-identical)', () => {
  const input =
    HEADER +
    '\n' +
    JSON.stringify({
      method: 'state.updated',
      params: {
        events: [
          { user_info: { email: 'first' + '@' + 'example.test' } },
          { user_info: { email: 'second' + '@' + 'example.test' } },
        ],
      },
    }) +
    '\n';
  const first = runScrub(input);
  assert.ok(first.out !== null && first.out.length > 0, 'first pass must produce output');
  const second = runScrub(first.out);
  assert.equal(second.out, first.out, 'second scrub pass must be a fixed point');
});

test('path prefixes passed via --redact-path become the <path> placeholder', () => {
  const input =
    HEADER + '\n' + JSON.stringify({ id: 9, result: { note: '/tmp' + '/scrub-private' + '/notes/f.md' } }) + '\n';
  const { out } = runScrub(input, ['--redact-path', '/tmp' + '/scrub-private']);
  const parsed = dataLine(out);
  assert.equal(parsed?.result?.note, '<path>' + '/notes/f.md');
});

// --- WR-03: boundary-aware home rewrite ----------------------------------------
// A path that merely SHARES A PREFIX with the home directory (home followed
// directly by a name character) is a different directory — it must pass through
// scrubbing verbatim, never be falsified into a tilde path. Sibling paths are
// built dynamically from os.homedir() (never literal machine paths).

test('WR-03: a sibling path sharing only the home prefix survives verbatim (home + name + /segment)', () => {
  const home = os.homedir();
  const sibling = home + 'x' + '/otherproj/file.txt';
  const input = HEADER + '\n' + JSON.stringify({ id: 10, result: { cwd: sibling } }) + '\n';
  const { out } = runScrub(input);
  const parsed = dataLine(out);
  assert.equal(parsed?.result?.cwd, sibling, 'a path that merely shares the home prefix must pass through verbatim');
  assert.ok(!(out ?? '').includes('~x'), 'the falsified tilde+suffix rewrite must never appear');
});

test('WR-03: a bare sibling of the home directory survives verbatim (home + name)', () => {
  const home = os.homedir();
  const sibling = home + 'x';
  const input = HEADER + '\n' + JSON.stringify({ id: 11, result: { cwd: sibling } }) + '\n';
  const { out } = runScrub(input);
  const parsed = dataLine(out);
  assert.equal(parsed?.result?.cwd, sibling, 'home + one name character is a different directory — verbatim');
  assert.ok(!(out ?? '').includes('~x'), 'the falsified tilde rewrite must never appear');
});

test('WR-03: the boundary rewrite is idempotent — scrubbing scrubbed sibling output is a fixed point', () => {
  const home = os.homedir();
  const input =
    HEADER +
    '\n' +
    JSON.stringify({ id: 12, result: { cwd: home + 'x' + '/otherproj/file.txt', bare: home } }) +
    '\n';
  const first = runScrub(input);
  assert.ok(first.out !== null && first.out.length > 0, 'first pass must produce output');
  const second = runScrub(first.out);
  assert.equal(second.out, first.out, 'second scrub pass must be a fixed point');
});

// --- IN-04: capture-header validation on line 1 --------------------------------
// Line 1 of a capture must parse as JSON with a `capture` key; otherwise the
// scrubber fails loudly instead of passing an unvalidated header through.
// --no-header is the escape hatch for legitimately header-less raw captures.

test('IN-04: a first line that is not JSON fails loudly (non-zero exit, message names the problem)', () => {
  const input = 'not json at all\n' + JSON.stringify({ id: 13, note: 'plain data line' }) + '\n';
  const { res, out } = runScrub(input);
  assert.notEqual(res.status, 0, 'header-less non-JSON input must not exit 0');
  assert.match(res.stderr ?? '', /capture/, 'stderr must name the failed capture-header contract');
  assert.ok(out === null || out === '', 'no scrubbed output is written for invalid input');
});

test('IN-04: a JSON first line without a capture key fails loudly', () => {
  const input =
    JSON.stringify({ date: '2026-11-19T00:00:00Z', desktopVersion: '3.18.2' }) +
    '\n' +
    JSON.stringify({ id: 14, note: 'plain data line' }) +
    '\n';
  const { res, out } = runScrub(input);
  assert.notEqual(res.status, 0, 'JSON line 1 without a capture key must not exit 0');
  assert.match(res.stderr ?? '', /capture/, 'stderr must name the missing capture-key contract');
  assert.ok(out === null || out === '', 'no scrubbed output is written for invalid input');
});

test('IN-04: --no-header scrubs line 1 like a data line and exits 0', () => {
  const home = os.homedir();
  const input =
    JSON.stringify({ method: 'state.updated', params: { cwd: home + '/proj/e3' } }) +
    '\n' +
    JSON.stringify({ id: 15, note: 'plain data line' }) +
    '\n';
  const { res, out } = runScrub(input, ['--no-header']);
  assert.equal(res.status, 0, 'expected exit 0 with --no-header, saw: ' + (res.stderr || '').slice(0, 200));
  const firstLine = dataLine(out, 0);
  assert.equal(firstLine?.params?.cwd, '~' + '/proj/e3', 'line 1 must be scrubbed like a data line');
});
