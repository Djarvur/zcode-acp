#!/usr/bin/env node
// experiments/app-server-probe.mjs — scenario-flow spawn/JSONL harness (Pattern 2,
// validated live against the shipped 3.18.2 binary during phase research).
//
// NEVER wire this into CI: it spawns the real desktop bundle and (for the turn
// flow) spends account quota — see experiments/README.md ("Ground rules").
// No experiment here ever mutates account state: there is deliberately no
// key-minting endpoint call anywhere in this file.
//
// Usage:
//   node experiments/app-server-probe.mjs \
//     --electron <bundle-root>/zcode \
//     --cjs <bundle-root>/resources/glm/zcode.cjs \
//     --builtin <builtin-config-file> \
//     --flow <flow-name> \
//     --raw-out <scratch-path-outside-repo> \
//     --window-ms 30000 [--quiet-ms 2500] [--workspace-dir <dir>]
//
// Behavior:
//   - spawns `<electron> <cjs> app-server --stdio` with ELECTRON_RUN_AS_NODE=1 and
//     ZCODE_BUILTIN_PROVIDER_CONFIG_FILE=<builtin>, stdio pipes, own process group;
//   - HOLDS STDIN OPEN until the collection window ends — writing a request and
//     closing stdin early makes the app-server exit cleanly BEFORE answering
//     (exit 0 + empty capture is the warning sign);
//   - appends timestamped stdout lines to the raw file (one JSON object per wire
//     line: {"ts":"<ISO>","line":"<raw wire line>"});
//   - mirrors child stderr to this process's stderr, capped at 64 KiB;
//   - answers server->host REQUESTS (the app-server is host-managed):
//     session/requestRuntimePreferences gets a minimal preferences result,
//     everything else gets -32601; every reply is recorded as a hostSent line;
//   - the FIRST line of the raw file is the capture header JSON with the builtin
//     revision READ AT CAPTURE TIME (stale-evidence guard against desktop
//     auto-update mid-phase);
//   - clean shutdown: stdin.end(), then a process-group SIGKILL backstop.
//
// Machine paths arrive as CLI arguments only — nothing machine-private is
// hardcoded here, and a --raw-out INSIDE this repository is refused (raw captures
// are born outside the repo; experiments/scrub-capture.mjs is the only path into
// experiments/captures/).

import { spawn } from 'node:child_process';
import { appendFileSync, readFileSync, writeFileSync } from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import { dirname } from 'node:path';
import { fileURLToPath } from 'node:url';

const SCRIPT_DIR = dirname(fileURLToPath(import.meta.url));
const REPO_ROOT = path.resolve(SCRIPT_DIR, '..');
const STDERR_CAP_BYTES = 64 * 1024;
const GRACE_MS = 4000; // between stdin.end() and the SIGKILL backstop

// Host-obligation replies. The app-server is host-managed: some server->host
// REQUESTS (method + id) must be answered or the internal protocol client closes
// and in-flight calls fail with -32603. Minimal answers per the 3.17.x reference
// host fixture (mcp-apps-host-fixture/agent.ts) and the strict result schema
// (zcodeSessionRuntimePreferencesResultSchema — the remaining fields default at
// the protocol parse boundary). Unknown host requests get -32601, like the
// fixture. All replies are recorded into the raw capture (hostSent lines).
function hostReplyFor(method) {
  if (method === 'session/requestRuntimePreferences') {
    return {
      result: {
        nativeSearchEnhancementsEnabled: false,
        memoryEnabled: false,
      },
    };
  }
  return { error: { code: -32601, message: `probe harness: unsupported host request: ${method}` } };
}

// ---------------------------------------------------------------------------
// Runtime state shared with the flow steps. Steps may be static
// ({ atMs, msg }) or dynamic ({ atMs, make(state) }) — the latter are
// evaluated at fire time so a probe can reference values learned from
// earlier replies (e.g. the sessionId out of a session/create result).
// ---------------------------------------------------------------------------
const state = {
  replies: {}, // id -> parsed reply wire object (result OR error)
  sessionId: null, // first result.session.sessionId observed
  createModel: null, // {providerId, modelId} out of the create reply (session default)
  notifications: [], // { method, keys } per server->host notification line
  seqKeyHits: {}, // sequence-ish top-level key -> times seen in notifications
  stateUpdatedPatchShapes: [], // unique JSON key-sets of state.updated params.patch
  sendReplyTs: null, // when the session/send reply arrived (latency-floor anchor)
  turnSettledTs: null, // when the session last LEFT status "running" per state.updated
  turnStatusSeen: [], // distinct state.updated patch.status values, in arrival order
  endReason: null, // quiet-window | window-expired | child-exit
};
const SEQ_WATCH_KEYS = ['seq', 'sequence', 'eventSeq', 'sequenceNumber', 'revision', 'afterSeq'];

function observeReply(wire) {
  if (wire == null || typeof wire !== 'object') return;
  // IN-04 (verdict-falsification guard): a frame carrying `method` is a
  // server->host REQUEST (method + id) or a notification (method, no id) — never
  // a reply; replies are {"id","result"|"error"} without a method
  // (docs/protocol/app-server-dialect.md §1, envelope grammar). Recorded wire
  // truth shows reverse requests arrive with string ids ("server-1", …) that
  // cannot collide with the numeric probe ids — but the id scheme is
  // server-owned, not a contract, so the shape check is the real guard: storing
  // a request in state.replies could shadow the real reply for a colliding id
  // (classifyReply would report "accepted" for a call the server never answered)
  // or anchor sendReplyTs early and open the quiet-window latency floor
  // prematurely. Only method-less frames are candidate replies.
  if (typeof wire.method === 'string') return;
  if (wire.id != null) state.replies[wire.id] = wire;
  if (PROBED_METHODS[wire.id] === 'session/send' && state.sendReplyTs == null) {
    state.sendReplyTs = Date.now();
  }
  const sess = wire && wire.result && wire.result.session;
  if (sess && typeof sess.sessionId === 'string' && !state.sessionId) {
    state.sessionId = sess.sessionId;
  }
  if (sess && sess.model && typeof sess.model === 'object' && !state.createModel) {
    state.createModel = { providerId: sess.model.providerId, modelId: sess.model.modelId };
  }
}

function observeNotification(wire) {
  if (wire == null || typeof wire !== 'object' || typeof wire.method !== 'string') return;
  const keys = wire.params && typeof wire.params === 'object' ? Object.keys(wire.params) : [];
  state.notifications.push({ method: wire.method, keys });
  for (const k of keys) {
    if (SEQ_WATCH_KEYS.includes(k)) state.seqKeyHits[k] = (state.seqKeyHits[k] || 0) + 1;
  }
  if (wire.method === 'state.updated' && wire.params.patch && typeof wire.params.patch === 'object') {
    const shape = JSON.stringify(Object.keys(wire.params.patch).sort());
    if (!state.stateUpdatedPatchShapes.includes(shape)) state.stateUpdatedPatchShapes.push(shape);
    if (wire.params.patch.status !== undefined && !state.turnStatusSeen.includes(wire.params.patch.status)) {
      state.turnStatusSeen.push(wire.params.patch.status);
    }
    if (wire.params.patch.status !== undefined && wire.params.patch.status !== 'running' && state.sendReplyTs != null) {
      state.turnSettledTs = Date.now();
    }
  }
}

// Verdict line for one probed request id: an accepted result, a refused error
// (code + message — a refusal IS the verdict), or no reply within the window.
function classifyReply(id, method) {
  const wire = state.replies[id];
  if (!wire) return { method, probeId: id, verdict: 'no-reply' };
  if (wire.error) {
    return {
      method,
      probeId: id,
      verdict: 'refused',
      errorCode: wire.error.code,
      errorMessage: String(wire.error.message || '').slice(0, 300),
    };
  }
  const keys = wire.result && typeof wire.result === 'object' ? Object.keys(wire.result) : [];
  return { method, probeId: id, verdict: 'accepted', resultKeys: keys };
}

function appendVerdicts(rawOut, flowName) {
  const ts = new Date().toISOString();
  appendFileSync(
    rawOut,
    JSON.stringify({
      ts,
      verdict: {
        kind: 'per-method',
        probed: Object.entries(PROBED_METHODS).map(([id, method]) => classifyReply(Number(id), method)),
      },
    }) + '\n',
  );
  const notifCounts = {};
  for (const n of state.notifications) notifCounts[n.method] = (notifCounts[n.method] || 0) + 1;
  appendFileSync(
    rawOut,
    JSON.stringify({
      ts,
      verdict: {
        kind: 'surface-summary',
        sessionCreateAccepted: state.sessionId != null,
        envelopeMethodsObserved: notifCounts,
        sequenceKeyPresence: state.seqKeyHits,
      },
    }) + '\n',
  );
  if (flowName === 'turn') {
    // Turn-completion terminal-signal summary (E2): which payloads arrived at the
    // end of the turn, whether permission/user-input interactions fired (expected
    // no for a no-tool turn), and the state.updated patch shapes seen.
    const turnFamily = {};
    for (const n of state.notifications) {
      if (/turn|complet|fail/i.test(n.method)) turnFamily[n.method] = (turnFamily[n.method] || 0) + 1;
    }
    const permission = {};
    for (const n of state.notifications) {
      if (/^interaction\/(requestPermission|requestUserInput)$/.test(n.method)) {
        permission[n.method] = (permission[n.method] || 0) + 1;
      }
    }
    const stateUpdatedCount = (notifCounts['state.updated'] || 0);
    appendFileSync(
      rawOut,
      JSON.stringify({
        ts,
        verdict: {
          kind: 'turn-summary',
          endReason: state.endReason,
          setModel: classifyReply(4, 'session/setModel'),
          send: classifyReply(5, 'session/send'),
          turnCompletionFamily: turnFamily,
          interactionRequests: permission,
          stateUpdated: { count: stateUpdatedCount, patchKeyShapes: state.stateUpdatedPatchShapes },
          turnStatusSeen: state.turnStatusSeen,
        },
      }) + '\n',
    );
  }
}

// Probed request ids -> method names. Flows that probe specific methods list
// them here so the per-method verdict lines are emitted even if the flow's
// dynamic step never fired (recorded honestly as no-reply).
const PROBED_METHODS = {};

// ---------------------------------------------------------------------------
// Flow registry — the dispatch seam. Plans 01-02 add `subscribe` and `turn`
// here; each flow returns a list of { atMs, msg } or { atMs, make } steps.
// ---------------------------------------------------------------------------
const FLOWS = {
  probe(ctx) {
    return [
      { atMs: 1500, msg: { id: 1, method: 'provider/refreshRegistry', params: { reason: 'probe' } } },
    ];
  },

  // E3 (DISC-04): path-as-key accepted, then an arbitrary host-chosen string key.
  // Registry-level only — no turn, no quota.
  workspacekey(ctx) {
    const ws = ctx.workspaceDir;
    return [
      {
        atMs: 1500,
        msg: { id: 1, method: 'session/create', params: { workspace: { workspacePath: ws, workspaceKey: ws } } },
      },
      {
        atMs: 4000,
        msg: {
          id: 2,
          method: 'session/create',
          params: { workspace: { workspacePath: ws, workspaceKey: 'probe-arbitrary-host-key-01' } },
        },
      },
    ];
  },

  // E1 (DISC-01): session/subscribe / v4/conversation/subscribe / session/events
  // surface probe. Registry-level only — NO session/send anywhere in this flow
  // (an idle window after subscribe is the point: whatever event envelopes the
  // server pushes on its own are the evidence). Create params follow the
  // 3.17.x zcodeSessionCreateParamsSchema (workspace required; mode/persistence
  // optional) re-read before this run; subscribe params follow
  // zcodeSessionSubscribeParamsSchema {sessionId, deliveryKind, afterSeq?,
  // includeSnapshot?}. The sessionId comes from the create reply at fire time.
  // A refusal is itself the verdict — the error reply is recorded with its code.
  'subscribe'(ctx) {
    const ws = ctx.workspaceDir;
    PROBED_METHODS[3] = 'session/subscribe';
    PROBED_METHODS[4] = 'v4/conversation/subscribe';
    PROBED_METHODS[5] = 'session/events';
    return [
      { atMs: 1500, msg: { id: 1, method: 'provider/refreshRegistry', params: { reason: 'probe' } } },
      {
        atMs: 5000,
        msg: {
          id: 2,
          method: 'session/create',
          params: {
            workspace: { workspacePath: ws, workspaceKey: ws },
            mode: 'build',
            persistence: 'immediate',
          },
        },
      },
      {
        atMs: 8500,
        make: (st) =>
          st.sessionId
            ? { id: 3, method: 'session/subscribe', params: { sessionId: st.sessionId, deliveryKind: 'desktop-continuous' } }
            : null,
      },
      {
        atMs: 11000,
        make: (st) =>
          st.sessionId
            ? { id: 4, method: 'v4/conversation/subscribe', params: { sessionId: st.sessionId, deliveryKind: 'desktop-continuous' } }
            : null,
      },
      {
        atMs: 13000,
        make: (st) => (st.sessionId ? { id: 5, method: 'session/events', params: { sessionId: st.sessionId } } : null),
      },
    ];
  },

  // E2 (DISC-01): turn-completion terminal signals. THE ONLY quota-spending flow:
  // exactly one tiny real turn ("Reply with exactly: OK"), manual tier, never
  // from CI. Subscribe first (E1 verdict: accepted), then a best-effort
  // session/setModel carrying the session's CURRENT default model (documents the
  // create-drops-reasoningLevel workaround with a live id; persistAsWorkspaceLastUsed
  // is forced false — no account-config mutation), then the send. Run with
  // --end-mode quiet-window --quiet-ms 2500: the run ends after 2.5 s without new
  // stdout lines once the send has fired (the field-verified fallback heuristic).
  // Reuses existing key material only; there is no key-minting endpoint call
  // anywhere in this file and no flow mints anything.
  'turn'(ctx) {
    const ws = ctx.workspaceDir;
    PROBED_METHODS[2] = 'session/create';
    PROBED_METHODS[3] = 'session/subscribe';
    PROBED_METHODS[4] = 'session/setModel';
    PROBED_METHODS[5] = 'session/send';
    return [
      { atMs: 1500, msg: { id: 1, method: 'provider/refreshRegistry', params: { reason: 'probe' } } },
      {
        atMs: 5000,
        msg: {
          id: 2,
          method: 'session/create',
          params: {
            workspace: { workspacePath: ws, workspaceKey: ws },
            mode: 'build',
            persistence: 'immediate',
          },
        },
      },
      {
        atMs: 8500,
        make: (st) =>
          st.sessionId
            ? { id: 3, method: 'session/subscribe', params: { sessionId: st.sessionId, deliveryKind: 'desktop-continuous' } }
            : null,
      },
      {
        atMs: 11000,
        make: (st) =>
          st.sessionId && st.createModel
            ? {
                id: 4,
                method: 'session/setModel',
                params: {
                  sessionId: st.sessionId,
                  model: { providerId: st.createModel.providerId, modelId: st.createModel.modelId },
                  persistAsWorkspaceLastUsed: false,
                },
              }
            : null,
      },
      {
        atMs: 13500,
        make: (st) =>
          st.sessionId
            ? { id: 5, method: 'session/send', params: { sessionId: st.sessionId, content: 'Reply with exactly: OK' } }
            : null,
      },
    ];
  },
};

// Experiment ids per flow (D-04: [V-EXP <id>] resolves to
// experiments/captures/<id>*). Flows added by plan 01-02 are pre-registered here
// so the seam is explicit.
const CAPTURE_IDS = {
  probe: 'EXP-PROBE',
  workspacekey: 'EXP-WORKSPACEKEY-01',
  subscribe: 'EXP-SUBSCRIBE-01', // flow lands in plan 01-02
  turn: 'EXP-TURN-01', // flow lands in plan 01-02
};

// --- args ------------------------------------------------------------------
function parseArgs(argv) {
  const out = {
    electron: null,
    cjs: null,
    builtin: null,
    flow: null,
    rawOut: null,
    windowMs: null,
    quietMs: 2500,
    endMode: 'fixed',
    workspaceDir: null,
    desktopVersion: 'unknown',
  };
  const required = ['electron', 'cjs', 'builtin', 'flow', 'rawOut', 'windowMs'];
  const map = {
    '--electron': 'electron',
    '--cjs': 'cjs',
    '--builtin': 'builtin',
    '--flow': 'flow',
    '--raw-out': 'rawOut',
    '--window-ms': 'windowMs',
    '--quiet-ms': 'quietMs',
    '--end-mode': 'endMode',
    '--workspace-dir': 'workspaceDir',
    '--desktop-version': 'desktopVersion',
  };
  for (let i = 0; i < argv.length; i++) {
    const key = map[argv[i]];
    if (!key || i + 1 >= argv.length) {
      process.stderr.write(`app-server-probe: missing or unknown flag: ${argv[i]}\n`);
      return null;
    }
    out[key] = argv[++i];
  }
  for (const k of required) {
    if (out[k] == null) {
      process.stderr.write(`app-server-probe: missing required flag --${k.replace(/[A-Z]/g, (c) => '-' + c.toLowerCase())}\n`);
      return null;
    }
  }
  out.windowMs = Number(out.windowMs);
  out.quietMs = Number(out.quietMs);
  if (!Number.isFinite(out.windowMs) || out.windowMs <= 0) {
    process.stderr.write('app-server-probe: --window-ms must be a positive number\n');
    return null;
  }
  if (!Number.isFinite(out.quietMs) || out.quietMs <= 0) {
    process.stderr.write('app-server-probe: --quiet-ms must be a positive number\n');
    return null;
  }
  if (out.endMode !== 'fixed' && out.endMode !== 'quiet-window') {
    process.stderr.write('app-server-probe: --end-mode must be "fixed" or "quiet-window"\n');
    return null;
  }
  // Header honesty (WR-04): an unset or empty value stays "unknown" — the
  // capture header never asserts a desktop version the probe did not read.
  if (!out.desktopVersion) out.desktopVersion = 'unknown';
  if (out.workspaceDir == null && out.rawOut) {
    // Default: the scratch directory holding the raw capture (guarded to be
    // outside the repo below).
    out.workspaceDir = dirname(path.resolve(out.rawOut));
  }
  return out;
}

function usage() {
  process.stderr.write(
    'usage: node experiments/app-server-probe.mjs --electron <bundle-root>/zcode \\\n' +
      '  --cjs <bundle-root>/resources/glm/zcode.cjs --builtin <builtin-config-file> \\\n' +
      '  --flow <probe|workspacekey|subscribe|turn> --raw-out <scratch-path-outside-repo> \\\n' +
      '  --window-ms 30000 [--quiet-ms 2500] [--end-mode fixed|quiet-window] [--workspace-dir <dir>] \\\n' +
      '  [--desktop-version <value>]\n',
  );
}

const flags = parseArgs(process.argv.slice(2));
if (!flags) {
  usage();
  process.exit(1);
}

// --- scrub-first guard: raw captures are born OUTSIDE the repo ----------------
const rawOut = path.resolve(flags.rawOut);
if (rawOut === REPO_ROOT || rawOut.startsWith(REPO_ROOT + path.sep)) {
  process.stderr.write(
    'app-server-probe: refusing --raw-out inside the repository — raw captures are\n' +
      'born outside the repo (D-02); point it at a scratch directory and scrub into\n' +
      'experiments/captures/ with experiments/scrub-capture.mjs.\n',
  );
  process.exit(1);
}

if (!Object.hasOwn(FLOWS, flags.flow)) {
  const pending = Object.keys(CAPTURE_IDS).filter((k) => !Object.hasOwn(FLOWS, k));
  process.stderr.write(
    `app-server-probe: unknown flow "${flags.flow}" (known: ${Object.keys(FLOWS).join(', ')}` +
      `${pending.length ? `; not implemented yet: ${pending.join(', ')}` : ''})\n`,
  );
  process.exit(1);
}

// --- WR-02: the probe knows its own steps — never accept a window that expires
// --- before the flow's last (load-bearing) step fires. At the documented 12 s
// --- example the window backstop won the race for both 01-02 flows (last steps
// --- at 13000/13500 ms), the late send() then wrote into an already-ended
// --- stdin, and the run completed exit 0 with its final probe never delivered.
// --- Validated BEFORE the spawn so a rejected window never starts the desktop
// --- app-server (and never writes a capture header).
const steps = FLOWS[flags.flow]({
  workspaceDir: flags.workspaceDir,
  windowMs: flags.windowMs,
  quietMs: flags.quietMs,
});
const lastAtMs = Math.max(...steps.map((s) => s.atMs));
if (flags.windowMs <= lastAtMs) {
  process.stderr.write(
    `app-server-probe: --window-ms ${flags.windowMs} expires before the "${flags.flow}" flow's ` +
      `last step (${lastAtMs} ms) — the final probe would be swallowed by the window backstop; raise the window\n`,
  );
  process.exit(1);
}

// --- capture header (builtinRevision + desktopVersion read AT CAPTURE TIME) ---
let builtinRevision = 'unknown';
try {
  const parsed = JSON.parse(readFileSync(flags.builtin, 'utf8'));
  if (parsed && typeof parsed.revision !== 'undefined') builtinRevision = String(parsed.revision);
} catch {
  // recorded as "unknown" — the header still dates and anchors the capture
}

const captureId = CAPTURE_IDS[flags.flow];
writeFileSync(
  rawOut,
  JSON.stringify({
    capture: captureId,
    date: new Date().toISOString(),
    desktopVersion: flags.desktopVersion,
    builtinRevision,
    host: os.release(),
  }) + '\n',
);

// --- spawn (own process group so the backstop can kill the whole tree) --------
const child = spawn(flags.electron, [flags.cjs, 'app-server', '--stdio'], {
  env: {
    ...process.env,
    ELECTRON_RUN_AS_NODE: '1',
    ZCODE_BUILTIN_PROVIDER_CONFIG_FILE: flags.builtin,
  },
  stdio: ['pipe', 'pipe', 'pipe'],
  detached: true,
});

child.stdin.on('error', () => {}); // EPIPE after the child exits is expected

let capturedLines = 0;
let stdoutBuf = '';
child.stdout.setEncoding('utf8');
child.stdout.on('data', (chunk) => {
  stdoutBuf += chunk;
  let idx;
  while ((idx = stdoutBuf.indexOf('\n')) >= 0) {
    const line = stdoutBuf.slice(0, idx);
    stdoutBuf = stdoutBuf.slice(idx + 1);
    appendFileSync(rawOut, JSON.stringify({ ts: new Date().toISOString(), line }) + '\n');
    capturedLines++;
    if (quietArmed) resetQuietTimer(); // quiet-window end mode: every line postpones the end
    // Answer server->host REQUESTS (method + id); notifications (no id) need no
    // reply. Every reply is recorded in the capture as a hostSent line.
    // IN-04: a request is never fed to observeReply (see the guard inside it) —
    // only method-less frames are candidate replies.
    try {
      const wire = JSON.parse(line);
      if (wire && typeof wire.method === 'string' && wire.id != null) {
        const reply = { id: wire.id, ...hostReplyFor(wire.method) };
        send(reply);
        appendFileSync(rawOut, JSON.stringify({ ts: new Date().toISOString(), hostSent: reply }) + '\n');
      } else {
        observeReply(wire);
        observeNotification(wire);
      }
    } catch {
      // non-JSON stdout line — keep it in the capture, no reply possible
    }
  }
});

let stderrCapped = 0;
child.stderr.setEncoding('utf8');
child.stderr.on('data', (chunk) => {
  if (stderrCapped < STDERR_CAP_BYTES) {
    stderrCapped += chunk.length;
    process.stderr.write(chunk.slice(0, Math.max(0, STDERR_CAP_BYTES - (stderrCapped - chunk.length))));
  }
});

const exited = new Promise((resolve) => {
  child.on('exit', (code, sig) => resolve({ code, sig }));
  child.on('error', (err) => resolve({ spawnError: err.message }));
});

function send(obj) {
  child.stdin.write(JSON.stringify(obj) + '\n');
}

function killGroup() {
  try {
    process.kill(-child.pid, 'SIGKILL');
  } catch {
    // already gone — fine
  }
}

// --- quiet-window end mode (E2) ------------------------------------------------
// In quiet-window mode the run ends --quiet-ms after the LAST stdout line, once
// every flow step has fired — the field-verified fallback heuristic for "the turn
// is over" when no dedicated terminal envelope is trusted. Two live-verified gates
// hold the close until the turn can actually be over (both corrections came from
// real runs that otherwise cancelled the turn mid-generation):
// 1. latency floor — the close is held until QUIET_SEND_FLOOR_MULTIPLE x quiet-ms
//    after the session/send reply (the send->first-output gap is request latency,
//    not silence between parts);
// 2. settle gate — the close is held until a state.updated patch reports a status
//    other than "running" AFTER the send (the session-status patch channel is
//    observed, not assumed to be the terminal signal itself — the capture records
//    everything around it, which is exactly the E2 evidence).
// windowMs stays as the absolute backstop. In fixed mode (default) the run always
// lasts windowMs.
const QUIET_SEND_FLOOR_MULTIPLE = 3;
let quietTimer = null;
let quietArmed = false;
let finishRun = null;
const finished = new Promise((resolve) => {
  finishRun = resolve;
});
function closeQuiet() {
  // Gate semantics:
  // - send never answered -> no turn is running; close on quiet immediately;
  // - send answered but the latency floor has not elapsed -> hold (gate 1);
  // - status never left "running" -> HOLD (gate 2): a turn that never settles is
  //   collected until the windowMs backstop, never quietly cut short mid-generation;
  // - settled >= quietMs ago and past the floor -> the turn is over; close.
  if (state.sendReplyTs == null) {
    state.endReason = state.endReason || 'quiet-window';
    finishRun();
    return;
  }
  const floorOk = Date.now() - state.sendReplyTs >= flags.quietMs * QUIET_SEND_FLOOR_MULTIPLE;
  const settled = state.turnSettledTs != null && Date.now() - state.turnSettledTs >= flags.quietMs;
  if (!floorOk || !settled) {
    quietTimer = setTimeout(closeQuiet, flags.quietMs);
    return;
  }
  state.endReason = state.endReason || 'quiet-window';
  finishRun();
}
function resetQuietTimer() {
  if (quietTimer) clearTimeout(quietTimer);
  quietTimer = setTimeout(closeQuiet, flags.quietMs);
}
function armQuietWindow() {
  if (flags.endMode !== 'quiet-window' || quietArmed) return;
  quietArmed = true;
  resetQuietTimer();
}

// --- run the flow (steps computed and window-validated before the spawn) --------
const timers = steps.map((s, i) =>
  setTimeout(() => {
    // Dynamic steps are evaluated at fire time against the accumulated state;
    // a step whose prerequisite never arrived (e.g. no session id) sends
    // nothing and its probe id is recorded as no-reply in the verdicts.
    const msg = s.msg !== undefined ? s.msg : s.make ? s.make(state) : null;
    if (msg) send(msg);
    if (i === steps.length - 1) armQuietWindow();
  }, s.atMs),
);

const sleep = (ms) => new Promise((r) => setTimeout(r, ms));
let exitInfo = await Promise.race([
  exited.then((e) => {
    state.endReason = state.endReason || 'child-exit';
    return e;
  }),
  finished.then(() => null),
  sleep(flags.windowMs).then(() => {
    state.endReason = state.endReason || 'window-expired';
    return null;
  }),
]);

let killed = false;
if (!exitInfo) {
  // collection window over: record verdicts, clean shutdown, SIGKILL backstop
  appendVerdicts(rawOut, flags.flow);
  child.stdin.end();
  exitInfo = await Promise.race([exited, sleep(GRACE_MS).then(() => null)]);
  if (!exitInfo) {
    killGroup();
    killed = true;
    exitInfo = await exited;
  }
} else {
  appendVerdicts(rawOut, flags.flow);
  child.stdin.end();
}
if (quietTimer) clearTimeout(quietTimer);
for (const t of timers) clearTimeout(t);

if (exitInfo.spawnError) {
  process.stderr.write(`app-server-probe: spawn failed: ${exitInfo.spawnError}\n`);
  process.exit(1);
}

// WR-01: key the no-wire-truth guard on capturedLines alone — a child that
// crashes (exit 1) with zero captured lines is the same EOF-pitfall/no-wire-truth
// condition as a child that exits 0 with zero lines, and must fail the run too.
const earlyZero = capturedLines === 0;
process.stderr.write(
  `app-server-probe: flow=${flags.flow} capture=${captureId} exit=${exitInfo.code}` +
    ` sig=${exitInfo.sig || '-'} lines=${capturedLines} killed=${killed}` +
    ` end=${state.endReason || 'n/a'} raw=${rawOut}\n`,
);
if (earlyZero) {
  process.stderr.write(
    'app-server-probe: WARNING — zero captured lines is the stdin-EOF pitfall\n' +
      'signature (or a bad --electron/--cjs path, or a child that crashed on boot);\n' +
      'the run produced no wire truth.\n',
  );
  process.exit(1);
}
process.exit(0);
